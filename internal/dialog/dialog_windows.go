//go:build windows

package dialog

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func ShowError(title, msg string) {
	user32, err := syscall.LoadDLL("user32.dll")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] %s\n", title, msg)
		return
	}
	defer user32.Release()

	msgBox, err := user32.FindProc("MessageBoxW")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] %s\n", title, msg)
		return
	}

	titlePtr, _ := syscall.UTF16PtrFromString(title)
	msgPtr, _ := syscall.UTF16PtrFromString(msg)

	msgBox.Call(
		0,
		uintptr(unsafe.Pointer(msgPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		0x00000010, // MB_ICONERROR
	)
}
