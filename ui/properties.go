package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"corexs-designer/internal/parser"
)

// PropertyPanel seçilmiş elementin xüsusiyyətlərini göstərir
type PropertyPanel struct {
	container    *fyne.Container
	currentElem  *parser.HTMLElement
	onChanged    func(*parser.HTMLElement)

	// Labels
	tagLabel  *widget.Label
	idLabel   *widget.Label

	// Typography
	colorEntry      *widget.Entry
	bgColorEntry    *widget.Entry
	fontSizeEntry   *widget.Entry
	fontWeightSelect *widget.Select
	fontFamilyEntry *widget.Entry
	textAlignSelect *widget.Select

	// Spacing
	marginEntry  *widget.Entry
	paddingEntry *widget.Entry

	// Size
	widthEntry  *widget.Entry
	heightEntry *widget.Entry

	// Position
	positionSelect *widget.Select
	topEntry       *widget.Entry
	leftEntry      *widget.Entry
	rightEntry     *widget.Entry
	bottomEntry    *widget.Entry

	// Border
	borderRadiusEntry *widget.Entry
	borderEntry       *widget.Entry

	// Digər
	opacityEntry  *widget.Entry
	displaySelect *widget.Select

	// Inner text
	innerTextEntry *widget.Entry
}

func NewPropertyPanel(onChanged func(*parser.HTMLElement)) *PropertyPanel {
	p := &PropertyPanel{
		onChanged: onChanged,
	}
	p.build()
	return p
}

func (p *PropertyPanel) build() {
	// Info
	p.tagLabel = widget.NewLabelWithStyle("Element seçilməyib", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	p.idLabel = widget.NewLabel("")

	// Typography
	p.colorEntry = p.newEntry("məs: #ff0000", func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.Color = v
			p.notify()
		}
	})
	p.bgColorEntry = p.newEntry("məs: #ffffff", func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.BackgroundColor = v
			p.notify()
		}
	})
	p.fontSizeEntry = p.newEntry("məs: 16px", func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.FontSize = v
			p.notify()
		}
	})
	p.fontWeightSelect = widget.NewSelect([]string{"", "normal", "bold", "100", "200", "300", "400", "500", "600", "700", "800", "900"}, func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.FontWeight = v
			p.notify()
		}
	})
	p.fontFamilyEntry = p.newEntry("məs: Arial, sans-serif", func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.FontFamily = v
			p.notify()
		}
	})
	p.textAlignSelect = widget.NewSelect([]string{"", "left", "center", "right", "justify"}, func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.TextAlign = v
			p.notify()
		}
	})

	// Spacing
	p.marginEntry = p.newEntry("məs: 10px 20px", func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.Margin = v
			p.notify()
		}
	})
	p.paddingEntry = p.newEntry("məs: 10px", func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.Padding = v
			p.notify()
		}
	})

	// Size
	p.widthEntry = p.newEntry("məs: 100px / 50%", func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.Width = v
			p.notify()
		}
	})
	p.heightEntry = p.newEntry("məs: 200px", func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.Height = v
			p.notify()
		}
	})

	// Position
	p.positionSelect = widget.NewSelect([]string{"", "static", "relative", "absolute", "fixed", "sticky"}, func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.Position = v
			p.notify()
		}
	})
	p.topEntry = p.newEntry("məs: 10px", func(v string) {
		if p.currentElem != nil { p.currentElem.Style.Top = v; p.notify() }
	})
	p.leftEntry = p.newEntry("məs: 20px", func(v string) {
		if p.currentElem != nil { p.currentElem.Style.Left = v; p.notify() }
	})
	p.rightEntry = p.newEntry("məs: 0px", func(v string) {
		if p.currentElem != nil { p.currentElem.Style.Right = v; p.notify() }
	})
	p.bottomEntry = p.newEntry("məs: 0px", func(v string) {
		if p.currentElem != nil { p.currentElem.Style.Bottom = v; p.notify() }
	})

	// Border
	p.borderRadiusEntry = p.newEntry("məs: 8px", func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.BorderRadius = v
			p.notify()
		}
	})
	p.borderEntry = p.newEntry("məs: 1px solid #ccc", func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.Border = v
			p.notify()
		}
	})

	// Display & Opacity
	p.displaySelect = widget.NewSelect([]string{"", "block", "inline", "inline-block", "flex", "grid", "none"}, func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.Display = v
			p.notify()
		}
	})
	p.opacityEntry = p.newEntry("0.0 - 1.0", func(v string) {
		if p.currentElem != nil {
			p.currentElem.Style.Opacity = v
			p.notify()
		}
	})

	// Inner text
	p.innerTextEntry = widget.NewMultiLineEntry()
	p.innerTextEntry.SetPlaceHolder("Element mətni...")
	p.innerTextEntry.OnChanged = func(v string) {
		if p.currentElem != nil {
			p.currentElem.InnerText = v
			p.notify()
		}
	}

	// Layout qur
	infoBox := container.NewVBox(
		p.tagLabel,
		p.idLabel,
	)

	typoBox := container.NewVBox(
		sectionLabel("🎨 Rəng & Şrift"),
		formRow("Rəng:", p.colorEntry),
		formRow("Arxa Plan:", p.bgColorEntry),
		formRow("Şrift Ölçüsü:", p.fontSizeEntry),
		formRow("Qalınlıq:", p.fontWeightSelect),
		formRow("Şrift:", p.fontFamilyEntry),
		formRow("Mətn Hizası:", p.textAlignSelect),
	)

	spacingBox := container.NewVBox(
		sectionLabel("📐 Boşluq"),
		formRow("Margin:", p.marginEntry),
		formRow("Padding:", p.paddingEntry),
	)

	sizeBox := container.NewVBox(
		sectionLabel("📏 Ölçü"),
		formRow("En:", p.widthEntry),
		formRow("Hündürlük:", p.heightEntry),
	)

	posBox := container.NewVBox(
		sectionLabel("📍 Mövqe"),
		formRow("Position:", p.positionSelect),
		formRow("Top:", p.topEntry),
		formRow("Left:", p.leftEntry),
		formRow("Right:", p.rightEntry),
		formRow("Bottom:", p.bottomEntry),
	)

	borderBox := container.NewVBox(
		sectionLabel("🔲 Kənar"),
		formRow("Border:", p.borderEntry),
		formRow("Border Radius:", p.borderRadiusEntry),
	)

	otherBox := container.NewVBox(
		sectionLabel("⚙️ Digər"),
		formRow("Display:", p.displaySelect),
		formRow("Opacity:", p.opacityEntry),
	)

	textBox := container.NewVBox(
		sectionLabel("✏️ Mətn"),
		p.innerTextEntry,
	)

	scrollContent := container.NewVBox(
		infoBox,
		widget.NewSeparator(),
		typoBox,
		widget.NewSeparator(),
		spacingBox,
		widget.NewSeparator(),
		sizeBox,
		widget.NewSeparator(),
		posBox,
		widget.NewSeparator(),
		borderBox,
		widget.NewSeparator(),
		otherBox,
		widget.NewSeparator(),
		textBox,
		layout.NewSpacer(),
	)

	scroll := container.NewScroll(scrollContent)
	scroll.SetMinSize(fyne.NewSize(280, 0))

	p.container = container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("⚙  Xüsusiyyətlər", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
		),
		nil, nil, nil,
		scroll,
	)
}

// LoadElement seçilmiş elementi yükləyir
func (p *PropertyPanel) LoadElement(elem *parser.HTMLElement) {
	p.currentElem = elem
	if elem == nil {
		p.tagLabel.SetText("Element seçilməyib")
		p.idLabel.SetText("")
		return
	}

	// Tag məlumatı
	displayName := "<" + elem.Tag + ">"
	if elem.ID != "" {
		displayName += " #" + elem.ID
	}
	p.tagLabel.SetText(displayName)

	classes := ""
	if len(elem.Classes) > 0 {
		classes = "." 
		for i, c := range elem.Classes {
			if i > 0 { classes += " ." }
			classes += c
		}
	}
	p.idLabel.SetText(classes)

	// Stil dəyərlərini doldur
	s := elem.Style
	p.setEntryText(p.colorEntry, s.Color)
	p.setEntryText(p.bgColorEntry, s.BackgroundColor)
	p.setEntryText(p.fontSizeEntry, s.FontSize)
	p.setSelectValue(p.fontWeightSelect, s.FontWeight)
	p.setEntryText(p.fontFamilyEntry, s.FontFamily)
	p.setSelectValue(p.textAlignSelect, s.TextAlign)
	p.setEntryText(p.marginEntry, s.Margin)
	p.setEntryText(p.paddingEntry, s.Padding)
	p.setEntryText(p.widthEntry, s.Width)
	p.setEntryText(p.heightEntry, s.Height)
	p.setSelectValue(p.positionSelect, s.Position)
	p.setEntryText(p.topEntry, s.Top)
	p.setEntryText(p.leftEntry, s.Left)
	p.setEntryText(p.rightEntry, s.Right)
	p.setEntryText(p.bottomEntry, s.Bottom)
	p.setEntryText(p.borderRadiusEntry, s.BorderRadius)
	p.setEntryText(p.borderEntry, s.Border)
	p.setSelectValue(p.displaySelect, s.Display)
	p.setEntryText(p.opacityEntry, s.Opacity)
	p.setEntryText(p.innerTextEntry, elem.InnerText)
}

func (p *PropertyPanel) Container() *fyne.Container {
	return p.container
}

func (p *PropertyPanel) notify() {
	if p.onChanged != nil && p.currentElem != nil {
		p.onChanged(p.currentElem)
	}
}

func (p *PropertyPanel) newEntry(placeholder string, onChange func(string)) *widget.Entry {
	e := widget.NewEntry()
	e.SetPlaceHolder(placeholder)
	e.OnChanged = onChange
	return e
}

func (p *PropertyPanel) setEntryText(e *widget.Entry, val string) {
	e.OnChanged = nil
	e.SetText(val)
}

func (p *PropertyPanel) setSelectValue(s *widget.Select, val string) {
	s.SetSelected(val)
}

// Yardımçı funksiyalar
func sectionLabel(text string) *widget.Label {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

func formRow(label string, w fyne.CanvasObject) *fyne.Container {
	lbl := widget.NewLabel(label)
	lbl.TextStyle = fyne.TextStyle{}
	return container.New(layout.NewFormLayout(), lbl, w)
}

// ColorPreview rəng önizləməsi üçün (gələcək inkişaf)
func colorPreview(color string) fyne.CanvasObject {
	_ = color
	_ = theme.ColorNameBackground
	return widget.NewLabel(color)
}
