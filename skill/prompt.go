package skill

import (
	"fmt"
	"strings"
)

// FormatSkillsAsXML formats a list of skills as an XML block for injection
// into LLM system instructions.
func FormatSkillsAsXML(skills []*Skill) string {
	var b strings.Builder
	b.WriteString("<available_skills>\n")
	for _, s := range skills {
		fmt.Fprintf(&b, "  <skill name=%q description=%q />\n", s.Name(), s.Description())
	}
	b.WriteString("</available_skills>")
	return b.String()
}
