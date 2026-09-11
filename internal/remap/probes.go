package remap

import "strconv"

// RewriteProbes copies processes and rewrites http probe ports that still
// name the project's listen port so process-compose (not remapped on Linux
// unless supervised) checks the host port.
func RewriteProbes(processes map[string]any, maps []Mapping) map[string]any {
	if len(processes) == 0 || len(maps) == 0 {
		return processes
	}
	lut := map[int]int{}
	for _, m := range maps {
		if m.Listen > 0 && m.Host > 0 && m.Listen != m.Host {
			lut[m.Listen] = m.Host
		}
	}
	if len(lut) == 0 {
		return processes
	}
	out, ok := copyAny(processes).(map[string]any)
	if !ok {
		return processes
	}
	for _, raw := range out {
		proc, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		rewriteProbe(proc, "readiness_probe", lut)
		rewriteProbe(proc, "liveness_probe", lut)
	}
	return out
}

func rewriteProbe(proc map[string]any, key string, lut map[int]int) {
	probe, ok := proc[key].(map[string]any)
	if !ok {
		return
	}
	httpGet, ok := probe["http_get"].(map[string]any)
	if !ok {
		return
	}
	port, ok := asPort(httpGet["port"])
	if !ok {
		return
	}
	if host, found := lut[port]; found {
		httpGet["port"] = host
	}
}

func asPort(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, t > 0
	case int64:
		return int(t), t > 0
	case float64:
		return int(t), t > 0
	case string:
		n, err := strconv.Atoi(t)
		return n, err == nil && n > 0
	default:
		return 0, false
	}
}

func copyAny(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = copyAny(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = copyAny(val)
		}
		return out
	default:
		return v
	}
}
