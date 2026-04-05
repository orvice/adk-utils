package skilltoolset

import (
	"strings"
	"testing"

	"github.com/orvice/adk-utils/skill"
)

func makeSkill(name, desc, instructions string) *skill.Skill {
	return &skill.Skill{
		Frontmatter: skill.Frontmatter{
			Name:        name,
			Description: desc,
		},
		Instructions: instructions,
		Resources: skill.Resources{
			References: map[string]string{"api.md": "# API docs"},
			Assets:     map[string]string{"template.txt": "Hello {{name}}"},
			Scripts:    map[string]string{"setup.sh": "#!/bin/bash"},
		},
	}
}

func TestNewSkillToolset_DuplicateName(t *testing.T) {
	skills := []*skill.Skill{
		makeSkill("weather", "Weather", ""),
		makeSkill("weather", "Weather2", ""),
	}
	_, err := NewSkillToolset(skills)
	if err == nil {
		t.Fatal("expected error for duplicate skill name")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error = %q, want containing 'duplicate'", err)
	}
}

func TestSkillToolset_Tools(t *testing.T) {
	ts, err := NewSkillToolset([]*skill.Skill{makeSkill("weather", "Weather", "")})
	if err != nil {
		t.Fatal(err)
	}

	tools, err := ts.Tools(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	names := map[string]bool{}
	for _, tl := range tools {
		names[tl.Name()] = true
	}
	for _, expected := range []string{"list_skills", "load_skill", "load_skill_resource"} {
		if !names[expected] {
			t.Errorf("missing tool %q", expected)
		}
	}
}

func TestSkillToolset_Name(t *testing.T) {
	ts, err := NewSkillToolset(nil)
	if err != nil {
		t.Fatal(err)
	}
	if ts.Name() != "skill_toolset" {
		t.Errorf("Name = %q, want %q", ts.Name(), "skill_toolset")
	}
}

// --- list_skills (handler) ---

func TestListSkills(t *testing.T) {
	ts, _ := NewSkillToolset([]*skill.Skill{
		makeSkill("weather", "Weather data", ""),
		makeSkill("calendar", "Calendar events", ""),
	})

	result, err := ts.listSkills(nil, ListSkillsArgs{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Output, "weather") || !strings.Contains(result.Output, "calendar") {
		t.Errorf("output = %q, expected both skill names", result.Output)
	}
}

func TestListSkills_Empty(t *testing.T) {
	ts, _ := NewSkillToolset(nil)
	result, err := ts.listSkills(nil, ListSkillsArgs{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Output, "<available_skills>") {
		t.Error("expected <available_skills> tag even for empty list")
	}
}

// --- load_skill (handler) ---

func TestLoadSkill(t *testing.T) {
	ts, _ := NewSkillToolset([]*skill.Skill{
		makeSkill("weather", "Weather", "Use this to get weather"),
	})

	result, err := ts.loadSkill(nil, LoadSkillArgs{Name: "weather"})
	if err != nil {
		t.Fatal(err)
	}
	if result.SkillName != "weather" {
		t.Errorf("SkillName = %q, want weather", result.SkillName)
	}
	if result.Instructions != "Use this to get weather" {
		t.Errorf("Instructions = %q", result.Instructions)
	}
	if result.Frontmatter == nil {
		t.Error("expected frontmatter")
	}
}

func TestLoadSkill_NotFound(t *testing.T) {
	ts, _ := NewSkillToolset([]*skill.Skill{makeSkill("weather", "Weather", "")})
	result, err := ts.loadSkill(nil, LoadSkillArgs{Name: "nonexistent"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorCode != "SKILL_NOT_FOUND" {
		t.Errorf("ErrorCode = %q, want SKILL_NOT_FOUND", result.ErrorCode)
	}
}

func TestLoadSkill_MissingName(t *testing.T) {
	ts, _ := NewSkillToolset([]*skill.Skill{makeSkill("weather", "Weather", "")})
	result, err := ts.loadSkill(nil, LoadSkillArgs{})
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorCode != "MISSING_SKILL_NAME" {
		t.Errorf("ErrorCode = %q, want MISSING_SKILL_NAME", result.ErrorCode)
	}
}

// --- load_skill_resource (handler) ---

func TestLoadSkillResource_Reference(t *testing.T) {
	ts, _ := NewSkillToolset([]*skill.Skill{makeSkill("weather", "Weather", "")})
	result, err := ts.loadSkillResource(nil, LoadSkillResourceArgs{
		SkillName: "weather",
		Path:      "references/api.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "# API docs" {
		t.Errorf("Content = %q", result.Content)
	}
}

func TestLoadSkillResource_Asset(t *testing.T) {
	ts, _ := NewSkillToolset([]*skill.Skill{makeSkill("weather", "Weather", "")})
	result, err := ts.loadSkillResource(nil, LoadSkillResourceArgs{
		SkillName: "weather",
		Path:      "assets/template.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "Hello {{name}}" {
		t.Errorf("Content = %q", result.Content)
	}
}

func TestLoadSkillResource_Script(t *testing.T) {
	ts, _ := NewSkillToolset([]*skill.Skill{makeSkill("weather", "Weather", "")})
	result, err := ts.loadSkillResource(nil, LoadSkillResourceArgs{
		SkillName: "weather",
		Path:      "scripts/setup.sh",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "#!/bin/bash" {
		t.Errorf("Content = %q", result.Content)
	}
}

func TestLoadSkillResource_InvalidPath(t *testing.T) {
	ts, _ := NewSkillToolset([]*skill.Skill{makeSkill("weather", "Weather", "")})
	result, err := ts.loadSkillResource(nil, LoadSkillResourceArgs{
		SkillName: "weather",
		Path:      "other/file.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorCode != "INVALID_RESOURCE_PATH" {
		t.Errorf("ErrorCode = %q, want INVALID_RESOURCE_PATH", result.ErrorCode)
	}
}

func TestLoadSkillResource_NotFound(t *testing.T) {
	ts, _ := NewSkillToolset([]*skill.Skill{makeSkill("weather", "Weather", "")})
	result, err := ts.loadSkillResource(nil, LoadSkillResourceArgs{
		SkillName: "weather",
		Path:      "references/missing.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorCode != "RESOURCE_NOT_FOUND" {
		t.Errorf("ErrorCode = %q, want RESOURCE_NOT_FOUND", result.ErrorCode)
	}
}

func TestLoadSkillResource_SkillNotFound(t *testing.T) {
	ts, _ := NewSkillToolset([]*skill.Skill{makeSkill("weather", "Weather", "")})
	result, err := ts.loadSkillResource(nil, LoadSkillResourceArgs{
		SkillName: "nonexistent",
		Path:      "references/api.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorCode != "SKILL_NOT_FOUND" {
		t.Errorf("ErrorCode = %q, want SKILL_NOT_FOUND", result.ErrorCode)
	}
}

// --- Tool descriptions contain skill guidance ---

func TestToolDescription_ContainsGuidance(t *testing.T) {
	ts, _ := NewSkillToolset([]*skill.Skill{makeSkill("weather", "Weather", "")})
	tools, _ := ts.Tools(nil)

	for _, tl := range tools {
		if tl.Name() == "list_skills" {
			desc := tl.Description()
			if !strings.Contains(desc, "list_skills") {
				t.Error("list_skills description should contain usage guidance")
			}
			return
		}
	}
	t.Error("list_skills tool not found")
}
