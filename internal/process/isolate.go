package process

import (
	"github.com/Mrjwj34/lane/internal/config"
	"github.com/Mrjwj34/lane/internal/netns"
)

func IsolatePlan(cfg *config.Config, allocated map[string]int) (maps []netns.Mapping, isolate bool) {
	if cfg == nil || !cfg.IsolateNet() {
		return nil, false
	}
	out := make([]netns.Mapping, 0, len(cfg.Ports))
	for _, p := range cfg.Ports {
		host := allocated[p.Name]
		if host == 0 {
			continue
		}
		listen := p.Listen
		if listen == 0 {
			listen = host
		}
		out = append(out, netns.Mapping{Name: p.Name, Host: host, Listen: listen})
	}
	return out, true
}
