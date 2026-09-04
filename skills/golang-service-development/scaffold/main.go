// Command scaffold generates a new Go microservice from the embedded
// templates/ directory. Replaces the previous sed-based new-service.sh.
//
// Usage:
//
//	go run ./scaffold [--force] [capability flags] <name> [target-parent-dir]
//	go run ./scaffold --force demo   # regenerate demo-service/ from templates
//
// Capability flags default to ALL OFF — a minimal shell that runs without
// Postgres. --example implies --db (a CRUD domain needs a database).
//
//	go run ./scaffold ping                  # minimal: empty proto service, no DB
//	go run ./scaffold pay --db --example    # CRUD service with Postgres
//
// Templates use Go text/template syntax. Available variables:
//
//	{{.Name}}        lowercase service name (e.g., "user")
//	{{.Pascal}}      PascalCase        (e.g., "User")
//	{{.Module}}      Go module path    (e.g., "user-service")
//	{{.Plural}}      lowercase plural  (e.g., "users")
//	{{.NameUpper}}   SCREAMING_SNAKE   (e.g., "USER")
//	{{.EnvPrefix}}   configx env prefix (e.g., "USER_SERVICE")
//	{{.DB}} {{.Redis}} {{.Thirdcall}} {{.Example}}  bool capability switches
//
// Template function:
//
//	{{envvar "DATABASE_HOST"}}  -> ${USER_SERVICE_DATABASE_HOST}
//	Used by config.example.yaml to emit ${VAR} placeholders.
//
// File paths in templates/ may also use these variables — they get rendered
// before writing (so api/proto/{{.Name}}/v1/{{.Name}}.proto.tmpl becomes
// api/proto/user/v1/user.proto). Some files are skipped based on capability
// switches; see skipRules below.
package main

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
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
	// EnvPrefix is the configx env prefix (NAME_SERVICE), used by configx
	// WithEnvPrefix and as the ${VAR} namespace in config.example.yaml.
	EnvPrefix string // "USER_SERVICE"

	// Capability switches (default all false = minimal shell that runs
	// without Postgres). See skipRules for file-level gating and the mixed
	// templates for internal {{if .X}} fragments.
	DB        bool
	Redis     bool
	Thirdcall bool
	Example   bool
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
		EnvPrefix: strings.ToUpper(name) + "_SERVICE",
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
	var db, redis, thirdcall, example bool
	args := []string{}
	for _, a := range os.Args[1:] {
		switch {
		case a == "--force" || a == "-f":
			force = true
		case a == "--db":
			db = true
		case a == "--no-db":
			db = false
		case a == "--redis":
			redis = true
		case a == "--no-redis":
			redis = false
		case a == "--thirdcall":
			thirdcall = true
		case a == "--no-thirdcall":
			thirdcall = false
		case a == "--example":
			example = true
		case a == "--no-example":
			example = false
		case a == "-h" || a == "--help":
			printUsage()
			os.Exit(0)
		case strings.HasPrefix(a, "-"):
			fail(fmt.Errorf("unknown flag %q", a))
		default:
			args = append(args, a)
		}
	}
	if len(args) < 1 {
		printUsage()
		os.Exit(2)
	}

	// --example implies --db (a CRUD domain needs a database). Enforce the
	// invariant here — the single source of truth — so templates can use
	// {{if .DB}} and {{if .Example}} independently without guarding every
	// combination.
	if example && !db {
		db = true
		fmt.Fprintln(os.Stderr, "note: --example implies --db (enabling database)")
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
	spec.DB = db
	spec.Redis = redis
	spec.Thirdcall = thirdcall
	spec.Example = example

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

	// Hand off Docker packaging to the golang-service-docker renderer so the
	// scaffold ships a docker-skill-standard Dockerfile/compose/.env rather than
	// a naive one. render.sh reads the just-generated config.example.yaml
	// (${VAR}) and .env.example (defaults) and stays idempotent for rerenders.
	renderDocker(spec, target, repoRoot)

	fmt.Printf("Created %s\n\n", target)
	fmt.Println("Next:")
	fmt.Printf("  cd %s\n", target)
	fmt.Println("  make tidy        # generates go.sum")
	fmt.Println("  make proto       # generates gen/")
	if spec.DB {
		fmt.Println("  make generate    # generates internal/store/generated/")
		fmt.Println("  make migrate     # creates DB tables (needs PostgreSQL)")
		fmt.Println("  make run         # local: cp .env.example .env first (edit DATABASE_HOST -> localhost)")
	} else {
		fmt.Println("  make run         # local: minimal shell, runs without a database")
	}
	fmt.Println()
	fmt.Println("Docker (Dockerfile + docker-compose.yaml generated):")
	fmt.Println("  make docker-up         # build + start the compose stack")
	fmt.Println("    # behind a firewall? set GOPROXY=https://goproxy.cn,direct in .env first")
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: scaffold [--force] [capability flags] <name> [target-parent-dir]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "capability flags (default: all off = minimal shell that runs without Postgres):")
	fmt.Fprintln(os.Stderr, "  --db / --no-db                PostgreSQL via dbx (postgres-only; go-common has no mysql path)")
	fmt.Fprintln(os.Stderr, "  --redis / --no-redis          Redis via redisx")
	fmt.Fprintln(os.Stderr, "  --thirdcall / --no-thirdcall  gid-service dependency (snowflake ID generator)")
	fmt.Fprintln(os.Stderr, "  --example / --no-example      CRUD starter domain (implies --db)")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  --force                       overwrite target if it exists (used for regenerating demo-service)")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

// skipRules maps a raw template path (relative to templates/, no .tmpl suffix,
// with the {{.Name}} literal intact) to the Spec capability that gates it.
// The {{.Name}} literal in the path is the signal — e.g.
// internal/store/models/{{.Name}}.go is obviously example-gated. Paths not in
// this table are always rendered. A path is matched BEFORE path rendering.
var skipRules = map[string]string{
	// DB-only (migrate plumbing + gorm gen dir). Migrate lives in pkg/handler
	// (re-exported as pkg.Migrate) so embedded module users can call it too;
	// cmd/server/migrate.go just wires cfg + db and delegates.
	"cmd/server/migrate.go":              "DB",
	"pkg/handler/migrate.go":             "DB",
	"pkg/handler/migrate_test.go":        "DB",
	"internal/store/generated/README.md": "DB",
	"internal/store/models/register.go":  "DB",
	// Example-only (the {{.Pascal}} CRUD domain).
	"internal/service/{{.Name}}/{{.Name}}.go": "Example",
	"internal/store/models/{{.Name}}.go":      "Example",
	"internal/store/dal/{{.Name}}.go":         "Example",
	"pkg/xcodes/{{.Name}}.go":                 "Example",
}

// skipFor reports whether the template at rawRel should be skipped for spec.
func skipFor(rawRel string, spec Spec) bool {
	switch skipRules[rawRel] {
	case "DB":
		return !spec.DB
	case "Redis":
		return !spec.Redis
	case "Thirdcall":
		return !spec.Thirdcall
	case "Example":
		return !spec.Example
	}
	return false
}

// renderAll walks templates/ and writes each .tmpl file to target/, rendering
// both path and content against spec. Files gated by a capability that is off
// are skipped (see skipRules).
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

		// Skip files whose capability is off. Match the raw rel (before path
		// rendering) so the {{.Name}} literal in skipRules aligns.
		if skipFor(rel, spec) {
			return nil
		}

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
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		// envvar renders a ${ENV_PREFIX_VAR} placeholder for config.example.yaml.
		// Building it inside an action avoids the ${{ triple-brace parse conflict
		// that arises when a literal "${" sits directly before a {{action}}.
		"envvar": func(varName string) string {
			return "${" + spec.EnvPrefix + "_" + varName + "}"
		},
	}).Parse(src)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, spec); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderDocker shells out to the golang-service-docker renderer to add Docker
// packaging (Dockerfile, docker-compose.yaml, .dockerignore, Makefile targets)
// to the freshly generated service. Best-effort: if the renderer is absent
// (docker skill not installed) or fails, the Go skeleton is still complete and
// the user can run render.sh manually later. --database and --redis are driven
// by the capability switches so the compose stack matches the generated code.
func renderDocker(spec Spec, target, repoRoot string) {
	renderSh := filepath.Join(repoRoot, "skills", "golang-service-docker", "scripts", "render.sh")
	if _, err := os.Stat(renderSh); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %s not found; skipping Docker packaging\n", renderSh)
		return
	}
	dbKind := "none"
	if spec.DB {
		dbKind = "postgres"
	}
	dockerArgs := []string{
		renderSh,
		"--target", target,
		"--service-name", spec.Module,
		"--binary-name", spec.Module,
		"--build-path", "./cmd/server",
		"--grpc-port", "9000",
		"--http-port", "8080",
		"--env-prefix", spec.EnvPrefix,
		"--database", dbKind,
		"--config-mode", "copy",
		"--config-source", "config.example.yaml",
	}
	if spec.Redis {
		dockerArgs = append(dockerArgs, "--redis")
	}
	cmd := exec.Command("bash", dockerArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: docker packaging render failed: %v\n", err)
	}
}
