package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"corexs-designer/internal/parser"
)

// ProjectFile is the .corexsd JSON structure
type ProjectFile struct {
	Version     string                                      `json:"version"`
	CreatedAt   string                                      `json:"createdAt"`
	SavedAt     string                                      `json:"savedAt"`
	HTMLPath    string                                      `json:"htmlPath"`
	CSSPath     string                                      `json:"cssPath"`
	HTMLContent string                                      `json:"htmlContent"`
	Changes     map[string]map[string]map[string]string     `json:"changes"`
	AnimChanges map[string]*AnimChange                      `json:"animChanges"`
	ActiveBP    string                                      `json:"activeBP"`
}

const ProjectVersion = "2026.1"

// handleSaveProject — called from bridge Handle
func (b *Bridge) handleSaveProject(payload json.RawMessage) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		b.sendError("Save project: " + err.Error())
		return
	}
	if p.Path == "" {
		// Ask for path via save dialog
		p.Path = b.OnFileSave("project.corexsd",
			"CorexS Project\x00*.corexsd\x00All Files\x00*.*\x00\x00")
		if p.Path == "" {
			return
		}
	}
	if err := b.saveProject(p.Path); err != nil {
		b.sendError("Save failed: " + err.Error())
		return
	}
	b.projectPath = p.Path
	b.send("project-saved", map[string]interface{}{
		"path": p.Path,
		"name": filepath.Base(p.Path),
	})
	// Add to recent
	b.addRecent(p.Path)
}

func (b *Bridge) handleSaveProjectAs() {
	path := b.OnFileSave("project.corexsd",
		"CorexS Project\x00*.corexsd\x00All Files\x00*.*\x00\x00")
	if path == "" {
		return
	}
	if err := b.saveProject(path); err != nil {
		b.sendError("Save failed: " + err.Error())
		return
	}
	b.projectPath = path
	b.send("project-saved", map[string]interface{}{
		"path": path,
		"name": filepath.Base(path),
	})
	b.addRecent(path)
}

func (b *Bridge) handleOpenProject() {
	path := b.OnFileOpen("CorexS Project\x00*.corexsd\x00All Files\x00*.*\x00\x00")
	if path == "" {
		return
	}
	b.loadProject(path)
}

func (b *Bridge) handleOpenRecent(payload json.RawMessage) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return
	}
	b.loadProject(p.Path)
}

func (b *Bridge) handleNewProject() {
	b.parseResult = nil
	for _, bp := range BreakpointOrder {
		b.respChanges[bp] = make(map[string]map[string]string)
	}
	b.animChanges = make(map[string]*AnimChange)
	b.cssImported = make(map[string]string)
	b.cssImportedRaw = ""
	b.activeBP = BreakpointDesktop
	b.projectPath = ""
	b.htmlFilePath = ""
	b.cssFilePath = ""
	b.send("project-new", map[string]interface{}{})
}

// ── Core save/load ────────────────────────────────────────────────────────────

func (b *Bridge) saveProject(path string) error {
	htmlContent := ""
	if b.parseResult != nil {
		htmlContent = b.parseResult.RawHTML
	}
	proj := ProjectFile{
		Version:     ProjectVersion,
		CreatedAt:   b.projectCreatedAt,
		SavedAt:     time.Now().Format("2006-01-02 15:04:05"),
		HTMLPath:    b.htmlFilePath,
		CSSPath:     b.cssFilePath,
		HTMLContent: htmlContent,
		Changes:     b.respChanges,
		AnimChanges: b.animChanges,
		ActiveBP:    b.activeBP,
	}
	if proj.CreatedAt == "" {
		proj.CreatedAt = proj.SavedAt
	}
	data, err := json.MarshalIndent(proj, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (b *Bridge) loadProject(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		b.sendError("Cannot open project: " + err.Error())
		return
	}
	var proj ProjectFile
	if err := json.Unmarshal(data, &proj); err != nil {
		b.sendError("Invalid project file: " + err.Error())
		return
	}

	// Restore HTML
	htmlContent := proj.HTMLContent
	if htmlContent == "" && proj.HTMLPath != "" {
		if d, err := os.ReadFile(proj.HTMLPath); err == nil {
			htmlContent = string(d)
		}
	}
	if htmlContent != "" {
		result, err := parser.ParseHTML(htmlContent)
		if err != nil {
			b.sendError("HTML parse error: " + err.Error())
			return
		}
		b.parseResult = result
	}

	// Restore CSS — auto-import from cssPath
	b.cssImported = make(map[string]string)
	b.cssImportedRaw = ""
	var restoredCSS string
	var restoredCSSName string
	if proj.CSSPath != "" {
		if d, err := os.ReadFile(proj.CSSPath); err == nil {
			b.cssImportedRaw = string(d)
			b.cssImported = parseCSS(b.cssImportedRaw)
			b.cssFilePath = proj.CSSPath
			restoredCSS = string(d)
			restoredCSSName = filepath.Base(proj.CSSPath)
		}
	}

	// Restore changes
	b.respChanges = make(map[string]map[string]map[string]string)
	for _, bp := range BreakpointOrder {
		b.respChanges[bp] = make(map[string]map[string]string)
	}
	for bp, bpMap := range proj.Changes {
		if _, ok := b.respChanges[bp]; !ok {
			b.respChanges[bp] = make(map[string]map[string]string)
		}
		for id, props := range bpMap {
			b.respChanges[bp][id] = props
		}
	}

	b.animChanges = proj.AnimChanges
	if b.animChanges == nil {
		b.animChanges = make(map[string]*AnimChange)
	}
	b.activeBP = proj.ActiveBP
	if b.activeBP == "" {
		b.activeBP = BreakpointDesktop
	}
	b.htmlFilePath = proj.HTMLPath
	b.cssFilePath = proj.CSSPath
	b.projectPath = path
	b.projectCreatedAt = proj.CreatedAt
	b.addRecent(path)

	// Build applied styles map for JS restore: elemID → cssString
	appliedStyles := map[string]string{}
	if b.parseResult != nil {
		if bpMap, ok := b.respChanges[BreakpointDesktop]; ok {
			for id, props := range bpMap {
				if len(props) > 0 {
					appliedStyles[id] = propsMapToCSS(props)
				}
			}
		}
	}

	// Send full reload to JS
	b.send("project-loaded", map[string]interface{}{
		"path":         path,
		"name":         filepath.Base(path),
		"hasHTML":      htmlContent != "",
		"html":         htmlContent,
		"bp":           b.activeBP,
		"changes":      b.totalChangeCount(),
		"animChanges":   b.animChanges,   // so JS can restore elemAnimState
		"appliedStyles": appliedStyles,  // so JS can re-apply styles to iframe
		"cssText":       restoredCSS,    // auto-inject into iframe
		"cssName":       restoredCSSName,
	})
	if b.parseResult != nil {
		b.handleGetTree()
	}
}

// ── Recent projects ───────────────────────────────────────────────────────────

func (b *Bridge) addRecent(path string) {
	// Remove if already exists
	newList := []string{path}
	for _, p := range b.recentProjects {
		if p != path {
			newList = append(newList, p)
		}
	}
	if len(newList) > 10 {
		newList = newList[:10]
	}
	b.recentProjects = newList
	b.send("recent-updated", map[string]interface{}{
		"recent": b.recentProjects,
	})
}

func (b *Bridge) handleGetRecent() {
	b.send("recent-updated", map[string]interface{}{
		"recent": b.recentProjects,
	})
}
