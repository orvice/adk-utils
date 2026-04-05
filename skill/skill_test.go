package skill

import "testing"

func TestFrontmatter_Validate_Valid(t *testing.T) {
	fm := Frontmatter{Name: "weather-skill", Description: "Queries weather data"}
	if err := fm.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFrontmatter_Validate_EmptyName(t *testing.T) {
	fm := Frontmatter{Description: "test"}
	if err := fm.Validate(); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestFrontmatter_Validate_InvalidName(t *testing.T) {
	tests := []string{
		"Weather Skill",
		"UPPER",
		"has_underscore",
		"has.dot",
		"-leading-hyphen",
		"trailing-hyphen-",
	}
	for _, name := range tests {
		fm := Frontmatter{Name: name, Description: "test"}
		if err := fm.Validate(); err == nil {
			t.Errorf("expected error for name %q", name)
		}
	}
}

func TestFrontmatter_Validate_NameTooLong(t *testing.T) {
	long := ""
	for i := 0; i < 65; i++ {
		long += "a"
	}
	fm := Frontmatter{Name: long, Description: "test"}
	if err := fm.Validate(); err == nil {
		t.Error("expected error for name > 64 chars")
	}
}

func TestFrontmatter_Validate_EmptyDescription(t *testing.T) {
	fm := Frontmatter{Name: "my-skill"}
	if err := fm.Validate(); err == nil {
		t.Error("expected error for empty description")
	}
}

func TestResources_GetReference(t *testing.T) {
	r := Resources{References: map[string]string{"api.md": "content"}}
	if v, ok := r.GetReference("api.md"); !ok || v != "content" {
		t.Errorf("GetReference = (%q, %v), want (%q, true)", v, ok, "content")
	}
	if _, ok := r.GetReference("missing"); ok {
		t.Error("expected false for missing reference")
	}
}

func TestResources_GetAsset(t *testing.T) {
	r := Resources{Assets: map[string]string{"tmpl.txt": "data"}}
	if v, ok := r.GetAsset("tmpl.txt"); !ok || v != "data" {
		t.Errorf("GetAsset = (%q, %v), want (%q, true)", v, ok, "data")
	}
	if _, ok := r.GetAsset("missing"); ok {
		t.Error("expected false for missing asset")
	}
}

func TestResources_GetScript(t *testing.T) {
	r := Resources{Scripts: map[string]string{"run.sh": "#!/bin/bash"}}
	if v, ok := r.GetScript("run.sh"); !ok || v != "#!/bin/bash" {
		t.Errorf("GetScript = (%q, %v), want (%q, true)", v, ok, "#!/bin/bash")
	}
	if _, ok := r.GetScript("missing"); ok {
		t.Error("expected false for missing script")
	}
}

func TestSkill_NameDescription(t *testing.T) {
	s := Skill{Frontmatter: Frontmatter{Name: "my-skill", Description: "Does things"}}
	if s.Name() != "my-skill" {
		t.Errorf("Name = %q, want %q", s.Name(), "my-skill")
	}
	if s.Description() != "Does things" {
		t.Errorf("Description = %q, want %q", s.Description(), "Does things")
	}
}
