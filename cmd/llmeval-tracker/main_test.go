package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPruneParents(t *testing.T) {
	tests := map[string]string{
		"TestFoo":         "pass",
		"TestFoo/Bar":     "pass",
		"TestFoo/Baz":     "fail",
		"TestStandalone":  "pass",
		"TestA":           "fail",
		"TestA/Sub1":      "pass",
		"TestA/Sub1/Leaf": "pass",
		"TestA/Sub2":      "fail",
	}

	pruned := pruneParents(tests)

	// Parents should be removed
	assert.NotContains(t, pruned, "TestFoo")
	assert.NotContains(t, pruned, "TestA")
	assert.NotContains(t, pruned, "TestA/Sub1")

	// Leaves and standalone should remain
	assert.Equal(t, "pass", pruned["TestFoo/Bar"])
	assert.Equal(t, "fail", pruned["TestFoo/Baz"])
	assert.Equal(t, "pass", pruned["TestStandalone"])
	assert.Equal(t, "pass", pruned["TestA/Sub1/Leaf"])
	assert.Equal(t, "fail", pruned["TestA/Sub2"])
}

func TestPruneParents_NoSubtests(t *testing.T) {
	tests := map[string]string{
		"TestAlpha": "pass",
		"TestBeta":  "fail",
	}

	pruned := pruneParents(tests)

	assert.Equal(t, "pass", pruned["TestAlpha"])
	assert.Equal(t, "fail", pruned["TestBeta"])
}

func TestCountResults(t *testing.T) {
	tests := map[string]string{
		"TestA": "pass",
		"TestB": "pass",
		"TestC": "fail",
		"TestD": "skip",
	}

	pass, fail, skip := countResults(tests)
	assert.Equal(t, 2, pass)
	assert.Equal(t, 1, fail)
	assert.Equal(t, 1, skip)
}

func TestBuildTrend(t *testing.T) {
	results := []string{"pass", "fail", "pass", "skip", ""}
	trend := buildTrend(results)
	assert.Equal(t, "\u2713\u2717\u2713-\u00b7", trend)
}

func TestPassRate(t *testing.T) {
	assert.InDelta(t, 75.0, passRate(3, 4), 0.01)
	assert.InDelta(t, 100.0, passRate(5, 5), 0.01)
	assert.InDelta(t, 0.0, passRate(0, 3), 0.01)
	assert.InDelta(t, 0.0, passRate(0, 0), 0.01)
}

func TestCollectTestNames(t *testing.T) {
	records := []RunRecord{
		{Tests: map[string]string{"TestA": "pass", "TestB": "fail"}},
		{Tests: map[string]string{"TestB": "pass", "TestC": "pass"}},
	}

	names := collectTestNames(records)
	assert.Equal(t, []string{"TestA", "TestB", "TestC"}, names)
}

func TestLoadRecords_NonExistent(t *testing.T) {
	records, err := loadRecords("/nonexistent/path/results.jsonl")
	require.NoError(t, err)
	assert.Nil(t, records)
}

func TestLoadRecords_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "results.jsonl")

	f, err := os.Create(path)
	require.NoError(t, err)

	records := []RunRecord{
		{
			Timestamp: time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC),
			Commit:    "abc1234",
			Tests:     map[string]string{"TestA": "pass", "TestB": "fail"},
		},
		{
			Timestamp: time.Date(2026, 2, 27, 11, 0, 0, 0, time.UTC),
			Commit:    "def5678",
			Tests:     map[string]string{"TestA": "pass", "TestB": "pass"},
		},
	}

	enc := json.NewEncoder(f)
	for _, rec := range records {
		require.NoError(t, enc.Encode(rec))
	}
	require.NoError(t, f.Close())

	loaded, err := loadRecords(path)
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	assert.Equal(t, "abc1234", loaded[0].Commit)
	assert.Equal(t, "fail", loaded[0].Tests["TestB"])
	assert.Equal(t, "pass", loaded[1].Tests["TestB"])
}

func TestParseGoTestJSON(t *testing.T) {
	input := strings.Join([]string{
		`{"Time":"2026-02-27T10:00:00Z","Action":"run","Package":"pkg","Test":"TestFoo"}`,
		`{"Time":"2026-02-27T10:00:01Z","Action":"run","Package":"pkg","Test":"TestFoo/Bar"}`,
		`{"Time":"2026-02-27T10:00:02Z","Action":"pass","Package":"pkg","Test":"TestFoo/Bar"}`,
		`{"Time":"2026-02-27T10:00:03Z","Action":"run","Package":"pkg","Test":"TestFoo/Baz"}`,
		`{"Time":"2026-02-27T10:00:04Z","Action":"fail","Package":"pkg","Test":"TestFoo/Baz"}`,
		`{"Time":"2026-02-27T10:00:05Z","Action":"fail","Package":"pkg","Test":"TestFoo"}`,
		`{"Time":"2026-02-27T10:00:06Z","Action":"pass","Package":"pkg","Test":"TestStandalone"}`,
	}, "\n")

	tests := parseTestEvents(input)
	pruned := pruneParents(tests)

	assert.Equal(t, "pass", pruned["TestFoo/Bar"])
	assert.Equal(t, "fail", pruned["TestFoo/Baz"])
	assert.Equal(t, "pass", pruned["TestStandalone"])
	assert.NotContains(t, pruned, "TestFoo")
}

// parseTestEvents is a helper that mimics the record stdin parsing logic.
func parseTestEvents(input string) map[string]string {
	tests := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewBufferString(input))
	for scanner.Scan() {
		var ev TestEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
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
	return tests
}
