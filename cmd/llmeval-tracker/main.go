// llmeval-tracker records and reports on LLM eval test results.
//
// Usage:
//
//	go test -v -json -tags=llmeval ./test/llmeval/... | go run ./cmd/llmeval-tracker record
//	go run ./cmd/llmeval-tracker report [--last N] [--flaky]
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultResultsFile = "test/llmeval/results.jsonl"

// TestEvent mirrors go test -json output.
type TestEvent struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Output  string    `json:"Output"`
	Elapsed float64   `json:"Elapsed"`
}

// RunRecord is one line in results.jsonl.
type RunRecord struct {
	Timestamp   time.Time         `json:"timestamp"`
	Commit      string            `json:"commit"`
	DurationSec float64           `json:"duration_secs"`
	Tests       map[string]string `json:"tests"` // test name -> "pass"|"fail"|"skip"
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "record":
		cmdRecord()
	case "report":
		cmdReport()
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  go test -v -json -tags=llmeval ./test/llmeval/... | llmeval-tracker record")
	fmt.Fprintln(os.Stderr, "  llmeval-tracker report [--last N] [--flaky]")
}

// cmdRecord reads go test -json from stdin, writes a RunRecord to results.jsonl.
func cmdRecord() {
	resultsPath := resultsFile()
	tests := make(map[string]string)

	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer for long output lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var ev TestEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue // skip non-JSON lines
		}

		if ev.Test == "" {
			continue
		}

		switch ev.Action {
		case "pass":
			tests[ev.Test] = "pass"
		case "fail":
			tests[ev.Test] = "fail"
		case "skip":
			tests[ev.Test] = "skip"
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	// Remove parent tests, keep only leaf subtests.
	pruned := pruneParents(tests)

	if len(pruned) == 0 {
		fmt.Fprintln(os.Stderr, "No test results found in input.")
		os.Exit(1)
	}

	record := RunRecord{
		Timestamp: time.Now().UTC(),
		Commit:    gitCommit(),
		Tests:     pruned,
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(resultsPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	f, err := os.OpenFile(resultsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", resultsPath, err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(record); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing record: %v\n", err)
		os.Exit(1)
	}

	pass, fail, skip := 0, 0, 0
	for _, v := range pruned {
		switch v {
		case "pass":
			pass++
		case "fail":
			fail++
		case "skip":
			skip++
		}
	}
	fmt.Fprintf(os.Stderr, "Recorded %d tests (%d pass, %d fail, %d skip) -> %s\n",
		len(pruned), pass, fail, skip, resultsPath)
}

// cmdReport reads results.jsonl and prints a stability report.
func cmdReport() {
	resultsPath := resultsFile()
	last := 10
	flakyOnly := false

	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--last":
			if i+1 < len(os.Args) {
				_, _ = fmt.Sscanf(os.Args[i+1], "%d", &last)
				i++
			}
		case "--flaky":
			flakyOnly = true
		}
	}

	records, err := loadRecords(resultsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading %s: %v\n", resultsPath, err)
		os.Exit(1)
	}

	if len(records) == 0 {
		fmt.Println("No results recorded yet.")
		return
	}

	// Use only last N records
	if len(records) > last {
		records = records[len(records)-last:]
	}

	// Collect all test names
	testNames := collectTestNames(records)

	// Build per-test history
	type testHistory struct {
		name    string
		results []string // "pass", "fail", "skip", "" (not present)
		passes  int
		fails   int
		total   int // non-skip, non-empty
	}

	var histories []testHistory
	for _, name := range testNames {
		h := testHistory{name: name}
		for _, rec := range records {
			result, ok := rec.Tests[name]
			if !ok {
				h.results = append(h.results, "")
				continue
			}
			h.results = append(h.results, result)
			switch result {
			case "pass":
				h.passes++
				h.total++
			case "fail":
				h.fails++
				h.total++
			}
		}
		histories = append(histories, h)
	}

	// Sort: flakiest first, then alphabetical
	sort.Slice(histories, func(i, j int) bool {
		ri := passRate(histories[i].passes, histories[i].total)
		rj := passRate(histories[j].passes, histories[j].total)
		if ri != rj {
			return ri < rj
		}
		return histories[i].name < histories[j].name
	})

	// Print header
	fmt.Printf("\nLLM Eval Test Stability Report (last %d runs)\n", len(records))
	fmt.Println(strings.Repeat("=", 80))

	// Run-over-run summary
	fmt.Printf("\nRuns:\n")
	for i, rec := range records {
		pass, fail, skip := countResults(rec.Tests)
		total := pass + fail
		pct := 0.0
		if total > 0 {
			pct = float64(pass) * 100 / float64(total)
		}
		commit := rec.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		fmt.Printf("  #%-3d  %s  %s  %3d pass  %2d fail  %2d skip  (%.0f%%)\n",
			i+1, rec.Timestamp.Format("2006-01-02 15:04"), commit,
			pass, fail, skip, pct)
	}

	// Per-test table
	if flakyOnly {
		fmt.Printf("\nFlaky tests (< 100%% pass rate):\n")
	} else {
		fmt.Printf("\nPer-test results:\n")
	}
	fmt.Println(strings.Repeat("-", 80))

	maxNameLen := 0
	for _, h := range histories {
		if len(h.name) > maxNameLen {
			maxNameLen = len(h.name)
		}
	}
	if maxNameLen > 55 {
		maxNameLen = 55
	}

	printed := 0
	for _, h := range histories {
		if h.total == 0 {
			continue
		}
		rate := passRate(h.passes, h.total)
		if flakyOnly && rate >= 100.0 {
			continue
		}

		name := h.name
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		trend := buildTrend(h.results)

		fmt.Printf("  %-*s  %3d/%-3d (%5.1f%%)  %s\n",
			maxNameLen, name, h.passes, h.total, rate, trend)
		printed++
	}

	if printed == 0 && flakyOnly {
		fmt.Println("  All tests passed 100% of the time!")
	}

	fmt.Println()
}

func passRate(passes, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(passes) * 100 / float64(total)
}

func buildTrend(results []string) string {
	var sb strings.Builder
	for _, r := range results {
		switch r {
		case "pass":
			sb.WriteRune('\u2713')
		case "fail":
			sb.WriteRune('\u2717')
		case "skip":
			sb.WriteRune('-')
		default:
			sb.WriteRune('\u00b7')
		}
	}
	return sb.String()
}

func countResults(tests map[string]string) (pass, fail, skip int) {
	for _, v := range tests {
		switch v {
		case "pass":
			pass++
		case "fail":
			fail++
		case "skip":
			skip++
		}
	}
	return
}

func collectTestNames(records []RunRecord) []string {
	seen := make(map[string]bool)
	for _, rec := range records {
		for name := range rec.Tests {
			seen[name] = true
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func loadRecords(path string) ([]RunRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var records []RunRecord
	dec := json.NewDecoder(f)
	for {
		var rec RunRecord
		if err := dec.Decode(&rec); err == io.EOF {
			break
		} else if err != nil {
			continue
		}
		records = append(records, rec)
	}
	return records, nil
}

// pruneParents removes parent test entries when leaf subtests exist.
func pruneParents(tests map[string]string) map[string]string {
	names := make([]string, 0, len(tests))
	for name := range tests {
		names = append(names, name)
	}
	sort.Strings(names)

	parents := make(map[string]bool)
	for _, name := range names {
		for _, other := range names {
			if other != name && strings.HasPrefix(other, name+"/") {
				parents[name] = true
				break
			}
		}
	}

	pruned := make(map[string]string, len(tests)-len(parents))
	for name, result := range tests {
		if !parents[name] {
			pruned[name] = result
		}
	}
	return pruned
}

func gitCommit() string {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func resultsFile() string {
	if v := os.Getenv("LLMEVAL_RESULTS"); v != "" {
		return v
	}
	return defaultResultsFile
}
