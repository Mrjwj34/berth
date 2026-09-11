package process

import (
	"testing"

	"github.com/Mrjwj34/lane/internal/config"
)

func TestPortMapsOnlyWhenListenSet(t *testing.T) {
	if maps := PortMaps(config.Ports{{Name: "web"}}, map[string]int{"web": 20100}); maps != nil {
		t.Fatalf("unexpected maps: %+v", maps)
	}
	maps := PortMaps(config.Ports{{Name: "api", Listen: 8080}, {Name: "web"}}, map[string]int{"api": 20100, "web": 20101})
	if len(maps) != 2 {
		t.Fatalf("maps: %+v", maps)
	}
	if maps[0].Host != 20100 || maps[0].Listen != 8080 {
		t.Fatalf("api map: %+v", maps[0])
	}
	if maps[1].Host != 20101 || maps[1].Listen != 20101 {
		t.Fatalf("web identity map: %+v", maps[1])
	}
}
