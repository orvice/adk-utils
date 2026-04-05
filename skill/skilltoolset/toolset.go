// Package skilltoolset implements a tool.Toolset that exposes skills
// to an LLM agent through progressive disclosure.
package skilltoolset

import (
	"fmt"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/orvice/adk-utils/skill"
)

// SkillToolset implements tool.Toolset, providing list_skills, load_skill,
// and load_skill_resource tools for LLM agents.
type SkillToolset struct {
	skills map[string]*skill.Skill
	tools  []tool.Tool
}

// NewSkillToolset creates a new SkillToolset from the given skills.
// Returns an error if duplicate skill names are found.
func NewSkillToolset(skills []*skill.Skill) (*SkillToolset, error) {
	m := make(map[string]*skill.Skill, len(skills))
	for _, s := range skills {
		if _, exists := m[s.Name()]; exists {
			return nil, fmt.Errorf("duplicate skill name: %q", s.Name())
		}
		m[s.Name()] = s
	}

	ts := &SkillToolset{skills: m}

	listTool, err := functiontool.New(
		functiontool.Config{
			Name: "list_skills",
			Description: "Lists all available skills with their names and descriptions.\n\n" +
				"You have access to skills. Skills provide specialized capabilities you can use.\n" +
				"Use list_skills to see available skills, load_skill to get instructions for a specific skill,\n" +
				"and load_skill_resource to access a skill's reference files, assets, or scripts.",
		},
		ts.listSkills,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create list_skills tool: %w", err)
	}

	loadTool, err := functiontool.New(
		functiontool.Config{
			Name:        "load_skill",
			Description: "Loads a skill's full instructions by name.",
		},
		ts.loadSkill,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create load_skill tool: %w", err)
	}

	resTool, err := functiontool.New(
		functiontool.Config{
			Name:        "load_skill_resource",
			Description: "Loads a resource file from a skill. Path must start with references/, assets/, or scripts/.",
		},
		ts.loadSkillResource,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create load_skill_resource tool: %w", err)
	}

	ts.tools = []tool.Tool{listTool, loadTool, resTool}
	return ts, nil
}

// Name implements tool.Toolset.
func (ts *SkillToolset) Name() string {
	return "skill_toolset"
}

// Tools implements tool.Toolset.
func (ts *SkillToolset) Tools(_ agent.ReadonlyContext) ([]tool.Tool, error) {
	return ts.tools, nil
}

// skillSlice returns all skills as a slice (for XML formatting).
func (ts *SkillToolset) skillSlice() []*skill.Skill {
	out := make([]*skill.Skill, 0, len(ts.skills))
	for _, s := range ts.skills {
		out = append(out, s)
	}
	return out
}

// --- Args / Result types ---

// ListSkillsArgs is the (empty) input for list_skills.
type ListSkillsArgs struct{}

// ListSkillsResult is the output of list_skills.
type ListSkillsResult struct {
	Output string `json:"output"`
}

func (ts *SkillToolset) listSkills(_ tool.Context, _ ListSkillsArgs) (ListSkillsResult, error) {
	return ListSkillsResult{
		Output: skill.FormatSkillsAsXML(ts.skillSlice()),
	}, nil
}

// LoadSkillArgs is the input for load_skill.
type LoadSkillArgs struct {
	Name string `json:"name" jsonschema:"The name of the skill to load."`
}

// LoadSkillResult is the output of load_skill.
type LoadSkillResult struct {
	SkillName    string            `json:"skill_name,omitempty"`
	Instructions string            `json:"instructions,omitempty"`
	Frontmatter  map[string]string `json:"frontmatter,omitempty"`
	Error        string            `json:"error,omitempty"`
	ErrorCode    string            `json:"error_code,omitempty"`
}

func (ts *SkillToolset) loadSkill(_ tool.Context, args LoadSkillArgs) (LoadSkillResult, error) {
	if args.Name == "" {
		return LoadSkillResult{Error: "name parameter is required", ErrorCode: "MISSING_SKILL_NAME"}, nil
	}

	s, found := ts.skills[args.Name]
	if !found {
		return LoadSkillResult{
			Error:     fmt.Sprintf("skill %q not found", args.Name),
			ErrorCode: "SKILL_NOT_FOUND",
		}, nil
	}

	return LoadSkillResult{
		SkillName:    s.Name(),
		Instructions: s.Instructions,
		Frontmatter: map[string]string{
			"name":          s.Frontmatter.Name,
			"description":   s.Frontmatter.Description,
			"license":       s.Frontmatter.License,
			"compatibility": s.Frontmatter.Compatibility,
		},
	}, nil
}

// LoadSkillResourceArgs is the input for load_skill_resource.
type LoadSkillResourceArgs struct {
	SkillName string `json:"skill_name" jsonschema:"The name of the skill."`
	Path      string `json:"path" jsonschema:"Resource path (must start with references/, assets/, or scripts/)."`
}

// LoadSkillResourceResult is the output of load_skill_resource.
type LoadSkillResourceResult struct {
	SkillName string `json:"skill_name,omitempty"`
	Path      string `json:"path,omitempty"`
	Content   string `json:"content,omitempty"`
	Error     string `json:"error,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
}

var validPrefixes = []string{"references/", "assets/", "scripts/"}

func (ts *SkillToolset) loadSkillResource(_ tool.Context, args LoadSkillResourceArgs) (LoadSkillResourceResult, error) {
	if args.SkillName == "" {
		return LoadSkillResourceResult{Error: "skill_name is required", ErrorCode: "MISSING_SKILL_NAME"}, nil
	}
	if args.Path == "" {
		return LoadSkillResourceResult{Error: "path is required", ErrorCode: "MISSING_PATH"}, nil
	}

	validPrefix := false
	for _, p := range validPrefixes {
		if strings.HasPrefix(args.Path, p) {
			validPrefix = true
			break
		}
	}
	if !validPrefix {
		return LoadSkillResourceResult{
			Error:     "path must start with references/, assets/, or scripts/",
			ErrorCode: "INVALID_RESOURCE_PATH",
		}, nil
	}

	s, found := ts.skills[args.SkillName]
	if !found {
		return LoadSkillResourceResult{
			Error:     fmt.Sprintf("skill %q not found", args.SkillName),
			ErrorCode: "SKILL_NOT_FOUND",
		}, nil
	}

	var content string
	var resourceFound bool

	switch {
	case strings.HasPrefix(args.Path, "references/"):
		content, resourceFound = s.Resources.GetReference(strings.TrimPrefix(args.Path, "references/"))
	case strings.HasPrefix(args.Path, "assets/"):
		content, resourceFound = s.Resources.GetAsset(strings.TrimPrefix(args.Path, "assets/"))
	case strings.HasPrefix(args.Path, "scripts/"):
		content, resourceFound = s.Resources.GetScript(strings.TrimPrefix(args.Path, "scripts/"))
	}

	if !resourceFound {
		return LoadSkillResourceResult{
			Error:     fmt.Sprintf("resource %q not found in skill %q", args.Path, args.SkillName),
			ErrorCode: "RESOURCE_NOT_FOUND",
		}, nil
	}

	return LoadSkillResourceResult{
		SkillName: args.SkillName,
		Path:      args.Path,
		Content:   content,
	}, nil
}

var _ tool.Toolset = (*SkillToolset)(nil)
