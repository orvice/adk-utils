package skill

import (
	"strings"
	"testing"
)

func TestFormatSkillsAsXML(t *testing.T) {
	skills := []*Skill{
		{Frontmatter: Frontmatter{Name: "weather", Description: "Weather data"}},
		{Frontmatter: Frontmatter{Name: "calendar", Description: "Calendar events"}},
	}

	xml := FormatSkillsAsXML(skills)

	if !strings.Contains(xml, "<available_skills>") {
		t.Error("expected <available_skills> tag")
	}
	if !strings.Contains(xml, "</available_skills>") {
		t.Error("expected </available_skills> tag")
	}
	if !strings.Contains(xml, `name="weather"`) {
		t.Error("expected weather skill")
	}
	if !strings.Contains(xml, `name="calendar"`) {
		t.Error("expected calendar skill")
	}
}

func TestFormatSkillsAsXML_Empty(t *testing.T) {
	xml := FormatSkillsAsXML(nil)
	if !strings.Contains(xml, "<available_skills>") {
		t.Error("expected <available_skills> tag")
	}
	if !strings.Contains(xml, "</available_skills>") {
		t.Error("expected </available_skills> tag")
	}
}
