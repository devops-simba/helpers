package helpers

import (
	"fmt"
	"strings"
)

type FormatNode struct {
	FormatString string      `json:"fmt,omitempty"`
	NoArg        bool        `json:"noArg,omitempty"`
	Arg          interface{} `json:"arg,omitempty"`
}

func (this FormatNode) Format() string {
	if this.FormatString == "" {
		return this.Arg.(string)
	} else if this.NoArg {
		return fmt.Sprintf(this.FormatString)
	} else {
		return fmt.Sprintf(this.FormatString, this.Arg)
	}
}

type FormatInfo []FormatNode

func (this FormatInfo) Format() string {
	builder := strings.Builder{}
	for i := 0; i < len(this); i++ {
		builder.WriteString(this[i].Format())
	}
	return builder.String()
}

func ParseFormatString(format string, args ...interface{}) FormatInfo {
	i := 0
	arg := 0
	end := len(format)
	result := FormatInfo{}
	for i < end {
		lastI := i
		for i < end && format[i] != '%' {
			i++
		}
		if i != lastI {
			result = append(result, FormatNode{Arg: format[lastI:i]})
		}
		if i >= end {
			break
		}

		lastI = i
		i++
		found := false
		for i < end {
			char := format[i]
			i++
			if ('a' <= char && char <= 'z') || char == '%' {
				// this is the verb
				found = true
				break
			}
		}
		if !found {
			panic("Invalid format string")
		}

		node := FormatNode{
			FormatString: format[lastI:i],
		}
		if arg < len(args) {
			node.Arg = args[arg]
			arg++
		} else {
			node.NoArg = true
		}

		result = append(result, node)
	}

	return result
}
