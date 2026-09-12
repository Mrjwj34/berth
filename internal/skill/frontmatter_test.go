package skill

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The skill body is consumed by harnesses and by skill installers that parse the
// YAML frontmatter before they will load it. A description containing ": " is
// still valid prose but invalid YAML, and a frontmatter that does not parse
// makes the skill invisible: `skills add Mrjwj34/berth` reports "No valid skills
// found" while every berth command keeps working. Parse it here so a wording
// change cannot silently break installation.
func TestSkillFrontmatterParses(t *testing.T) {
	data, err := Content()
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		t.Fatalf("SKILL.md must start with a frontmatter block, got %q", firstLine(text))
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		t.Fatal("frontmatter block is not terminated")
	}
	block := text[4 : 4+end]

	var front struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(block), &front); err != nil {
		t.Fatalf("frontmatter is not valid YAML: %v\n%s", err, block)
	}
	if front.Name != "berth" {
		t.Fatalf("frontmatter name = %q, want %q (it must match the install directory)", front.Name, "berth")
	}
	if len(front.Description) < 40 {
		t.Fatalf("frontmatter description is too short to match a skill catalog: %q", front.Description)
	}
	if strings.Contains(front.Description, "::") {
		t.Fatalf("description contains a double colon, which usually means a broken sentence: %q", front.Description)
	}
}

func firstLine(text string) string {
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		return text[:i]
	}
	return text
}
