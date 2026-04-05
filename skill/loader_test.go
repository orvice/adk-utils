package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSkill_AllComponents(t *testing.T) {
	s, err := LoadSkill("testdata/weather-skill")
	if err != nil {
		t.Fatalf("LoadSkill: %v", err)
	}

	if s.Name() != "weather-skill" {
		t.Errorf("Name = %q, want %q", s.Name(), "weather-skill")
	}
	if s.Description() != "Queries weather data for any location" {
		t.Errorf("Description = %q, want %q", s.Description(), "Queries weather data for any location")
	}
	if s.Frontmatter.License != "MIT" {
		t.Errorf("License = %q, want %q", s.Frontmatter.License, "MIT")
	}
	if s.Instructions == "" {
		t.Error("Instructions should not be empty")
	}

	if _, ok := s.Resources.GetReference("api.md"); !ok {
		t.Error("expected reference api.md")
	}
	if _, ok := s.Resources.GetAsset("template.txt"); !ok {
		t.Error("expected asset template.txt")
	}
	if _, ok := s.Resources.GetScript("setup.sh"); !ok {
		t.Error("expected script setup.sh")
	}
}

func TestLoadSkill_OnlySKILLMD(t *testing.T) {
	s, err := LoadSkill("testdata/calendar-skill")
	if err != nil {
		t.Fatalf("LoadSkill: %v", err)
	}

	if s.Name() != "calendar-skill" {
		t.Errorf("Name = %q, want %q", s.Name(), "calendar-skill")
	}
	if len(s.Resources.References) != 0 {
		t.Errorf("expected empty References, got %d", len(s.Resources.References))
	}
	if len(s.Resources.Assets) != 0 {
		t.Errorf("expected empty Assets, got %d", len(s.Resources.Assets))
	}
	if len(s.Resources.Scripts) != 0 {
		t.Errorf("expected empty Scripts, got %d", len(s.Resources.Scripts))
	}
}

func TestLoadSkill_NoSKILLMD(t *testing.T) {
	_, err := LoadSkill("testdata/no-skill")
	if err == nil {
		t.Fatal("expected error for directory without SKILL.md")
	}
}

func TestLoadSkillDir(t *testing.T) {
	skills, err := LoadSkillDir("testdata")
	if err != nil {
		t.Fatalf("LoadSkillDir: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}

	names := map[string]bool{}
	for _, s := range skills {
		names[s.Name()] = true
	}
	if !names["weather-skill"] || !names["calendar-skill"] {
		t.Errorf("expected weather-skill and calendar-skill, got %v", names)
	}
}

func TestLoadSkillDir_Empty(t *testing.T) {
	dir := t.TempDir()
	skills, err := LoadSkillDir(dir)
	if err != nil {
		t.Fatalf("LoadSkillDir: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
}

func TestParseFrontmatter(t *testing.T) {
	content := "---\nname: test-skill\ndescription: A test\n---\n# Body\nHello"
	fm, body, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	if fm.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", fm.Name, "test-skill")
	}
	if body != "# Body\nHello" {
		t.Errorf("body = %q, want %q", body, "# Body\nHello")
	}
}

func TestParseFrontmatter_MissingDelimiter(t *testing.T) {
	_, _, err := ParseFrontmatter("no frontmatter here")
	if err == nil {
		t.Fatal("expected error for missing frontmatter")
	}
}

func TestFSLoader_AsInterface(t *testing.T) {
	var loader Loader = NewFSLoader("testdata")

	// LoadSkill via interface
	s, err := loader.LoadSkill("testdata/weather-skill")
	if err != nil {
		t.Fatalf("LoadSkill: %v", err)
	}
	if s.Name() != "weather-skill" {
		t.Errorf("Name = %q, want %q", s.Name(), "weather-skill")
	}

	// LoadAll via interface
	skills, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}
}

func TestLoadSkill_InvalidName(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: Bad Name\ndescription: test\n---\nbody"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadSkill(dir)
	if err == nil {
		t.Fatal("expected validation error for invalid name")
	}
}
