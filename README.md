GhostDeck TUI Slide Deck
========================

Neon/cyberpunk terminal slide presenter for internal AI demos.

Requirements
------------
- Go 1.25+
- Terminal font: Fira Code Nerd Font (recommended)

Quick start
-----------
1) Update `slides.md` (use `---` to separate slides)
2) Run:

```bash
go mod tidy
go run . -file slides.md
```

Controls
--------
- next: right / l / n / space / enter
- prev: left / h / p / backspace
- quit: q / ctrl+c

Markdown rules
--------------
- `---` starts a new slide
- `#` and `##` headings render with ANSI art fonts
- `-` bullets render as list items
- A slide with only one text block auto-renders as a big H1

Frontmatter
-----------
Optional YAML frontmatter at the top of `slides.md` lets you choose fonts:

```yaml
---
h1Font: pixeloperator
h2Font: fivebyfive
---
```

You can also override any `ansifonts` render options per heading:

```yaml
---
h1Font: pixeloperator
h2Font: fivebyfive
h1Options:
  alignment: center
  textColor: "#ff4fd8"
  scaleFactor: 1.0
  charSpacing: 1
  uppercase: true
h2Options:
  alignment: left
  textColor: "#00f0ff"
  useGradient: true
  gradientColor: "#5effff"
  gradientDirection: leftRight
  shadowEnabled: true
  shadowHorizontalOffset: 1
  shadowVerticalOffset: 1
  shadowStyle: dark
  uppercase: true
---
```

Notes
-----
- The presenter keeps a 16:9 content area centered with an inset.
- Background uses animated ASCII effects and a rotating torus.
