package parser

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

type ElementStyle struct {
	Color           string `json:"color"`
	BackgroundColor string `json:"backgroundColor"`
	FontSize        string `json:"fontSize"`
	FontWeight      string `json:"fontWeight"`
	FontFamily      string `json:"fontFamily"`
	Width           string `json:"width"`
	Height          string `json:"height"`
	Margin          string `json:"margin"`
	MarginTop       string `json:"marginTop"`
	MarginRight     string `json:"marginRight"`
	MarginBottom    string `json:"marginBottom"`
	MarginLeft      string `json:"marginLeft"`
	Padding         string `json:"padding"`
	PaddingTop      string `json:"paddingTop"`
	PaddingRight    string `json:"paddingRight"`
	PaddingBottom   string `json:"paddingBottom"`
	PaddingLeft     string `json:"paddingLeft"`
	Position        string `json:"position"`
	Top             string `json:"top"`
	Left            string `json:"left"`
	Right           string `json:"right"`
	Bottom          string `json:"bottom"`
	Display         string `json:"display"`
	TextAlign       string `json:"textAlign"`
	BorderRadius    string `json:"borderRadius"`
	Border          string `json:"border"`
	BorderColor     string `json:"borderColor"`
	BorderWidth     string `json:"borderWidth"`
	BorderStyle     string `json:"borderStyle"`
	Opacity         string `json:"opacity"`
	LineHeight      string `json:"lineHeight"`
	LetterSpacing   string `json:"letterSpacing"`
	FlexDirection   string `json:"flexDirection"`
	JustifyContent  string `json:"justifyContent"`
	AlignItems      string `json:"alignItems"`
	BoxShadow       string `json:"boxShadow"`
	Overflow        string `json:"overflow"`
	ZIndex          string `json:"zIndex"`
	Transform       string `json:"transform"`
	Transition      string `json:"transition"`
	Cursor          string `json:"cursor"`
	TextDecoration  string `json:"textDecoration"`
}

type HTMLElement struct {
	UniqueKey  string            `json:"id"`
	Tag        string            `json:"tag"`
	ElemID     string            `json:"elemId"`
	Classes    []string          `json:"classes"`
	Attributes map[string]string `json:"attributes"`
	InnerText  string            `json:"innerText"`
	Style      ElementStyle      `json:"style"`
	Children   []*HTMLElement    `json:"children"`
	Parent     *HTMLElement      `json:"-"`
}

type ParseResult struct {
	Root     *HTMLElement
	Elements map[string]*HTMLElement
	RawHTML  string
}

var counter int

func ParseHTML(htmlContent string) (*ParseResult, error) {
	counter = 0
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	result := &ParseResult{
		RawHTML:  htmlContent,
		Elements: make(map[string]*HTMLElement),
	}

	root := &HTMLElement{
		UniqueKey:  "corexs-root",
		Tag:        "root",
		Attributes: make(map[string]string),
	}
	result.Root = root

	bodyNode := findTag(doc, "body")
	if bodyNode == nil {
		bodyNode = doc
	}

	walkNode(bodyNode, root, result.Elements)
	return result, nil
}

func findTag(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findTag(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func walkNode(n *html.Node, parent *HTMLElement, elements map[string]*HTMLElement) {
	if n.Type != html.ElementNode {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkNode(c, parent, elements)
		}
		return
	}

	counter++
	elem := &HTMLElement{
		UniqueKey:  fmt.Sprintf("cx-%s-%d", n.Data, counter),
		Tag:        n.Data,
		Attributes: make(map[string]string),
		Parent:     parent,
	}

	for _, attr := range n.Attr {
		switch attr.Key {
		case "id":
			elem.ElemID = attr.Val
		case "class":
			elem.Classes = strings.Fields(attr.Val)
		case "style":
			elem.Style = ParseInlineStyle(attr.Val)
		default:
			elem.Attributes[attr.Key] = attr.Val
		}
	}

	// Direct text uşaqları
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			t := strings.TrimSpace(c.Data)
			if t != "" {
				elem.InnerText += t + " "
			}
		}
	}
	elem.InnerText = strings.TrimSpace(elem.InnerText)

	parent.Children = append(parent.Children, elem)
	elements[elem.UniqueKey] = elem

	if n.Data != "script" && n.Data != "style" {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkNode(c, elem, elements)
		}
	}
}

func ParseInlineStyle(s string) ElementStyle {
	style := ElementStyle{}
	for _, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(strings.ToLower(kv[0]))
		v := strings.TrimSpace(kv[1])
		ApplyStyleProp(&style, k, v)
	}
	return style
}

func ApplyStyleProp(s *ElementStyle, k, v string) {
	switch k {
	case "color":
		s.Color = v
	case "background-color", "background":
		s.BackgroundColor = v
	case "font-size":
		s.FontSize = v
	case "font-weight":
		s.FontWeight = v
	case "font-family":
		s.FontFamily = v
	case "width":
		s.Width = v
	case "height":
		s.Height = v
	case "margin":
		s.Margin = v
	case "margin-top":
		s.MarginTop = v
	case "margin-right":
		s.MarginRight = v
	case "margin-bottom":
		s.MarginBottom = v
	case "margin-left":
		s.MarginLeft = v
	case "padding":
		s.Padding = v
	case "padding-top":
		s.PaddingTop = v
	case "padding-right":
		s.PaddingRight = v
	case "padding-bottom":
		s.PaddingBottom = v
	case "padding-left":
		s.PaddingLeft = v
	case "position":
		s.Position = v
	case "top":
		s.Top = v
	case "left":
		s.Left = v
	case "right":
		s.Right = v
	case "bottom":
		s.Bottom = v
	case "display":
		s.Display = v
	case "text-align":
		s.TextAlign = v
	case "border-radius":
		s.BorderRadius = v
	case "border":
		s.Border = v
	case "border-color":
		s.BorderColor = v
	case "border-width":
		s.BorderWidth = v
	case "border-style":
		s.BorderStyle = v
	case "opacity":
		s.Opacity = v
	case "line-height":
		s.LineHeight = v
	case "letter-spacing":
		s.LetterSpacing = v
	case "flex-direction":
		s.FlexDirection = v
	case "justify-content":
		s.JustifyContent = v
	case "align-items":
		s.AlignItems = v
	case "box-shadow":
		s.BoxShadow = v
	case "overflow":
		s.Overflow = v
	case "z-index":
		s.ZIndex = v
	case "transform":
		s.Transform = v
	case "transition":
		s.Transition = v
	case "cursor":
		s.Cursor = v
	case "text-decoration":
		s.TextDecoration = v
	}
}

func StyleToCSS(s ElementStyle) string {
	var parts []string
	add := func(prop, val string) {
		if val != "" {
			parts = append(parts, prop+":"+val)
		}
	}
	add("color", s.Color)
	add("background-color", s.BackgroundColor)
	add("font-size", s.FontSize)
	add("font-weight", s.FontWeight)
	add("font-family", s.FontFamily)
	add("width", s.Width)
	add("height", s.Height)
	add("margin", s.Margin)
	add("margin-top", s.MarginTop)
	add("margin-right", s.MarginRight)
	add("margin-bottom", s.MarginBottom)
	add("margin-left", s.MarginLeft)
	add("padding", s.Padding)
	add("padding-top", s.PaddingTop)
	add("padding-right", s.PaddingRight)
	add("padding-bottom", s.PaddingBottom)
	add("padding-left", s.PaddingLeft)
	add("position", s.Position)
	add("top", s.Top)
	add("left", s.Left)
	add("right", s.Right)
	add("bottom", s.Bottom)
	add("display", s.Display)
	add("text-align", s.TextAlign)
	add("border-radius", s.BorderRadius)
	add("border", s.Border)
	add("border-color", s.BorderColor)
	add("border-width", s.BorderWidth)
	add("border-style", s.BorderStyle)
	add("opacity", s.Opacity)
	add("line-height", s.LineHeight)
	add("letter-spacing", s.LetterSpacing)
	add("flex-direction", s.FlexDirection)
	add("justify-content", s.JustifyContent)
	add("align-items", s.AlignItems)
	add("box-shadow", s.BoxShadow)
	add("overflow", s.Overflow)
	add("z-index", s.ZIndex)
	add("transform", s.Transform)
	add("transition", s.Transition)
	add("cursor", s.Cursor)
	add("text-decoration", s.TextDecoration)
	return strings.Join(parts, ";")
}

// GetSelector returns the best CSS selector for an element.
// Priority: #id > .class > tag path (no data-cx attributes)
func GetSelector(elem *HTMLElement) string {
	// 1. ID — most specific
	if elem.ElemID != "" {
		return "#" + elem.ElemID
	}
	// 2. Class(es)
	if len(elem.Classes) > 0 {
		return elem.Tag + "." + strings.Join(elem.Classes, ".")
	}
	// 3. Build a path selector using tag + nth-of-type
	// Walk up to find a stable anchor (id or class on ancestor)
	return buildPathSelector(elem)
}

// buildPathSelector creates a CSS path like: section.hero > h1:nth-of-type(1)
func buildPathSelector(elem *HTMLElement) string {
	parts := []string{}
	cur := elem
	for cur != nil && cur.Tag != "root" && cur.Tag != "" {
		seg := nthSelector(cur)
		parts = append([]string{seg}, parts...)
		// Stop if we hit an element with id or class — that's our anchor
		if cur.Parent != nil && (cur.Parent.ElemID != "" || len(cur.Parent.Classes) > 0 || cur.Parent.Tag == "body") {
			// Prepend anchor
			anc := cur.Parent
			var ancSel string
			if anc.ElemID != "" {
				ancSel = "#" + anc.ElemID
			} else if len(anc.Classes) > 0 {
				ancSel = anc.Tag + "." + strings.Join(anc.Classes, ".")
			} else if anc.Tag == "body" {
				ancSel = "body"
			}
			if ancSel != "" {
				parts = append([]string{ancSel}, parts...)
				break
			}
		}
		cur = cur.Parent
	}
	if len(parts) == 0 {
		return elem.Tag
	}
	return strings.Join(parts, " > ")
}

// nthSelector returns "tag" or "tag:nth-of-type(n)" for an element among its siblings
func nthSelector(elem *HTMLElement) string {
	if elem.Parent == nil {
		return elem.Tag
	}
	// Count siblings of same tag
	n := 0
	total := 0
	for _, sib := range elem.Parent.Children {
		if sib.Tag == elem.Tag {
			total++
			if sib.UniqueKey == elem.UniqueKey {
				n = total
			}
		}
	}
	if total == 1 {
		return elem.Tag // no need for nth
	}
	return elem.Tag + ":nth-of-type(" + itoa(n) + ")"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// StyleToPropsMap converts ElementStyle to a map[prop]value for bridge use
func StyleToPropsMap(s ElementStyle) map[string]string {
	m := make(map[string]string)
	add := func(k, v string) {
		if v != "" {
			m[k] = v
		}
	}
	add("color", s.Color)
	add("background-color", s.BackgroundColor)
	add("font-size", s.FontSize)
	add("font-weight", s.FontWeight)
	add("font-family", s.FontFamily)
	add("width", s.Width)
	add("height", s.Height)
	add("margin", s.Margin)
	add("margin-top", s.MarginTop)
	add("margin-right", s.MarginRight)
	add("margin-bottom", s.MarginBottom)
	add("margin-left", s.MarginLeft)
	add("padding", s.Padding)
	add("padding-top", s.PaddingTop)
	add("padding-right", s.PaddingRight)
	add("padding-bottom", s.PaddingBottom)
	add("padding-left", s.PaddingLeft)
	add("position", s.Position)
	add("top", s.Top)
	add("left", s.Left)
	add("right", s.Right)
	add("bottom", s.Bottom)
	add("display", s.Display)
	add("text-align", s.TextAlign)
	add("border-radius", s.BorderRadius)
	add("border", s.Border)
	add("opacity", s.Opacity)
	add("line-height", s.LineHeight)
	add("letter-spacing", s.LetterSpacing)
	add("flex-direction", s.FlexDirection)
	add("justify-content", s.JustifyContent)
	add("align-items", s.AlignItems)
	add("box-shadow", s.BoxShadow)
	add("overflow", s.Overflow)
	add("z-index", s.ZIndex)
	add("transform", s.Transform)
	add("transition", s.Transition)
	add("cursor", s.Cursor)
	add("text-decoration", s.TextDecoration)
	return m
}
