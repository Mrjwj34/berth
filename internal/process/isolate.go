package process

import (
	"github.com/Mrjwj34/lane/internal/config"
	"github.com/Mrjwj34/lane/internal/netns"
)

func PortMaps(ports config.Ports, allocated map[string]int) []netns.Mapping {
	if !ports.NeedsIsolate() {
		return nil
	}
	out := make([]netns.Mapping, 0, len(ports))
	for _, p := range ports {
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
	return out
}
