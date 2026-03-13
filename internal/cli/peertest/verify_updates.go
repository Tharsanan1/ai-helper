package peertest

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/tharsanan1/ai-helper/internal/util"
	"gopkg.in/yaml.v3"
)

type githubIssuePayload struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type peerTestIssueEnvelope struct {
	PeerTestUpdates peerTestIssueSpec `yaml:"peer_test_updates"`
}

type peerTestIssueSpec struct {
	ProductVersion string                `yaml:"product_version"`
	Updates        []peerTestIssueUpdate `yaml:"updates"`
}

type peerTestIssueUpdate struct {
	UpdateID     string                     `yaml:"update_id"`
	Type         string                     `yaml:"type"`
	Deliverables []peerTestIssueDeliverable `yaml:"deliverables"`
}

type peerTestIssueDeliverable struct {
	Path   string `yaml:"path"`
	Action string `yaml:"action"`
}

type jarReference struct {
	Key      string
	Path     string
	Version  string
	UpdateID string
}

type jarConflictGroup struct {
	Key           string
	LatestPath    string
	LatestVersion string
	References    []jarReference
}

type missingDeliverable struct {
	UpdateID string
	Path     string
}

func runPeerTestVerifyUpdates(productCfg *resolvedProductConfig, issueURL, issueNumber string) error {
	printProgress("Fetching peer test issue for update verification")
	owner, repo, number, err := parseGitHubIssueURL(issueURL)
	if err != nil {
		return err
	}
	issue, err := fetchGitHubIssue(owner, repo, number)
	if err != nil {
		return err
	}
	printDone("Fetched peer test issue")

	printProgress("Parsing peer test update metadata")
	spec, err := extractPeerTestIssueSpec(issue.Body)
	if err != nil {
		return err
	}
	if strings.TrimSpace(spec.ProductVersion) != "" && strings.TrimSpace(spec.ProductVersion) != productCfg.version {
		return fmt.Errorf("issue product_version %s does not match --product-version %s", spec.ProductVersion, productCfg.version)
	}
	printDone("Parsed peer test update metadata")

	issueDir := resolveIssueDir(productCfg, issueNumber)
	productRoot, err := findExtractedProductDir(issueDir)
	if err != nil {
		return err
	}

	if util.GlobalContext.IsDryRun() {
		fmt.Println("Dry run: would verify peer test issue updates with these settings:")
		fmt.Printf("  Issue: %s/%s#%d\n", owner, repo, issue.Number)
		fmt.Printf("  Issue title: %s\n", issue.Title)
		fmt.Printf("  Product root: %s\n", productRoot)
		fmt.Printf("  Updates: %d\n", len(spec.Updates))
		return nil
	}

	printProgress("Verifying issue deliverables against %s", productRoot)
	report := verifyPeerTestIssueAgainstProduct(productRoot, spec)
	printVerificationReport(report)

	if len(report.Missing) > 0 || len(report.LatestJarMissing) > 0 {
		return fmt.Errorf("peer test issue verification found missing deliverables")
	}
	printDone("Peer test issue verification completed")
	return nil
}

func parseGitHubIssueURL(issueURL string) (owner, repo string, number int, err error) {
	parsed, err := url.Parse(issueURL)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid --peertest-issue URL %q: %w", issueURL, err)
	}
	if !strings.EqualFold(parsed.Host, "github.com") {
		return "", "", 0, fmt.Errorf("invalid --peertest-issue URL: expected a github.com issue URL")
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 || parts[2] != "issues" {
		return "", "", 0, fmt.Errorf("invalid --peertest-issue URL: expected https://github.com/<owner>/<repo>/issues/<number>")
	}
	number, err = strconv.Atoi(parts[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid --peertest-issue URL: expected numeric issue number")
	}
	return parts[0], parts[1], number, nil
}

func fetchGitHubIssue(owner, repo string, number int) (*githubIssuePayload, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("gh is required for --verify-updates but was not found in PATH")
	}

	cmd := exec.Command("gh", "issue", "view", strconv.Itoa(number), "--repo", owner+"/"+repo, "--json", "number,title,body")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to fetch github issue via gh: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("failed to fetch github issue via gh: %w", err)
	}

	var issue githubIssuePayload
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to decode gh issue output: %w", err)
	}
	return &issue, nil
}

var fencedYAMLBlockPattern = regexp.MustCompile("(?s)```(?:yaml|yml)?\\s*(.*?)```")

func extractPeerTestIssueSpec(body string) (*peerTestIssueSpec, error) {
	matches := fencedYAMLBlockPattern.FindAllStringSubmatch(body, -1)
	for _, match := range matches {
		block := strings.TrimSpace(match[1])
		if !strings.Contains(block, "peer_test_updates:") {
			continue
		}

		var envelope peerTestIssueEnvelope
		if err := yaml.Unmarshal([]byte(block), &envelope); err != nil {
			return nil, fmt.Errorf("failed to parse peer_test_updates YAML block: %w", err)
		}
		if len(envelope.PeerTestUpdates.Updates) == 0 {
			return nil, fmt.Errorf("peer_test_updates YAML block does not contain any updates")
		}
		for i := range envelope.PeerTestUpdates.Updates {
			update := &envelope.PeerTestUpdates.Updates[i]
			update.UpdateID = strings.TrimSpace(update.UpdateID)
			if update.UpdateID == "" {
				return nil, fmt.Errorf("peer_test_updates contains an update without update_id")
			}
			for j := range update.Deliverables {
				update.Deliverables[j].Path = strings.TrimSpace(update.Deliverables[j].Path)
			}
		}
		return &envelope.PeerTestUpdates, nil
	}
	return nil, fmt.Errorf("issue does not contain a peer_test_updates YAML block")
}

type peerTestVerificationReport struct {
	ExactDuplicates  map[string][]string
	JarConflicts     []jarConflictGroup
	Missing          []missingDeliverable
	LatestJarMissing []missingDeliverable
	UpdateCounts     map[string]int
}

func verifyPeerTestIssueAgainstProduct(productRoot string, spec *peerTestIssueSpec) peerTestVerificationReport {
	report := peerTestVerificationReport{
		ExactDuplicates: make(map[string][]string),
		UpdateCounts:    make(map[string]int),
	}

	pathToUpdates := map[string]map[string]struct{}{}
	jarGroups := map[string][]jarReference{}
	conflictLatestPath := map[string]string{}
	conflictPaths := map[string]struct{}{}
	latestJarMissingPaths := map[string]struct{}{}

	for _, update := range spec.Updates {
		for _, deliverable := range update.Deliverables {
			path := strings.TrimSpace(deliverable.Path)
			if path == "" {
				continue
			}
			report.UpdateCounts[update.UpdateID]++
			if _, ok := pathToUpdates[path]; !ok {
				pathToUpdates[path] = map[string]struct{}{}
			}
			pathToUpdates[path][update.UpdateID] = struct{}{}

			if ref, ok := parseVersionedJarReference(path, update.UpdateID); ok {
				jarGroups[ref.Key] = append(jarGroups[ref.Key], ref)
			}
		}
	}

	for path, updates := range pathToUpdates {
		if len(updates) > 1 {
			report.ExactDuplicates[path] = sortedSetKeys(updates)
		}
	}

	for key, refs := range jarGroups {
		versions := map[string]struct{}{}
		for _, ref := range refs {
			versions[ref.Version] = struct{}{}
		}
		if len(versions) <= 1 {
			continue
		}
		latest := refs[0]
		for _, ref := range refs[1:] {
			if compareVersionStrings(ref.Version, latest.Version) > 0 {
				latest = ref
			}
		}
		report.JarConflicts = append(report.JarConflicts, jarConflictGroup{
			Key:           key,
			LatestPath:    latest.Path,
			LatestVersion: latest.Version,
			References:    append([]jarReference(nil), refs...),
		})
		conflictLatestPath[key] = latest.Path
		for _, ref := range refs {
			conflictPaths[ref.Path] = struct{}{}
		}
		if !pathExists(filepath.Join(productRoot, filepath.FromSlash(latest.Path))) {
			report.LatestJarMissing = append(report.LatestJarMissing, missingDeliverable{
				UpdateID: latest.UpdateID,
				Path:     latest.Path,
			})
			latestJarMissingPaths[latest.Path] = struct{}{}
		}
	}

	for _, update := range spec.Updates {
		for _, deliverable := range update.Deliverables {
			path := strings.TrimSpace(deliverable.Path)
			if path == "" {
				continue
			}
			if _, ok := conflictPaths[path]; ok {
				ref, ok := parseVersionedJarReference(path, update.UpdateID)
				if ok && conflictLatestPath[ref.Key] != path {
					continue
				}
			}
			if _, alreadyReported := latestJarMissingPaths[path]; alreadyReported {
				continue
			}
			if !pathExists(filepath.Join(productRoot, filepath.FromSlash(path))) {
				report.Missing = append(report.Missing, missingDeliverable{
					UpdateID: update.UpdateID,
					Path:     path,
				})
			}
		}
	}

	sort.Slice(report.JarConflicts, func(i, j int) bool { return report.JarConflicts[i].Key < report.JarConflicts[j].Key })
	sort.Slice(report.Missing, func(i, j int) bool { return report.Missing[i].Path < report.Missing[j].Path })
	sort.Slice(report.LatestJarMissing, func(i, j int) bool { return report.LatestJarMissing[i].Path < report.LatestJarMissing[j].Path })
	return report
}

var versionedJarPattern = regexp.MustCompile(`^(.+)_([0-9][A-Za-z0-9._-]*)\.jar$`)

func parseVersionedJarReference(path, updateID string) (jarReference, bool) {
	base := filepath.Base(path)
	match := versionedJarPattern.FindStringSubmatch(base)
	if len(match) != 3 {
		return jarReference{}, false
	}
	dir := filepath.ToSlash(filepath.Dir(path))
	return jarReference{
		Key:      filepath.ToSlash(filepath.Join(dir, match[1])),
		Path:     filepath.ToSlash(path),
		Version:  match[2],
		UpdateID: updateID,
	}, true
}

var versionTokenPattern = regexp.MustCompile(`[0-9]+|[A-Za-z]+`)

func compareVersionStrings(a, b string) int {
	aTokens := versionTokenPattern.FindAllString(a, -1)
	bTokens := versionTokenPattern.FindAllString(b, -1)
	maxLen := len(aTokens)
	if len(bTokens) > maxLen {
		maxLen = len(bTokens)
	}

	for i := 0; i < maxLen; i++ {
		if i >= len(aTokens) {
			return -1
		}
		if i >= len(bTokens) {
			return 1
		}
		aTok := aTokens[i]
		bTok := bTokens[i]
		aNum, aErr := strconv.Atoi(aTok)
		bNum, bErr := strconv.Atoi(bTok)
		switch {
		case aErr == nil && bErr == nil:
			if aNum < bNum {
				return -1
			}
			if aNum > bNum {
				return 1
			}
		default:
			aNorm := strings.ToLower(aTok)
			bNorm := strings.ToLower(bTok)
			if aNorm < bNorm {
				return -1
			}
			if aNorm > bNorm {
				return 1
			}
		}
	}
	return 0
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func sortedSetKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func printVerificationReport(report peerTestVerificationReport) {
	fmt.Println()
	printSeparator()
	fmt.Println("Peer Test Update Verification")
	printSeparator()

	if len(report.Missing) == 0 && len(report.LatestJarMissing) == 0 {
		printDone("All required deliverables were found in the product")
	} else {
		fmt.Println("Missing deliverables:")
		for _, item := range report.Missing {
			fmt.Printf("  - [%s] %s\n", item.UpdateID, item.Path)
		}
		for _, item := range report.LatestJarMissing {
			fmt.Printf("  - [latest missing %s] %s\n", item.UpdateID, item.Path)
		}
	}

	if len(report.ExactDuplicates) > 0 {
		fmt.Println()
		fmt.Println("Duplicate exact paths across updates:")
		paths := make([]string, 0, len(report.ExactDuplicates))
		for path := range report.ExactDuplicates {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			fmt.Printf("  - %s (updates: %s)\n", path, strings.Join(report.ExactDuplicates[path], ", "))
		}
	}

	if len(report.JarConflicts) > 0 {
		fmt.Println()
		fmt.Println("Same jar mentioned with different versions:")
		for _, group := range report.JarConflicts {
			fmt.Printf("  - %s -> latest %s\n", group.Key, group.LatestPath)
			for _, ref := range group.References {
				fmt.Printf("      [%s] %s\n", ref.UpdateID, ref.Path)
			}
		}
	}

	fmt.Println()
	fmt.Println("Summary by update:")
	ids := make([]string, 0, len(report.UpdateCounts))
	for id := range report.UpdateCounts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		fmt.Printf("  - %s: %d deliverable(s)\n", id, report.UpdateCounts[id])
	}
	printSeparator()
}
