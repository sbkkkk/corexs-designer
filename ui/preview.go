package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"corexs-designer/internal/parser"
)

// PreviewPanel mərkəzdə HTML önizləməsini göstərir
type PreviewPanel struct {
	container      *fyne.Container
	richText       *widget.RichText
	currentHTML    string
	selectedElem   *parser.HTMLElement
	allElements    []*parser.HTMLElement
	onElemClick    func(*parser.HTMLElement)
	statusLabel    *widget.Label
}

func NewPreviewPanel(onElemClick func(*parser.HTMLElement)) *PreviewPanel {
	p := &PreviewPanel{
		onElemClick: onElemClick,
	}
	p.build()
	return p
}

func (p *PreviewPanel) build() {
	p.richText = widget.NewRichTextFromMarkdown("### HTML faylı açın\n\nFayl > HTML Aç menyusundan HTML faylı seçin.")
	p.richText.Wrapping = fyne.TextWrapWord

	p.statusLabel = widget.NewLabel("Hazır")

	header := widget.NewLabelWithStyle("👁  Önizləmə", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Element seçimi üçün siyahı
	p.container = container.NewBorder(
		container.NewVBox(header, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), p.statusLabel),
		nil, nil,
		container.NewScroll(p.richText),
	)
}

// LoadHTML HTML mətnini yükləyir və göstərir
func (p *PreviewPanel) LoadHTML(htmlContent string, elements []*parser.HTMLElement) {
	p.currentHTML = htmlContent
	p.allElements = elements
	p.renderPreview()
}

// HighlightElement seçilmiş elementi vurğulayır
func (p *PreviewPanel) HighlightElement(elem *parser.HTMLElement) {
	p.selectedElem = elem
	p.renderPreview()
	if elem != nil {
		p.statusLabel.SetText(fmt.Sprintf("Seçildi: <%s> %s", elem.Tag, elem.ID))
	}
}

func (p *PreviewPanel) renderPreview() {
	if p.currentHTML == "" {
		return
	}

	// Sadə markdown preview yarat
	// Real-world: webview widget istifadə edilərdi
	var sb strings.Builder
	sb.WriteString("## HTML Strukturu\n\n")
	sb.WriteString("```\n")

	// Elementləri siyahı şəklində göstər
	for _, elem := range p.allElements {
		if isHiddenTag(elem.Tag) {
			continue
		}
		depth := getDepth(elem)
		indent := strings.Repeat("  ", depth)

		line := indent + "<" + elem.Tag
		if elem.ID != "" {
			line += " id=\"" + elem.ID + "\""
		}
		if len(elem.Classes) > 0 {
			line += " class=\"" + strings.Join(elem.Classes, " ") + "\""
		}
		line += ">"

		if elem.InnerText != "" {
			text := elem.InnerText
			if len(text) > 30 {
				text = text[:30] + "..."
			}
			line += " " + text
		}

		if p.selectedElem != nil && elem.UniqueKey == p.selectedElem.UniqueKey {
			line += "  ◄ SEÇİLMİŞ"
		}

		sb.WriteString(line + "\n")
	}

	sb.WriteString("```\n\n")

	// Seçilmiş element üçün stil göstər
	if p.selectedElem != nil {
		sb.WriteString("### Seçilmiş Element Stili\n\n")
		sb.WriteString("```css\n")
		styleStr := parser.StyleToString(p.selectedElem.Style)
		if styleStr == "" {
			sb.WriteString("/* Stil yoxdur */\n")
		} else {
			selector := parser.GetSelector(p.selectedElem)
			sb.WriteString(selector + " {\n")
			parts := strings.Split(styleStr, "; ")
			for _, part := range parts {
				if part != "" {
					sb.WriteString("    " + part + ";\n")
				}
			}
			sb.WriteString("}\n")
		}
		sb.WriteString("```\n")
	}

	p.richText.ParseMarkdown(sb.String())
}

func (p *PreviewPanel) Container() *fyne.Container {
	return p.container
}

func getDepth(elem *parser.HTMLElement) int {
	depth := 0
	current := elem.Parent
	for current != nil && current.Tag != "root" {
		depth++
		current = current.Parent
	}
	return depth
}
