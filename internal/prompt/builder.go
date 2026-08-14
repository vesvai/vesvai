package prompt

import (
	"fmt"
	"sort"
	"strings"
)

const (
	OrderFrontmatter = -10
	OrderRole        = iota * 10
	OrderPersona
	OrderObjective
	OrderGoal
	OrderEnvironment
	OrderStrengths  = 75
	OrderGuidelines = 115
	OrderContext
	OrderMemory
	OrderKnowledge
	OrderSkills
	OrderTools
	OrderWorkflow
	OrderInstructions
	OrderRules
	OrderRequirements
	OrderConstraints
	OrderLimitations
	OrderWarnings
	OrderFailureCases
	OrderSuccessCriteria
	OrderChecklist
	OrderStyle
	OrderNotes
	OrderThinking
	OrderOutputFormat
	OrderSchema
	OrderOutput
	OrderExamples
	OrderXML
	OrderCustom = 1000
)

type Builder struct {
	sections []section
	vars     map[string]string
}

type section struct {
	name    string
	content string
	order   int
}

type Example struct {
	User      string
	Assistant string
}

type Step struct {
	Title       string
	Description string
	Bullets     []string
}

func New() *Builder {
	return &Builder{
		vars: make(map[string]string),
	}
}

func (b *Builder) Section(name, content string, order int) *Builder {
	b.sections = append(b.sections, section{name: name, content: content, order: order})
	return b
}

func (b *Builder) CustomList(name, title string, items []string, numbered bool, order int) *Builder {
	if len(items) == 0 {
		return b
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	for i, item := range items {
		if numbered {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
		} else {
			sb.WriteString(fmt.Sprintf("- %s\n", item))
		}
	}
	return b.Section(name, strings.TrimSpace(sb.String()), order)
}

func (b *Builder) textSection(name, title string, content ...string) *Builder {
	if len(content) == 0 {
		return b
	}
	body := strings.Join(content, " ")
	return b.Section(name, fmt.Sprintf("# %s\n\n%s", title, body), b.nameToOrder(name))
}

func (b *Builder) bulletList(name string, items []string) *Builder {
	return b.CustomList(name, strings.ToTitle(name[:1])+name[1:], items, false, b.nameToOrder(name))
}

func (b *Builder) numberedList(name string, items []string) *Builder {
	return b.CustomList(name, strings.ToTitle(name[:1])+name[1:], items, true, b.nameToOrder(name))
}

func (b *Builder) Var(key, value string) *Builder {
	b.vars[key] = value
	return b
}

func (b *Builder) Variables(vars map[string]string) *Builder {
	for k, v := range vars {
		b.vars[k] = v
	}
	return b
}

func (b *Builder) If(condition bool, fn func(*Builder)) *Builder {
	if condition {
		fn(b)
	}
	return b
}
func (b *Builder) Frontmatter(content string) *Builder {
	return b.Section("frontmatter", content, OrderFrontmatter)
}
func (b *Builder) Strengths(items ...string) *Builder  { return b.bulletList("strengths", items) }
func (b *Builder) Guidelines(items ...string) *Builder { return b.bulletList("guidelines", items) }
func (b *Builder) Role(text ...string) *Builder        { return b.textSection("role", "Role", text...) }
func (b *Builder) Persona(text ...string) *Builder {
	return b.textSection("persona", "Persona", text...)
}
func (b *Builder) Objective(text ...string) *Builder {
	return b.textSection("objective", "Objective", text...)
}
func (b *Builder) Goal(text ...string) *Builder { return b.textSection("goal", "Goal", text...) }
func (b *Builder) Environment(text ...string) *Builder {
	return b.textSection("environment", "Environment", text...)
}
func (b *Builder) Memory(text ...string) *Builder { return b.textSection("memory", "Memory", text...) }
func (b *Builder) Knowledge(text ...string) *Builder {
	return b.textSection("knowledge", "Knowledge", text...)
}
func (b *Builder) Style(text ...string) *Builder { return b.textSection("style", "Style", text...) }
func (b *Builder) Notes(text ...string) *Builder { return b.textSection("notes", "Notes", text...) }
func (b *Builder) Thinking(text ...string) *Builder {
	return b.textSection("thinking", "Thinking Process", text...)
}
func (b *Builder) OutputFormat(text ...string) *Builder {
	return b.textSection("outputFormat", "Output Format", text...)
}
func (b *Builder) Schema(text ...string) *Builder { return b.textSection("schema", "Schema", text...) }
func (b *Builder) Output(text ...string) *Builder { return b.textSection("output", "Output", text...) }
func (b *Builder) Metadata(text ...string) *Builder {
	return b.textSection("metadata", "Metadata", text...)
}

func (b *Builder) Context(items ...string) *Builder      { return b.bulletList("context", items) }
func (b *Builder) Constraints(items ...string) *Builder  { return b.bulletList("constraints", items) }
func (b *Builder) Limitations(items ...string) *Builder  { return b.bulletList("limitations", items) }
func (b *Builder) Warnings(items ...string) *Builder     { return b.bulletList("warnings", items) }
func (b *Builder) FailureCases(items ...string) *Builder { return b.bulletList("failureCases", items) }
func (b *Builder) SuccessCriteria(items ...string) *Builder {
	return b.bulletList("successCriteria", items)
}
func (b *Builder) Checklist(items ...string) *Builder { return b.bulletList("checklist", items) }

func (b *Builder) Workflow(items ...string) *Builder { return b.numberedList("workflow", items) }
func (b *Builder) Instructions(items ...string) *Builder {
	return b.numberedList("instructions", items)
}
func (b *Builder) Rules(items ...string) *Builder { return b.numberedList("rules", items) }
func (b *Builder) Requirements(items ...string) *Builder {
	return b.numberedList("requirements", items)
}

func (b *Builder) Examples(examples ...Example) *Builder {
	if len(examples) == 0 {
		return b
	}
	var sb strings.Builder
	sb.WriteString("# Examples\n\n")
	for i, ex := range examples {
		sb.WriteString(fmt.Sprintf("## Example %d\n\n", i+1))
		sb.WriteString(fmt.Sprintf("**User:** %s\n\n", ex.User))
		sb.WriteString(fmt.Sprintf("**Assistant:** %s\n\n", ex.Assistant))
	}
	return b.Section("examples", strings.TrimSpace(sb.String()), OrderExamples)
}

func (b *Builder) XML(tag, content string) *Builder {
	xmlContent := fmt.Sprintf("<%s>\n%s\n</%s>", tag, content, tag)
	return b.Section(tag, xmlContent, OrderXML)
}

func (b *Builder) Steps(name, title string, steps []Step) *Builder {
	if len(steps) == 0 {
		return b
	}

	var sb strings.Builder
	if title != "" {
		sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	}

	for i, step := range steps {
		sb.WriteString(fmt.Sprintf("%d. **%s**", i+1, step.Title))
		if step.Description != "" {
			sb.WriteString(fmt.Sprintf(": %s", step.Description))
		}
		sb.WriteString("\n")

		for _, bullet := range step.Bullets {
			sb.WriteString(fmt.Sprintf("   - %s\n", bullet))
		}

		if i < len(steps)-1 {
			sb.WriteString("\n")
		}
	}

	return b.Section(name, strings.TrimSpace(sb.String()), b.nameToOrder(name))
}

func (b *Builder) Process(steps ...Step) *Builder {
	return b.Steps("workflow", "Your Process", steps)
}

func (b *Builder) Build() string {
	sort.SliceStable(b.sections, func(i, j int) bool {
		return b.sections[i].order < b.sections[j].order
	})

	var replacerArgs []string
	for k, v := range b.vars {
		replacerArgs = append(replacerArgs, fmt.Sprintf("{{.%s}}", k), v)
	}
	var replacer *strings.Replacer
	if len(replacerArgs) > 0 {
		replacer = strings.NewReplacer(replacerArgs...)
	}

	var sb strings.Builder
	for i, s := range b.sections {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		content := s.content
		if replacer != nil {
			content = replacer.Replace(content)
		}
		sb.WriteString(content)
	}

	return sb.String()
}

func (b *Builder) nameToOrder(name string) int {
	switch name {
	case "role":
		return OrderRole
	case "persona":
		return OrderPersona
	case "objective":
		return OrderObjective
	case "goal":
		return OrderGoal
	case "environment":
		return OrderEnvironment
	case "context":
		return OrderContext
	case "memory":
		return OrderMemory
	case "knowledge":
		return OrderKnowledge
	case "skills":
		return OrderSkills
	case "tools":
		return OrderTools
	case "workflow":
		return OrderWorkflow
	case "instructions":
		return OrderInstructions
	case "rules":
		return OrderRules
	case "requirements":
		return OrderRequirements
	case "constraints":
		return OrderConstraints
	case "limitations":
		return OrderLimitations
	case "warnings":
		return OrderWarnings
	case "failureCases":
		return OrderFailureCases
	case "successCriteria":
		return OrderSuccessCriteria
	case "checklist":
		return OrderChecklist
	case "style":
		return OrderStyle
	case "notes":
		return OrderNotes
	case "thinking":
		return OrderThinking
	case "outputFormat":
		return OrderOutputFormat
	case "schema":
		return OrderSchema
	case "output":
		return OrderOutput
	case "examples":
		return OrderExamples
	default:
		return OrderCustom
	}
}
