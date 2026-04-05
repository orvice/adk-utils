package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseFrontmatter splits a SKILL.md content string into YAML frontmatter and
// the markdown body. The frontmatter is delimited by "---" lines at the start.
// This is exported so that custom Loader implementations can reuse the parsing logic.
func ParseFrontmatter(content string) (*Frontmatter, string, error) {
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

// FSLoader loads skills from the local filesystem.
// It implements the Loader interface.
type FSLoader struct {
	// Dir is the parent directory containing skill subdirectories.
	// Used by LoadAll to enumerate skills.
	Dir string
}

// NewFSLoader creates a filesystem-based Loader.
// dir is the parent directory containing skill subdirectories (used by LoadAll).
func NewFSLoader(dir string) *FSLoader {
	return &FSLoader{Dir: dir}
}

// LoadSkill loads a single skill from the given directory path.
// The directory must contain a SKILL.md file and may optionally contain
// references/, assets/, and scripts/ subdirectories.
func (l *FSLoader) LoadSkill(dir string) (*Skill, error) {
	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return nil, fmt.Errorf("reading SKILL.md in %s: %w", dir, err)
	}

	fm, instructions, err := ParseFrontmatter(string(data))
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

// LoadAll loads all skills from subdirectories of l.Dir.
// Each subdirectory that contains a SKILL.md file is loaded as a skill.
// Subdirectories without SKILL.md are silently skipped.
func (l *FSLoader) LoadAll() ([]*Skill, error) {
	entries, err := os.ReadDir(l.Dir)
	if err != nil {
		return nil, fmt.Errorf("reading skill directory %s: %w", l.Dir, err)
	}

	var skills []*Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		subdir := filepath.Join(l.Dir, e.Name())
		if _, err := os.Stat(filepath.Join(subdir, "SKILL.md")); os.IsNotExist(err) {
			continue
		}
		s, err := l.LoadSkill(subdir)
		if err != nil {
			return nil, fmt.Errorf("loading skill from %s: %w", subdir, err)
		}
		skills = append(skills, s)
	}
	return skills, nil
}

var _ Loader = (*FSLoader)(nil)

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

// --- Legacy convenience functions (delegate to FSLoader) ---

// LoadSkill loads a single skill from a directory.
// This is a convenience wrapper around FSLoader.LoadSkill.
func LoadSkill(dir string) (*Skill, error) {
	return (&FSLoader{}).LoadSkill(dir)
}

// LoadSkillDir loads all skills from subdirectories of the given directory.
// This is a convenience wrapper around FSLoader.LoadAll.
func LoadSkillDir(dir string) ([]*Skill, error) {
	return NewFSLoader(dir).LoadAll()
}
