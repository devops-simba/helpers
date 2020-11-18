package helpers

import (
	"fmt"
	"html"
)

const (
	ConsoleContextName = "console"
	HTMLContextName    = "HTML"
	NoColorContextName = "NoColor"
	formatEscape       = "\033["
)

type ColorContext interface {
	Name() string
	Write(color Color, text string) string
	Writef(color Color, format string, a ...interface{}) string
	Print(color Color, text string) (int, error)
	Printf(color Color, format string, a ...interface{}) (int, error)
}

type ConsoleContext struct{}

func (this ConsoleContext) Name() string { return ConsoleContextName }
func (this ConsoleContext) Write(color Color, text string) string {
	return formatEscape + color.Render(this) + "m" + text + formatEscape + "0m"
}
func (this ConsoleContext) Writef(color Color, format string, a ...interface{}) string {
	return this.Write(color, fmt.Sprintf(format, a...))
}
func (this ConsoleContext) Print(color Color, text string) (int, error) {
	return fmt.Print(this.Write(color, text))
}
func (this ConsoleContext) Printf(color Color, format string, a ...interface{}) (int, error) {
	return fmt.Print(this.Writef(color, format, a...))
}

type HTMLContext struct{}

func (this HTMLContext) Name() string { return HTMLContextName }
func (this HTMLContext) Write(color Color, text string) string {
	return "<span style='" + color.Render(this) + "'>" + html.EscapeString(text) + "</span>"
}
func (this HTMLContext) Writef(color Color, format string, a ...interface{}) string {
	return this.Write(color, fmt.Sprintf(format, a...))
}
func (this HTMLContext) Print(color Color, text string) (int, error) {
	return fmt.Print(this.Write(color, text))
}
func (this HTMLContext) Printf(color Color, format string, a ...interface{}) (int, error) {
	return fmt.Print(this.Writef(color, format, a...))
}

type NoColorContext struct{}

func (this NoColorContext) Name() string                          { return NoColorContextName }
func (this NoColorContext) Write(color Color, text string) string { return text }
func (this NoColorContext) Writef(color Color, format string, a ...interface{}) string {
	return this.Write(color, fmt.Sprintf(format, a...))
}

var (
	Console = ConsoleContext{}
	HTML    = HTMLContext{}
	NoColor = NoColorContext{}
)

type Color interface {
	Render(context ColorContext) string
	AsForeground() Color
	AsBackground() Color
}

type RGBColor uint32

func RGB(red, green, blue uint8) RGBColor {
	var n uint32
	n = uint32(blue) | (uint32(green) << 8) | (uint32(red) << 16)
	return RGBColor(n)
}
func (this RGBColor) isBackground() bool { return uint32(this&0xFF000000) != 0 }
func (this RGBColor) Red() uint8         { return uint8((this >> 16) & 0xFF) }
func (this RGBColor) Green() uint8       { return uint8((this >> 8) & 0xFF) }
func (this RGBColor) Blue() uint8        { return uint8(this & 0xFF) }
func (this RGBColor) Render(context ColorContext) string {
	switch context.Name() {
	case ConsoleContextName:
		if this.isBackground() {
			return fmt.Sprintf("48;2;%d;%d;%d", this.Red(), this.Green(), this.Blue())
		} else {
			return fmt.Sprintf("38;2;%d;%d;%d", this.Red(), this.Green(), this.Blue())
		}
	case HTMLContextName:
		if this.isBackground() {
			return fmt.Sprintf("background-color: #%06X", this&0xFFFFFF)
		} else {
			return fmt.Sprintf("color: #%06X", this&0xFFFFFF)
		}
	case NoColorContextName:
		return ""
	default:
		panic("Unknown context")
	}
}
func (this RGBColor) AsForeground() Color { return RGBColor(this & 0x00FFFFFF) }
func (this RGBColor) AsBackground() Color { return RGBColor(this | 0xFF000000) }
func (this RGBColor) String() string      { return fmt.Sprintf("#%06X", this&0xFFFFFF) }

type MixedColor struct {
	Foreground Color
	Background Color
}

func Mixed(foreground, background Color) MixedColor {
	return MixedColor{
		Foreground: foreground.AsForeground(),
		Background: background.AsBackground(),
	}
}
func (this MixedColor) AsForeground() Color { return this.Foreground }
func (this MixedColor) AsBackground() Color { return this.Background }
func (this MixedColor) Render(context ColorContext) string {
	switch context.Name() {
	case ConsoleContextName:
		return this.Foreground.Render(context) + ";" + this.Background.Render(context)
	case HTMLContextName:
		return this.Foreground.Render(context) + ";" + this.Background.Render(context)
	case NoColorContextName:
		return ""
	default:
		panic("Unknown context")
	}
}

const (
	AliceBlue            = RGBColor(0xF0F8FF)
	AntiqueWhite         = RGBColor(0xFAEBD7)
	Aqua                 = RGBColor(0x00FFFF)
	Aquamarine           = RGBColor(0x7FFFD4)
	Azure                = RGBColor(0xF0FFFF)
	Beige                = RGBColor(0xF5F5DC)
	Bisque               = RGBColor(0xFFE4C4)
	Black                = RGBColor(0x000000)
	BlanchedAlmond       = RGBColor(0xFFEBCD)
	Blue                 = RGBColor(0x0000FF)
	BlueViolet           = RGBColor(0x8A2BE2)
	Brown                = RGBColor(0xA52A2A)
	BurlyWood            = RGBColor(0xDEB887)
	CadetBlue            = RGBColor(0x5F9EA0)
	Chartreuse           = RGBColor(0x7FFF00)
	Chocolate            = RGBColor(0xD2691E)
	Coral                = RGBColor(0xFF7F50)
	CornflowerBlue       = RGBColor(0x6495ED)
	Cornsilk             = RGBColor(0xFFF8DC)
	Crimson              = RGBColor(0xDC143C)
	Cyan                 = RGBColor(0x00FFFF)
	DarkBlue             = RGBColor(0x00008B)
	DarkCyan             = RGBColor(0x008B8B)
	DarkGoldenRod        = RGBColor(0xB8860B)
	DarkGray             = RGBColor(0xA9A9A9)
	DarkGrey             = RGBColor(0xA9A9A9)
	DarkGreen            = RGBColor(0x006400)
	DarkKhaki            = RGBColor(0xBDB76B)
	DarkMagenta          = RGBColor(0x8B008B)
	DarkOliveGreen       = RGBColor(0x556B2F)
	DarkOrange           = RGBColor(0xFF8C00)
	DarkOrchid           = RGBColor(0x9932CC)
	DarkRed              = RGBColor(0x8B0000)
	DarkSalmon           = RGBColor(0xE9967A)
	DarkSeaGreen         = RGBColor(0x8FBC8F)
	DarkSlateBlue        = RGBColor(0x483D8B)
	DarkSlateGray        = RGBColor(0x2F4F4F)
	DarkSlateGrey        = RGBColor(0x2F4F4F)
	DarkTurquoise        = RGBColor(0x00CED1)
	DarkViolet           = RGBColor(0x9400D3)
	DeepPink             = RGBColor(0xFF1493)
	DeepSkyBlue          = RGBColor(0x00BFFF)
	DimGray              = RGBColor(0x696969)
	DimGrey              = RGBColor(0x696969)
	DodgerBlue           = RGBColor(0x1E90FF)
	FireBrick            = RGBColor(0xB22222)
	FloralWhite          = RGBColor(0xFFFAF0)
	ForestGreen          = RGBColor(0x228B22)
	Fuchsia              = RGBColor(0xFF00FF)
	Gainsboro            = RGBColor(0xDCDCDC)
	GhostWhite           = RGBColor(0xF8F8FF)
	Gold                 = RGBColor(0xFFD700)
	GoldenRod            = RGBColor(0xDAA520)
	Gray                 = RGBColor(0x808080)
	Grey                 = RGBColor(0x808080)
	Green                = RGBColor(0x008000)
	GreenYellow          = RGBColor(0xADFF2F)
	HoneyDew             = RGBColor(0xF0FFF0)
	HotPink              = RGBColor(0xFF69B4)
	IndianRed            = RGBColor(0xCD5C5C)
	Indigo               = RGBColor(0x4B0082)
	Ivory                = RGBColor(0xFFFFF0)
	Khaki                = RGBColor(0xF0E68C)
	Lavender             = RGBColor(0xE6E6FA)
	LavenderBlush        = RGBColor(0xFFF0F5)
	LawnGreen            = RGBColor(0x7CFC00)
	LemonChiffon         = RGBColor(0xFFFACD)
	LightBlue            = RGBColor(0xADD8E6)
	LightCoral           = RGBColor(0xF08080)
	LightCyan            = RGBColor(0xE0FFFF)
	LightGoldenRodYellow = RGBColor(0xFAFAD2)
	LightGray            = RGBColor(0xD3D3D3)
	LightGrey            = RGBColor(0xD3D3D3)
	LightGreen           = RGBColor(0x90EE90)
	LightPink            = RGBColor(0xFFB6C1)
	LightSalmon          = RGBColor(0xFFA07A)
	LightSeaGreen        = RGBColor(0x20B2AA)
	LightSkyBlue         = RGBColor(0x87CEFA)
	LightSlateGray       = RGBColor(0x778899)
	LightSlateGrey       = RGBColor(0x778899)
	LightSteelBlue       = RGBColor(0xB0C4DE)
	LightYellow          = RGBColor(0xFFFFE0)
	Lime                 = RGBColor(0x00FF00)
	LimeGreen            = RGBColor(0x32CD32)
	Linen                = RGBColor(0xFAF0E6)
	Magenta              = RGBColor(0xFF00FF)
	Maroon               = RGBColor(0x800000)
	MediumAquaMarine     = RGBColor(0x66CDAA)
	MediumBlue           = RGBColor(0x0000CD)
	MediumOrchid         = RGBColor(0xBA55D3)
	MediumPurple         = RGBColor(0x9370DB)
	MediumSeaGreen       = RGBColor(0x3CB371)
	MediumSlateBlue      = RGBColor(0x7B68EE)
	MediumSpringGreen    = RGBColor(0x00FA9A)
	MediumTurquoise      = RGBColor(0x48D1CC)
	MediumVioletRed      = RGBColor(0xC71585)
	MidnightBlue         = RGBColor(0x191970)
	MintCream            = RGBColor(0xF5FFFA)
	MistyRose            = RGBColor(0xFFE4E1)
	Moccasin             = RGBColor(0xFFE4B5)
	NavajoWhite          = RGBColor(0xFFDEAD)
	Navy                 = RGBColor(0x000080)
	OldLace              = RGBColor(0xFDF5E6)
	Olive                = RGBColor(0x808000)
	OliveDrab            = RGBColor(0x6B8E23)
	Orange               = RGBColor(0xFFA500)
	OrangeRed            = RGBColor(0xFF4500)
	Orchid               = RGBColor(0xDA70D6)
	PaleGoldenRod        = RGBColor(0xEEE8AA)
	PaleGreen            = RGBColor(0x98FB98)
	PaleTurquoise        = RGBColor(0xAFEEEE)
	PaleVioletRed        = RGBColor(0xDB7093)
	PapayaWhip           = RGBColor(0xFFEFD5)
	PeachPuff            = RGBColor(0xFFDAB9)
	Peru                 = RGBColor(0xCD853F)
	Pink                 = RGBColor(0xFFC0CB)
	Plum                 = RGBColor(0xDDA0DD)
	PowderBlue           = RGBColor(0xB0E0E6)
	Purple               = RGBColor(0x800080)
	RebeccaPurple        = RGBColor(0x663399)
	Red                  = RGBColor(0xFF0000)
	RosyBrown            = RGBColor(0xBC8F8F)
	RoyalBlue            = RGBColor(0x4169E1)
	SaddleBrown          = RGBColor(0x8B4513)
	Salmon               = RGBColor(0xFA8072)
	SandyBrown           = RGBColor(0xF4A460)
	SeaGreen             = RGBColor(0x2E8B57)
	SeaShell             = RGBColor(0xFFF5EE)
	Sienna               = RGBColor(0xA0522D)
	Silver               = RGBColor(0xC0C0C0)
	SkyBlue              = RGBColor(0x87CEEB)
	SlateBlue            = RGBColor(0x6A5ACD)
	SlateGray            = RGBColor(0x708090)
	SlateGrey            = RGBColor(0x708090)
	Snow                 = RGBColor(0xFFFAFA)
	SpringGreen          = RGBColor(0x00FF7F)
	SteelBlue            = RGBColor(0x4682B4)
	Tan                  = RGBColor(0xD2B48C)
	Teal                 = RGBColor(0x008080)
	Thistle              = RGBColor(0xD8BFD8)
	Tomato               = RGBColor(0xFF6347)
	Turquoise            = RGBColor(0x40E0D0)
	Violet               = RGBColor(0xEE82EE)
	Wheat                = RGBColor(0xF5DEB3)
	White                = RGBColor(0xFFFFFF)
	WhiteSmoke           = RGBColor(0xF5F5F5)
	Yellow               = RGBColor(0xFFFF00)
	YellowGreen          = RGBColor(0x9ACD32)
)
