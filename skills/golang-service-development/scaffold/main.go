// Command scaffold generates a new Go microservice from the embedded
// templates/ directory. Replaces the previous sed-based new-service.sh.
//
// Usage:
//
//	go run ./scaffold <name> [target-parent-dir]
//	go run ./scaffold --force demo   # regenerate demo-service/ from templates
//
// Templates use Go text/template syntax. Available variables:
//
//	{{.Name}}        lowercase service name (e.g., "user")
//	{{.Pascal}}      PascalCase        (e.g., "User")
//	{{.Module}}      Go module path    (e.g., "user-service")
//	{{.Plural}}      lowercase plural  (e.g., "users")
//	{{.NameUpper}}   SCREAMING_SNAKE   (e.g., "USER")
//
// File paths in templates/ may also use these variables — they get rendered
// before writing (so api/proto/{{.Name}}/v1/{{.Name}}.proto.tmpl becomes
// api/proto/user/v1/user.proto).
package main

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

//go:embed all:templates
var templatesFS embed.FS

// Spec carries every derived identifier for a service. Templates reference
// these via {{.Name}} etc.
type Spec struct {
	Name      string // "user"
	Pascal    string // "User"
	Module    string // "user-service"
	Plural    string // "users"
	NameUpper string // "USER"
}

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// newSpec validates the name and computes every derived field.
//
// targetParent is where the new <name>-service/ directory will be created.
func newSpec(name, targetParent string) (Spec, string, error) {
	if !nameRe.MatchString(name) {
		return Spec{}, "", fmt.Errorf("name %q must match ^[a-z][a-z0-9]*$ (lowercase letters/digits, no hyphens)", name)
	}
	absParent, err := filepath.Abs(targetParent)
	if err != nil {
		return Spec{}, "", err
	}
	target := filepath.Join(absParent, name+"-service")

	return Spec{
		Name:      name,
		Pascal:    strings.ToUpper(name[:1]) + name[1:],
		Module:    name + "-service",
		Plural:    name + "s",
		NameUpper: strings.ToUpper(name),
	}, target, nil
}

// repoRootFromCwd walks up from the current working directory to find the
// dev-skills repo root (identified by containing skills/golang-service-development/).
// The wrapper script always cd's into scaffold/ before running, so cwd is
// <repo-root>/skills/golang-service-development/scaffold/.
func repoRootFromCwd() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "skills", "golang-service-development")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("could not locate dev-skills root from cwd %s (looking for skills/golang-service-development/)", cwd)
}

func main() {
	force := false
	args := []string{}
	for _, a := range os.Args[1:] {
		switch {
		case a == "--force" || a == "-f":
			force = true
		case strings.HasPrefix(a, "-"):
			fail(fmt.Errorf("unknown flag %q", a))
		default:
			args = append(args, a)
		}
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: scaffold [--force] <name> [target-parent-dir]")
		fmt.Fprintln(os.Stderr, "  --force  overwrite target if it exists (used for regenerating demo-service)")
		os.Exit(2)
	}

	repoRoot, err := repoRootFromCwd()
	if err != nil {
		fail(err)
	}

	// Default target parent: dev-skills's parent (so new services land
	// alongside the monorepo, not inside it).
	targetParent := filepath.Join(repoRoot, "..")
	if len(args) >= 2 {
		targetParent = args[1]
	}

	spec, target, err := newSpec(args[0], targetParent)
	if err != nil {
		fail(err)
	}

	if force {
		if err := os.RemoveAll(target); err != nil {
			fail(fmt.Errorf("remove existing target: %w", err))
		}
	} else if _, err := os.Stat(target); err == nil {
		fail(fmt.Errorf("target %s already exists (use --force to overwrite)", target))
	}

	if err := renderAll(spec, target); err != nil {
		fail(err)
	}

	fmt.Printf("Created %s\n\n", target)
	fmt.Println("Next:")
	fmt.Printf("  cd %s\n", target)
	fmt.Println("  make tidy        # generates go.sum")
	fmt.Println("  make proto       # generates gen/")
	fmt.Println("  make generate    # generates internal/store/generated/")
	fmt.Println("  make migrate     # creates DB tables (needs PostgreSQL)")
	fmt.Println("  make run         # starts :9000 (gRPC) + :8080 (HTTP)")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

// renderAll walks templates/ and writes each .tmpl file to target/, rendering
// both path and content against spec.
func renderAll(spec Spec, target string) error {
	return fs.WalkDir(templatesFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel := strings.TrimPrefix(path, "templates/")
		if !strings.HasSuffix(rel, ".tmpl") {
			return nil
		}
		rel = strings.TrimSuffix(rel, ".tmpl")

		// Render path (file path may contain {{.Name}}).
		outRel, err := renderString("path", rel, spec)
		if err != nil {
			return fmt.Errorf("render path %q: %w", rel, err)
		}
		outPath := filepath.Join(target, outRel)

		// Read + render content.
		content, err := templatesFS.ReadFile(path)
		if err != nil {
			return err
		}
		rendered, err := renderString(rel, string(content), spec)
		if err != nil {
			return fmt.Errorf("render content %q: %w", rel, err)
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(outPath, []byte(rendered), 0o644)
	})
}

func renderString(name, src string, spec Spec) (string, error) {
	tmpl, err := template.New(name).Parse(src)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, spec); err != nil {
		return "", err
	}
	return buf.String(), nil
}
