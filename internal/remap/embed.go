package remap

import (
	_ "embed"
	"runtime"
)

//go:embed preload.c.src
var preloadSrc string

//go:embed wrap.c.src
var wrapSrc string

//go:embed launch.c.src
var launchSrc string

func libName() string {
	if runtime.GOOS == "darwin" {
		return "liblane_remap.dylib"
	}
	return "liblane_remap.so"
}
