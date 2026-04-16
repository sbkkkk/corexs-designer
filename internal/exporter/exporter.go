package exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"corexs-designer/internal/parser"
)

type Change struct {
	Elem     *parser.HTMLElement
	Selector string
}

type Exporter struct {
	changes map[string]*Change
}

func New() *Exporter {
	return &Exporter{changes: make(map[string]*Change)}
}

func (e *Exporter) Record(elem *parser.HTMLElement) {
	e.changes[elem.UniqueKey] = &Change{
		Elem:     elem,
		Selector: parser.GetSelector(elem),
	}
}

func (e *Exporter) Count() int { return len(e.changes) }

func (e *Exporter) Clear() { e.changes = make(map[string]*Change) }

func (e *Exporter) ExportCSS(outputPath string) error {
	if len(e.changes) == 0 {
		return fmt.Errorf("dəyişiklik yoxdur")
	}

	var sb strings.Builder
	sb.WriteString("/*\n")
	sb.WriteString(" * Corexs Designer — Generated CSS\n")
	sb.WriteString(fmt.Sprintf(" * %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(" */\n\n")

	for _, c := range e.changes {
		styleStr := parser.StyleToCSS(c.Elem.Style)
		if styleStr == "" {
			continue
		}
		if c.Elem.InnerText != "" {
			txt := c.Elem.InnerText
			if len(txt) > 50 {
				txt = txt[:50] + "..."
			}
			sb.WriteString(fmt.Sprintf("/* \"%s\" */\n", txt))
		}
		sb.WriteString(c.Selector + " {\n")
		for _, prop := range strings.Split(styleStr, ";") {
			if prop != "" {
				sb.WriteString("  " + prop + ";\n")
			}
		}
		sb.WriteString("}\n\n")
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(sb.String()), 0644)
}
