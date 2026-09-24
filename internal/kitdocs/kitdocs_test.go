package kitdocs

import (
	"reflect"
	"testing"
	"testing/fstest"
)

func TestCommandNamesReadsTheGeneratedReference(t *testing.T) {
	kit := fstest.MapFS{"commands.md": {Data: []byte(
		"Commands are typed as `ffauto:<family>.<name>|<arg>`.\n" +
			"| `ffauto:construction.place` | `<blueprintString>\\|<x>\\|<z>` | Places. |\n" +
			"| `ffauto:help` | `[family]` | |\n" +
			"- `ffauto:craft.queue`\n- `ffauto:help`\n")}}
	names, ok := CommandNames(kit)
	if !ok || !reflect.DeepEqual(names, []string{"construction.place", "craft.queue", "help"}) {
		t.Fatalf("got %v, %v", names, ok)
	}
	if _, ok := CommandNames(fstest.MapFS{}); ok {
		t.Fatal("a kit without commands.md must report no reference")
	}
}

func TestSkillsParseFrontmatter(t *testing.T) {
	kit := fstest.MapFS{
		".claude/skills/b-skill/SKILL.md": {Data: []byte("---\nname: build-a-mall\ndescription: >\n  Build an automated\n  line of buildings.\ngameVersion: 0.50\n---\nBody here.\n")},
		".claude/skills/a-skill/SKILL.md": {Data: []byte("---\ndescription: \"Quoted: yes\"\n---\n\nText.\n")},
		".claude/skills/empty/README.md":  {Data: []byte("not a skill")},
	}
	skills := Skills(kit)
	if len(skills) != 2 {
		t.Fatalf("got %d skills", len(skills))
	}
	if s := skills[0]; s.Name != "a-skill" || s.Description != "Quoted: yes" || s.Body != "\nText.\n" {
		t.Fatalf("a-skill: %+v", s)
	}
	if s := skills[1]; s.Name != "build-a-mall" || s.Description != "Build an automated line of buildings." || s.Body != "Body here.\n" {
		t.Fatalf("build-a-mall: %+v", s)
	}
}
