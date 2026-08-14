package filesystem

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type IgnoreRules struct {
	rules   []IgnoreRule
	sources []string
}

func LoadIgnoreRules(rootDir string) (*IgnoreRules, error) {
	ir := &IgnoreRules{}

	ir.rules = append(ir.rules, IgnoreRule{
		Pattern: ".git",
		Regex:   regexp.MustCompile(`(^|/)\.git(/|$)`),
		DirOnly: true,
		Source:  "default",
	})
	ir.addSource("default")

	gitignorePath := filepath.Join(rootDir, ".gitignore")
	if err := ir.AddFromFile(gitignorePath, ".gitignore"); err != nil {
		return nil, fmt.Errorf("failed to load .gitignore: %w", err)
	}

	vesvaignorePath := filepath.Join(rootDir, ".vesvaignore")
	if err := ir.AddFromFile(vesvaignorePath, ".vesvaignore"); err != nil {
		return nil, fmt.Errorf("failed to load .vesvaignore: %w", err)
	}

	return ir, nil
}

func (ir *IgnoreRules) AddFromFile(filePath, source string) error {
	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		rule, err := parseIgnoreLine(line, source)
		if err != nil {
			continue
		}
		ir.rules = append(ir.rules, *rule)
	}

	ir.addSource(source)

	return scanner.Err()
}

func (ir *IgnoreRules) addSource(source string) {
	for _, s := range ir.sources {
		if s == source {
			return
		}
	}
	ir.sources = append(ir.sources, source)
}

func (ir *IgnoreRules) IsIgnored(relPath string, isDir bool) bool {
	relPath = cleanPath(relPath)

	for _, source := range ir.sources {
		if ir.isIgnoredBySource(relPath, isDir, source) {
			return true
		}
	}

	return false
}

func (ir *IgnoreRules) isIgnoredBySource(relPath string, isDir bool, source string) bool {
	result := false

	for _, rule := range ir.rules {
		if rule.Source != source {
			continue
		}

		matched := false

		if rule.DirOnly {
			if isDir {
				matched = rule.Regex.MatchString(relPath)
			} else {
				dir := filepath.Dir(relPath)
				for dir != "." && dir != "/" {
					if rule.Regex.MatchString(dir) {
						matched = true
						break
					}
					dir = filepath.Dir(dir)
				}
			}
		} else {
			matched = rule.Regex.MatchString(relPath)
			if !matched && !isDir {
				dir := filepath.Dir(relPath)
				for dir != "." && dir != "/" {
					if rule.Regex.MatchString(dir) {
						matched = true
						break
					}
					dir = filepath.Dir(dir)
				}
			}
		}

		if matched {
			if rule.Negate {
				result = false
			} else {
				result = true
			}
		}
	}

	return result
}

func parseIgnoreLine(line, source string) (*IgnoreRule, error) {
	negate := false
	if strings.HasPrefix(line, "!") {
		negate = true
		line = line[1:]
	}

	dirOnly := false
	if strings.HasSuffix(line, "/") {
		dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty pattern after processing")
	}

	regexPattern := toRegexPattern(line)
	regex, err := regexp.Compile("^" + regexPattern + "$")
	if err != nil {
		return nil, fmt.Errorf("invalid pattern %q: %w", line, err)
	}

	return &IgnoreRule{
		Pattern: line,
		Regex:   regex,
		Negate:  negate,
		DirOnly: dirOnly,
		Source:  source,
	}, nil
}
