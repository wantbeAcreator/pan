//go:build windows

package dialog

import (
	"syscall"
	"unsafe"
)

func ShowError(title, msg string) {
	user32, _ := syscall.LoadDLL("user32.dll")
	defer user32.Release()
	msgBox, _ := user32.FindProc("MessageBoxW")

	titlePtr, _ := syscall.UTF16PtrFromString(title)
	msgPtr, _ := syscall.UTF16PtrFromString(msg)

	msgBox.Call(
		0,
		uintptr(unsafe.Pointer(msgPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		0x00000010, // MB_ICONERROR
	)
}
