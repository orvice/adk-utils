// Package skill defines the data model for ADK skills.
//
// A skill represents a capability that can be loaded by an LLM agent
// through progressive disclosure: L1 metadata (name/description),
// L2 instructions (SKILL.md body), and L3 resources (references/assets/scripts).
package skill

import (
	"fmt"
	"regexp"
)

// kebabCaseRe matches valid kebab-case identifiers: lowercase letters, digits, and hyphens.
var kebabCaseRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

const (
	maxNameLen        = 64
	maxDescriptionLen = 1024
)

// Frontmatter holds the L1 metadata parsed from the YAML header of SKILL.md.
type Frontmatter struct {
	Name          string         `yaml:"name"`
	Description   string         `yaml:"description"`
	License       string         `yaml:"license,omitempty"`
	Compatibility string         `yaml:"compatibility,omitempty"`
	Metadata      map[string]any `yaml:"metadata,omitempty"`
}

// Validate checks that the frontmatter fields meet the required constraints.
func (f *Frontmatter) Validate() error {
	if f.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if len(f.Name) > maxNameLen {
		return fmt.Errorf("skill name must be at most %d characters, got %d", maxNameLen, len(f.Name))
	}
	if !kebabCaseRe.MatchString(f.Name) {
		return fmt.Errorf("skill name %q must be lowercase kebab-case (e.g. my-skill)", f.Name)
	}
	if f.Description == "" {
		return fmt.Errorf("skill description is required")
	}
	if len(f.Description) > maxDescriptionLen {
		return fmt.Errorf("skill description must be at most %d characters, got %d", maxDescriptionLen, len(f.Description))
	}
	return nil
}

// Resources holds the L3 content files associated with a skill.
type Resources struct {
	References map[string]string
	Assets     map[string]string
	Scripts    map[string]string
}

// GetReference returns the content of a reference file by key.
func (r *Resources) GetReference(key string) (string, bool) {
	v, ok := r.References[key]
	return v, ok
}

// GetAsset returns the content of an asset file by key.
func (r *Resources) GetAsset(key string) (string, bool) {
	v, ok := r.Assets[key]
	return v, ok
}

// GetScript returns the content of a script file by key.
func (r *Resources) GetScript(key string) (string, bool) {
	v, ok := r.Scripts[key]
	return v, ok
}

// Skill represents a complete skill with metadata, instructions, and resources.
type Skill struct {
	Frontmatter  Frontmatter
	Instructions string
	Resources    Resources
}

// Name returns the skill's name from its frontmatter.
func (s *Skill) Name() string {
	return s.Frontmatter.Name
}

// Description returns the skill's description from its frontmatter.
func (s *Skill) Description() string {
	return s.Frontmatter.Description
}
