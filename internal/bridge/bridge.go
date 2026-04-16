package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"corexs-designer/internal/parser"
)

// ── CSS block type ────────────────────────────────────────────────────────────

// CSSBlock represents one parsed CSS rule block
type CSSBlock struct {
	Selector string
	PropKeys []string // props in insertion order
	PropVals map[string]string
	IsRaw    bool
	Raw      string
}

// ── Breakpoints ───────────────────────────────────────────────────────────────

const BreakpointDesktop = "desktop"

var BreakpointOrder = []string{
	BreakpointDesktop,
	"1440", "1280", "1024", "768", "480", "375",
}

var BreakpointLabel = map[string]string{
	BreakpointDesktop: "Desktop (base)",
	"1440":            "1440px · Large Desktop",
	"1280":            "1280px · Standard",
	"1024":            "1024px · Small Desktop",
	"768":             "768px · Tablet",
	"480":             "480px · Mobile L",
	"375":             "375px · Mobile S",
}

// ── Types ─────────────────────────────────────────────────────────────────────

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// AnimChange holds animation data per element
type AnimChange struct {
	AnimID    string `json:"animId"`
	Keyframes string `json:"kf"`
	CSS       string `json:"css"`     // full animation CSS value
	Trigger   string `json:"trigger"` // loop | onenter | onclick
}

// ── Bridge ────────────────────────────────────────────────────────────────────

type Bridge struct {
	parseResult *parser.ParseResult

	// responsive changes: breakpoint → elemID → {css-prop: value}
	respChanges map[string]map[string]map[string]string

	// animation changes: elemID → AnimChange (not breakpoint-specific)
	animChanges map[string]*AnimChange

	// imported CSS: selector → declarations string
	cssImported    map[string]string
	cssImportedRaw string // raw text of imported CSS for ordered parsing

	// current active breakpoint
	activeBP string

	// Project state
	projectPath      string
	projectCreatedAt string
	htmlFilePath     string
	cssFilePath      string
	recentProjects   []string

	SendToJS   func(string)
	OnFileOpen func(filter string) string
	OnFileSave func(defaultName string, filter string) string
}

func New(
	sendToJS func(string),
	onFileOpen func(filter string) string,
	onFileSave func(defaultName string, filter string) string,
) *Bridge {
	b := &Bridge{
		respChanges: make(map[string]map[string]map[string]string),
		animChanges: make(map[string]*AnimChange),
		cssImported: make(map[string]string),
		activeBP:    BreakpointDesktop,
		SendToJS:    sendToJS,
		OnFileOpen:  onFileOpen,
		OnFileSave:  onFileSave,
	}
	for _, bp := range BreakpointOrder {
		b.respChanges[bp] = make(map[string]map[string]string)
	}
	return b
}

// ── Handle ────────────────────────────────────────────────────────────────────

func (b *Bridge) Handle(raw string) {
	var msg Message
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		b.sendError("JSON parse error: " + err.Error())
		return
	}
	switch msg.Type {
	case "open-html":
		b.handleOpenHTML()
	case "import-css":
		b.handleImportCSS()
	case "select-element":
		b.handleSelect(msg.Payload)
	case "style-change":
		b.handleStyleChange(msg.Payload)
	case "set-breakpoint":
		b.handleSetBreakpoint(msg.Payload)
	case "anim-change":
		b.handleAnimChange(msg.Payload)
	case "export-css":
		b.handleExportCSS()
	case "export-css-html":
		b.handleExportCSSHTML()
	case "text-change":
		b.handleTextChange(msg.Payload)
	case "get-tree":
		b.handleGetTree()
	case "new-project":
		b.handleNewProject()
	case "save-project":
		b.handleSaveProject(msg.Payload)
	case "save-project-as":
		b.handleSaveProjectAs()
	case "open-project":
		b.handleOpenProject()
	case "open-recent":
		b.handleOpenRecent(msg.Payload)
	case "get-recent":
		b.handleGetRecent()
	default:
		b.sendError("Unknown message: " + msg.Type)
	}
}

// ── Open HTML ─────────────────────────────────────────────────────────────────

func (b *Bridge) handleOpenHTML() {
	path := b.OnFileOpen("HTML Files\x00*.html;*.htm\x00All Files\x00*.*\x00\x00")
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		b.sendError("Cannot read file: " + err.Error())
		return
	}
	result, err := parser.ParseHTML(string(data))
	if err != nil {
		b.sendError("HTML parse error: " + err.Error())
		return
	}
	b.parseResult = result
	b.htmlFilePath = path
	for _, bp := range BreakpointOrder {
		b.respChanges[bp] = make(map[string]map[string]string)
	}
	b.animChanges = make(map[string]*AnimChange)
	b.cssImported = make(map[string]string)
	b.cssImportedRaw = ""
	b.activeBP = BreakpointDesktop

	b.send("load-html", map[string]interface{}{
		"html":     string(data),
		"filename": filepath.Base(path),
	})
	b.handleGetTree()
}

// ── Breakpoint ────────────────────────────────────────────────────────────────

func (b *Bridge) handleSetBreakpoint(payload json.RawMessage) {
	var p struct {
		BP string `json:"bp"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}
	b.activeBP = p.BP
	if _, ok := b.respChanges[b.activeBP]; !ok {
		b.respChanges[b.activeBP] = make(map[string]map[string]string)
	}

	// Build merged styles for ALL elements at the new breakpoint
	// so the iframe can re-render each element correctly.
	allStyles := map[string]string{} // elemID → merged CSS string
	if b.parseResult != nil {
		for id := range b.parseResult.Elements {
			merged := b.getMergedStyle(id)
			if len(merged) > 0 {
				allStyles[id] = propsMapToCSS(merged)
			}
		}
	}

	b.send("breakpoint-set", map[string]interface{}{
		"bp":        b.activeBP,
		"label":     BreakpointLabel[b.activeBP],
		"count":     b.totalChangeCount(),
		"allStyles": allStyles, // full re-render map
	})
}

// ── Import CSS ────────────────────────────────────────────────────────────────

func (b *Bridge) handleImportCSS() {
	path := b.OnFileOpen("CSS Files\x00*.css\x00All Files\x00*.*\x00\x00")
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		b.sendError("Cannot read CSS file: " + err.Error())
		return
	}
	cssText := string(data)
	b.cssFilePath = path
	b.cssImportedRaw = cssText // save raw for ordered export
	parsed := parseCSS(cssText)
	for sel, props := range parsed {
		b.cssImported[sel] = props
	}
	if b.parseResult != nil {
		b.applyImportedCSS(parsed)
	}
	b.send("css-imported", map[string]interface{}{
		"filename": filepath.Base(path),
		"rules":    len(parsed),
		"css":      cssText,
	})
}

func (b *Bridge) applyImportedCSS(parsed map[string]string) {
	for _, elem := range b.parseResult.Elements {
		selector := parser.GetSelector(elem)
		if props, ok := parsed[selector]; ok {
			if _, ok2 := b.respChanges[BreakpointDesktop][elem.UniqueKey]; !ok2 {
				b.respChanges[BreakpointDesktop][elem.UniqueKey] = make(map[string]string)
			}
			for _, decl := range strings.Split(props, ";") {
				decl = strings.TrimSpace(decl)
				if decl == "" {
					continue
				}
				kv := strings.SplitN(decl, ":", 2)
				if len(kv) != 2 {
					continue
				}
				k := strings.TrimSpace(kv[0])
				v := strings.TrimSpace(kv[1])
				parser.ApplyStyleProp(&elem.Style, k, v)
				b.respChanges[BreakpointDesktop][elem.UniqueKey][k] = v
			}
			b.send("apply-style", map[string]interface{}{
				"id":    elem.UniqueKey,
				"css":   parser.StyleToCSS(elem.Style),
				"count": b.totalChangeCount(),
			})
		}
	}
}

// ── Select element ────────────────────────────────────────────────────────────

func (b *Bridge) handleSelect(payload json.RawMessage) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}
	if b.parseResult == nil {
		return
	}
	elem, ok := b.parseResult.Elements[p.ID]
	if !ok {
		return
	}
	merged := b.getMergedStyle(p.ID)
	b.send("element-selected", map[string]interface{}{
		"id":        elem.UniqueKey,
		"tag":       elem.Tag,
		"elemId":    elem.ElemID,
		"classes":   elem.Classes,
		"innerText": elem.InnerText,
		"style":     merged,
		"baseStyle": elem.Style,
		"selector":  parser.GetSelector(elem),
		"activeBP":  b.activeBP,
	})
}

// getMergedStyle returns the effective props map for an element at the active BP.
//
// Cascade logic (matches how browsers apply @media max-width):
//   - Always start with desktop (base) styles
//   - Then apply overrides for breakpoints that are LARGER than or equal to activeBP
//     (because @media max-width:768px applies when screen ≤ 768px,
//     so if we are at 768px, the 768px rules apply, and so do 480px and 375px rules
//     since they also match a 768px-wide viewport... but we only apply the exact BP)
//   - Actually: apply ONLY the exact matching breakpoint override on top of desktop
//     This is what makes responsive editing predictable:
//     desktop base + only the selected BP's overrides
func (b *Bridge) getMergedStyle(elemID string) map[string]string {
	merged := make(map[string]string)

	// 1. Start with desktop base
	if props, ok := b.respChanges[BreakpointDesktop][elemID]; ok {
		for k, v := range props {
			merged[k] = v
		}
	}

	// 2. If we are at desktop, done
	if b.activeBP == BreakpointDesktop {
		return merged
	}

	// 3. Apply the active breakpoint's overrides on top
	//    This replaces desktop values with mobile-specific ones
	if props, ok := b.respChanges[b.activeBP][elemID]; ok {
		for k, v := range props {
			if v == "" {
				delete(merged, k) // empty string = remove the property
			} else {
				merged[k] = v
			}
		}
	}

	return merged
}

// ── Style change ──────────────────────────────────────────────────────────────

func (b *Bridge) handleStyleChange(payload json.RawMessage) {
	var p struct {
		ID    string `json:"id"`
		Prop  string `json:"prop"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}
	if b.parseResult == nil {
		return
	}
	elem, ok := b.parseResult.Elements[p.ID]
	if !ok {
		return
	}
	bp := b.activeBP
	if _, ok := b.respChanges[bp]; !ok {
		b.respChanges[bp] = make(map[string]map[string]string)
	}
	if _, ok := b.respChanges[bp][p.ID]; !ok {
		b.respChanges[bp][p.ID] = make(map[string]string)
	}
	if p.Value == "" {
		delete(b.respChanges[bp][p.ID], p.Prop)
	} else {
		b.respChanges[bp][p.ID][p.Prop] = p.Value
	}
	if bp == BreakpointDesktop {
		parser.ApplyStyleProp(&elem.Style, p.Prop, p.Value)
	}
	merged := b.getMergedStyle(p.ID)
	cssStr := propsMapToCSS(merged)
	b.send("apply-style", map[string]interface{}{
		"id":    p.ID,
		"css":   cssStr,
		"count": b.totalChangeCount(),
		"bp":    bp,
	})
}

// ── Animation change ──────────────────────────────────────────────────────────

func (b *Bridge) handleAnimChange(payload json.RawMessage) {
	var p struct {
		ID      string `json:"id"`
		AnimID  string `json:"animId"`
		KF      string `json:"kf"`
		CSS     string `json:"css"`
		Trigger string `json:"trigger"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}
	if p.AnimID == "" {
		delete(b.animChanges, p.ID)
	} else {
		b.animChanges[p.ID] = &AnimChange{
			AnimID:    p.AnimID,
			Keyframes: p.KF,
			CSS:       p.CSS,
			Trigger:   p.Trigger,
		}
	}
	// Confirm back
	b.send("anim-saved", map[string]interface{}{
		"id":    p.ID,
		"count": len(b.animChanges),
	})
}

// ── Text change ───────────────────────────────────────────────────────────────

func (b *Bridge) handleTextChange(payload json.RawMessage) {
	var p struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}
	if b.parseResult == nil {
		return
	}
	if elem, ok := b.parseResult.Elements[p.ID]; ok {
		elem.InnerText = p.Text
	}
}

// ── Tree ──────────────────────────────────────────────────────────────────────

func (b *Bridge) handleGetTree() {
	if b.parseResult == nil {
		return
	}
	b.send("tree", map[string]interface{}{
		"root": buildTree(b.parseResult.Root),
	})
}

// ── Export CSS ────────────────────────────────────────────────────────────────

func (b *Bridge) handleExportCSS() {
	if b.totalChangeCount() == 0 && len(b.cssImported) == 0 && len(b.animChanges) == 0 {
		b.send("export-result", map[string]interface{}{
			"ok":  false,
			"msg": "No changes to export.",
		})
		return
	}
	path := b.OnFileSave("styles.css", "CSS Files\x00*.css\x00All Files\x00*.*\x00\x00")
	if path == "" {
		return
	}
	css := b.buildCSS()
	if err := os.WriteFile(path, []byte(css), 0644); err != nil {
		b.send("export-result", map[string]interface{}{"ok": false, "msg": err.Error()})
		return
	}
	b.send("export-result", map[string]interface{}{
		"ok":    true,
		"path":  path,
		"count": b.totalChangeCount(),
		"mode":  "css",
	})
}

// ── Export CSS + HTML ─────────────────────────────────────────────────────────

func (b *Bridge) handleExportCSSHTML() {
	if b.parseResult == nil {
		b.send("export-result", map[string]interface{}{"ok": false, "msg": "No HTML loaded."})
		return
	}
	cssPath := b.OnFileSave("styles.css", "CSS Files\x00*.css\x00All Files\x00*.*\x00\x00")
	if cssPath == "" {
		return
	}
	if err := os.WriteFile(cssPath, []byte(b.buildCSS()), 0644); err != nil {
		b.send("export-result", map[string]interface{}{"ok": false, "msg": err.Error()})
		return
	}
	htmlPath := b.OnFileSave("index.html", "HTML Files\x00*.html\x00All Files\x00*.*\x00\x00")
	if htmlPath == "" {
		return
	}
	exportedHTML := injectCSSLink(b.parseResult.RawHTML, filepath.Base(cssPath))
	if err := os.WriteFile(htmlPath, []byte(exportedHTML), 0644); err != nil {
		b.send("export-result", map[string]interface{}{"ok": false, "msg": err.Error()})
		return
	}
	b.send("export-result", map[string]interface{}{
		"ok":       true,
		"path":     cssPath,
		"htmlPath": htmlPath,
		"count":    b.totalChangeCount(),
		"mode":     "css+html",
	})
}

// ── CSS Builder ───────────────────────────────────────────────────────────────
// Strategy:
//   1. Start with imported (original) CSS as the base
//   2. Parse it into a selector→{prop:val} map
//   3. Apply our session changes on top (overwrite changed props, add new selectors)
//   4. Rebuild keyframes + media queries
//   5. Output: the complete updated CSS file

func (b *Bridge) buildCSS() string {
	var sb strings.Builder
	sb.WriteString("/*\n")
	sb.WriteString(" * CorexS Designer — Generated CSS\n")
	sb.WriteString(fmt.Sprintf(" * %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(" */\n\n")

	// ── Step 1: Parse original CSS into structured form ──────────────────────
	// origBlocks preserves order: []CSSBlock{Selector, PropKeys, PropVals, IsRaw, Raw}

	var origBlocks []CSSBlock
	origSelectorIndex := map[string]int{} // selector → index in origBlocks

	if len(b.cssImported) > 0 {
		// We stored selector→propString in cssImported
		// But we need the original ORDER. Since parseCSS gives us a map (unordered),
		// we need to re-read the original raw CSS to preserve order.
		// cssImportedRaw holds the raw text.
		if b.cssImportedRaw != "" {
			origBlocks, origSelectorIndex = parseCSSOrdered(b.cssImportedRaw)
		} else {
			// Fallback: reconstruct from map (unordered)
			for sel, props := range b.cssImported {
				idx := len(origBlocks)
				origSelectorIndex[sel] = idx
				keys, vals := parsePropsOrdered(props)
				origBlocks = append(origBlocks, CSSBlock{
					Selector: sel,
					PropKeys: keys,
					PropVals: vals,
				})
			}
		}
	}

	// ── Step 2: Apply session changes to origBlocks ───────────────────────────
	// Build a selector→changedProps map from respChanges[desktop]
	selectorChanges := map[string]map[string]string{} // selector → {prop: val}

	if b.parseResult != nil {
		if bpMap, ok := b.respChanges[BreakpointDesktop]; ok {
			for id, props := range bpMap {
				if len(props) == 0 {
					continue
				}
				elem, ok := b.parseResult.Elements[id]
				if !ok {
					continue
				}
				sel := parser.GetSelector(elem)
				if _, exists := selectorChanges[sel]; !exists {
					selectorChanges[sel] = make(map[string]string)
				}
				for k, v := range props {
					selectorChanges[sel][k] = v
				}
			}
		}
	}

	// Apply changes to existing blocks
	for sel, changes := range selectorChanges {
		if idx, exists := origSelectorIndex[sel]; exists {
			blk := &origBlocks[idx]
			for k, v := range changes {
				if _, has := blk.PropVals[k]; has {
					// Update existing prop
					blk.PropVals[k] = v
				} else {
					// Add new prop
					blk.PropKeys = append(blk.PropKeys, k)
					blk.PropVals[k] = v
				}
			}
		} else {
			// New selector — add at end
			keys, vals := make([]string, 0), make(map[string]string)
			for k, v := range changes {
				keys = append(keys, k)
				vals[k] = v
			}
			sort.Strings(keys)
			origSelectorIndex[sel] = len(origBlocks)
			origBlocks = append(origBlocks, CSSBlock{
				Selector: sel,
				PropKeys: keys,
				PropVals: vals,
			})
		}
	}

	// ── Step 3: Animation changes ──────────────────────────────────────────────
	// Collect animation selectors from animChanges
	type animEntry struct {
		sel  string
		anim *AnimChange
	}
	var animEntries []animEntry
	if b.parseResult != nil {
		for id, ac := range b.animChanges {
			if elem, ok := b.parseResult.Elements[id]; ok {
				animEntries = append(animEntries, animEntry{parser.GetSelector(elem), ac})
			}
		}
	}

	// ── Step 4: Write keyframes ────────────────────────────────────────────────
	// Collect from both original (raw blocks) and new animations
	// New animation keyframes go at top
	newKFs := make(map[string]bool)
	for _, ae := range animEntries {
		if ae.anim.Keyframes != "" {
			newKFs[ae.anim.Keyframes] = true
		}
	}
	if len(newKFs) > 0 {
		sb.WriteString("/* ── Animation Keyframes ── */\n")
		for kf := range newKFs {
			sb.WriteString(kf + "\n")
		}
		sb.WriteString("\n")
	}

	// ── Step 5: Write all blocks in original order ─────────────────────────────
	for _, blk := range origBlocks {
		if blk.IsRaw {
			// @keyframes, @media, etc — write as-is
			sb.WriteString(blk.Raw)
			if !strings.HasSuffix(strings.TrimSpace(blk.Raw), "\n") {
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
			continue
		}
		if len(blk.PropKeys) == 0 {
			continue
		}
		sb.WriteString(blk.Selector + " {\n")
		for _, k := range blk.PropKeys {
			v := blk.PropVals[k]
			if v != "" {
				sb.WriteString("  " + k + ": " + v + ";\n")
			}
		}
		sb.WriteString("}\n\n")
	}

	// ── Step 6: Write animation-only entries (no original selector) ────────────
	for _, ae := range animEntries {
		// Check if already handled via selectorChanges (which would be in origBlocks)
		alreadyWritten := false
		for _, blk := range origBlocks {
			if blk.Selector == ae.sel {
				alreadyWritten = true
				break
			}
		}
		if !alreadyWritten {
			sb.WriteString(ae.sel + " {\n")
			writeAnimCSS(&sb, ae.anim, "  ")
			sb.WriteString("}\n\n")
		}
	}
	// For selectors that are in origBlocks AND have animation, inject animation
	// into their block output
	// NOTE: above loop already wrote them. We need to handle this differently:
	// Inject anim into origBlocks before writing.
	// Actually, let's do a second pass: add animation to origBlocks entries.
	// Reset and rewrite.
	// Easier: do it in the write loop above. Let's rebuild properly.

	// ── Actually: rebuild cleanly injecting anim into blocks ──────────────────
	// Reset sb and redo
	sb.Reset()
	sb.WriteString("/*\n")
	sb.WriteString(" * CorexS Designer — Generated CSS\n")
	sb.WriteString(fmt.Sprintf(" * %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(" */\n\n")

	// Keyframes first
	if len(newKFs) > 0 {
		sb.WriteString("/* ── Animation Keyframes ── */\n")
		for kf := range newKFs {
			sb.WriteString(kf + "\n")
		}
		sb.WriteString("\n")
	}

	// Build anim lookup: selector → *AnimChange
	animBySel := map[string]*AnimChange{}
	for _, ae := range animEntries {
		animBySel[ae.sel] = ae.anim
	}

	writtenSels := map[string]bool{}

	for _, blk := range origBlocks {
		if blk.IsRaw {
			sb.WriteString(blk.Raw)
			if !strings.HasSuffix(strings.TrimSpace(blk.Raw), "\n") {
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
			continue
		}
		anim := animBySel[blk.Selector]
		if len(blk.PropKeys) == 0 && anim == nil {
			continue
		}
		writtenSels[blk.Selector] = true
		sb.WriteString(blk.Selector + " {\n")
		for _, k := range blk.PropKeys {
			v := blk.PropVals[k]
			if v != "" {
				sb.WriteString("  " + k + ": " + v + ";\n")
			}
		}
		if anim != nil {
			writeAnimCSS(&sb, anim, "  ")
		}
		sb.WriteString("}\n\n")
	}

	// New selectors not in original
	for sel, changes := range selectorChanges {
		if writtenSels[sel] {
			continue
		}
		sb.WriteString(sel + " {\n")
		keys := make([]string, 0, len(changes))
		for k := range changes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if v := changes[k]; v != "" {
				sb.WriteString("  " + k + ": " + v + ";\n")
			}
		}
		if anim := animBySel[sel]; anim != nil {
			writeAnimCSS(&sb, anim, "  ")
		}
		writtenSels[sel] = true
		sb.WriteString("}\n\n")
	}

	// Animation-only new selectors
	for sel, anim := range animBySel {
		if writtenSels[sel] {
			continue
		}
		sb.WriteString(sel + " {\n")
		writeAnimCSS(&sb, anim, "  ")
		sb.WriteString("}\n\n")
	}

	// ── Responsive overrides (from session, not in original) ──────────────────
	for _, bp := range BreakpointOrder {
		if bp == BreakpointDesktop {
			continue
		}
		bpMap, ok := b.respChanges[bp]
		if !ok || len(bpMap) == 0 {
			continue
		}
		label := BreakpointLabel[bp]
		sb.WriteString(fmt.Sprintf("/* ── %s ── */\n", label))
		sb.WriteString(fmt.Sprintf("@media (max-width: %spx) {\n\n", bp))
		b.writeBreakpointRules(&sb, bp, bpMap)
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

// writeAnimCSS writes animation properties into a CSS rule block
func writeAnimCSS(sb *strings.Builder, ac *AnimChange, indent string) {
	lines := strings.Split(ac.CSS, ";")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		sb.WriteString(indent + line + ";\n")
	}
	sb.WriteString(indent + "/* animation trigger: " + ac.Trigger + " */\n")
}

func (b *Bridge) writeBreakpointRules(sb *strings.Builder, bp string, bpMap map[string]map[string]string) {
	type entry struct {
		sel   string
		props map[string]string
		text  string
	}
	var entries []entry
	for id, props := range bpMap {
		if len(props) == 0 {
			continue
		}
		sel := id
		text := ""
		if b.parseResult != nil {
			if elem, ok := b.parseResult.Elements[id]; ok {
				sel = parser.GetSelector(elem)
				text = elem.InnerText
			}
		}
		entries = append(entries, entry{sel, props, text})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].sel < entries[j].sel })

	indent := "  "

	for _, e := range entries {
		if e.text != "" {
			t := e.text
			if len(t) > 50 {
				t = t[:50] + "..."
			}
			sb.WriteString(indent + fmt.Sprintf("/* \"%s\" */\n", t))
		}
		sb.WriteString(indent + e.sel + " {\n")
		var keys []string
		for k := range e.props {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if v := e.props[k]; v != "" {
				sb.WriteString(indent + "  " + k + ": " + v + ";\n")
			}
		}
		sb.WriteString(indent + "}\n\n")
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (b *Bridge) totalChangeCount() int {
	total := 0
	for _, bpMap := range b.respChanges {
		total += len(bpMap)
	}
	return total + len(b.animChanges)
}

func (b *Bridge) send(t string, payload interface{}) {
	data, err := json.Marshal(map[string]interface{}{"type": t, "payload": payload})
	if err != nil {
		return
	}
	b.SendToJS(fmt.Sprintf("window.__corexsReceive(%s)", string(data)))
}

func (b *Bridge) sendError(msg string) {
	b.send("error", map[string]interface{}{"msg": msg})
}

func parsePx(bp string) int {
	n := 0
	for _, c := range bp {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func propsMapToCSS(props map[string]string) string {
	var parts []string
	for k, v := range props {
		if k != "" && v != "" {
			parts = append(parts, k+":"+v)
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}

// parseCSSOrdered parses a CSS file preserving block order.
// Returns ordered blocks and a selector→index map for fast lookup.
// @keyframes, @media etc are stored as raw blocks.
func parseCSSOrdered(css string) ([]CSSBlock, map[string]int) {
	var blocks []CSSBlock
	idx := map[string]int{}

	i := 0
	n := len(css)

	for i < n {
		// Skip whitespace and comments
		i = skipWS(css, i)
		if i >= n {
			break
		}

		// Comment
		if i+1 < n && css[i] == '/' && css[i+1] == '*' {
			end := strings.Index(css[i+2:], "*/")
			if end < 0 {
				break
			}
			i = i + 2 + end + 2
			continue
		}

		// @rule
		if css[i] == '@' {
			// Find matching brace, handling nested braces
			braceStart := strings.Index(css[i:], "{")
			if braceStart < 0 {
				// No brace — e.g. @import; just find semicolon
				semi := strings.Index(css[i:], ";")
				if semi < 0 {
					break
				}
				rawText := css[i : i+semi+1]
				blocks = append(blocks, CSSBlock{IsRaw: true, Raw: rawText})
				i += semi + 1
				continue
			}
			// Find matching closing brace (may be nested for @media)
			bStart := i + braceStart
			depth := 0
			j := bStart
			for j < n {
				if css[j] == '{' {
					depth++
				} else if css[j] == '}' {
					depth--
					if depth == 0 {
						break
					}
				}
				j++
			}
			rawText := css[i : j+1]
			blocks = append(blocks, CSSBlock{IsRaw: true, Raw: rawText})
			i = j + 1
			continue
		}

		// Regular rule: find selector and block
		bracePos := strings.Index(css[i:], "{")
		if bracePos < 0 {
			break
		}
		sel := strings.TrimSpace(css[i : i+bracePos])
		closePos := strings.Index(css[i+bracePos:], "}")
		if closePos < 0 {
			break
		}
		body := css[i+bracePos+1 : i+bracePos+closePos]

		keys, vals := parsePropsOrdered(body)

		if sel != "" {
			bidx := len(blocks)
			idx[sel] = bidx
			blocks = append(blocks, CSSBlock{
				Selector: sel,
				PropKeys: keys,
				PropVals: vals,
			})
		}
		i = i + bracePos + closePos + 1
	}

	return blocks, idx
}

func skipWS(s string, i int) int {
	for i < len(s) {
		c := s[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		i++
	}
	return i
}

// parsePropsOrdered parses "prop: val; prop2: val2" preserving insertion order.
func parsePropsOrdered(body string) ([]string, map[string]string) {
	var keys []string
	vals := map[string]string{}
	for _, line := range strings.Split(body, ";") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if k == "" {
			continue
		}
		if _, exists := vals[k]; !exists {
			keys = append(keys, k)
		}
		vals[k] = v
	}
	return keys, vals
}

func parseCSS(css string) map[string]string {
	result := make(map[string]string)
	for {
		start := strings.Index(css, "/*")
		if start < 0 {
			break
		}
		end := strings.Index(css[start:], "*/")
		if end < 0 {
			break
		}
		css = css[:start] + css[start+end+2:]
	}
	for {
		openBrace := strings.Index(css, "{")
		if openBrace < 0 {
			break
		}
		closeBrace := strings.Index(css[openBrace:], "}")
		if closeBrace < 0 {
			break
		}
		selector := strings.TrimSpace(css[:openBrace])
		body := strings.TrimSpace(css[openBrace+1 : openBrace+closeBrace])
		if selector != "" && body != "" {
			for _, sel := range strings.Split(selector, ",") {
				sel = strings.TrimSpace(sel)
				if sel != "" {
					if existing, ok := result[sel]; ok {
						result[sel] = existing + ";" + body
					} else {
						result[sel] = body
					}
				}
			}
		}
		css = css[openBrace+closeBrace+1:]
	}
	return result
}

func injectCSSLink(htmlContent, cssFilename string) string {
	linkTag := `<link rel="stylesheet" href="` + cssFilename + `">`
	if idx := strings.Index(strings.ToLower(htmlContent), "</head>"); idx >= 0 {
		return htmlContent[:idx] + "  " + linkTag + "\n" + htmlContent[idx:]
	}
	return linkTag + "\n" + htmlContent
}

// ── Tree builder ──────────────────────────────────────────────────────────────

type TreeNode struct {
	ID       string      `json:"id"`
	Tag      string      `json:"tag"`
	ElemID   string      `json:"elemId"`
	Classes  []string    `json:"classes"`
	Text     string      `json:"text"`
	Children []*TreeNode `json:"children"`
}

func buildTree(elem *parser.HTMLElement) *TreeNode {
	if elem == nil {
		return nil
	}
	skip := map[string]bool{
		"script": true, "style": true, "meta": true,
		"link": true, "head": true, "title": true,
		"base": true, "noscript": true,
	}
	node := &TreeNode{
		ID: elem.UniqueKey, Tag: elem.Tag,
		ElemID: elem.ElemID, Classes: elem.Classes,
		Text: trunc(elem.InnerText, 30),
	}
	for _, child := range elem.Children {
		if skip[child.Tag] {
			continue
		}
		node.Children = append(node.Children, buildTree(child))
	}
	return node
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
