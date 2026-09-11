package remap

// Mapping remaps a process listen port onto a unique host port.
// The process still calls bind(:Listen); the interceptor binds Host instead.
type Mapping struct {
	Name   string `json:"name"`
	Host   int    `json:"host"`
	Listen int    `json:"listen"`
}
