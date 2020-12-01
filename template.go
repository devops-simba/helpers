package helpers

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"text/template"
)

const (
	T_NoColorName          = "none"
	unknownColorNameFormat = "'%s' is not a known color name"
)

var (
	defaultTemplateContext         = atomic.Value{}
	ErrorCantDereferenceNilPointer = StringError("Can't dereference nil pointer")
	ErrorInvalidColorCode          = StringError("Invalid color code")
)

func GetDefaultTemplateContext() ColorContext {
	context := defaultTemplateContext.Load()
	if context == nil {
		return MonoColor
	}
	return context.(ColorContext)
}
func SetDefaultTemplateContext(context ColorContext) {
	defaultTemplateContext.Store(context)
}

type TemplateColorContext interface {
	GetContext() ColorContext
	GetColorMap() *ColorNameMap
	GetDefaultColor() Color
}

type TT_JoinedScope struct {
	Inner interface{}
	Outer interface{}
}

// THF_Deref dereference a pointer
func THF_Deref(pointer interface{}) (interface{}, error) {
	if pointer == nil {
		return nil, ErrorCantDereferenceNilPointer
	}

	rv := reflect.ValueOf(pointer)
	if rv.IsNil() {
		return nil, ErrorCantDereferenceNilPointer
	}
	if rv.Kind() == reflect.Ptr {
		return rv.Elem().Interface(), nil
	}
	return nil, fmt.Errorf("Expected a pointer but received %T", pointer)
}

// THF_MakeDict get an even number of parameters(first is key and second is value) and create a dictionary from it
func THF_MakeDict(values ...interface{}) (map[interface{}]interface{}, error) {
	if (len(values) & 1) == 1 {
		return nil, StringError("Invalid number of arguments for MakeDict")
	}

	result := make(map[interface{}]interface{})
	if len(values) == 0 {
		return result, nil
	}
	for i := 0; i < len(values); i += 2 {
		key := values[i]
		value := values[i+1]

		result[key] = value
	}

	return result, nil
}

// THF_JoinScope join an inner and an outer scope and create a joined scope from it
func THF_JoinScope(outer, inner interface{}) TT_JoinedScope {
	return TT_JoinedScope{Inner: inner, Outer: outer}
}

// THF_Quote quote an input string, escaping '"' and '\' character
func THF_Quote(value interface{}) (string, error) {
	s, ok := value.(string)
	if !ok {
		s = fmt.Sprint(value)
	}

	buffer, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(buffer), nil
}

// THF_QuoteAndJoin quote all items in the collection and then join them with separator
func THF_QuoteAndJoin(values []string, sep string) (string, error) {
	builder := &strings.Builder{}
	for i := 0; i < len(values); i++ {
		if i != 0 {
			builder.WriteString(sep)
		}
		q, err := THF_Quote(values[i])
		if err != nil {
			return "", err
		}
		builder.WriteString(q)
	}
	return builder.String(), nil
}

// THF_Color return a color using its name or RGB code
func THF_Color(codeOrName interface{}) (Color, error) {
	switch v := codeOrName.(type) {
	case int:
		if v < 0 || v > 0xFFFFFF {
			return nil, ErrorInvalidColorCode
		}
		return RGBColor(uint32(v)), nil
	case uint:
		if v > 0xFFFFFF {
			return nil, ErrorInvalidColorCode
		}
		return RGBColor(v), nil
	case string:
		if v == T_NoColorName {
			return NoColor, nil
		}
		if v == "" {
			return nil, fmt.Errorf(unknownColorNameFormat, v)
		}
		if v[0] == '#' {
			if code, err := strconv.ParseUint(v[1:], 16, 24); err != nil {
				return RGBColor(uint32(code)), nil
			}
		}

		if code := GetColorCodeByName(v); code != NoColorCode {
			return code.ToColor(), nil
		}
		return nil, fmt.Errorf(unknownColorNameFormat, v)
	case RGBCode:
		return v.ToColor(), nil
	default:
		if color, ok := codeOrName.(Color); ok {
			return color, nil
		}
	}
	return nil, fmt.Errorf("%T is not a color code or color name", codeOrName)
}

// THF_ColorC if `context` implemented `TemplateColorContext`, lookup `colorName` in color map of the context
// otherwise return result of calling `THF_Color` with `colorName`
func THF_ColorC(context interface{}, colorName string) (Color, error) {
	if tcc, ok := context.(TemplateColorContext); ok {
		if colorName == "" {
			return tcc.GetDefaultColor(), nil
		}
		if colorName == T_NoColorName {
			return NoColor, nil
		}
		if colorName[0] == '#' {
			if code, err := strconv.ParseUint(colorName[1:], 16, 24); err == nil {
				return RGBColor(uint32(code)), nil
			}
		}

		code := tcc.GetColorMap().GetColorCodeByName(colorName)
		if code != NoColorCode {
			return RGBColor(code), nil
		}

		return nil, fmt.Errorf(unknownColorNameFormat, colorName)
	} else {
		return THF_Color(colorName)
	}
}

// THF_WithColor return a colored content
func THF_WithColor(codeOrName interface{}, content interface{}) (interface{}, error) {
	color, err := THF_Color(codeOrName)
	if err != nil {
		return nil, err
	}

	return BindContentToContext(GetDefaultTemplateContext(), CContent(color, content)), nil
}

// THF_WithColorC return a colored content using the provided `context`
func THF_WithColorC(context interface{}, colorName string, content interface{}) (interface{}, error) {
	if tcc, ok := context.(TemplateColorContext); ok {
		color, err := THF_ColorC(tcc, colorName)
		if err != nil {
			return nil, err
		}

		return BindContentToContext(tcc.GetContext(), CContent(color, content)), nil
	}
	return THF_WithColor(colorName, content)
}

// THF_CFormat return a colored formatted string
func THF_CFormat(color interface{}, format string, args ...interface{}) (interface{}, error) {
	return THF_WithColor(color, CreateFormatContent(format, args...))
}

// THF_CFormatC return a colored formatted string
func THF_CFormatC(context interface{}, colorName string, format string, args ...interface{}) (interface{}, error) {
	return THF_WithColorC(context, colorName, CreateFormatContent(format, args...))
}

var globalFuncs = template.FuncMap{
	"Json":         json.Marshal,
	"Join":         strings.Join,
	"Deref":        THF_Deref,
	"Quote":        THF_Quote,
	"QuoteAndJoin": THF_QuoteAndJoin,
	"JoinScope":    THF_JoinScope,
	"MakeDict":     THF_MakeDict,
	"Color":        THF_Color,
	"ColorC":       THF_ColorC,
	"WithColor":    THF_WithColor,
	"WithColorC":   THF_WithColorC,
	"CFormat":      THF_CFormat,
	"CFormatC":     THF_CFormatC,
}

func GetGlobalTemplateFuncs() template.FuncMap { return globalFuncs }
func RegisterTemplateFunc(name string, f interface{}) {
	if f == nil || name == "" {
		panic("Invalid argument")
	}

	globalFuncs[name] = f
}

func ParseTemplate(name string, body string) (*template.Template, error) {
	return template.New(name).Funcs(globalFuncs).Parse(body)
}
