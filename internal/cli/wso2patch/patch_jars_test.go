package wso2patch

import (
	"archive/zip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsRuntimeRelevantChange(t *testing.T) {
	tests := []struct {
		path     string
		relevant bool
	}{
		{path: "src/main/java/A.java", relevant: true},
		{path: "src/test/java/A.java", relevant: false},
		{path: "docs/readme.md", relevant: false},
		{path: ".github/workflows/ci.yml", relevant: false},
		{path: "README.md", relevant: false},
		{path: "src/main/resources/config.yaml", relevant: true},
	}

	for _, tt := range tests {
		got := isRuntimeRelevantChange(tt.path)
		if got != tt.relevant {
			t.Fatalf("isRuntimeRelevantChange(%q)=%v, want %v", tt.path, got, tt.relevant)
		}
	}
}

func TestResolveProductDir(t *testing.T) {
	root := t.TempDir()

	_, err := resolveProductDir(root, "")
	if err == nil || !strings.Contains(err.Error(), "no wso2am-*") {
		t.Fatalf("expected missing product error, got %v", err)
	}

	only := filepath.Join(root, "wso2am-4.5.0")
	mustMkdir(t, only)
	got, err := resolveProductDir(root, "")
	if err != nil {
		t.Fatalf("expected auto-detect success, got %v", err)
	}
	if got != only {
		t.Fatalf("expected %s, got %s", only, got)
	}

	mustMkdir(t, filepath.Join(root, "wso2am-4.6.0"))
	_, err = resolveProductDir(root, "")
	if err == nil || !strings.Contains(err.Error(), "multiple wso2am-*") {
		t.Fatalf("expected multiple product error, got %v", err)
	}

	got, err = resolveProductDir(root, "wso2am-4.5.0")
	if err != nil {
		t.Fatalf("expected explicit path success, got %v", err)
	}
	if got != only {
		t.Fatalf("expected %s, got %s", only, got)
	}
}

func TestFindProductJarMatch(t *testing.T) {
	root := t.TempDir()
	plugins := filepath.Join(root, "repository", "components", "plugins")
	mustMkdir(t, plugins)

	jar1 := filepath.Join(plugins, "org.wso2.carbon.apimgt.impl_9.31.86.jar")
	mustWriteFile(t, jar1, []byte("a"))

	match, err := findProductJarMatch(root, "org.wso2.carbon.apimgt.impl", "org.wso2.carbon.apimgt.impl")
	if err != nil {
		t.Fatalf("expected match, got %v", err)
	}
	if match != jar1 {
		t.Fatalf("expected %s, got %s", jar1, match)
	}

	_, err = findProductJarMatch(root, "org.wso2.carbon.apimgt.unknown", "org.wso2.carbon.apimgt.unknown")
	if err == nil || !strings.Contains(err.Error(), "no destination jar found") {
		t.Fatalf("expected no destination error, got %v", err)
	}

	jar2 := filepath.Join(root, "updates", "org.wso2.carbon.apimgt.impl_9.31.87.jar")
	mustMkdir(t, filepath.Dir(jar2))
	mustWriteFile(t, jar2, []byte("b"))
	_, err = findProductJarMatch(root, "org.wso2.carbon.apimgt.impl", "org.wso2.carbon.apimgt.impl")
	if err == nil || !strings.Contains(err.Error(), "multiple destination jars") {
		t.Fatalf("expected ambiguous destination error, got %v", err)
	}
}

func TestFindProductJarMatch_IgnoresPatchDir(t *testing.T) {
	root := t.TempDir()
	plugins := filepath.Join(root, "repository", "components", "plugins")
	patches := filepath.Join(root, "repository", "components", "patches", "patch9999")
	mustMkdir(t, plugins)
	mustMkdir(t, patches)

	pluginsJar := filepath.Join(plugins, "org.wso2.carbon.apimgt.impl_9.31.86.jar")
	patchJar := filepath.Join(patches, "org.wso2.carbon.apimgt.impl_9.31.87.jar")
	mustWriteFile(t, pluginsJar, []byte("plugins"))
	mustWriteFile(t, patchJar, []byte("patch"))

	match, err := findProductJarMatch(root, "org.wso2.carbon.apimgt.impl", "org.wso2.carbon.apimgt.impl")
	if err != nil {
		t.Fatalf("expected match, got %v", err)
	}
	if match != pluginsJar {
		t.Fatalf("expected plugins jar %s, got %s", pluginsJar, match)
	}
}

func TestReadBundleMetadata(t *testing.T) {
	jarPath := filepath.Join(t.TempDir(), "test.jar")
	createJarWithManifest(t, jarPath, "Bundle-SymbolicName: org.wso2.carbon.apimgt.impl;singleton:=true\nBundle-Version: 9.28.116.407\n")

	meta, err := readBundleMetadata(jarPath)
	if err != nil {
		t.Fatalf("expected manifest parse success, got %v", err)
	}
	if meta.SymbolicName != "org.wso2.carbon.apimgt.impl" {
		t.Fatalf("expected symbolic name org.wso2.carbon.apimgt.impl, got %q", meta.SymbolicName)
	}
	if meta.Version != "9.28.116.407" {
		t.Fatalf("expected bundle version 9.28.116.407, got %q", meta.Version)
	}
}

func TestReadPomInfo(t *testing.T) {
	dir := t.TempDir()
	pomPath := filepath.Join(dir, "pom.xml")
	pomContent := `<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>org.example</groupId>
  <artifactId>example-module</artifactId>
  <version>1.0.0</version>
  <packaging>pom</packaging>
</project>`
	mustWriteFile(t, pomPath, []byte(pomContent))

	info, err := readPomInfo(pomPath)
	if err != nil {
		t.Fatalf("expected pom parse success, got %v", err)
	}
	if info.ArtifactID != "example-module" {
		t.Fatalf("unexpected artifactId: %q", info.ArtifactID)
	}
	if info.Packaging != "pom" {
		t.Fatalf("unexpected packaging: %q", info.Packaging)
	}
}

func TestShouldSkipModulePackaging(t *testing.T) {
	if !shouldSkipModulePackaging("pom") {
		t.Fatal("expected pom packaging to be skipped")
	}
	if !shouldSkipModulePackaging(" POM ") {
		t.Fatal("expected case-insensitive pom packaging to be skipped")
	}
	if !shouldSkipModulePackaging("war") {
		t.Fatal("expected war packaging to be skipped")
	}
	if !shouldSkipModulePackaging(" WAR ") {
		t.Fatal("expected case-insensitive war packaging to be skipped")
	}
	if shouldSkipModulePackaging("jar") {
		t.Fatal("did not expect jar packaging to be skipped")
	}
}

func TestModuleRootsFromChanges_RuntimeFilter(t *testing.T) {
	repo := t.TempDir()
	modA := filepath.Join(repo, "components", "a", "module-a")
	modB := filepath.Join(repo, "components", "b", "module-b")
	mustMkdir(t, modA)
	mustMkdir(t, modB)
	mustWriteFile(t, filepath.Join(modA, "pom.xml"), []byte("<project><artifactId>module-a</artifactId></project>"))
	mustWriteFile(t, filepath.Join(modB, "pom.xml"), []byte("<project><artifactId>module-b</artifactId></project>"))

	changed := []string{
		"components/a/module-a/src/test/java/A.java",
		"components/b/module-b/src/main/java/B.java",
	}

	modules, err := moduleRootsFromChanges(repo, changed)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(modules) != 1 {
		t.Fatalf("expected 1 runtime-relevant module, got %d (%v)", len(modules), modules)
	}
	if modules[0] != modB {
		t.Fatalf("expected module %s, got %s", modB, modules[0])
	}
}

func TestEnsureGitSnapshotLifecycle(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}

	product := t.TempDir()
	mustWriteFile(t, filepath.Join(product, "a.txt"), []byte("one"))

	ref1, err := ensureGitSnapshot(product)
	if err != nil {
		t.Fatalf("expected initial snapshot success, got %v", err)
	}
	if ref1 == "" {
		t.Fatal("expected non-empty first ref")
	}

	mustWriteFile(t, filepath.Join(product, "a.txt"), []byte("two"))
	ref2, err := ensureGitSnapshot(product)
	if err != nil {
		t.Fatalf("expected dirty snapshot success, got %v", err)
	}
	if ref2 == "" || ref2 == ref1 {
		t.Fatalf("expected new ref on dirty snapshot, ref1=%s ref2=%s", ref1, ref2)
	}

	dirty, err := isGitDirty(product)
	if err != nil {
		t.Fatalf("failed to query dirty state: %v", err)
	}
	if dirty {
		t.Fatal("expected clean git state after auto-commit snapshot")
	}

	ref3, err := ensureGitSnapshot(product)
	if err != nil {
		t.Fatalf("expected clean snapshot success, got %v", err)
	}
	if ref3 != ref2 {
		t.Fatalf("expected unchanged ref on clean repo, ref2=%s ref3=%s", ref2, ref3)
	}
}

func TestResolveCarbonPatchDir(t *testing.T) {
	product := "/tmp/wso2am-4.2.0"
	got := resolveCarbonPatchDir(product)
	want := filepath.Join(product, "repository", "components", "patches", defaultCarbonPatchDirName)
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func createJarWithManifest(t *testing.T, jarPath, manifestBody string) {
	t.Helper()
	f, err := os.Create(jarPath)
	if err != nil {
		t.Fatalf("failed to create jar file %s: %v", jarPath, err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	mf, err := zw.Create("META-INF/MANIFEST.MF")
	if err != nil {
		t.Fatalf("failed to create manifest entry: %v", err)
	}
	manifest := "Manifest-Version: 1.0\n" + manifestBody + "\n"
	if _, err := mf.Write([]byte(manifest)); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close jar writer: %v", err)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}
