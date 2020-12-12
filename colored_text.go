package helpers

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	NoColorCode RGBCode = 0xFFFFFFFF
	// NoColor means that content does not have any color of its own and get color of its contex
	NoColor   NoColorT    = false
	TTY       TTYContext  = true
	MonoColor TTYContext  = false
	HTML      HTMLContext = true
)

//region RGBCode: RGB representation of a color
type RGBCode uint32

func (this RGBCode) Red() uint8     { return uint8((this >> 16) & 0xFF) }
func (this RGBCode) Green() uint8   { return uint8((this >> 8) & 0xFF) }
func (this RGBCode) Blue() uint8    { return uint8((this >> 0) & 0xFF) }
func (this RGBCode) String() string { return fmt.Sprintf("#%X", uint32(this&0xFFFFFF)) }
func (this RGBCode) ToColor() Color {
	if this == NoColorCode {
		return NoColor
	}
	if (this & 0xFF000000) != 0 {
		panic("Invalid color code")
	}
	return RGBColor(this)
}

// endregion

type ColorCoverage int

const (
	NoCoverage ColorCoverage = iota
	Foreground
	Background
	Both
)

type ColorName struct {
	Foreground string
	Background string
}

func (this ColorName) IsEmpty() bool { return this.Foreground == "" && this.Background == "" }

type Color interface {
	Code() RGBCode
	Coverage() ColorCoverage
	AsForeground() Color
	AsBackground() Color
	HtmlColorName() ColorName
	TerminalColorName() ColorName
}

//region NoColorT: Implementation of a nil value for ``Color`` interface
type NoColorT bool

func (this NoColorT) Code() RGBCode                { return NoColorCode }
func (this NoColorT) Coverage() ColorCoverage      { return NoCoverage }
func (this NoColorT) AsForeground() Color          { return this }
func (this NoColorT) AsBackground() Color          { return this }
func (this NoColorT) HtmlColorName() ColorName     { return ColorName{} }
func (this NoColorT) TerminalColorName() ColorName { return ColorName{} }

//endregion

//region RGBColor: Implementation of a ``Color`` that work with an ``RGBCode``
type RGBColor uint32

func (this RGBColor) Code() RGBCode       { return RGBCode(uint32(this & 0xFFFFFF)) }
func (this RGBColor) AsForeground() Color { return this & 0xFFFFFF }
func (this RGBColor) AsBackground() Color { return RGBColor(this | 0x80000000) }
func (this RGBColor) Coverage() ColorCoverage {
	if (this & 0x80000000) != 0 {
		return Background
	} else {
		return Foreground
	}
}
func (this RGBColor) HtmlColorName() ColorName {
	htmlColorName := ""
	if htmlColorName = GetColorNameByCode(this.Code()); htmlColorName == "" {
		htmlColorName = this.Code().String()
	}
	if this.Coverage() == Background {
		return ColorName{Background: htmlColorName}
	}
	return ColorName{Foreground: htmlColorName}
}
func (this RGBColor) TerminalColorName() ColorName {
	code := this.Code()
	if this.Coverage() == Background {
		return ColorName{Background: fmt.Sprintf("48;2;%d;%d;%d", code.Red(), code.Green(), code.Blue())}
	} else {
		return ColorName{Foreground: fmt.Sprintf("38;2;%d;%d;%d", code.Red(), code.Green(), code.Blue())}
	}
}

//endregion

//region MixedColor
type MixedColor struct {
	foreground Color
	background Color
}

func MixColors(foreground, background Color) MixedColor {
	return MixedColor{
		foreground: foreground.AsForeground(),
		background: background.AsBackground(),
	}
}

func (this MixedColor) Coverage() ColorCoverage { return Both }
func (this MixedColor) Code() RGBCode           { return this.foreground.Code() }
func (this MixedColor) AsForeground() Color     { return this.foreground }
func (this MixedColor) AsBackground() Color     { return this.background }
func (this MixedColor) HtmlColorName() ColorName {
	return ColorName{
		Foreground: this.foreground.HtmlColorName().Foreground,
		Background: this.background.HtmlColorName().Background,
	}
}
func (this MixedColor) TerminalColorName() ColorName {
	return ColorName{
		Foreground: this.foreground.TerminalColorName().Foreground,
		Background: this.background.TerminalColorName().Background,
	}
}

//endregion

//region ColoredWriter
type ColoredWriter struct {
	context ColorContext
	w       io.Writer
	color   Color
}

func NewColoredWriterWithColor(context ColorContext, w io.Writer, color Color) *ColoredWriter {
	return &ColoredWriter{context: context, w: w, color: color}
}
func NewColoredWriter(context ColorContext, w io.Writer) *ColoredWriter {
	return NewColoredWriterWithColor(context, w, NoColor)
}
func (this *ColoredWriter) GetWriter() io.Writer { return this.w }
func (this *ColoredWriter) GetColor() Color      { return this.color }
func (this *ColoredWriter) SetColor(color Color) (oldColor Color) {
	oldColor = this.color
	if color.Code() != NoColorCode {
		this.color = color
	}
	return oldColor
}
func (this *ColoredWriter) Write(b []byte) error {
	return this.context.Write(this, b)
}
func (this *ColoredWriter) WriteString(s string) error { return this.Write([]byte(s)) }
func (this *ColoredWriter) WriteContent(content interface{}) error {
	if content == nil {
		content = "<nil>"
	}
	if buf, ok := content.([]byte); ok {
		return this.Write(buf)
	} else if s, ok := content.(string); ok {
		return this.WriteString(s)
	} else if cc, ok := content.(ColoredContent); ok {
		return cc.Render(this)
	} else {
		s := fmt.Sprintf("%v", content)
		return this.WriteString(s)
	}
}

//endregion

type ColoredContent interface {
	Render(w *ColoredWriter) error
}

//region ColoredValue: a simple value that bind a content with a ``Color``
type ColoredValue struct {
	Color   Color
	Content interface{}
}

func (this ColoredValue) Render(w *ColoredWriter) error {
	oldColor := w.SetColor(this.Color)
	defer w.SetColor(oldColor)

	return w.WriteContent(this.Content)
}

//endregion

//region FormatContent: A formatter that support ``ColoredContent`` as its argument
type FormatContent FormatInfo

func CreateFormatContent(format string, args ...interface{}) FormatContent {
	result := ParseFormatString(format, args...)
	return FormatContent(result)
}
func (this FormatContent) Render(w *ColoredWriter) error {
	for i := 0; i < len(this); i++ {
		var err error
		node := this[i]
		if node.FormatString == "" {
			err = w.WriteContent(node.Arg)
		} else if node.NoArg {
			value := node.Format()
			err = w.WriteContent(value)
		} else if ccontent, ok := node.Arg.(ColoredContent); ok {
			err = w.WriteContent(ccontent)
		} else {
			value := node.Format()
			err = w.WriteString(value)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

//endregion

type ColorContext interface {
	Name() string
	Write(w *ColoredWriter, b []byte) error
}

//region TTYContext: A ``ColorContext`` that support ``TTY`` coloring and ``MonoColor``
type TTYContext bool

var (
	ttyStartColor = []byte("\033[")
	ttyEndColor   = []byte{'m'}
	ttyResetColor = []byte("\033[0m")
)

func writeTerminalColor(w io.Writer, color string) error {
	if _, err := w.Write(ttyStartColor); err != nil {
		return err
	}
	if _, err := w.Write([]byte(color)); err != nil {
		return err
	}

	if _, err := w.Write(ttyEndColor); err != nil {
		return err
	}

	return nil
}

func (this TTYContext) Name() string {
	if this {
		return "TTY"
	} else {
		return "MonoColor"
	}
}
func (this TTYContext) Write(w *ColoredWriter, b []byte) error {
	var err error
	requireReset := false
	if this {
		if clr := w.GetColor().TerminalColorName(); !clr.IsEmpty() {
			requireReset = true

			if clr.Foreground != "" {
				if err = writeTerminalColor(w.GetWriter(), clr.Foreground); err != nil {
					return err
				}
			}
			if clr.Background != "" {
				if err = writeTerminalColor(w.GetWriter(), clr.Background); err != nil {
					return err
				}
			}
		}
	}

	if _, err = w.GetWriter().Write(b); err != nil {
		return err
	}

	if requireReset {
		_, err = w.GetWriter().Write(ttyResetColor)
		return err
	}
	return nil
}

//endregion

//region HTMLContext: a ``ColorContext`` that support HTML coloring
type HTMLContext bool

var (
	htmlColorStartFormat = `<span style="%s">`
	htmlEndColor         = []byte("</span>")
)

func (this HTMLContext) Name() string { return "HTML" }
func (this HTMLContext) Write(w *ColoredWriter, b []byte) error {
	var err error
	requireReset := false
	if clr := w.GetColor().HtmlColorName(); !clr.IsEmpty() {
		requireReset = true
		clrHeader := `<span style="`
		if clr.Foreground != "" {
			clrHeader += "color: " + clr.Foreground
		}
		if clr.Background != "" {
			clrHeader += "background-color: " + clr.Background
		}
		clrHeader += `">`
		if _, err = w.GetWriter().Write([]byte(clrHeader)); err != nil {
			return err
		}
	}

	if _, err = w.GetWriter().Write(b); err != nil {
		return err
	}

	if requireReset {
		_, err = w.GetWriter().Write(htmlEndColor)
		return err
	}
	return nil
}

//endregion

// Get default context that must used to write content to a writer.
// This will return ``TTY`` if w is a TTY and ``MonoColor`` otherwise
func GetDefaultContext(w io.Writer) ColorContext {
	if f, ok := w.(*os.File); ok && IsTerminal(f) {
		return TTY
	} else {
		return MonoColor
	}
}

// CContent Make a content colored, so you may write it to a ColorContext
func CContent(color Color, content interface{}) ColoredValue {
	if color == nil {
		color = NoColor
	}
	if cv, ok := content.(ColoredValue); ok {
		return ColoredValue{
			Color:   color,
			Content: cv.Content,
		}
	}
	return ColoredValue{
		Color:   color,
		Content: content,
	}
}

// CFormat Return a formatted value that can be written to a ColorContext as colored.
// Most important thing here is it will keep the color of arguments
func CFormat(color Color, format string, args ...interface{}) ColoredValue {
	if color == nil {
		color = NoColor
	}
	return ColoredValue{
		Color:   color,
		Content: CreateFormatContent(format, args...),
	}
}

// CWrite write a content to ``w`` using ``context`` or default context of ``w``
func CWrite(w io.Writer, content interface{}, context ColorContext) error {
	if context == nil {
		context = GetDefaultContext(w)
	}

	cw := NewColoredWriter(context, w)
	return cw.WriteContent(content)
}

// CWritec write a content with specified color to ``w`` using ``context`` or default context of ``w``
func CWritec(w io.Writer, color Color, content interface{}, context ColorContext) error {
	if context == nil {
		context = GetDefaultContext(w)
	}

	cw := NewColoredWriterWithColor(context, w, color)
	return cw.WriteContent(content)
}

// CWritef write a formatted content to ``w``
func CWritef(w io.Writer, context ColorContext, format string, args ...interface{}) error {
	return CWrite(w, CreateFormatContent(format, args...), context)
}

// CWritefc write a formatted content with specified color to ``w``
func CWritefc(w io.Writer, context ColorContext, color Color, format string, args ...interface{}) error {
	return CWrite(w, CFormat(color, format, args...), context)
}

const (
	AliceBlue            RGBColor = 0xF0F8FF
	AntiqueWhite         RGBColor = 0xFAEBD7
	Aqua                 RGBColor = 0x00FFFF
	Aquamarine           RGBColor = 0x7FFFD4
	Azure                RGBColor = 0xF0FFFF
	Beige                RGBColor = 0xF5F5DC
	Bisque               RGBColor = 0xFFE4C4
	Black                RGBColor = 0x000000
	BlanchedAlmond       RGBColor = 0xFFEBCD
	Blue                 RGBColor = 0x0000FF
	BlueViolet           RGBColor = 0x8A2BE2
	Brown                RGBColor = 0xA52A2A
	BurlyWood            RGBColor = 0xDEB887
	CadetBlue            RGBColor = 0x5F9EA0
	Chartreuse           RGBColor = 0x7FFF00
	Chocolate            RGBColor = 0xD2691E
	Coral                RGBColor = 0xFF7F50
	CornflowerBlue       RGBColor = 0x6495ED
	Cornsilk             RGBColor = 0xFFF8DC
	Crimson              RGBColor = 0xDC143C
	Cyan                 RGBColor = 0x00FFFF
	DarkBlue             RGBColor = 0x00008B
	DarkCyan             RGBColor = 0x008B8B
	DarkGoldenRod        RGBColor = 0xB8860B
	DarkGray             RGBColor = 0xA9A9A9
	DarkGrey             RGBColor = 0xA9A9A9
	DarkGreen            RGBColor = 0x006400
	DarkKhaki            RGBColor = 0xBDB76B
	DarkMagenta          RGBColor = 0x8B008B
	DarkOliveGreen       RGBColor = 0x556B2F
	DarkOrange           RGBColor = 0xFF8C00
	DarkOrchid           RGBColor = 0x9932CC
	DarkRed              RGBColor = 0x8B0000
	DarkSalmon           RGBColor = 0xE9967A
	DarkSeaGreen         RGBColor = 0x8FBC8F
	DarkSlateBlue        RGBColor = 0x483D8B
	DarkSlateGray        RGBColor = 0x2F4F4F
	DarkSlateGrey        RGBColor = 0x2F4F4F
	DarkTurquoise        RGBColor = 0x00CED1
	DarkViolet           RGBColor = 0x9400D3
	DeepPink             RGBColor = 0xFF1493
	DeepSkyBlue          RGBColor = 0x00BFFF
	DimGray              RGBColor = 0x696969
	DimGrey              RGBColor = 0x696969
	DodgerBlue           RGBColor = 0x1E90FF
	FireBrick            RGBColor = 0xB22222
	FloralWhite          RGBColor = 0xFFFAF0
	ForestGreen          RGBColor = 0x228B22
	Fuchsia              RGBColor = 0xFF00FF
	Gainsboro            RGBColor = 0xDCDCDC
	GhostWhite           RGBColor = 0xF8F8FF
	Gold                 RGBColor = 0xFFD700
	GoldenRod            RGBColor = 0xDAA520
	Gray                 RGBColor = 0x808080
	Grey                 RGBColor = 0x808080
	Green                RGBColor = 0x008000
	GreenYellow          RGBColor = 0xADFF2F
	HoneyDew             RGBColor = 0xF0FFF0
	HotPink              RGBColor = 0xFF69B4
	IndianRed            RGBColor = 0xCD5C5C
	Indigo               RGBColor = 0x4B0082
	Ivory                RGBColor = 0xFFFFF0
	Khaki                RGBColor = 0xF0E68C
	Lavender             RGBColor = 0xE6E6FA
	LavenderBlush        RGBColor = 0xFFF0F5
	LawnGreen            RGBColor = 0x7CFC00
	LemonChiffon         RGBColor = 0xFFFACD
	LightBlue            RGBColor = 0xADD8E6
	LightCoral           RGBColor = 0xF08080
	LightCyan            RGBColor = 0xE0FFFF
	LightGoldenRodYellow RGBColor = 0xFAFAD2
	LightGray            RGBColor = 0xD3D3D3
	LightGrey            RGBColor = 0xD3D3D3
	LightGreen           RGBColor = 0x90EE90
	LightPink            RGBColor = 0xFFB6C1
	LightSalmon          RGBColor = 0xFFA07A
	LightSeaGreen        RGBColor = 0x20B2AA
	LightSkyBlue         RGBColor = 0x87CEFA
	LightSlateGray       RGBColor = 0x778899
	LightSlateGrey       RGBColor = 0x778899
	LightSteelBlue       RGBColor = 0xB0C4DE
	LightYellow          RGBColor = 0xFFFFE0
	Lime                 RGBColor = 0x00FF00
	LimeGreen            RGBColor = 0x32CD32
	Linen                RGBColor = 0xFAF0E6
	Magenta              RGBColor = 0xFF00FF
	Maroon               RGBColor = 0x800000
	MediumAquaMarine     RGBColor = 0x66CDAA
	MediumBlue           RGBColor = 0x0000CD
	MediumOrchid         RGBColor = 0xBA55D3
	MediumPurple         RGBColor = 0x9370DB
	MediumSeaGreen       RGBColor = 0x3CB371
	MediumSlateBlue      RGBColor = 0x7B68EE
	MediumSpringGreen    RGBColor = 0x00FA9A
	MediumTurquoise      RGBColor = 0x48D1CC
	MediumVioletRed      RGBColor = 0xC71585
	MidnightBlue         RGBColor = 0x191970
	MintCream            RGBColor = 0xF5FFFA
	MistyRose            RGBColor = 0xFFE4E1
	Moccasin             RGBColor = 0xFFE4B5
	NavajoWhite          RGBColor = 0xFFDEAD
	Navy                 RGBColor = 0x000080
	OldLace              RGBColor = 0xFDF5E6
	Olive                RGBColor = 0x808000
	OliveDrab            RGBColor = 0x6B8E23
	Orange               RGBColor = 0xFFA500
	OrangeRed            RGBColor = 0xFF4500
	Orchid               RGBColor = 0xDA70D6
	PaleGoldenRod        RGBColor = 0xEEE8AA
	PaleGreen            RGBColor = 0x98FB98
	PaleTurquoise        RGBColor = 0xAFEEEE
	PaleVioletRed        RGBColor = 0xDB7093
	PapayaWhip           RGBColor = 0xFFEFD5
	PeachPuff            RGBColor = 0xFFDAB9
	Peru                 RGBColor = 0xCD853F
	Pink                 RGBColor = 0xFFC0CB
	Plum                 RGBColor = 0xDDA0DD
	PowderBlue           RGBColor = 0xB0E0E6
	Purple               RGBColor = 0x800080
	RebeccaPurple        RGBColor = 0x663399
	Red                  RGBColor = 0xFF0000
	RosyBrown            RGBColor = 0xBC8F8F
	RoyalBlue            RGBColor = 0x4169E1
	SaddleBrown          RGBColor = 0x8B4513
	Salmon               RGBColor = 0xFA8072
	SandyBrown           RGBColor = 0xF4A460
	SeaGreen             RGBColor = 0x2E8B57
	SeaShell             RGBColor = 0xFFF5EE
	Sienna               RGBColor = 0xA0522D
	Silver               RGBColor = 0xC0C0C0
	SkyBlue              RGBColor = 0x87CEEB
	SlateBlue            RGBColor = 0x6A5ACD
	SlateGray            RGBColor = 0x708090
	SlateGrey            RGBColor = 0x708090
	Snow                 RGBColor = 0xFFFAFA
	SpringGreen          RGBColor = 0x00FF7F
	SteelBlue            RGBColor = 0x4682B4
	Tan                  RGBColor = 0xD2B48C
	Teal                 RGBColor = 0x008080
	Thistle              RGBColor = 0xD8BFD8
	Tomato               RGBColor = 0xFF6347
	Turquoise            RGBColor = 0x40E0D0
	Violet               RGBColor = 0xEE82EE
	Wheat                RGBColor = 0xF5DEB3
	White                RGBColor = 0xFFFFFF
	WhiteSmoke           RGBColor = 0xF5F5F5
	Yellow               RGBColor = 0xFFFF00
	YellowGreen          RGBColor = 0x9ACD32
)

type ColorNameMap struct {
	colorNamesByCode map[RGBCode]string
	colorsByName     map[string]RGBCode
}

func NewColorNameMap(colorNamesByCode map[RGBCode]string) *ColorNameMap {
	result := &ColorNameMap{
		colorNamesByCode: make(map[RGBCode]string),
		colorsByName:     make(map[string]RGBCode),
	}

	if colorNamesByCode != nil {
		for code, name := range colorNamesByCode {
			result.SetColorCodeName(code, name)
		}
	}

	return result
}
func (this *ColorNameMap) GetColorNameByCode(code RGBCode) string {
	if name, ok := this.colorNamesByCode[code]; ok {
		return name
	}
	return ""
}
func (this *ColorNameMap) GetColorCodeByName(name string) RGBCode {
	iname := strings.ToLower(name)
	if code, ok := this.colorsByName[iname]; ok {
		return code
	}
	return NoColorCode
}
func (this *ColorNameMap) SetColorCodeName(code RGBCode, name string) *ColorNameMap {
	this.colorNamesByCode[code] = name
	iname := strings.ToLower(name)
	this.colorsByName[iname] = code
	return this
}
func (this *ColorNameMap) AddName(name string, code RGBCode) *ColorNameMap {
	iname := strings.ToLower(name)
	this.colorsByName[iname] = code
	return this
}
func (this *ColorNameMap) Clone() *ColorNameMap {
	result := NewColorNameMap(nil)
	for code, name := range this.colorNamesByCode {
		result.colorNamesByCode[code] = name
	}
	for name, code := range this.colorsByName {
		result.colorsByName[name] = code
	}
	return result
}

var globalColorMap = NewColorNameMap(map[RGBCode]string{
	AliceBlue.Code():            "AliceBlue",
	AntiqueWhite.Code():         "AntiqueWhite",
	Aqua.Code():                 "Aqua",
	Aquamarine.Code():           "Aquamarine",
	Azure.Code():                "Azure",
	Beige.Code():                "Beige",
	Bisque.Code():               "Bisque",
	Black.Code():                "Black",
	BlanchedAlmond.Code():       "BlanchedAlmond",
	Blue.Code():                 "Blue",
	BlueViolet.Code():           "BlueViolet",
	Brown.Code():                "Brown",
	BurlyWood.Code():            "BurlyWood",
	CadetBlue.Code():            "CadetBlue",
	Chartreuse.Code():           "Chartreuse",
	Chocolate.Code():            "Chocolate",
	Coral.Code():                "Coral",
	CornflowerBlue.Code():       "CornflowerBlue",
	Cornsilk.Code():             "Cornsilk",
	Crimson.Code():              "Crimson",
	Cyan.Code():                 "Cyan",
	DarkBlue.Code():             "DarkBlue",
	DarkCyan.Code():             "DarkCyan",
	DarkGoldenRod.Code():        "DarkGoldenRod",
	DarkGray.Code():             "DarkGray",
	DarkGrey.Code():             "DarkGrey",
	DarkGreen.Code():            "DarkGreen",
	DarkKhaki.Code():            "DarkKhaki",
	DarkMagenta.Code():          "DarkMagenta",
	DarkOliveGreen.Code():       "DarkOliveGreen",
	DarkOrange.Code():           "DarkOrange",
	DarkOrchid.Code():           "DarkOrchid",
	DarkRed.Code():              "DarkRed",
	DarkSalmon.Code():           "DarkSalmon",
	DarkSeaGreen.Code():         "DarkSeaGreen",
	DarkSlateBlue.Code():        "DarkSlateBlue",
	DarkSlateGray.Code():        "DarkSlateGray",
	DarkSlateGrey.Code():        "DarkSlateGrey",
	DarkTurquoise.Code():        "DarkTurquoise",
	DarkViolet.Code():           "DarkViolet",
	DeepPink.Code():             "DeepPink",
	DeepSkyBlue.Code():          "DeepSkyBlue",
	DimGray.Code():              "DimGray",
	DimGrey.Code():              "DimGrey",
	DodgerBlue.Code():           "DodgerBlue",
	FireBrick.Code():            "FireBrick",
	FloralWhite.Code():          "FloralWhite",
	ForestGreen.Code():          "ForestGreen",
	Fuchsia.Code():              "Fuchsia",
	Gainsboro.Code():            "Gainsboro",
	GhostWhite.Code():           "GhostWhite",
	Gold.Code():                 "Gold",
	GoldenRod.Code():            "GoldenRod",
	Gray.Code():                 "Gray",
	Grey.Code():                 "Grey",
	Green.Code():                "Green",
	GreenYellow.Code():          "GreenYellow",
	HoneyDew.Code():             "HoneyDew",
	HotPink.Code():              "HotPink",
	IndianRed.Code():            "IndianRed",
	Indigo.Code():               "Indigo",
	Ivory.Code():                "Ivory",
	Khaki.Code():                "Khaki",
	Lavender.Code():             "Lavender",
	LavenderBlush.Code():        "LavenderBlush",
	LawnGreen.Code():            "LawnGreen",
	LemonChiffon.Code():         "LemonChiffon",
	LightBlue.Code():            "LightBlue",
	LightCoral.Code():           "LightCoral",
	LightCyan.Code():            "LightCyan",
	LightGoldenRodYellow.Code(): "LightGoldenRodYellow",
	LightGray.Code():            "LightGray",
	LightGrey.Code():            "LightGrey",
	LightGreen.Code():           "LightGreen",
	LightPink.Code():            "LightPink",
	LightSalmon.Code():          "LightSalmon",
	LightSeaGreen.Code():        "LightSeaGreen",
	LightSkyBlue.Code():         "LightSkyBlue",
	LightSlateGray.Code():       "LightSlateGray",
	LightSlateGrey.Code():       "LightSlateGrey",
	LightSteelBlue.Code():       "LightSteelBlue",
	LightYellow.Code():          "LightYellow",
	Lime.Code():                 "Lime",
	LimeGreen.Code():            "LimeGreen",
	Linen.Code():                "Linen",
	Magenta.Code():              "Magenta",
	Maroon.Code():               "Maroon",
	MediumAquaMarine.Code():     "MediumAquaMarine",
	MediumBlue.Code():           "MediumBlue",
	MediumOrchid.Code():         "MediumOrchid",
	MediumPurple.Code():         "MediumPurple",
	MediumSeaGreen.Code():       "MediumSeaGreen",
	MediumSlateBlue.Code():      "MediumSlateBlue",
	MediumSpringGreen.Code():    "MediumSpringGreen",
	MediumTurquoise.Code():      "MediumTurquoise",
	MediumVioletRed.Code():      "MediumVioletRed",
	MidnightBlue.Code():         "MidnightBlue",
	MintCream.Code():            "MintCream",
	MistyRose.Code():            "MistyRose",
	Moccasin.Code():             "Moccasin",
	NavajoWhite.Code():          "NavajoWhite",
	Navy.Code():                 "Navy",
	OldLace.Code():              "OldLace",
	Olive.Code():                "Olive",
	OliveDrab.Code():            "OliveDrab",
	Orange.Code():               "Orange",
	OrangeRed.Code():            "OrangeRed",
	Orchid.Code():               "Orchid",
	PaleGoldenRod.Code():        "PaleGoldenRod",
	PaleGreen.Code():            "PaleGreen",
	PaleTurquoise.Code():        "PaleTurquoise",
	PaleVioletRed.Code():        "PaleVioletRed",
	PapayaWhip.Code():           "PapayaWhip",
	PeachPuff.Code():            "PeachPuff",
	Peru.Code():                 "Peru",
	Pink.Code():                 "Pink",
	Plum.Code():                 "Plum",
	PowderBlue.Code():           "PowderBlue",
	Purple.Code():               "Purple",
	RebeccaPurple.Code():        "RebeccaPurple",
	Red.Code():                  "Red",
	RosyBrown.Code():            "RosyBrown",
	RoyalBlue.Code():            "RoyalBlue",
	SaddleBrown.Code():          "SaddleBrown",
	Salmon.Code():               "Salmon",
	SandyBrown.Code():           "SandyBrown",
	SeaGreen.Code():             "SeaGreen",
	SeaShell.Code():             "SeaShell",
	Sienna.Code():               "Sienna",
	Silver.Code():               "Silver",
	SkyBlue.Code():              "SkyBlue",
	SlateBlue.Code():            "SlateBlue",
	SlateGray.Code():            "SlateGray",
	SlateGrey.Code():            "SlateGrey",
	Snow.Code():                 "Snow",
	SpringGreen.Code():          "SpringGreen",
	SteelBlue.Code():            "SteelBlue",
	Tan.Code():                  "Tan",
	Teal.Code():                 "Teal",
	Thistle.Code():              "Thistle",
	Tomato.Code():               "Tomato",
	Turquoise.Code():            "Turquoise",
	Violet.Code():               "Violet",
	Wheat.Code():                "Wheat",
	White.Code():                "White",
	WhiteSmoke.Code():           "WhiteSmoke",
	Yellow.Code():               "Yellow",
	YellowGreen.Code():          "YellowGreen",
})

func GetGlobalColorMap() *ColorNameMap           { return globalColorMap }
func GetColorNameByCode(code RGBCode) string     { return globalColorMap.GetColorNameByCode(code) }
func GetColorCodeByName(name string) RGBCode     { return globalColorMap.GetColorCodeByName(name) }
func SetColorCodeName(code RGBCode, name string) { globalColorMap.SetColorCodeName(code, name) }

type ContentWithContext struct {
	Context ColorContext
	Content interface{}
}

func BindContentToContext(context ColorContext, content interface{}) ContentWithContext {
	return ContentWithContext{Context: context, Content: content}
}

func (this ContentWithContext) Render(w *ColoredWriter) error {
	return w.WriteContent(this.Content)
}
func (this ContentWithContext) String() string {
	builder := &strings.Builder{}
	CWrite(builder, this.Content, this.Context)
	return builder.String()
}
