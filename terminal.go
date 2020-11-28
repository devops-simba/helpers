package helpers

import (
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

func IsTerminal(f *os.File) bool {
	return terminal.IsTerminal(int(f.Fd()))
}
