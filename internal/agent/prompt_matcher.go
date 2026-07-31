package agent

import (
	"regexp"
	"strings"
)

type PromptVariant struct {
	Pattern string `json:"pattern"`
	Prompt  string `json:"prompt"`
}

type PromptMatcher struct {
	variants []PromptVariant
	default_ string
}

func NewPromptMatcher(defaultPrompt string, variants []PromptVariant) *PromptMatcher {
	return &PromptMatcher{
		variants: variants,
		default_: defaultPrompt,
	}
}

func (m *PromptMatcher) Match(model string) string {
	for _, v := range m.variants {
		if matchPattern(v.Pattern, model) {
			return v.Prompt
		}
	}
	return m.default_
}

func matchPattern(pattern, value string) bool {
	if pattern == value {
		return true
	}

	if strings.HasPrefix(pattern, "^") {
		return matchRegex(pattern, value)
	}

	if strings.Contains(pattern, "*") {
		return matchWildcard(pattern, value)
	}

	return false
}

func matchWildcard(pattern, value string) bool {
	escaped := regexp.QuoteMeta(pattern)
	escaped = strings.ReplaceAll(escaped, `\*`, ".*")

	regex := "^" + escaped + "$"
	return matchRegex(regex, value)
}

func matchRegex(pattern, value string) bool {
	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return false
	}
	return matched
}
