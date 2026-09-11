package config

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Port is a named host-published port. Listen, when set, is the TCP port the
// project process already binds — lane remaps it and does not require the app
// to read LANE_PORT_*.
type Port struct {
	Name   string `yaml:"name" json:"name"`
	Listen int    `yaml:"listen,omitempty" json:"listen,omitempty"`
}

// Ports unmarshals both `ports: [web, api]` and a mapping with listen ports:
//
//	ports:
//	  api: 8080
//	  web:
//	    listen: 5173
type Ports []Port

func (p Ports) Names() []string {
	names := make([]string, len(p))
	for i, port := range p {
		names[i] = port.Name
	}
	return names
}

func (p Ports) ListenMap() map[string]int {
	out := map[string]int{}
	for _, port := range p {
		if port.Listen > 0 {
			out[port.Name] = port.Listen
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (p Ports) NeedsIsolate() bool {
	for _, port := range p {
		if port.Listen > 0 {
			return true
		}
	}
	return false
}

func (p *Ports) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		*p = nil
		return nil
	}
	switch value.Kind {
	case yaml.SequenceNode:
		out := make(Ports, 0, len(value.Content))
		for _, node := range value.Content {
			port, err := decodePortNode(node)
			if err != nil {
				return err
			}
			out = append(out, port)
		}
		*p = out
		return nil
	case yaml.MappingNode:
		if len(value.Content)%2 != 0 {
			return fmt.Errorf("ports: invalid mapping")
		}
		out := make(Ports, 0, len(value.Content)/2)
		for i := 0; i < len(value.Content); i += 2 {
			name := strings.TrimSpace(value.Content[i].Value)
			port, err := decodePortValue(name, value.Content[i+1])
			if err != nil {
				return err
			}
			out = append(out, port)
		}
		*p = out
		return nil
	case yaml.ScalarNode:
		if value.Tag == "!!null" || strings.TrimSpace(value.Value) == "" {
			*p = nil
			return nil
		}
		return fmt.Errorf("ports: expected a list or a mapping")
	default:
		return fmt.Errorf("ports: expected a list or a mapping")
	}
}

func decodePortNode(node *yaml.Node) (Port, error) {
	if node.Kind == yaml.ScalarNode {
		return Port{Name: strings.TrimSpace(node.Value)}, nil
	}
	var aux struct {
		Name   string `yaml:"name"`
		Listen int    `yaml:"listen"`
	}
	if err := node.Decode(&aux); err != nil {
		return Port{}, fmt.Errorf("ports: %w", err)
	}
	if strings.TrimSpace(aux.Name) == "" {
		return Port{}, fmt.Errorf("ports: missing name")
	}
	return Port{Name: strings.TrimSpace(aux.Name), Listen: aux.Listen}, nil
}

func decodePortValue(name string, node *yaml.Node) (Port, error) {
	if name == "" {
		return Port{}, fmt.Errorf("ports: empty name")
	}
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag == "!!null" || strings.TrimSpace(node.Value) == "" || node.Value == "~" {
			return Port{Name: name}, nil
		}
		listen, err := strconv.Atoi(strings.TrimSpace(node.Value))
		if err != nil {
			return Port{}, fmt.Errorf("ports.%s: listen port must be an integer", name)
		}
		return Port{Name: name, Listen: listen}, nil
	case yaml.MappingNode:
		var aux struct {
			Listen int `yaml:"listen"`
		}
		if err := node.Decode(&aux); err != nil {
			return Port{}, fmt.Errorf("ports.%s: %w", name, err)
		}
		return Port{Name: name, Listen: aux.Listen}, nil
	default:
		return Port{}, fmt.Errorf("ports.%s: expected a listen port or {listen: N}", name)
	}
}
