package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Analyzer extracts knowledge from a codebase
type Analyzer struct {
	workspacePath string
}

// NewAnalyzer creates a new codebase analyzer
func NewAnalyzer(workspacePath string) *Analyzer {
	return &Analyzer{
		workspacePath: workspacePath,
	}
}

// Analyze performs a comprehensive analysis of the codebase and returns facts
func (a *Analyzer) Analyze() (*WorkspaceMemory, error) {
	memory := &WorkspaceMemory{
		Facts:   make([]Fact, 0),
		Notes:   make([]Note, 0),
		Version: 1,
	}

	// Analyze project structure
	if err := a.analyzeProjectStructure(memory); err != nil {
		return nil, fmt.Errorf("failed to analyze project structure: %w", err)
	}

	// Analyze language and framework
	if err := a.analyzeLanguageAndFramework(memory); err != nil {
		return nil, fmt.Errorf("failed to analyze language and framework: %w", err)
	}

	// Analyze directory structure
	if err := a.analyzeDirectoryStructure(memory); err != nil {
		return nil, fmt.Errorf("failed to analyze directory structure: %w", err)
	}

	// Analyze configuration files
	if err := a.analyzeConfigurationFiles(memory); err != nil {
		return nil, fmt.Errorf("failed to analyze configuration files: %w", err)
	}

	// Analyze dependencies
	if err := a.analyzeDependencies(memory); err != nil {
		return nil, fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	return memory, nil
}

func (a *Analyzer) analyzeProjectStructure(memory *WorkspaceMemory) error {
	// Check for common project files
	projectFiles := map[string]FactType{
		"go.mod":       FactTypeDependency,
		"go.sum":       FactTypeDependency,
		"package.json": FactTypeDependency,
		"Cargo.toml":   FactTypeDependency,
		"requirements.txt": FactTypeDependency,
		"pyproject.toml": FactTypeDependency,
		"Makefile":     FactTypePattern,
		"Dockerfile":   FactTypePattern,
		"docker-compose.yml": FactTypePattern,
		".gitignore":   FactTypeConvention,
	}

	for filename, factType := range projectFiles {
		filePath := filepath.Join(a.workspacePath, filename)
		if _, err := os.Stat(filePath); err == nil {
			memory.Facts = append(memory.Facts, Fact{
				ID:         fmt.Sprintf("project-file-%s", strings.ReplaceAll(filename, ".", "-")),
				Type:       factType,
				Title:      fmt.Sprintf("Project uses %s", filename),
				Content:    fmt.Sprintf("The project includes %s, indicating %s", filename, describeProjectFile(filename)),
				Confidence: 0.9,
				Source:     filename,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			})
		}
	}

	return nil
}

func (a *Analyzer) analyzeLanguageAndFramework(memory *WorkspaceMemory) error {
	// Go project
	goModPath := filepath.Join(a.workspacePath, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		data, err := os.ReadFile(goModPath)
		if err == nil {
			content := string(data)
			memory.Facts = append(memory.Facts, Fact{
				ID:         "language-go",
				Type:       FactTypeArchitecture,
				Title:      "Project uses Go",
				Content:    fmt.Sprintf("Go module project. Module declaration:\n%s", extractGoModule(content)),
				Confidence: 1.0,
				Source:     "go.mod",
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			})

			// Extract Go version
			if version := extractGoVersion(content); version != "" {
				memory.Facts = append(memory.Facts, Fact{
					ID:         "go-version",
					Type:       FactTypeDependency,
					Title:      fmt.Sprintf("Go version %s", version),
					Content:    fmt.Sprintf("Project requires Go %s or later", version),
					Confidence: 1.0,
					Source:     "go.mod",
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				})
			}
		}
	}

	// Node.js project
	packageJSONPath := filepath.Join(a.workspacePath, "package.json")
	if _, err := os.Stat(packageJSONPath); err == nil {
		data, err := os.ReadFile(packageJSONPath)
		if err == nil {
			var pkg struct {
				Name    string            `json:"name"`
				Version string            `json:"version"`
				Deps    map[string]string `json:"dependencies"`
			}
			if json.Unmarshal(data, &pkg) == nil {
				memory.Facts = append(memory.Facts, Fact{
					ID:         "language-nodejs",
					Type:       FactTypeArchitecture,
					Title:      "Project uses Node.js",
					Content:    fmt.Sprintf("Node.js project: %s v%s", pkg.Name, pkg.Version),
					Confidence: 1.0,
					Source:     "package.json",
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				})
			}
		}
	}

	return nil
}

func (a *Analyzer) analyzeDirectoryStructure(memory *WorkspaceMemory) error {
	entries, err := os.ReadDir(a.workspacePath)
	if err != nil {
		return err
	}

	directories := make([]string, 0)
	files := make([]string, 0)

	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			directories = append(directories, entry.Name())
		} else if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	// Analyze common directory patterns
	commonDirs := map[string]string{
		"cmd":     "Command-line applications",
		"internal": "Private application packages",
		"pkg":     "Public library packages",
		"api":     "API definitions or handlers",
		"web":     "Web-related code",
		"ui":      "User interface code",
		"frontend": "Frontend application",
		"backend": "Backend application",
		"src":     "Source code",
		"lib":     "Library code",
		"test":    "Test files",
		"tests":   "Test files",
		"docs":    "Documentation",
		"scripts": "Build and utility scripts",
		"tools":   "Development tools",
		"config":  "Configuration files",
		"migrations": "Database migrations",
		"models":  "Data models",
		"handlers": "Request handlers",
		"services": "Business logic services",
		"repository": "Data access layer",
	}

	detectedDirs := make([]string, 0)
	for _, dir := range directories {
		if desc, ok := commonDirs[dir]; ok {
			detectedDirs = append(detectedDirs, fmt.Sprintf("%s (%s)", dir, desc))
		}
	}

	if len(detectedDirs) > 0 {
		memory.Facts = append(memory.Facts, Fact{
			ID:         "directory-structure",
			Type:       FactTypeArchitecture,
			Title:      "Project directory structure",
			Content:    fmt.Sprintf("Detected directories: %s", strings.Join(detectedDirs, ", ")),
			Confidence: 0.8,
			Source:     "directory scan",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		})
	}

	return nil
}

func (a *Analyzer) analyzeConfigurationFiles(memory *WorkspaceMemory) error {
	configFiles := []string{
		".golangci.yml",
		".golangci.yaml",
		"eslint.config.js",
		".eslintrc.js",
		".eslintrc.json",
		"tsconfig.json",
		"jest.config.js",
		"vitest.config.ts",
		"pytest.ini",
		"setup.cfg",
		".editorconfig",
		".prettierrc",
		".prettierrc.json",
	}

	for _, configFile := range configFiles {
		filePath := filepath.Join(a.workspacePath, configFile)
		if _, err := os.Stat(filePath); err == nil {
			memory.Facts = append(memory.Facts, Fact{
				ID:         fmt.Sprintf("config-%s", strings.ReplaceAll(configFile, ".", "-")),
				Type:       FactTypeConvention,
				Title:      fmt.Sprintf("Uses %s", configFile),
				Content:    fmt.Sprintf("Project includes %s for code formatting/linting configuration", configFile),
				Confidence: 0.9,
				Source:     configFile,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			})
		}
	}

	return nil
}

func (a *Analyzer) analyzeDependencies(memory *WorkspaceMemory) error {
	// Analyze Go dependencies
	goModPath := filepath.Join(a.workspacePath, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		data, err := os.ReadFile(goModPath)
		if err == nil {
			content := string(data)
			deps := extractGoDependencies(content)
			if len(deps) > 0 {
				memory.Facts = append(memory.Facts, Fact{
					ID:         "go-dependencies",
					Type:       FactTypeDependency,
					Title:      "Go dependencies",
					Content:    fmt.Sprintf("Key dependencies: %s", strings.Join(deps, ", ")),
					Confidence: 0.9,
					Source:     "go.mod",
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				})
			}
		}
	}

	return nil
}

// Helper functions

func describeProjectFile(filename string) string {
	switch filename {
	case "go.mod", "go.sum":
		return "Go module dependency management"
	case "package.json":
		return "Node.js package management"
	case "Cargo.toml":
		return "Rust package management"
	case "requirements.txt":
		return "Python package management"
	case "pyproject.toml":
		return "Python project configuration"
	case "Makefile":
		return "Build automation"
	case "Dockerfile":
		return "Container deployment"
	case "docker-compose.yml":
		return "Multi-container orchestration"
	case ".gitignore":
		return "Version control exclusions"
	default:
		return "project configuration"
	}
}

func extractGoModule(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return line
		}
	}
	return ""
}

func extractGoVersion(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			return strings.TrimPrefix(line, "go ")
		}
	}
	return ""
}

func extractGoDependencies(content string) []string {
	deps := make([]string, 0)
	inRequire := false

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		if inRequire && line != "" && !strings.HasPrefix(line, "//") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				dep := parts[0]
				// Extract just the package name (last segment)
				if idx := strings.LastIndex(dep, "/"); idx != -1 {
					dep = dep[idx+1:]
				}
				deps = append(deps, dep)
			}
		}
	}
	return deps
}
