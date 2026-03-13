package wso2patch

import (
	"archive/zip"
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/config"
	"github.com/tharsanan1/ai-helper/internal/util"
)

var (
	patchJarsRepo    string
	patchJarsProduct string
	patchJarsYes     bool
)

const defaultCarbonPatchDirName = "patch9999"

type jarPatchPlan struct {
	ModulePath           string
	ModuleRel            string
	ArtifactID           string
	SymbolicName         string
	SourceBundleVersion  string
	ProductBundleVersion string
	HasVersionMismatch   bool
	SourceJar            string
	ProductJar           string
	PatchJar             string
}

type pomProject struct {
	ArtifactID string `xml:"artifactId"`
	Packaging  string `xml:"packaging"`
}

type pomInfo struct {
	ArtifactID string
	Packaging  string
}

var patchJarsCmd = &cobra.Command{
	Use:   "patch-jars",
	Short: "Create Carbon patch jars from changed modules using git-safe snapshots",
	Long: `Create Carbon patch jars from a selected repo worktree into a WSO2 product directory.

The product directory is protected by git snapshots: aihelper will initialize/commit
product state before patching and auto-roll back on failure.`,
	Example: `  aihelper wso2-patch patch-jars --repo carbon-apimgt
  aihelper wso2-patch patch-jars --repo carbon-apimgt --product ./wso2am-4.5.0
  aihelper wso2-patch patch-jars --repo carbon-apimgt --yes`,
	RunE: runPatchJars,
}

func init() {
	patchJarsCmd.Flags().StringVar(&patchJarsRepo, "repo", "", "Repo in patch root to patch from (required), e.g. carbon-apimgt")
	patchJarsCmd.Flags().StringVar(&patchJarsProduct, "product", "", "Product directory path (optional if exactly one wso2am-* exists in patch root)")
	patchJarsCmd.Flags().BoolVar(&patchJarsYes, "yes", false, "Apply without confirmation prompt")
	_ = patchJarsCmd.MarkFlagRequired("repo")
}

func runPatchJars(cmd *cobra.Command, args []string) error {
	repoFlag := strings.TrimSpace(patchJarsRepo)
	if repoFlag == "" {
		return fmt.Errorf("--repo is required")
	}

	cfgManager, err := utilConfigManager()
	if err != nil {
		return err
	}

	repoNames := resolveRepoNamesFromConfig(cfgManager)
	if len(repoNames) == 0 {
		return fmt.Errorf("wso2-patch.repos is empty; configure at least one repository")
	}
	if !containsString(repoNames, repoFlag) {
		return fmt.Errorf("unknown repo %q. Available repos: %s", repoFlag, strings.Join(repoNames, ", "))
	}

	patchRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	patchRoot, err = filepath.Abs(patchRoot)
	if err != nil {
		return fmt.Errorf("failed to resolve current directory: %w", err)
	}

	if err := validatePatchRoot(patchRoot, repoNames); err != nil {
		return err
	}

	repoPath := filepath.Join(patchRoot, repoFlag)
	if _, err := runGit(repoPath, "rev-parse", "--is-inside-work-tree"); err != nil {
		return fmt.Errorf("repo %q is not a valid git repository at %s", repoFlag, repoPath)
	}

	productDir, err := resolveProductDir(patchRoot, patchJarsProduct)
	if err != nil {
		return err
	}
	patchDir := resolveCarbonPatchDir(productDir)

	printProgress("Collecting changed files from %s", repoFlag)
	changedFiles, err := collectChangedFiles(repoPath)
	if err != nil {
		return err
	}
	if len(changedFiles) == 0 {
		return fmt.Errorf("no changed files found in repo %s", repoFlag)
	}

	modulePaths, err := moduleRootsFromChanges(repoPath, changedFiles)
	if err != nil {
		return err
	}
	if len(modulePaths) == 0 {
		return fmt.Errorf("no runtime-relevant changed modules found for repo %s", repoFlag)
	}

	plans := make([]jarPatchPlan, 0, len(modulePaths))
	for _, modulePath := range modulePaths {
		moduleRel, _ := filepath.Rel(repoPath, modulePath)
		pom, err := readPomInfo(filepath.Join(modulePath, "pom.xml"))
		if err != nil {
			return fmt.Errorf("failed to resolve artifactId for module %s: %w", moduleRel, err)
		}
		if shouldSkipModulePackaging(pom.Packaging) {
			printDone("Skipping aggregator module %s (packaging=%s)", filepath.ToSlash(moduleRel), strings.TrimSpace(pom.Packaging))
			continue
		}

		sourceJar, err := findBuiltModuleJar(modulePath, pom.ArtifactID)
		if err != nil {
			return fmt.Errorf("module %s: %w", moduleRel, err)
		}

		sourceMeta, err := readBundleMetadata(sourceJar)
		if err != nil {
			return fmt.Errorf("module %s: failed to read source jar manifest: %w", moduleRel, err)
		}

		productJar, err := findProductJarMatch(productDir, sourceMeta.SymbolicName, pom.ArtifactID)
		if err != nil {
			return fmt.Errorf("module %s: %w", moduleRel, err)
		}

		productMeta, err := readBundleMetadata(productJar)
		if err != nil {
			return fmt.Errorf("module %s: failed to read product jar manifest: %w", moduleRel, err)
		}

		versionMismatch := sourceMeta.Version != "" && productMeta.Version != "" && sourceMeta.Version != productMeta.Version
		patchJar := filepath.Join(patchDir, filepath.Base(productJar))

		plans = append(plans, jarPatchPlan{
			ModulePath:           modulePath,
			ModuleRel:            filepath.ToSlash(moduleRel),
			ArtifactID:           pom.ArtifactID,
			SymbolicName:         sourceMeta.SymbolicName,
			SourceBundleVersion:  sourceMeta.Version,
			ProductBundleVersion: productMeta.Version,
			HasVersionMismatch:   versionMismatch,
			SourceJar:            sourceJar,
			ProductJar:           productJar,
			PatchJar:             patchJar,
		})
	}

	if len(plans) == 0 {
		return fmt.Errorf("no patchable modules found in repo %s (changes may be only aggregator/docs/test files)", repoFlag)
	}

	sort.Slice(plans, func(i, j int) bool {
		return plans[i].ModuleRel < plans[j].ModuleRel
	})

	printPatchPlan(repoFlag, productDir, patchDir, plans)

	if utilDryRun() {
		fmt.Println("Dry run: no files changed.")
		return nil
	}

	if !patchJarsYes {
		ok, err := confirmProceed(fmt.Sprintf("Proceed with patching %d jar(s)?", len(plans)))
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("patching cancelled")
		}
	}

	printProgress("Ensuring git snapshot in product directory")
	prePatchRef, err := ensureGitSnapshot(productDir)
	if err != nil {
		return err
	}
	printDone("Pre-patch git ref: %s", prePatchRef)

	printProgress("Preparing patch directory %s", patchDir)
	if err := os.MkdirAll(patchDir, 0755); err != nil {
		return fmt.Errorf("failed to create patch directory %s: %w", patchDir, err)
	}
	printDone("Prepared patch directory")

	createdPatchFiles := make(map[string]bool, len(plans))
	patched := 0
	for _, plan := range plans {
		printProgress("Adding patch jar for %s", plan.ModuleRel)

		_, statErr := os.Stat(plan.PatchJar)
		createdPatchFiles[plan.PatchJar] = os.IsNotExist(statErr)
		if statErr != nil && !os.IsNotExist(statErr) {
			return fmt.Errorf("failed to inspect patch destination %s: %w", plan.PatchJar, statErr)
		}

		if err := copyFileAtomic(plan.SourceJar, plan.PatchJar); err != nil {
			cleanupErr := cleanupNewPatchFiles(createdPatchFiles)
			rollbackErr := rollbackToRef(productDir, prePatchRef)
			if rollbackErr != nil && cleanupErr != nil {
				return fmt.Errorf("patch failed for %s: %v; cleanup failed: %v; rollback failed: %v", plan.ModuleRel, err, cleanupErr, rollbackErr)
			}
			if rollbackErr != nil {
				return fmt.Errorf("patch failed for %s: %v; rollback failed: %v", plan.ModuleRel, err, rollbackErr)
			}
			if cleanupErr != nil {
				return fmt.Errorf("patch failed for %s: %v; rollback succeeded but cleanup failed: %v", plan.ModuleRel, err, cleanupErr)
			}
			return fmt.Errorf("patch failed for %s: %v; auto-rolled back to %s", plan.ModuleRel, err, prePatchRef)
		}
		patched++
		printDone("Patched %s", plan.ModuleRel)
	}

	if utilColorEnabled() {
		color.Green("\n✓ Created %d patch jar(s) in %s\n", patched, patchDir)
	} else {
		fmt.Printf("\nCreated %d patch jar(s) in %s\n", patched, patchDir)
	}
	for _, plan := range plans {
		fmt.Printf("  - %s\n", plan.SourceJar)
		fmt.Printf("    -> %s\n", plan.PatchJar)
		fmt.Printf("    (applies to %s)\n", plan.ProductJar)
	}

	fmt.Println()
	fmt.Printf("Restart the server to apply patches from %s.\n", patchDir)
	fmt.Println()
	fmt.Printf("Rollback command:\n  cd %s && git reset --hard %s && git clean -fd -- %s\n",
		productDir,
		prePatchRef,
		filepath.ToSlash(filepath.Join("repository", "components", "patches", defaultCarbonPatchDirName)),
	)

	return nil
}

func utilConfigManager() (*config.Config, error) {
	cfgManager, err := util.GlobalContext.GetConfigManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get config manager: %w", err)
	}

	cfg, err := cfgManager.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

func utilDryRun() bool {
	return util.GlobalContext.IsDryRun()
}

func utilColorEnabled() bool {
	return util.GlobalContext.IsColorEnabled()
}

func resolveProductDir(patchRoot, productFlag string) (string, error) {
	productFlag = strings.TrimSpace(productFlag)
	if productFlag != "" {
		resolved, err := expandPath(productFlag)
		if err != nil {
			return "", fmt.Errorf("failed to resolve --product path %q: %w", productFlag, err)
		}
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(patchRoot, resolved)
		}
		resolved, err = filepath.Abs(resolved)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute product path %q: %w", resolved, err)
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return "", fmt.Errorf("failed to inspect product path %s: %w", resolved, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("product path is not a directory: %s", resolved)
		}
		return resolved, nil
	}

	entries, err := os.ReadDir(patchRoot)
	if err != nil {
		return "", fmt.Errorf("failed to list patch root %s: %w", patchRoot, err)
	}
	candidates := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "wso2am-") {
			candidates = append(candidates, filepath.Join(patchRoot, name))
		}
	}

	switch len(candidates) {
	case 0:
		return "", fmt.Errorf("no wso2am-* product directory found under patch root %s; use --product", patchRoot)
	case 1:
		return candidates[0], nil
	default:
		sort.Strings(candidates)
		return "", fmt.Errorf("multiple wso2am-* product directories found (%s); use --product", strings.Join(candidates, ", "))
	}
}

func collectChangedFiles(repoPath string) ([]string, error) {
	out, err := runGit(repoPath, "status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to read git status in %s: %w", repoPath, err)
	}
	if strings.TrimSpace(out) == "" {
		return []string{}, nil
	}

	seen := make(map[string]struct{})
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if len(line) < 3 {
			continue
		}
		status := line[0:2]
		if status == "!!" {
			continue
		}
		if len(line) < 4 {
			continue
		}

		pathPart := strings.TrimSpace(line[3:])
		if pathPart == "" {
			continue
		}
		if strings.Contains(pathPart, " -> ") {
			parts := strings.Split(pathPart, " -> ")
			pathPart = strings.TrimSpace(parts[len(parts)-1])
		}
		pathPart = filepath.ToSlash(strings.TrimSpace(pathPart))
		if pathPart == "" {
			continue
		}
		seen[pathPart] = struct{}{}
	}

	files := make([]string, 0, len(seen))
	for f := range seen {
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
}

func moduleRootsFromChanges(repoPath string, changedFiles []string) ([]string, error) {
	runtimeRelevantByModule := make(map[string]bool)
	for _, changedFile := range changedFiles {
		modulePath, err := findNearestModuleRoot(repoPath, changedFile)
		if err != nil {
			continue
		}
		moduleRel, err := filepath.Rel(repoPath, modulePath)
		if err != nil {
			continue
		}
		moduleRel = filepath.ToSlash(moduleRel)

		changeRel := filepath.ToSlash(changedFile)
		moduleFileRel := changeRel
		if moduleRel != "." {
			prefix := moduleRel + "/"
			if strings.HasPrefix(changeRel, prefix) {
				moduleFileRel = strings.TrimPrefix(changeRel, prefix)
			}
		}

		if isRuntimeRelevantChange(moduleFileRel) {
			runtimeRelevantByModule[modulePath] = true
			continue
		}
		if _, exists := runtimeRelevantByModule[modulePath]; !exists {
			runtimeRelevantByModule[modulePath] = false
		}
	}

	modules := make([]string, 0)
	for modulePath, relevant := range runtimeRelevantByModule {
		if relevant {
			modules = append(modules, modulePath)
		}
	}
	sort.Strings(modules)
	return modules, nil
}

func isRuntimeRelevantChange(path string) bool {
	p := strings.ToLower(filepath.ToSlash(strings.TrimSpace(path)))
	if p == "" {
		return false
	}
	if strings.HasPrefix(p, ".github/") {
		return false
	}
	if strings.HasPrefix(p, "docs/") || strings.Contains(p, "/docs/") {
		return false
	}
	if strings.HasPrefix(p, "src/test/") || strings.Contains(p, "/src/test/") {
		return false
	}
	if strings.HasSuffix(p, ".md") {
		return false
	}
	if strings.HasPrefix(filepath.Base(p), "readme") {
		return false
	}
	return true
}

func findNearestModuleRoot(repoPath, changedFile string) (string, error) {
	cleanRepo := filepath.Clean(repoPath)
	absPath := filepath.Join(cleanRepo, filepath.FromSlash(changedFile))

	dir := absPath
	if info, err := os.Stat(absPath); err == nil {
		if !info.IsDir() {
			dir = filepath.Dir(absPath)
		}
	} else {
		dir = filepath.Dir(absPath)
	}

	for {
		if dir == "" {
			break
		}
		if _, err := os.Stat(filepath.Join(dir, "pom.xml")); err == nil {
			return dir, nil
		}
		if dir == cleanRepo {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no module root found for %s", changedFile)
}

func readPomInfo(pomPath string) (pomInfo, error) {
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return pomInfo{}, fmt.Errorf("failed to read pom file %s: %w", pomPath, err)
	}
	var project pomProject
	if err := xml.Unmarshal(data, &project); err != nil {
		return pomInfo{}, fmt.Errorf("failed to parse pom file %s: %w", pomPath, err)
	}
	artifactID := strings.TrimSpace(project.ArtifactID)
	if artifactID == "" {
		return pomInfo{}, fmt.Errorf("artifactId not found in %s", pomPath)
	}
	return pomInfo{
		ArtifactID: artifactID,
		Packaging:  strings.TrimSpace(project.Packaging),
	}, nil
}

func shouldSkipModulePackaging(packaging string) bool {
	switch strings.ToLower(strings.TrimSpace(packaging)) {
	case "pom", "war":
		return true
	default:
		return false
	}
}

func findBuiltModuleJar(moduleDir, artifactID string) (string, error) {
	targetDir := filepath.Join(moduleDir, "target")
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return "", fmt.Errorf("failed to read target directory %s: %w", targetDir, err)
	}

	candidates := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".jar") {
			continue
		}
		if strings.HasSuffix(name, "-sources.jar") || strings.HasSuffix(name, "-javadoc.jar") || strings.HasSuffix(name, "-tests.jar") {
			continue
		}
		candidates = append(candidates, filepath.Join(targetDir, name))
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no built jar found in %s; run maven build first", targetDir)
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}

	matching := make([]string, 0)
	for _, candidate := range candidates {
		base := filepath.Base(candidate)
		if base == artifactID+".jar" || strings.HasPrefix(base, artifactID+"-") {
			matching = append(matching, candidate)
		}
	}

	if len(matching) == 1 {
		return matching[0], nil
	}

	if len(matching) > 1 {
		sort.Strings(matching)
		return "", fmt.Errorf("ambiguous built jars for artifact %s: %s", artifactID, strings.Join(matching, ", "))
	}

	sort.Strings(candidates)
	return "", fmt.Errorf("multiple built jars found in %s and none match artifactId %s: %s", targetDir, artifactID, strings.Join(candidates, ", "))
}

type bundleMetadata struct {
	SymbolicName string
	Version      string
}

func readBundleMetadata(jarPath string) (bundleMetadata, error) {
	zr, err := zip.OpenReader(jarPath)
	if err != nil {
		return bundleMetadata{}, err
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.Name != "META-INF/MANIFEST.MF" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return bundleMetadata{}, err
		}
		defer rc.Close()

		manifest, err := parseManifest(rc)
		if err != nil {
			return bundleMetadata{}, err
		}
		name := strings.TrimSpace(manifest["Bundle-SymbolicName"])
		if strings.Contains(name, ";") {
			name = strings.TrimSpace(strings.Split(name, ";")[0])
		}
		return bundleMetadata{
			SymbolicName: name,
			Version:      strings.TrimSpace(manifest["Bundle-Version"]),
		}, nil
	}

	return bundleMetadata{}, nil
}

func parseManifest(r io.Reader) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(r)
	var currentKey string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			currentKey = ""
			continue
		}
		if strings.HasPrefix(line, " ") {
			if currentKey != "" {
				result[currentKey] += strings.TrimPrefix(line, " ")
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		result[key] = val
		currentKey = key
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func findProductJarMatch(productDir, symbolicName, artifactID string) (string, error) {
	prefixes := make([]string, 0, 2)
	if symbolicName != "" {
		prefixes = append(prefixes, symbolicName)
	}
	if artifactID != "" && artifactID != symbolicName {
		prefixes = append(prefixes, artifactID)
	}

	for _, prefix := range prefixes {
		matches, err := findJarMatchesByPrefix(productDir, prefix)
		if err != nil {
			return "", err
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
		if len(matches) > 1 {
			return "", fmt.Errorf("multiple destination jars matched prefix %q: %s", prefix, strings.Join(matches, ", "))
		}
	}

	return "", fmt.Errorf("no destination jar found in %s for symbolicName=%q artifactId=%q", productDir, symbolicName, artifactID)
}

func findJarMatchesByPrefix(root, prefix string) ([]string, error) {
	matches := make([]string, 0)
	patchesRoot := filepath.Join(filepath.Clean(root), "repository", "components", "patches")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && filepath.Clean(path) == patchesRoot {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".jar") {
			return nil
		}
		if strings.HasPrefix(d.Name(), prefix+"_") {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan product directory %s: %w", root, err)
	}
	sort.Strings(matches)
	return matches, nil
}

func ensureGitSnapshot(productDir string) (string, error) {
	isGit, err := isGitRepo(productDir)
	if err != nil {
		return "", err
	}
	if !isGit {
		if _, err := runGit(productDir, "init"); err != nil {
			return "", fmt.Errorf("failed to initialize git repo in %s: %w", productDir, err)
		}
		if err := commitAll(productDir, "aihelper: initial product snapshot"); err != nil {
			return "", err
		}
		return currentGitHead(productDir)
	}

	dirty, err := isGitDirty(productDir)
	if err != nil {
		return "", err
	}
	if dirty {
		msg := fmt.Sprintf("aihelper: pre-patch snapshot %s", time.Now().Format("20060102-150405"))
		if err := commitAll(productDir, msg); err != nil {
			return "", err
		}
	}

	head, err := currentGitHead(productDir)
	if err == nil {
		return head, nil
	}

	if err := commitAll(productDir, "aihelper: initial product snapshot"); err != nil {
		return "", err
	}
	return currentGitHead(productDir)
}

func isGitRepo(productDir string) (bool, error) {
	_, err := runGit(productDir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false, nil
	}
	return true, nil
}

func isGitDirty(productDir string) (bool, error) {
	out, err := runGit(productDir, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("failed to get git status in %s: %w", productDir, err)
	}
	return strings.TrimSpace(out) != "", nil
}

func commitAll(productDir, message string) error {
	if _, err := runGit(productDir, "add", "-A"); err != nil {
		return fmt.Errorf("failed to stage product files in %s: %w", productDir, err)
	}
	if _, err := runGit(productDir, "-c", "user.name=aihelper", "-c", "user.email=aihelper@local", "commit", "-m", message); err != nil {
		return fmt.Errorf("failed to commit git snapshot in %s: %w", productDir, err)
	}
	return nil
}

func currentGitHead(productDir string) (string, error) {
	head, err := runGit(productDir, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to resolve git HEAD in %s: %w", productDir, err)
	}
	head = strings.TrimSpace(head)
	if head == "" {
		return "", fmt.Errorf("empty git HEAD in %s", productDir)
	}
	return head, nil
}

func rollbackToRef(productDir, ref string) error {
	_, err := runGit(productDir, "reset", "--hard", ref)
	if err != nil {
		return fmt.Errorf("failed to rollback product directory %s to %s: %w", productDir, ref, err)
	}
	return nil
}

func copyFileAtomic(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source jar %s: %w", src, err)
	}
	defer srcFile.Close()

	dstMode := os.FileMode(0644)
	dstInfo, err := os.Stat(dst)
	if err == nil {
		dstMode = dstInfo.Mode()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect destination jar %s: %w", dst, err)
	}

	dstDir := filepath.Dir(dst)
	tmpFile, err := os.CreateTemp(dstDir, ".aihelper-patch-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file in %s: %w", dstDir, err)
	}
	tmpPath := tmpFile.Name()

	cleanup := func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}

	if _, err := io.Copy(tmpFile, srcFile); err != nil {
		cleanup()
		return fmt.Errorf("failed to copy jar bytes from %s to temp file: %w", src, err)
	}

	if err := tmpFile.Chmod(dstMode); err != nil {
		cleanup()
		return fmt.Errorf("failed to apply file mode on temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to atomically replace %s: %w", dst, err)
	}

	return nil
}

func printPatchPlan(repoName, productDir, patchDir string, plans []jarPatchPlan) {
	fmt.Println()
	if utilColorEnabled() {
		color.Cyan("============================================================")
		color.Cyan("PATCH PLAN")
		color.Cyan("============================================================")
	} else {
		fmt.Println("============================================================")
		fmt.Println("PATCH PLAN")
		fmt.Println("============================================================")
	}
	fmt.Printf("Repo: %s\n", repoName)
	fmt.Printf("Product: %s\n", productDir)
	fmt.Printf("Patch Dir: %s\n", patchDir)
	fmt.Printf("Jars to patch: %d\n", len(plans))
	versionMismatchCount := 0
	for _, p := range plans {
		if p.HasVersionMismatch {
			versionMismatchCount++
		}
	}
	if versionMismatchCount > 0 {
		fmt.Printf("Version mismatches: %d (review carefully)\n", versionMismatchCount)
	}
	fmt.Println()
	for _, p := range plans {
		fmt.Printf("[%s]\n", p.ModuleRel)
		fmt.Printf("  Artifact: %s\n", p.ArtifactID)
		if p.SymbolicName != "" {
			fmt.Printf("  Bundle-SymbolicName: %s\n", p.SymbolicName)
		}
		if p.SourceBundleVersion != "" || p.ProductBundleVersion != "" {
			fmt.Printf("  Bundle-Version: source=%s product=%s\n", emptyIfBlank(p.SourceBundleVersion), emptyIfBlank(p.ProductBundleVersion))
		}
		if p.HasVersionMismatch {
			fmt.Printf("  WARNING: bundle version mismatch may break OSGi resolution\n")
		}
		fmt.Printf("  Source: %s\n", p.SourceJar)
		fmt.Printf("  Product Jar: %s\n", p.ProductJar)
		fmt.Printf("  Patch Jar: %s\n", p.PatchJar)
		fmt.Println()
	}
}

func resolveCarbonPatchDir(productDir string) string {
	return filepath.Join(productDir, "repository", "components", "patches", defaultCarbonPatchDirName)
}

func cleanupNewPatchFiles(created map[string]bool) error {
	var errs []string
	for path, createdNow := range created {
		if !createdNow {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("%s (%v)", path, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to remove new patch files: %s", strings.Join(errs, ", "))
	}
	return nil
}

func emptyIfBlank(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return "<unknown>"
	}
	return v
}

func confirmProceed(message string) (bool, error) {
	fmt.Printf("%s [y/N] ", message)
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("failed to read confirmation input: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(text))
	return answer == "y" || answer == "yes", nil
}
