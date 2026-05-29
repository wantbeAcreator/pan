package osdetect

import "runtime"

func CurrentOS() string {
	return runtime.GOOS
}
