// Package kitdocs reads the embedded kit documents: the command reference, the guide and the
// skills. Every document is optional at build time; callers treat a missing one as absent.
package kitdocs

import (
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Paths inside the kit folder.
const (
	GuidePath    = "HowToPlay.md"
	CommandsPath = "commands.md"
	SkillsDir    = ".claude/skills"
)

// The generated reference (scripts/generate-ffauto-reference.py in the game repo) writes every
// command as `ffauto:<name>`, in the tables and in the undocumented list alike.
var commandRef = regexp.MustCompile("`ffauto:([A-Za-z0-9_]+(?:\\.[A-Za-z0-9_]+)*)")

// CommandNames returns the sorted, de-duplicated command names in commands.md, and false when the
// kit has no commands.md.
func CommandNames(kit fs.FS) ([]string, bool) {
	data, err := fs.ReadFile(kit, CommandsPath)
	if err != nil {
		return nil, false
	}
	seen := map[string]bool{}
	var names []string
	for _, m := range commandRef.FindAllStringSubmatch(string(data), -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			names = append(names, m[1])
		}
	}
	sort.Strings(names)
	return names, true
}

// Diff returns names live has that embedded lacks (added) and the reverse (removed), sorted.
func Diff(embedded, live []string) (added, removed []string) {
	in := func(set []string) map[string]bool {
		m := make(map[string]bool, len(set))
		for _, s := range set {
			m[s] = true
		}
		return m
	}
	e, l := in(embedded), in(live)
	for n := range l {
		if !e[n] {
			added = append(added, n)
		}
	}
	for n := range e {
		if !l[n] {
			removed = append(removed, n)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

// Skill is one kit/.claude/skills/<dir>/SKILL.md.
type Skill struct {
	Name        string
	Description string
	// Path is the SKILL.md path inside the kit.
	Path string
	// Text is the whole file; Body is the text after the frontmatter.
	Text string
	Body string
}

// Skills lists every skill folder with a SKILL.md, sorted by name.
func Skills(kit fs.FS) []Skill {
	entries, err := fs.ReadDir(kit, SkillsDir)
	if err != nil {
		return nil
	}
	var skills []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := path.Join(SkillsDir, e.Name(), "SKILL.md")
		data, err := fs.ReadFile(kit, p)
		if err != nil {
			continue
		}
		front, body := splitFrontmatter(string(data))
		s := Skill{Name: e.Name(), Path: p, Text: string(data), Body: body}
		if n := front["name"]; n != "" {
			s.Name = n
		}
		s.Description = front["description"]
		skills = append(skills, s)
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills
}

// splitFrontmatter parses the flat `key: value` YAML frontmatter skills use. Folded (`>`) and
// literal (`|`) scalars are joined from their indented lines; anything richer is ignored.
func splitFrontmatter(text string) (map[string]string, string) {
	front := map[string]string{}
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return front, text
	}
	rest := normalized[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return front, text
	}
	body := strings.TrimLeft(rest[end+len("\n---"):], "-")
	body = strings.TrimPrefix(body, "\n")

	lines := strings.Split(rest[:end], "\n")
	for i := 0; i < len(lines); i++ {
		key, value, ok := strings.Cut(lines[i], ":")
		if !ok || strings.HasPrefix(lines[i], " ") || strings.HasPrefix(lines[i], "\t") {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if value == ">" || value == "|" || value == ">-" || value == "|-" {
			var parts []string
			for i+1 < len(lines) && (strings.HasPrefix(lines[i+1], " ") || strings.HasPrefix(lines[i+1], "\t")) {
				i++
				parts = append(parts, strings.TrimSpace(lines[i]))
			}
			value = strings.Join(parts, " ")
		}
		front[key] = unquote(value)
	}
	return front, body
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}
