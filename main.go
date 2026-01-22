package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/superstarryeyes/bit/ansifonts"
	"gopkg.in/yaml.v3"
)

type tickMsg time.Time

type model struct {
	slides []Slide
	index  int
	width  int
	height int
	frame  int
}

type keyMap struct {
	Next key.Binding
	Prev key.Binding
	Quit key.Binding
}

var keys = keyMap{
	Next: key.NewBinding(
		key.WithKeys("right", "l", "n", "space", "enter"),
		key.WithHelp("->", "next"),
	),
	Prev: key.NewBinding(
		key.WithKeys("left", "h", "p", "backspace"),
		key.WithHelp("<-", "prev"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

var (
	colorBgDim       = lipgloss.Color("#045d68")
	colorBgBright    = lipgloss.Color("#00f0ff")
	colorPanelBg     = lipgloss.Color("#05080b")
	colorPanelEdge   = lipgloss.Color("#00c8ff")
	colorBlockBg     = lipgloss.Color("#0b1b22")
	colorNoiseDim    = lipgloss.Color("#0b2d36")
	colorNoiseBright = lipgloss.Color("#146a7b")
	colorTextMain    = lipgloss.Color("#bffcff")
	colorTextMuted   = lipgloss.Color("#4dd1ff")
	colorTextAccent  = lipgloss.Color("#ff4fd8")
)

var (
	bgDimStyle            = lipgloss.NewStyle().Foreground(colorBgDim)
	bgBrightStyle         = lipgloss.NewStyle().Foreground(colorBgBright)
	panelBgStyle          = lipgloss.NewStyle().Background(colorPanelBg)
	panelEdgeStyle        = lipgloss.NewStyle().Foreground(colorPanelEdge).Background(colorPanelBg)
	textMainStyle         = panelBgStyle.Foreground(colorTextMain)
	textMutedStyle        = panelBgStyle.Foreground(colorTextMuted)
	textHeadStyle         = panelBgStyle.Foreground(colorTextAccent).Bold(true)
	textHeadBlockStyle    = lipgloss.NewStyle().Foreground(colorTextAccent).Background(colorBlockBg).Bold(true)
	textSubStyle          = panelBgStyle.Foreground(colorBgBright).Bold(true)
	panelNoiseDimStyle    = panelBgStyle.Foreground(colorNoiseDim)
	panelNoiseBrightStyle = panelBgStyle.Foreground(colorNoiseBright)
)

var (
	ansiHeadingFontName    = "pixeloperator"
	ansiSubheadingFontName = "fivebyfive"
	ansiHeadingFont        *ansifonts.Font
	ansiSubheadingFont     *ansifonts.Font
	ansiHeadingOptions     ansifonts.RenderOptions
	ansiSubheadingOptions  ansifonts.RenderOptions
	ansiHeadingHeight      int
	ansiSubheadingHeight   int
	frontmatterH1Options   *FontOptionsYAML
	frontmatterH2Options   *FontOptionsYAML
	headingUppercase       = true
	subheadingUppercase    = true
)

func main() {
	filePath := flag.String("file", "slides.md", "markdown file with slides separated by ---")
	flag.Parse()

	data, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Printf("failed to read %s: %v\n", *filePath, err)
		os.Exit(1)
	}

	frontmatter, content := parseFrontMatter(string(data))
	applyFrontMatter(frontmatter)
	if frontmatterH1Options != nil && frontmatterH1Options.Uppercase != nil {
		headingUppercase = *frontmatterH1Options.Uppercase
	}
	if frontmatterH2Options != nil && frontmatterH2Options.Uppercase != nil {
		subheadingUppercase = *frontmatterH2Options.Uppercase
	}
	slides := parseSlides(content)
	initAnsiFonts()
	m := model{slides: slides}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tickMsg:
		m.frame++
		return m, tickCmd()
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Next):
			if len(m.slides) > 0 {
				m.index = (m.index + 1) % len(m.slides)
			}
		case key.Matches(msg, keys.Prev):
			if len(m.slides) > 0 {
				m.index = (m.index - 1 + len(m.slides)) % len(m.slides)
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading..."
	}
	return renderScreen(m)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func initAnsiFonts() {
	headingFont, err := ansifonts.LoadFont(ansiHeadingFontName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load heading font %q: %v\n", ansiHeadingFontName, err)
		return
	}
	subFont, err := ansifonts.LoadFont(ansiSubheadingFontName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load subheading font %q: %v\n", ansiSubheadingFontName, err)
		return
	}

	ansiHeadingFont = headingFont
	ansiSubheadingFont = subFont
	ansiHeadingOptions = ansifonts.DefaultRenderOptions()
	ansiHeadingOptions.Alignment = ansifonts.LeftAlign
	ansiHeadingOptions.TextColor = string(colorTextAccent)
	ansiHeadingOptions.CharSpacing = 1
	ansiHeadingOptions.WordSpacing = 1
	ansiHeadingOptions.LineSpacing = 0
	ansiHeadingOptions.ScaleFactor = 1.0
	ansiHeadingOptions = applyRenderOptions(ansiHeadingOptions, frontmatterH1Options, "h1")

	ansiSubheadingOptions = ansifonts.DefaultRenderOptions()
	ansiSubheadingOptions.Alignment = ansifonts.LeftAlign
	ansiSubheadingOptions.TextColor = string(colorBgBright)
	ansiSubheadingOptions.CharSpacing = 1
	ansiSubheadingOptions.WordSpacing = 1
	ansiSubheadingOptions.LineSpacing = 0
	ansiSubheadingOptions.ScaleFactor = 1.0
	ansiSubheadingOptions = applyRenderOptions(ansiSubheadingOptions, frontmatterH2Options, "h2")

	ansiHeadingHeight = fontHeight(ansiHeadingFont, ansiHeadingOptions)
	ansiSubheadingHeight = fontHeight(ansiSubheadingFont, ansiSubheadingOptions)
}

type FrontMatter struct {
	H1Font    string           `yaml:"h1Font"`
	H2Font    string           `yaml:"h2Font"`
	H1Options *FontOptionsYAML `yaml:"h1Options"`
	H2Options *FontOptionsYAML `yaml:"h2Options"`
}

type FontOptionsYAML struct {
	CharSpacing            *int     `yaml:"charSpacing"`
	WordSpacing            *int     `yaml:"wordSpacing"`
	LineSpacing            *int     `yaml:"lineSpacing"`
	Alignment              string   `yaml:"alignment"`
	TextColor              string   `yaml:"textColor"`
	GradientColor          string   `yaml:"gradientColor"`
	GradientDirection      string   `yaml:"gradientDirection"`
	UseGradient            *bool    `yaml:"useGradient"`
	ScaleFactor            *float64 `yaml:"scaleFactor"`
	ShadowEnabled          *bool    `yaml:"shadowEnabled"`
	ShadowHorizontalOffset *int     `yaml:"shadowHorizontalOffset"`
	ShadowVerticalOffset   *int     `yaml:"shadowVerticalOffset"`
	ShadowStyle            string   `yaml:"shadowStyle"`
	Uppercase              *bool    `yaml:"uppercase"`
}

func parseFrontMatter(md string) (FrontMatter, string) {
	var fm FrontMatter
	lines := strings.Split(md, "\n")
	idx := 0
	for idx < len(lines) && strings.TrimSpace(lines[idx]) == "" {
		idx++
	}
	if idx >= len(lines) || strings.TrimSpace(lines[idx]) != "---" {
		return fm, md
	}
	end := idx + 1
	for end < len(lines) && strings.TrimSpace(lines[end]) != "---" {
		end++
	}
	if end >= len(lines) {
		return fm, md
	}
	if err := yaml.Unmarshal([]byte(strings.Join(lines[idx+1:end], "\n")), &fm); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to parse frontmatter: %v\n", err)
		return FrontMatter{}, md
	}
	return fm, strings.Join(lines[end+1:], "\n")
}

func applyFrontMatter(fm FrontMatter) {
	if strings.TrimSpace(fm.H1Font) != "" {
		ansiHeadingFontName = strings.TrimSpace(fm.H1Font)
	}
	if strings.TrimSpace(fm.H2Font) != "" {
		ansiSubheadingFontName = strings.TrimSpace(fm.H2Font)
	}
	if fm.H1Options != nil {
		frontmatterH1Options = fm.H1Options
	}
	if fm.H2Options != nil {
		frontmatterH2Options = fm.H2Options
	}
}

type Slide struct {
	Blocks []Block
}

type BlockKind int

const (
	BlockHeading BlockKind = iota
	BlockText
	BlockBullet
	BlockSpacer
)

type Block struct {
	Kind  BlockKind
	Level int
	Text  string
}

type LineKind int

const (
	LineBody LineKind = iota
	LineHeading
	LineHeadingBlock
	LineHeadingANSI
	LineSubheading
	LineSubheadingANSI
	LineMuted
)

type Line struct {
	Text string
	Kind LineKind
}

func parseSlides(md string) []Slide {
	lines := strings.Split(md, "\n")
	var slides []Slide
	var buf []string

	flush := func() {
		slide := parseSlide(buf)
		if len(slide.Blocks) == 0 {
			return
		}
		slides = append(slides, slide)
		buf = nil
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			flush()
			continue
		}
		buf = append(buf, line)
	}
	flush()

	if len(slides) == 0 {
		slides = append(slides, Slide{Blocks: []Block{{Kind: BlockHeading, Level: 1, Text: "EMPTY"}}})
	}
	return slides
}

func parseSlide(lines []string) Slide {
	var blocks []Block
	var para []string

	flushPara := func() {
		if len(para) == 0 {
			return
		}
		text := strings.TrimSpace(strings.Join(para, " "))
		if text != "" {
			blocks = append(blocks, Block{Kind: BlockText, Text: text})
		}
		para = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flushPara()
			if len(blocks) == 0 || blocks[len(blocks)-1].Kind != BlockSpacer {
				blocks = append(blocks, Block{Kind: BlockSpacer})
			}
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			flushPara()
			level := 0
			for level < len(trimmed) && trimmed[level] == '#' {
				level++
			}
			text := strings.TrimSpace(trimmed[level:])
			blocks = append(blocks, Block{Kind: BlockHeading, Level: level, Text: text})
			continue
		}

		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
			flushPara()
			text := strings.TrimSpace(trimmed[2:])
			blocks = append(blocks, Block{Kind: BlockBullet, Text: text})
			continue
		}

		para = append(para, trimmed)
	}

	flushPara()
	return Slide{Blocks: blocks}
}

func renderScreen(m model) string {
	w, h := m.width, m.height
	contentW, contentH, offsetX, offsetY := fitContent(w, h)

	bg := buildBackground(w, h, m.frame)
	overlayTorus(bg, w, h, m.frame)
	overlayMapField(bg, w, h, m.frame)
	stampWords(bg, w, h, m.frame)

	var contentLines []string
	innerW := max(1, contentW-2)
	innerH := max(1, contentH-2)
	contentBg := sliceBackground(bg, offsetX+1, offsetY+1, innerW, innerH)
	if len(m.slides) > 0 {
		contentLines = renderContent(m.slides[m.index], contentW, contentH, m.index, len(m.slides), contentBg)
	} else {
		contentLines = renderContent(Slide{}, contentW, contentH, 0, 1, contentBg)
	}

	lines := make([]string, h)
	for y := 0; y < h; y++ {
		if y < offsetY || y >= offsetY+contentH {
			lines[y] = renderBgLine(bg[y])
			continue
		}
		left := ""
		if offsetX > 0 {
			left = renderBgLine(bg[y][:offsetX])
		}
		right := ""
		if offsetX+contentW < w {
			right = renderBgLine(bg[y][offsetX+contentW:])
		}
		lines[y] = left + contentLines[y-offsetY] + right
	}

	return strings.Join(lines, "\n")
}

func fitContent(w, h int) (contentW, contentH, offsetX, offsetY int) {
	if w < 20 || h < 10 {
		return w, h, 0, 0
	}
	insetX := max(2, w/20)
	insetY := max(1, h/20)
	availW := w - 4 - 2*insetX
	availH := h - 2 - 2*insetY
	targetAspect := panelAspect * cellAspectAdj
	contentW = availW
	contentH = int(float64(contentW) / targetAspect)
	if contentH > availH {
		contentH = availH
		contentW = int(float64(contentH) * targetAspect)
	}
	if contentW < 10 || contentH < 6 {
		return w, h, 0, 0
	}
	offsetX = (w - contentW) / 2
	offsetY = (h - contentH) / 2
	return contentW, contentH, offsetX, offsetY
}

func renderContent(slide Slide, contentW, contentH, index, total int, contentBg [][]rune) []string {
	innerW := max(1, contentW-2)
	innerH := max(1, contentH-2)
	innerBuf := make([][]rune, innerH)
	innerKinds := make([]LineKind, innerH)
	innerANSI := make([]string, innerH)
	for y := 0; y < innerH; y++ {
		innerBuf[y] = make([]rune, innerW)
		for x := 0; x < innerW; x++ {
			innerBuf[y][x] = ' '
		}
		innerKinds[y] = LineBody
	}

	if innerH > 0 {
		header := fmt.Sprintf(" GHOSTDECK // SLIDE %02d/%02d ", index+1, total)
		placeText(innerBuf[0], 1, header)
		innerKinds[0] = LineMuted
	}
	if innerH > 1 {
		footer := "<- prev  -> next  q quit"
		placeTextRight(innerBuf[innerH-1], innerW-2, footer)
		innerKinds[innerH-1] = LineMuted
	}

	textTop := 1
	textBottom := innerH - 2
	if textBottom >= textTop {
		textAreaW := max(1, innerW-4)
		h1Lines, h2Lines, bodyLines := renderSlideSections(slide, textAreaW)
		placeSlideSections(innerBuf, innerKinds, innerANSI, h1Lines, h2Lines, bodyLines, textTop, textBottom, textAreaW)
	}

	lines := make([]string, contentH)
	borderTop := "+" + strings.Repeat("-", contentW-2) + "+"
	borderBottom := borderTop
	lines[0] = panelEdgeStyle.Render(borderTop)
	for y := 1; y < contentH-1; y++ {
		bgRow := []rune(nil)
		if y-1 < len(contentBg) {
			bgRow = contentBg[y-1]
		}
		if innerANSI[y-1] != "" {
			mid := renderPanelAnsiLine(innerANSI[y-1], innerW)
			lines[y] = panelEdgeStyle.Render("|") + mid + panelEdgeStyle.Render("|")
			continue
		}
		mid := renderPanelLine(innerBuf[y-1], innerKinds[y-1], bgRow)
		lines[y] = panelEdgeStyle.Render("|") + mid + panelEdgeStyle.Render("|")
	}
	lines[contentH-1] = panelEdgeStyle.Render(borderBottom)
	return lines
}

func renderPanelLine(textRow []rune, kind LineKind, bgRow []rune) string {
	width := len(textRow)
	if width == 0 {
		return ""
	}
	line := padRight(string(textRow), width)
	if kind == LineHeadingBlock {
		return textHeadBlockStyle.Render(line)
	}
	if lineHasText(textRow) || len(bgRow) < width {
		return styleForLine(kind).Render(line)
	}
	return renderPanelNoiseLine(bgRow[:width])
}

func renderPanelAnsiLine(text string, width int) string {
	if width <= 0 {
		return ""
	}
	trimmed := trimToWidth(text, width)
	line := padRight(trimmed, width)
	return applyPanelBackgroundANSI(line)
}

func renderSlideSections(slide Slide, width int) ([]Line, []Line, []Line) {
	var h1Text string
	var h2Text string
	var bodyBlocks []Block

	if block, ok := soloTextBlock(slide); ok {
		h1Text = block.Text
	} else {
		for _, block := range slide.Blocks {
			if block.Kind == BlockHeading {
				text := strings.TrimSpace(block.Text)
				if text == "" {
					continue
				}
				if block.Level <= 1 && h1Text == "" {
					h1Text = text
					continue
				}
				if block.Level == 2 && h2Text == "" {
					h2Text = text
					continue
				}
			}
			bodyBlocks = append(bodyBlocks, block)
		}
	}

	h1Lines := renderHeadingLines(h1Text, width)
	h2Lines := renderSubheadingLines(h2Text, width)
	bodyLines := renderBodyBlocks(bodyBlocks, width)
	return h1Lines, h2Lines, bodyLines
}

func renderHeadingLines(text string, width int) []Line {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	text = maybeUppercaseForFont(text, ansiHeadingFont, headingUppercase)
	if ansiLines := renderAnsiLines(text, width, ansiHeadingFont, ansiHeadingOptions); len(ansiLines) > 0 {
		big := centerBlockLines(ansiLines, width)
		big = alignBlockLines(big, width, ansiHeadingOptions.Alignment)
		lines := make([]Line, 0, len(big))
		for _, line := range big {
			lines = append(lines, Line{Text: line, Kind: LineHeadingANSI})
		}
		return lines
	}

	big := centerBlockLines(renderBigText(text, width), width)
	big = alignBlockLines(big, width, ansiHeadingOptions.Alignment)
	lines := make([]Line, 0, len(big))
	for _, line := range big {
		lines = append(lines, Line{Text: line, Kind: LineHeadingBlock})
	}
	return lines
}

func renderSubheadingLines(text string, width int) []Line {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	text = maybeUppercaseForFont(text, ansiSubheadingFont, subheadingUppercase)
	if ansiLines := renderAnsiLines(text, width, ansiSubheadingFont, ansiSubheadingOptions); len(ansiLines) > 0 {
		big := centerBlockLines(ansiLines, width)
		big = alignBlockLines(big, width, ansiSubheadingOptions.Alignment)
		lines := make([]Line, 0, len(big))
		for _, line := range big {
			lines = append(lines, Line{Text: line, Kind: LineSubheadingANSI})
		}
		return lines
	}
	return []Line{{Text: alignLine(text, width, ansiSubheadingOptions.Alignment), Kind: LineSubheading}}
}

func maybeUppercaseForFont(text string, font *ansifonts.Font, preferUpper bool) string {
	if !preferUpper || font == nil {
		return text
	}
	if fontSupportsLowercase(font) {
		return text
	}
	return strings.ToUpper(text)
}

func fontSupportsLowercase(font *ansifonts.Font) bool {
	if font == nil || len(font.FontData.Characters) == 0 {
		return false
	}
	if _, ok := font.FontData.Characters["a"]; ok {
		return true
	}
	if _, ok := font.FontData.Characters["b"]; ok {
		return true
	}
	if _, ok := font.FontData.Characters["g"]; ok {
		return true
	}
	return false
}

func renderBodyBlocks(blocks []Block, width int) []Line {
	var lines []Line

	addSpacer := func() {
		if len(lines) == 0 || strings.TrimSpace(lines[len(lines)-1].Text) == "" {
			return
		}
		lines = append(lines, Line{Text: "", Kind: LineBody})
	}

	for _, block := range blocks {
		switch block.Kind {
		case BlockHeading:
			text := strings.TrimSpace(block.Text)
			if text == "" {
				continue
			}
			addSpacer()
			text = maybeUppercaseForFont(text, ansiSubheadingFont, subheadingUppercase)
			lines = append(lines, Line{Text: alignLine(text, width, ansiSubheadingOptions.Alignment), Kind: LineSubheading})
			lines = append(lines, Line{Text: "", Kind: LineBody})
		case BlockBullet:
			addSpacer()
			wrapped := wrapText(block.Text, max(1, width-4))
			for i, line := range wrapped {
				prefix := "- "
				if i > 0 {
					prefix = "  "
				}
				lines = append(lines, Line{Text: prefix + line, Kind: LineBody})
			}
		case BlockText:
			addSpacer()
			wrapped := wrapText(block.Text, width)
			for _, line := range wrapped {
				lines = append(lines, Line{Text: line, Kind: LineBody})
			}
		case BlockSpacer:
			lines = append(lines, Line{Text: "", Kind: LineBody})
		}
	}

	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1].Text) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func placeSlideSections(innerBuf [][]rune, innerKinds []LineKind, innerANSI []string, h1Lines, h2Lines, bodyLines []Line, top, bottom, width int) {
	if bottom < top {
		return
	}
	textAreaH := bottom - top + 1
	h1Height := ansiHeadingHeight
	if h1Height == 0 {
		h1Height = bigFontHeight
	}
	h2Height := ansiSubheadingHeight
	if h2Height == 0 {
		h2Height = 3
	}

	h1Reserve := max(h1Height, textAreaH/3)
	h2Reserve := max(h2Height, textAreaH/6)
	gap := 1
	if h1Reserve+h2Reserve+gap > textAreaH {
		overflow := h1Reserve + h2Reserve + gap - textAreaH
		if h2Reserve-overflow >= h2Height {
			h2Reserve -= overflow
		} else {
			remainder := overflow - (h2Reserve - h2Height)
			h2Reserve = h2Height
			h1Reserve = max(h1Height, h1Reserve-remainder)
		}
	}

	h1Start := top + max(0, (h1Reserve-len(h1Lines))/2)
	h2Top := top + h1Reserve
	h2Start := h2Top + max(0, (h2Reserve-len(h2Lines))/2)
	bodyTop := h2Top + h2Reserve + gap
	bodyBottom := bottom

	placeLines(innerBuf, innerKinds, innerANSI, h1Start, min(h1Reserve, textAreaH), h1Lines)
	placeLines(innerBuf, innerKinds, innerANSI, h2Start, min(h2Reserve, textAreaH), h2Lines)
	if bodyTop <= bodyBottom {
		placeLines(innerBuf, innerKinds, innerANSI, bodyTop, bodyBottom-bodyTop+1, bodyLines)
	}
}

func placeLines(innerBuf [][]rune, innerKinds []LineKind, innerANSI []string, startRow, maxRows int, lines []Line) {
	if maxRows <= 0 || len(lines) == 0 {
		return
	}
	row := startRow
	for _, line := range lines {
		if row >= len(innerBuf) || row >= startRow+maxRows {
			break
		}
		if lineIsANSI(line.Kind) {
			innerANSI[row] = strings.Repeat(" ", 2) + line.Text
			innerKinds[row] = line.Kind
		} else {
			placeText(innerBuf[row], 2, line.Text)
			if line.Kind != LineBody {
				innerKinds[row] = line.Kind
			}
		}
		row++
	}
}

func renderPanelNoiseLine(bgRow []rune) string {
	var b strings.Builder
	currentBright := isBright(bgRow[0])
	var seg strings.Builder
	flush := func(bright bool) {
		if seg.Len() == 0 {
			return
		}
		text := seg.String()
		seg.Reset()
		if bright {
			b.WriteString(panelNoiseBrightStyle.Render(text))
			return
		}
		b.WriteString(panelNoiseDimStyle.Render(text))
	}

	for _, r := range bgRow {
		out := softenRune(r)
		bright := isBright(out)
		if bright != currentBright {
			flush(currentBright)
			currentBright = bright
		}
		seg.WriteRune(out)
	}
	flush(currentBright)
	return b.String()
}

func styleForLine(kind LineKind) lipgloss.Style {
	switch kind {
	case LineHeading:
		return textHeadStyle
	case LineHeadingBlock:
		return textHeadBlockStyle
	case LineHeadingANSI:
		return textHeadStyle
	case LineSubheading:
		return textSubStyle
	case LineSubheadingANSI:
		return textSubStyle
	case LineMuted:
		return textMutedStyle
	default:
		return textMainStyle
	}
}

func renderAnsiLines(text string, maxWidth int, font *ansifonts.Font, options ansifonts.RenderOptions) []string {
	if font == nil {
		return nil
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	rendered := ansifonts.RenderTextWithOptions(text, font, options)
	for i := range rendered {
		rendered[i] = strings.TrimRight(rendered[i], " ")
		rendered[i] = trimToWidth(rendered[i], maxWidth)
	}
	return rendered
}

func fontHeight(font *ansifonts.Font, options ansifonts.RenderOptions) int {
	if font == nil {
		return 0
	}
	lines := ansifonts.RenderTextWithOptions("A", font, options)
	return len(lines)
}

func applyRenderOptions(base ansifonts.RenderOptions, override *FontOptionsYAML, label string) ansifonts.RenderOptions {
	if override == nil {
		return base
	}
	if override.CharSpacing != nil {
		base.CharSpacing = *override.CharSpacing
	}
	if override.WordSpacing != nil {
		base.WordSpacing = *override.WordSpacing
	}
	if override.LineSpacing != nil {
		base.LineSpacing = *override.LineSpacing
	}
	if override.TextColor != "" {
		base.TextColor = normalizeHexColor(override.TextColor)
	}
	if override.GradientColor != "" {
		base.GradientColor = normalizeHexColor(override.GradientColor)
	}
	if override.UseGradient != nil {
		base.UseGradient = *override.UseGradient
	}
	if override.ScaleFactor != nil {
		base.ScaleFactor = *override.ScaleFactor
	}
	if override.ShadowEnabled != nil {
		base.ShadowEnabled = *override.ShadowEnabled
	}
	if override.ShadowHorizontalOffset != nil {
		base.ShadowHorizontalOffset = *override.ShadowHorizontalOffset
	}
	if override.ShadowVerticalOffset != nil {
		base.ShadowVerticalOffset = *override.ShadowVerticalOffset
	}
	if override.Alignment != "" {
		if align, ok := parseAlignment(override.Alignment); ok {
			base.Alignment = align
		}
	}
	if override.GradientDirection != "" {
		if dir, ok := parseGradientDirection(override.GradientDirection); ok {
			base.GradientDirection = dir
		}
	}
	if override.ShadowStyle != "" {
		if style, ok := parseShadowStyle(override.ShadowStyle); ok {
			base.ShadowStyle = style
		}
	}
	if override.ShadowEnabled == nil && (override.ShadowHorizontalOffset != nil || override.ShadowVerticalOffset != nil || override.ShadowStyle != "") {
		base.ShadowEnabled = true
	}
	if err := base.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: invalid %s render options: %v\n", label, err)
		return base
	}
	return base
}

func normalizeHexColor(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if strings.HasPrefix(value, "#") {
		return value
	}
	if len(value) == 6 {
		return "#" + value
	}
	return value
}

func parseAlignment(value string) (ansifonts.TextAlignment, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "left", "leftalign":
		return ansifonts.LeftAlign, true
	case "center", "centeralign", "centre", "centrealign":
		return ansifonts.CenterAlign, true
	case "right", "rightalign":
		return ansifonts.RightAlign, true
	default:
		return ansifonts.LeftAlign, false
	}
}

func parseGradientDirection(value string) (ansifonts.GradientDirection, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "updown", "up-down":
		return ansifonts.UpDown, true
	case "downup", "down-up":
		return ansifonts.DownUp, true
	case "leftright", "left-right":
		return ansifonts.LeftRight, true
	case "rightleft", "right-left":
		return ansifonts.RightLeft, true
	default:
		return ansifonts.LeftRight, false
	}
}

func parseShadowStyle(value string) (ansifonts.ShadowStyle, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "light", "lightshade":
		return ansifonts.LightShade, true
	case "medium", "mediumshade":
		return ansifonts.MediumShade, true
	case "dark", "darkshade":
		return ansifonts.DarkShade, true
	default:
		return ansifonts.MediumShade, false
	}
}

func renderBlocks(slide Slide, width int) []Line {
	var lines []Line

	addSpacer := func() {
		if len(lines) == 0 || strings.TrimSpace(lines[len(lines)-1].Text) == "" {
			return
		}
		lines = append(lines, Line{Text: "", Kind: LineBody})
	}

	if block, ok := soloTextBlock(slide); ok {
		if ansiLines := renderAnsiLines(block.Text, width, ansiHeadingFont, ansiHeadingOptions); len(ansiLines) > 0 {
			big := centerBlockLines(ansiLines, width)
			big = alignBlockLines(big, width, ansiHeadingOptions.Alignment)
			for _, line := range big {
				lines = append(lines, Line{Text: line, Kind: LineHeadingANSI})
			}
		} else {
			big := centerBlockLines(renderBigText(block.Text, width), width)
			big = alignBlockLines(big, width, ansiHeadingOptions.Alignment)
			for _, line := range big {
				lines = append(lines, Line{Text: line, Kind: LineHeadingBlock})
			}
		}
		return lines
	}

	for _, block := range slide.Blocks {
		switch block.Kind {
		case BlockHeading:
			addSpacer()
			text := strings.TrimSpace(block.Text)
			if text == "" {
				continue
			}
			if block.Level <= 1 {
				if ansiLines := renderAnsiLines(text, width, ansiHeadingFont, ansiHeadingOptions); len(ansiLines) > 0 {
					big := centerBlockLines(ansiLines, width)
					big = alignBlockLines(big, width, ansiHeadingOptions.Alignment)
					for _, line := range big {
						lines = append(lines, Line{Text: line, Kind: LineHeadingANSI})
					}
				} else {
					big := centerBlockLines(renderBigText(text, width), width)
					big = alignBlockLines(big, width, ansiHeadingOptions.Alignment)
					for _, line := range big {
						lines = append(lines, Line{Text: line, Kind: LineHeadingBlock})
					}
				}
			} else if block.Level == 2 {
				if ansiLines := renderAnsiLines(text, width, ansiSubheadingFont, ansiSubheadingOptions); len(ansiLines) > 0 {
					big := centerBlockLines(ansiLines, width)
					big = alignBlockLines(big, width, ansiSubheadingOptions.Alignment)
					for _, line := range big {
						lines = append(lines, Line{Text: line, Kind: LineSubheadingANSI})
					}
				} else {
					lines = append(lines, Line{Text: alignLine(strings.ToUpper(text), width, ansiSubheadingOptions.Alignment), Kind: LineSubheading})
				}
			} else {
				lines = append(lines, Line{Text: centerLine(strings.ToUpper(text), width), Kind: LineSubheading})
			}
			lines = append(lines, Line{Text: "", Kind: LineBody})
		case BlockBullet:
			addSpacer()
			wrapped := wrapText(block.Text, max(1, width-4))
			for i, line := range wrapped {
				prefix := "- "
				if i > 0 {
					prefix = "  "
				}
				lines = append(lines, Line{Text: prefix + line, Kind: LineBody})
			}
		case BlockText:
			addSpacer()
			wrapped := wrapText(block.Text, width)
			for _, line := range wrapped {
				lines = append(lines, Line{Text: line, Kind: LineBody})
			}
		case BlockSpacer:
			lines = append(lines, Line{Text: "", Kind: LineBody})
		}
	}

	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1].Text) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		lines = append(lines, Line{Text: centerLine("(empty slide)", width), Kind: LineMuted})
	}
	return lines
}

func centerBlockLines(lines []string, width int) []string {
	if len(lines) == 0 || width <= 0 {
		return lines
	}
	maxWidth := 0
	for _, line := range lines {
		w := runeWidth(strings.TrimRight(line, " "))
		if w > maxWidth {
			maxWidth = w
		}
	}
	if maxWidth == 0 {
		return lines
	}
	if maxWidth > width {
		maxWidth = width
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		trimmed := strings.TrimRight(line, " ")
		trimmed = trimToWidth(trimmed, maxWidth)
		out[i] = padRight(trimmed, maxWidth)
	}
	return out
}

func lineHasText(row []rune) bool {
	for _, r := range row {
		if r != ' ' {
			return true
		}
	}
	return false
}

func lineIsANSI(kind LineKind) bool {
	return kind == LineHeadingANSI || kind == LineSubheadingANSI
}

func soloTextBlock(slide Slide) (Block, bool) {
	var found *Block
	for _, block := range slide.Blocks {
		if block.Kind == BlockSpacer {
			continue
		}
		if found != nil {
			return Block{}, false
		}
		if block.Kind != BlockText {
			return Block{}, false
		}
		b := block
		found = &b
	}
	if found == nil {
		return Block{}, false
	}
	return *found, true
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	line := ""
	lineWidth := 0
	for _, word := range words {
		wordWidth := runeWidth(word)
		if line == "" {
			line = word
			lineWidth = wordWidth
			continue
		}
		if lineWidth+1+wordWidth <= width {
			line = line + " " + word
			lineWidth += 1 + wordWidth
			continue
		}
		lines = append(lines, line)
		line = word
		lineWidth = wordWidth
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func renderBigText(text string, maxWidth int) []string {
	text = strings.ToUpper(text)
	lines := make([]string, bigFontHeight)
	for _, r := range text {
		glyph, ok := bigFont[r]
		if !ok {
			glyph = bigFont['?']
		}
		for i := 0; i < bigFontHeight; i++ {
			lines[i] += mapGlyphLine(glyph[i]) + " "
		}
	}
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
		lines[i] = trimToWidth(lines[i], maxWidth)
	}
	return lines
}

func trimToWidth(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}
	if runeWidth(text) <= maxWidth {
		return text
	}
	return ansi.Truncate(text, maxWidth, "")
}

func mapGlyphLine(line string) string {
	var b strings.Builder
	for _, r := range line {
		if r == '#' {
			b.WriteRune(bigFillRune)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func runeWidth(text string) int {
	return ansi.StringWidth(text)
}

func centerLine(text string, width int) string {
	if width <= 0 {
		return text
	}
	if runeWidth(text) >= width {
		return text
	}
	pad := (width - runeWidth(text)) / 2
	return strings.Repeat(" ", pad) + text
}

func alignLine(text string, width int, alignment ansifonts.TextAlignment) string {
	if width <= 0 {
		return text
	}
	w := runeWidth(text)
	if w >= width {
		return text
	}
	pad := 0
	switch alignment {
	case ansifonts.CenterAlign:
		pad = (width - w) / 2
	case ansifonts.RightAlign:
		pad = width - w
	default:
		pad = 0
	}
	return strings.Repeat(" ", pad) + text
}

func alignBlockLines(lines []string, width int, alignment ansifonts.TextAlignment) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = alignLine(line, width, alignment)
	}
	return out
}

func padRight(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if runeWidth(text) >= width {
		return text
	}
	return text + strings.Repeat(" ", width-runeWidth(text))
}

func placeText(row []rune, x int, text string) {
	if x < 0 {
		x = 0
	}
	for i, r := range []rune(text) {
		pos := x + i
		if pos < 0 || pos >= len(row) {
			break
		}
		row[pos] = r
	}
}

func placeTextRight(row []rune, rightEdge int, text string) {
	width := runeWidth(text)
	x := rightEdge - width
	if x < 0 {
		x = 0
	}
	placeText(row, x, text)
}

func buildBackground(w, h, frame int) [][]rune {
	buf := make([][]rune, h)
	for y := 0; y < h; y++ {
		row := make([]rune, w)
		for x := 0; x < w; x++ {
			row[x] = bgRune(x, y, frame)
		}
		buf[y] = row
	}
	return buf
}

func sliceBackground(bg [][]rune, startX, startY, width, height int) [][]rune {
	if width <= 0 || height <= 0 || len(bg) == 0 {
		return nil
	}
	if startY < 0 || startY+height > len(bg) {
		return nil
	}
	if startX < 0 {
		return nil
	}
	out := make([][]rune, height)
	for y := 0; y < height; y++ {
		row := bg[startY+y]
		if startX+width > len(row) {
			return nil
		}
		copyRow := make([]rune, width)
		copy(copyRow, row[startX:startX+width])
		out[y] = copyRow
	}
	return out
}

func bgRune(x, y, frame int) rune {
	if (x+frame/2)%16 == 0 {
		return '|'
	}
	if (y+frame/3)%8 == 0 {
		return '-'
	}
	if (x+y+frame/4)%23 == 0 {
		return '/'
	}
	seed := (x*31 + y*17 + frame*13) % 100
	if seed > 95 {
		return '*'
	}
	if seed > 85 {
		return '+'
	}
	if seed > 70 {
		return '.'
	}
	if seed > 55 {
		return ':'
	}
	return ' '
}

func overlaySphere(buf [][]rune, w, h, frame int) {
	if w < 20 || h < 10 {
		return
	}
	palette := []rune(".,:;=+*#%@")
	radius := float64(min(w, h)) * 0.26
	cx := float64(w)*0.28 + math.Sin(float64(frame)*0.02)*float64(w)*0.05
	cy := float64(h)*0.45 + math.Cos(float64(frame)*0.015)*float64(h)*0.04
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := (float64(x) - cx) / radius
			dy := (float64(y) - cy) / radius
			dist := dx*dx + dy*dy
			if dist > 1 {
				continue
			}
			shade := math.Sqrt(1 - dist)
			idx := int(shade * float64(len(palette)-1))
			if idx < 0 {
				idx = 0
			}
			if idx >= len(palette) {
				idx = len(palette) - 1
			}
			buf[y][x] = palette[idx]
		}
	}
}

func overlayMapField(buf [][]rune, w, h, frame int) {
	if w < 50 || h < 18 {
		return
	}
	fieldW := max(30, w/3)
	fieldH := max(12, h/3)
	x0 := w - fieldW - 3
	y0 := 2
	palette := []rune(" .,:;=+*#%@")
	zoom := 0.65 + 0.35*math.Sin(float64(frame)*0.012)
	scale := 0.08 + 0.06*zoom
	offsetX := float64(frame) * 0.04
	offsetY := float64(frame) * 0.03

	for y := 0; y < fieldH; y++ {
		for x := 0; x < fieldW; x++ {
			v := fieldValue(float64(x)+offsetX, float64(y)+offsetY, scale)
			idx := int(v * float64(len(palette)-1))
			if idx < 0 {
				idx = 0
			}
			if idx >= len(palette) {
				idx = len(palette) - 1
			}
			buf[y0+y][x0+x] = palette[idx]
		}
	}

	if fieldW > 6 && fieldH > 4 {
		for x := 0; x < fieldW; x++ {
			buf[y0][x0+x] = '='
			buf[y0+fieldH-1][x0+x] = '='
		}
		for y := 0; y < fieldH; y++ {
			buf[y0+y][x0] = '|'
			buf[y0+y][x0+fieldW-1] = '|'
		}
		buf[y0][x0] = '['
		buf[y0][x0+fieldW-1] = ']'
		buf[y0+fieldH-1][x0] = '['
		buf[y0+fieldH-1][x0+fieldW-1] = ']'

		label := " MAPSCAN "
		for i, r := range label {
			if 1+i >= fieldW-1 {
				break
			}
			buf[y0][x0+1+i] = r
		}
	}
}

func fieldValue(x, y, scale float64) float64 {
	fx := x * scale
	fy := y * scale
	v := math.Sin(fx*1.7+fy*1.3) + math.Sin(fx*0.7-fy*2.1) + math.Cos(fx*2.4+fy*1.9)
	return (v + 3) / 6
}

func overlayTorus(buf [][]rune, w, h, frame int) {
	palette := []rune(".,-~:;=!*#$@")
	zbuf := make([][]float64, h)
	for y := 0; y < h; y++ {
		zbuf[y] = make([]float64, w)
	}

	A := float64(frame) * 0.04
	B := float64(frame) * 0.02
	cx := float64(w) * 0.28
	cy := float64(h) * 0.72
	scaleX := float64(w) * 0.36
	scaleY := float64(h) * 0.22

	for theta := 0.0; theta < 2*math.Pi; theta += 0.06 {
		for phi := 0.0; phi < 2*math.Pi; phi += 0.018 {
			c := math.Sin(phi)
			d := math.Cos(theta)
			e := math.Sin(A)
			f := math.Sin(theta)
			g := math.Cos(A)
			hh := d + 2
			D := 1 / (c*hh*e + f*g + 5)
			l := math.Cos(phi)
			m := math.Cos(B)
			n := math.Sin(B)
			t := c*hh*g - f*e
			x := int(cx + scaleX*D*(l*hh*m-t*n))
			y := int(cy + scaleY*D*(l*hh*n+t*m))
			if x < 0 || x >= w || y < 0 || y >= h {
				continue
			}
			if D > zbuf[y][x] {
				zbuf[y][x] = D
				N := int(8 * ((f*e-c*d*g)*m - c*d*e - f*g - l*d*n))
				if N < 0 {
					N = 0
				}
				if N >= len(palette) {
					N = len(palette) - 1
				}
				buf[y][x] = palette[N]
			}
		}
	}
}

func stampWords(buf [][]rune, w, h, frame int) {
	words := []string{"HAXX0R", "DEMO", "GHOSTDECK", "NEON", "SYNTH", "KERNEL"}
	for i, word := range words {
		y := (i*3 + frame/6) % max(1, h)
		maxX := max(1, w-len(word)-1)
		x := (frame*2 + i*17) % maxX
		row := buf[y]
		for j, r := range word {
			if x+j >= 0 && x+j < w {
				row[x+j] = r
			}
		}
	}
}

func softenRune(r rune) rune {
	if r == ' ' {
		return ' '
	}
	switch r {
	case '$':
		return '#'
	case '%':
		return '+'
	default:
		return r
	}
}

func renderBgLine(chars []rune) string {
	if len(chars) == 0 {
		return ""
	}
	var b strings.Builder
	currentBright := isBright(chars[0])
	var seg strings.Builder
	flush := func(bright bool) {
		if seg.Len() == 0 {
			return
		}
		text := seg.String()
		seg.Reset()
		if bright {
			b.WriteString(bgBrightStyle.Render(text))
			return
		}
		b.WriteString(bgDimStyle.Render(text))
	}

	for _, r := range chars {
		bright := isBright(r)
		if bright != currentBright {
			flush(currentBright)
			currentBright = bright
		}
		seg.WriteRune(r)
	}
	flush(currentBright)
	return b.String()
}

func applyPanelBackgroundANSI(text string) string {
	bgCode := ansiBackgroundCode(string(colorPanelBg))
	if bgCode == "" {
		return text
	}
	reset := "\x1b[0m"
	withBg := strings.ReplaceAll(text, reset, reset+bgCode)
	return bgCode + withBg + reset
}

func ansiBackgroundCode(hex string) string {
	r, g, b, ok := parseHexColor(hex)
	if !ok {
		return ""
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

func parseHexColor(hex string) (int, int, int, bool) {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	r := int((v >> 16) & 0xFF)
	g := int((v >> 8) & 0xFF)
	b := int(v & 0xFF)
	return r, g, b, true
}

func isBright(r rune) bool {
	switch r {
	case '@', '#', '$', '*', '+', '%', '&', '=', '8', '0', 'O':
		return true
	default:
		return false
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const (
	bigFontHeight = 5
	bigFillRune   = '█'
	panelAspect   = 16.0 / 9.0
	cellAspectAdj = 2.0
)

var bigFont = map[rune][]string{
	' ': {"     ", "     ", "     ", "     ", "     "},
	'?': {" ### ", "#   #", "  ## ", "     ", "  #  "},
	'!': {"  #  ", "  #  ", "  #  ", "     ", "  #  "},
	'.': {"     ", "     ", "     ", "     ", "  #  "},
	':': {"     ", "  #  ", "     ", "  #  ", "     "},
	'-': {"     ", "     ", "#####", "     ", "     "},
	'/': {"    #", "   # ", "  #  ", " #   ", "#    "},
	'0': {" ### ", "#  ##", "# # #", "##  #", " ### "},
	'1': {"  #  ", " ##  ", "  #  ", "  #  ", " ### "},
	'2': {" ### ", "#   #", "   # ", "  #  ", "#####"},
	'3': {"#### ", "    #", " ### ", "    #", "#### "},
	'4': {"#   #", "#   #", "#####", "    #", "    #"},
	'5': {"#####", "#    ", "#### ", "    #", "#### "},
	'6': {" ### ", "#    ", "#### ", "#   #", " ### "},
	'7': {"#####", "   # ", "  #  ", " #   ", " #   "},
	'8': {" ### ", "#   #", " ### ", "#   #", " ### "},
	'9': {" ### ", "#   #", " ####", "    #", " ### "},
	'A': {" ### ", "#   #", "#####", "#   #", "#   #"},
	'B': {"#### ", "#   #", "#### ", "#   #", "#### "},
	'C': {" ####", "#    ", "#    ", "#    ", " ####"},
	'D': {"#### ", "#   #", "#   #", "#   #", "#### "},
	'E': {"#####", "#    ", "#### ", "#    ", "#####"},
	'F': {"#####", "#    ", "#### ", "#    ", "#    "},
	'G': {" ####", "#    ", "#  ##", "#   #", " ####"},
	'H': {"#   #", "#   #", "#####", "#   #", "#   #"},
	'I': {"#####", "  #  ", "  #  ", "  #  ", "#####"},
	'J': {"  ###", "   # ", "   # ", "#  # ", " ##  "},
	'K': {"#   #", "#  # ", "###  ", "#  # ", "#   #"},
	'L': {"#    ", "#    ", "#    ", "#    ", "#####"},
	'M': {"#   #", "## ##", "# # #", "#   #", "#   #"},
	'N': {"#   #", "##  #", "# # #", "#  ##", "#   #"},
	'O': {" ### ", "#   #", "#   #", "#   #", " ### "},
	'P': {"#### ", "#   #", "#### ", "#    ", "#    "},
	'Q': {" ### ", "#   #", "#   #", "#  ##", " ####"},
	'R': {"#### ", "#   #", "#### ", "#  # ", "#   #"},
	'S': {" ####", "#    ", " ### ", "    #", "#### "},
	'T': {"#####", "  #  ", "  #  ", "  #  ", "  #  "},
	'U': {"#   #", "#   #", "#   #", "#   #", " ### "},
	'V': {"#   #", "#   #", "#   #", " # # ", "  #  "},
	'W': {"#   #", "#   #", "# # #", "## ##", "#   #"},
	'X': {"#   #", " # # ", "  #  ", " # # ", "#   #"},
	'Y': {"#   #", " # # ", "  #  ", "  #  ", "  #  "},
	'Z': {"#####", "   # ", "  #  ", " #   ", "#####"},
}
