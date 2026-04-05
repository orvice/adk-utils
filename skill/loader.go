package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// parseFrontmatter splits a SKILL.md content into YAML frontmatter and markdown body.
// The frontmatter is delimited by "---" lines at the start of the file.
func parseFrontmatter(content string) (*Frontmatter, string, error) {
	const delim = "---"

	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, delim) {
		return nil, "", fmt.Errorf("SKILL.md must start with ---")
	}

	// Find the closing delimiter.
	rest := trimmed[len(delim):]
	idx := strings.Index(rest, "\n"+delim)
	if idx < 0 {
		return nil, "", fmt.Errorf("SKILL.md missing closing ---")
	}

	yamlBlock := rest[:idx]
	// Body starts after the closing "---\n".
	body := rest[idx+1+len(delim):]
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
	}

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return nil, "", fmt.Errorf("parsing SKILL.md frontmatter: %w", err)
	}

	return &fm, body, nil
}

// loadResourceDir reads all files in a directory into a string map keyed by filename.
// Returns an empty map if the directory does not exist.
func loadResourceDir(dir string) (map[string]string, error) {
	m := make(map[string]string)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading file %s: %w", e.Name(), err)
		}
		m[e.Name()] = string(data)
	}
	return m, nil
}

// LoadSkill loads a single skill from a directory containing a SKILL.md file
// and optional references/, assets/, and scripts/ subdirectories.
func LoadSkill(dir string) (*Skill, error) {
	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return nil, fmt.Errorf("reading SKILL.md in %s: %w", dir, err)
	}

	fm, instructions, err := parseFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("in %s: %w", dir, err)
	}

	if err := fm.Validate(); err != nil {
		return nil, fmt.Errorf("in %s: %w", dir, err)
	}

	refs, err := loadResourceDir(filepath.Join(dir, "references"))
	if err != nil {
		return nil, err
	}
	assets, err := loadResourceDir(filepath.Join(dir, "assets"))
	if err != nil {
		return nil, err
	}
	scripts, err := loadResourceDir(filepath.Join(dir, "scripts"))
	if err != nil {
		return nil, err
	}

	return &Skill{
		Frontmatter:  *fm,
		Instructions: instructions,
		Resources: Resources{
			References: refs,
			Assets:     assets,
			Scripts:    scripts,
		},
	}, nil
}

// LoadSkillDir loads all skills from subdirectories of the given directory.
// Each subdirectory that contains a SKILL.md file is loaded as a skill.
// Subdirectories without SKILL.md are silently skipped.
func LoadSkillDir(dir string) ([]*Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading skill directory %s: %w", dir, err)
	}

	var skills []*Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		subdir := filepath.Join(dir, e.Name())
		// Skip subdirectories without SKILL.md.
		if _, err := os.Stat(filepath.Join(subdir, "SKILL.md")); os.IsNotExist(err) {
			continue
		}
		s, err := LoadSkill(subdir)
		if err != nil {
			return nil, fmt.Errorf("loading skill from %s: %w", subdir, err)
		}
		skills = append(skills, s)
	}
	return skills, nil
}
