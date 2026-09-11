package process

import (
	"testing"

	"github.com/Mrjwj34/lane/internal/config"
)

func TestIsolatePlan(t *testing.T) {
	maps, isolate := IsolatePlan(&config.Config{Ports: config.Ports{{Name: "web"}}}, map[string]int{"web": 20100})
	if isolate || maps != nil {
		t.Fatalf("short-form ports should not isolate: isolate=%v maps=%+v", isolate, maps)
	}
	maps, isolate = IsolatePlan(&config.Config{Isolate: "net"}, map[string]int{})
	if !isolate || len(maps) != 0 {
		t.Fatalf("isolate: net with no ports: isolate=%v maps=%+v", isolate, maps)
	}
	maps, isolate = IsolatePlan(&config.Config{Ports: config.Ports{{Name: "api", Listen: 8080}, {Name: "web"}}, Isolate: "net"}, map[string]int{"api": 20100, "web": 20101})
	if !isolate || len(maps) != 2 {
		t.Fatalf("maps: %+v isolate=%v", maps, isolate)
	}
	if maps[0].Host != 20100 || maps[0].Listen != 8080 {
		t.Fatalf("api map: %+v", maps[0])
	}
	if maps[1].Host != 20101 || maps[1].Listen != 20101 {
		t.Fatalf("web identity map: %+v", maps[1])
	}
}
