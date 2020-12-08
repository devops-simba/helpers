package helpers

func IIF(condition bool, ifTrue, ifFalse interface{}) interface{} {
	if condition {
		return ifTrue
	} else {
		return ifFalse
	}
}
func IIFn(condition bool, ifTrue, ifFalse int) int {
	if condition {
		return ifTrue
	} else {
		return ifFalse
	}
}
func IIFs(condition bool, ifTrue, ifFalse string) string {
	if condition {
		return ifTrue
	} else {
		return ifFalse
	}
}
func IIFf(condition bool, ifTrue, ifFalse func()) func() {
	if condition {
		return ifTrue
	} else {
		return ifFalse
	}
}
