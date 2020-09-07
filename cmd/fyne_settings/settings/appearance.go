package settings

import (
	"encoding/json"
	"image/color"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
)

const (
	systemThemeName = "system default"
)

var (
	colorNames = []string{theme.COLOR_BLUE, theme.COLOR_GREEN, theme.COLOR_YELLOW,
		theme.COLOR_ORANGE, theme.COLOR_RED, theme.COLOR_GREY}
	colorValues = map[string]color.Color{
		theme.COLOR_BLUE:   color.NRGBA{R: 0x21, G: 0x96, B: 0xf3, A: 0xff},
		theme.COLOR_GREEN:  color.NRGBA{R: 0x8b, G: 0xc3, B: 0x4a, A: 0xff},
		theme.COLOR_YELLOW: color.NRGBA{R: 0xff, G: 0xeb, B: 0x3b, A: 0xff},
		theme.COLOR_ORANGE: color.NRGBA{R: 0xff, G: 0x98, B: 0x00, A: 0xff},
		theme.COLOR_RED:    color.NRGBA{R: 0xf4, G: 0x43, B: 0x36, A: 0xff},
		theme.COLOR_GREY:   color.NRGBA{R: 0x9e, G: 0x9e, B: 0x9e, A: 0xff},
	}
)

// Settings gives access to user interfaces to control Fyne settings
type Settings struct {
	fyneSettings app.SettingsSchema

	preview *canvas.Image
	colors  []fyne.CanvasObject
}

// NewSettings returns a new settings instance with the current configuration loaded
func NewSettings() *Settings {
	s := &Settings{}
	s.load()

	return s
}

// AppearanceIcon returns the icon for appearance settings
func (s *Settings) AppearanceIcon() fyne.Resource {
	return theme.NewThemedResource(appearanceIcon, nil)
}

// LoadAppearanceScreen creates a new settings screen to handle appearance configuration
func (s *Settings) LoadAppearanceScreen(w fyne.Window) fyne.CanvasObject {
	s.preview = canvas.NewImageFromResource(themeDarkPreview)
	s.preview.FillMode = canvas.ImageFillContain

	def := s.fyneSettings.ThemeName
	themeNames := []string{"dark", "light"}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		themeNames = append(themeNames, systemThemeName)
		if s.fyneSettings.ThemeName == "" {
			def = systemThemeName
		}
	}
	themes := widget.NewSelect(themeNames, s.chooseTheme)
	themes.SetSelected(def)

	scale := s.makeScaleGroup(w.Canvas().Scale())

	current := s.fyneSettings.PrimaryColor
	if current == "" {
		current = theme.COLOR_BLUE
	}
	for _, c := range colorNames {
		b := newColorButton(c, colorValues[c], s)
		s.colors = append(s.colors, b)
	}
	swatch := fyne.NewContainerWithLayout(layout.NewGridLayout(6), s.colors...)

	scale.Append(widget.NewGroup("Main Color", swatch))
	scale.Append(widget.NewGroup("Theme", themes))

	bottom := widget.NewHBox(layout.NewSpacer(),
		&widget.Button{Text: "Apply", Style: widget.PrimaryButton, OnTapped: func() {
			err := s.save()
			if err != nil {
				fyne.LogError("Failed on saving", err)
			}

			s.appliedScale(s.fyneSettings.Scale)
		}})

	return fyne.NewContainerWithLayout(layout.NewBorderLayout(scale, bottom, nil, nil),
		scale, bottom, s.preview)
}

func (s *Settings) chooseTheme(name string) {
	if name == systemThemeName {
		name = ""
	}
	s.fyneSettings.ThemeName = name

	switch name {
	case "light":
		s.preview.Resource = themeLightPreview
	default:
		s.preview.Resource = themeDarkPreview
	}
	canvas.Refresh(s.preview)
}

func (s *Settings) load() {
	err := s.loadFromFile(s.fyneSettings.StoragePath())
	if err != nil {
		fyne.LogError("Settings load error:", err)
	}
}

func (s *Settings) loadFromFile(path string) error {
	file, err := os.Open(path) // #nosec
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(filepath.Dir(path), 0700)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	decode := json.NewDecoder(file)

	return decode.Decode(&s.fyneSettings)
}

func (s *Settings) save() error {
	return s.saveToFile(s.fyneSettings.StoragePath())
}

func (s *Settings) saveToFile(path string) error {
	err := os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil { // this is not an exists error according to docs
		return err
	}

	data, err := json.Marshal(&s.fyneSettings)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 0644)
}

type colorButton struct {
	widget.BaseWidget
	name    string
	color   color.Color
	current bool

	s *Settings
}

func newColorButton(n string, c color.Color, s *Settings) *colorButton {
	b := &colorButton{name: n, color: c, s: s}
	b.ExtendBaseWidget(b)
	return b
}

func (c *colorButton) CreateRenderer() fyne.WidgetRenderer {
	r := canvas.NewRectangle(color.Transparent)
	r.StrokeWidth = 5

	if c.name == c.s.fyneSettings.PrimaryColor {
		r.StrokeColor = theme.PrimaryColor()
	}

	return &colorRenderer{c: c, rect: r, objs: []fyne.CanvasObject{r}}
}

func (c *colorButton) Tapped(_ *fyne.PointEvent) {
	c.s.fyneSettings.PrimaryColor = c.name
	for _, child := range c.s.colors {
		child.Refresh()
	}
}

type colorRenderer struct {
	c    *colorButton
	rect *canvas.Rectangle
	objs []fyne.CanvasObject
}

func (c *colorRenderer) Layout(s fyne.Size) {
	c.rect.Resize(s)
}

func (c *colorRenderer) MinSize() fyne.Size {
	return fyne.NewSize(20, 20)
}

func (c *colorRenderer) Refresh() {
	if c.c.name == c.c.s.fyneSettings.PrimaryColor {
		c.rect.StrokeColor = theme.PrimaryColor()
	} else {
		c.rect.StrokeColor = color.Transparent
	}

	c.rect.Refresh()
}

func (c *colorRenderer) BackgroundColor() color.Color {
	return c.c.color
}

func (c *colorRenderer) Objects() []fyne.CanvasObject {
	return c.objs
}

func (c *colorRenderer) Destroy() {
}
