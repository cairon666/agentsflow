//go:build e2e

package e2e

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestUseGoldenFiles(t *testing.T) {
	repo := repoRoot(t)
	bin := buildAgentsflowBinary(t, repo)
	scenarios := discoverGoldenScenarios(t, filepath.Join(repo, "e2e", "testdata"))

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			workDir := t.TempDir()
			homeDir := t.TempDir()
			runAgentsflowUse(t, bin, scenario, workDir, homeDir)

			actualDir := workDir
			if scenario.Scope == "global" {
				actualDir = homeDir
			}
			assertGoldenDir(t, scenario.GoldenDir, actualDir)
		})
	}
}

type goldenScenario struct {
	Name         string
	Target       string
	Scope        string
	TemplatePath string
	GoldenDir    string
}

func discoverGoldenScenarios(t *testing.T, testdataDir string) []goldenScenario {
	t.Helper()
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("read e2e testdata: %v", err)
	}

	scenarios := make([]goldenScenario, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		target, scope, ok := parseScenarioName(name)
		if !ok {
			continue
		}

		dir := filepath.Join(testdataDir, name)
		templatePath := filepath.Join(dir, "template.yaml")
		goldenDir := filepath.Join(dir, "golden")
		if !isFile(templatePath) || !isDir(goldenDir) {
			continue
		}
		scenarios = append(scenarios, goldenScenario{
			Name:         name,
			Target:       target,
			Scope:        scope,
			TemplatePath: templatePath,
			GoldenDir:    goldenDir,
		})
	}
	sort.Slice(scenarios, func(i, j int) bool {
		return scenarios[i].Name < scenarios[j].Name
	})
	if len(scenarios) == 0 {
		t.Fatalf("no e2e golden scenarios found in %s", testdataDir)
	}
	return scenarios
}

func parseScenarioName(name string) (target string, scope string, ok bool) {
	index := strings.LastIndex(name, "_")
	if index <= 0 || index == len(name)-1 {
		return "", "", false
	}
	target = name[:index]
	scope = name[index+1:]
	switch scope {
	case "project", "global":
		return target, scope, true
	default:
		return "", "", false
	}
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func runAgentsflowUse(t *testing.T, bin string, scenario goldenScenario, workDir, homeDir string) {
	t.Helper()
	cmd := exec.Command(
		bin,
		"use",
		scenario.TemplatePath,
		"--target", scenario.Target,
		"--bind", "main="+modelForTarget(scenario.Target),
		"--scope", scenario.Scope,
		"--yes",
	)
	cmd.Dir = workDir
	cmd.Env = testEnv(homeDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agentsflow use failed: %v\n%s", err, output)
	}
}

func modelForTarget(target string) string {
	switch target {
	case "claude":
		return "claude-test"
	case "opencode":
		return "opencode_test"
	default:
		return "gpt-test"
	}
}

func buildAgentsflowBinary(t *testing.T, repo string) string {
	t.Helper()
	if bin := strings.TrimSpace(os.Getenv("AGENTSFLOW_E2E_BIN")); bin != "" {
		return bin
	}

	bin := filepath.Join(t.TempDir(), executableName("agentsflow"))
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/agentsflow")
	cmd.Dir = repo
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build agentsflow binary: %v\n%s", err, output)
	}
	return bin
}

func assertGoldenDir(t *testing.T, goldenDir, actualDir string) {
	t.Helper()
	want := readTree(t, goldenDir)
	got := readTree(t, actualDir)

	wantPaths := sortedKeys(want)
	gotPaths := sortedKeys(got)
	if !sameStrings(wantPaths, gotPaths) {
		t.Fatalf("generated file set mismatch\nwant: %v\ngot:  %v", wantPaths, gotPaths)
	}

	for _, path := range wantPaths {
		if bytes.Equal(want[path], got[path]) {
			continue
		}
		t.Fatalf(
			"generated file %s does not match golden\n\nwant:\n%s\n\ngot:\n%s",
			path,
			want[path],
			got[path],
		)
	}
}

func readTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("relative path for %q: %w", path, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %q: %w", path, err)
		}
		files[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		t.Fatalf("read file tree %s: %v", root, err)
	}
	return files
}

func sortedKeys(values map[string][]byte) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate current test file")
	}
	return filepath.Dir(filepath.Dir(file))
}

func testEnv(homeDir string) []string {
	env := make([]string, 0, len(os.Environ())+4)
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "HOME=") ||
			strings.HasPrefix(item, "USERPROFILE=") ||
			strings.HasPrefix(item, "XDG_CONFIG_HOME=") {
			continue
		}
		env = append(env, item)
	}
	env = append(env,
		"HOME="+homeDir,
		"USERPROFILE="+homeDir,
		"XDG_CONFIG_HOME="+filepath.Join(homeDir, ".config"),
	)
	return env
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
