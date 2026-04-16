package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"corexs-designer/internal/parser"
)

// TreePanel sol paneldə HTML element ağacını göstərir
type TreePanel struct {
	container  *fyne.Container
	tree       *widget.Tree
	elements   map[string]*parser.HTMLElement
	childMap   map[string][]string
	onSelected func(*parser.HTMLElement)
}

func NewTreePanel(onSelected func(*parser.HTMLElement)) *TreePanel {
	t := &TreePanel{
		onSelected: onSelected,
		elements:   make(map[string]*parser.HTMLElement),
		childMap:   make(map[string][]string),
	}
	t.build()
	return t
}

func (t *TreePanel) build() {
	t.tree = &widget.Tree{
		ChildUIDs: func(uid string) []string {
			return t.childMap[uid]
		},
		IsBranch: func(uid string) bool {
			children, ok := t.childMap[uid]
			return ok && len(children) > 0
		},
		CreateNode: func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("element")
		},
		UpdateNode: func(uid string, branch bool, node fyne.CanvasObject) {
			label := node.(*widget.Label)
			elem, ok := t.elements[uid]
			if !ok {
				if uid == "root" {
					label.SetText("🌐 Document")
				}
				return
			}

			display := buildDisplayName(elem)
			label.SetText(display)
		},
		OnSelected: func(uid string) {
			elem, ok := t.elements[uid]
			if ok && t.onSelected != nil {
				t.onSelected(elem)
			}
		},
	}

	header := widget.NewLabelWithStyle("🌲  Element Ağacı", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	t.container = container.NewBorder(
		container.NewVBox(header, widget.NewSeparator()),
		nil, nil, nil,
		t.tree,
	)
}

// LoadElements parse edilmiş elementləri yükləyir
func (t *TreePanel) LoadElements(root *parser.HTMLElement) {
	t.elements = make(map[string]*parser.HTMLElement)
	t.childMap = make(map[string][]string)

	// Root node
	t.childMap[""] = []string{"root"}

	// Root-un uşaqlarını əlavə et
	t.buildTree("root", root)

	t.tree.Refresh()
}

func (t *TreePanel) buildTree(parentUID string, elem *parser.HTMLElement) {
	for _, child := range elem.Children {
		// Uşaqları görünüşdən gizlət (script, style, meta, head vs)
		if isHiddenTag(child.Tag) {
			continue
		}

		uid := child.UniqueKey
		t.elements[uid] = child
		t.childMap[parentUID] = append(t.childMap[parentUID], uid)

		// Rekursiv uşaqları əlavə et
		if len(child.Children) > 0 {
			t.buildTree(uid, child)
		}
	}
}

func (t *TreePanel) Container() *fyne.Container {
	return t.container
}

// buildDisplayName elementi üçün oxunaqlı ad yaradır
func buildDisplayName(elem *parser.HTMLElement) string {
	icon := getTagIcon(elem.Tag)
	name := icon + " <" + elem.Tag + ">"

	if elem.ID != "" {
		name += " #" + elem.ID
	} else if len(elem.Classes) > 0 {
		name += " ." + elem.Classes[0]
		if len(elem.Classes) > 1 {
			name += "..."
		}
	}

	if elem.InnerText != "" {
		text := strings.TrimSpace(elem.InnerText)
		if len(text) > 20 {
			text = text[:20] + "..."
		}
		name += "  \"" + text + "\""
	}

	return name
}

func getTagIcon(tag string) string {
	icons := map[string]string{
		"div":     "📦",
		"section": "📑",
		"article": "📄",
		"header":  "🔝",
		"footer":  "🔚",
		"nav":     "🧭",
		"main":    "🏠",
		"aside":   "↔",
		"h1":      "H1",
		"h2":      "H2",
		"h3":      "H3",
		"h4":      "H4",
		"h5":      "H5",
		"h6":      "H6",
		"p":       "¶",
		"span":    "✦",
		"a":       "🔗",
		"img":     "🖼",
		"ul":      "•",
		"ol":      "1.",
		"li":      "—",
		"table":   "⊞",
		"button":  "🔘",
		"input":   "✏",
		"form":    "📋",
		"label":   "🏷",
		"select":  "▼",
		"video":   "▶",
		"audio":   "🎵",
	}
	if icon, ok := icons[tag]; ok {
		return icon
	}
	return "◇"
}

func isHiddenTag(tag string) bool {
	hidden := map[string]bool{
		"script": true,
		"style":  true,
		"meta":   true,
		"link":   true,
		"head":   true,
		"title":  true,
		"base":   true,
		"noscript": true,
	}
	return hidden[tag]
}
