package database

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"
)

//go:embed scripts/*.sql
var scriptFS embed.FS

// InitParams holds the parameters used to render SQL init scripts.
type InitParams struct {
	DatabaseName        string
	JWTSecret           string
	AuthenticatorPassword string
	AuthAdminPassword     string
	StorageAdminPassword  string
}

// RenderInitScripts reads all embedded SQL files, sorts them by name,
// and renders each one as a Go template with the given params.
// Returns a slice of rendered SQL strings (one per file).
func RenderInitScripts(params InitParams) ([]string, error) {
	entries, err := scriptFS.ReadDir("scripts")
	if err != nil {
		return nil, fmt.Errorf("reading embedded scripts: %w", err)
	}

	// Sort by filename to ensure deterministic ordering
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var results []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := scriptFS.ReadFile("scripts/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("reading script %s: %w", entry.Name(), err)
		}

		tmpl, err := template.New(entry.Name()).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", entry.Name(), err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, params); err != nil {
			return nil, fmt.Errorf("rendering template %s: %w", entry.Name(), err)
		}

		results = append(results, buf.String())
	}

	return results, nil
}

// CombinedInitSQL renders all init scripts and concatenates them into a single SQL string.
func CombinedInitSQL(params InitParams) (string, error) {
	scripts, err := RenderInitScripts(params)
	if err != nil {
		return "", err
	}
	return strings.Join(scripts, "\n"), nil
}
