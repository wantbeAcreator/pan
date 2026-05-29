//go:build linux || darwin

package dialog

import (
	"fmt"
	"os"
)

func ShowError(title, msg string) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", title, msg)
}
