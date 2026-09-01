package architecture_test

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve architecture test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestMonitoringUsesLayeredSourceLayout(t *testing.T) {
	root := repositoryRoot(t)
	for _, required := range []string{"cmd/monitoring-server", "internal/application/monitoring", "internal/assembly/module", "internal/assembly/saas", "internal/transport/http/module", "internal/transport/http/saas", "module", "saas"} {
		if info, err := os.Stat(filepath.Join(root, required)); err != nil || !info.IsDir() {
			t.Errorf("required Monitoring boundary %q is missing", required)
		}
	}
	for _, forbidden := range []string{"application", "server", "domain", "infrastructure", "contract", "model", "repository", "service"} {
		if _, err := os.Stat(filepath.Join(root, forbidden)); !os.IsNotExist(err) {
			t.Errorf("legacy or unsupported top-level package %q must not exist", forbidden)
		}
	}
}

func TestPublicPackagesRemainThinFacades(t *testing.T) {
	root := repositoryRoot(t)
	for _, name := range []string{"module", "saas"} {
		entries, err := os.ReadDir(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") || entry.Name() == "module.go" || entry.Name() == "server.go" {
				continue
			}
			t.Errorf("public %s package must remain a thin facade; unexpected production file %q", name, entry.Name())
		}
	}
}

func TestMonitoringProductionDependenciesExcludeRuntimeAndPlane(t *testing.T) {
	root := repositoryRoot(t)
	// This test inspects dependency ownership; it is not a tagged-dependency
	// compilation test. -e keeps the package graph inspectable while a new
	// Foundation package is being developed locally ahead of its next tag.
	// TestMonitoringUsesTaggedDependencies independently rejects replace/local
	// path dependencies in the committed module definition.
	command := exec.Command("go", "list", "-e", "-json", "./...")
	command.Dir = root
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(output)
	for {
		var pkg struct {
			GoFiles    []string
			ImportPath string
			Imports    []string
		}
		if err := decoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		for _, imported := range pkg.Imports {
			if imported == "github.com/domainry/domainry-runtime" || strings.HasPrefix(imported, "github.com/domainry/domainry-runtime/") || imported == "github.com/domainry/domainry-plane" || strings.HasPrefix(imported, "github.com/domainry/domainry-plane/") {
				t.Errorf("%s imports forbidden host implementation %s", pkg.ImportPath, imported)
			}
		}
	}
	if err := command.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestMonitoringUsesTaggedDependencies(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "replace ") || strings.Contains(string(content), "../domainry-") {
		t.Fatal("Monitoring must consume released module tags, not local directory replacements")
	}
}
