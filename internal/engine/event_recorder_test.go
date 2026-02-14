package engine

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// EventRecorder is a test helper that records typed GameEvents emitted by the
// engine. Embed it in a mock UI to replace the old string-based approach.
type EventRecorder struct {
	Events []GameEvent
}

// Emit records the event.
func (r *EventRecorder) Emit(event GameEvent) {
	r.Events = append(r.Events, event)
}

// --- Generic query helpers ---

// OfType returns all recorded events matching type T.
func OfType[T GameEvent](r *EventRecorder) []T {
	return SliceOfType[T](r.Events)
}

// SliceOfType returns all events matching type T from a plain slice.
func SliceOfType[T GameEvent](events []GameEvent) []T {
	var out []T
	for _, e := range events {
		if typed, ok := e.(T); ok {
			out = append(out, typed)
		}
	}
	return out
}

// --- Testify-aware assertion helpers (EventRecorder) ---

// RequireFirst finds the first event of type T or fails the test immediately.
func RequireFirst[T GameEvent](t *testing.T, r *EventRecorder) T {
	t.Helper()
	return requireFirstFrom[T](t, r.Events)
}

// AssertHasEvent asserts that at least one event of type T was recorded.
func AssertHasEvent[T GameEvent](t *testing.T, r *EventRecorder) {
	t.Helper()
	assertHasEventIn[T](t, r.Events)
}

// AssertNoEvent asserts that no events of type T were recorded.
func AssertNoEvent[T GameEvent](t *testing.T, r *EventRecorder) {
	t.Helper()
	assertNoEventIn[T](t, r.Events)
}

// --- Testify-aware assertion helpers (plain slice) ---

// RequireFirstFrom finds the first event of type T in a slice or fails.
func RequireFirstFrom[T GameEvent](t *testing.T, events []GameEvent) T {
	t.Helper()
	return requireFirstFrom[T](t, events)
}

// AssertHasEventIn asserts that at least one event of type T exists in the slice.
func AssertHasEventIn[T GameEvent](t *testing.T, events []GameEvent) {
	t.Helper()
	assertHasEventIn[T](t, events)
}

// AssertNoEventIn asserts that no events of type T exist in the slice.
func AssertNoEventIn[T GameEvent](t *testing.T, events []GameEvent) {
	t.Helper()
	assertNoEventIn[T](t, events)
}

// --- Internal helpers ---

func requireFirstFrom[T GameEvent](t *testing.T, events []GameEvent) T {
	t.Helper()
	for _, e := range events {
		if typed, ok := e.(T); ok {
			return typed
		}
	}
	require.Failf(t, "event not found",
		"no %T in %d events: %s", *new(T), len(events), eventTypes(events))
	return *new(T) // unreachable
}

func assertHasEventIn[T GameEvent](t *testing.T, events []GameEvent) {
	t.Helper()
	for _, e := range events {
		if _, ok := e.(T); ok {
			return
		}
	}
	assert.Failf(t, "event not found",
		"no %T in %d events: %s", *new(T), len(events), eventTypes(events))
}

func assertNoEventIn[T GameEvent](t *testing.T, events []GameEvent) {
	t.Helper()
	found := SliceOfType[T](events)
	assert.Emptyf(t, found, "expected no %T events, found %d", *new(T), len(found))
}

// eventTypes returns a summary string of event types for diagnostic messages.
func eventTypes(events []GameEvent) string {
	if len(events) == 0 {
		return "(none)"
	}
	types := make([]string, len(events))
	for i, e := range events {
		types[i] = fmt.Sprintf("%T", e)
	}
	return fmt.Sprintf("%v", types)
}

// --- EventRecorder self-tests ---

func TestEventRecorder_OfType(t *testing.T) {
	r := &EventRecorder{}
	r.Emit(NarrativeEvent{Text: "first"})
	r.Emit(SystemMessageEvent{Message: "sys"})
	r.Emit(NarrativeEvent{Text: "second"})

	narrs := OfType[NarrativeEvent](r)
	require.Len(t, narrs, 2)
	assert.Equal(t, "first", narrs[0].Text)
	assert.Equal(t, "second", narrs[1].Text)

	sysMsgs := OfType[SystemMessageEvent](r)
	require.Len(t, sysMsgs, 1)
	assert.Equal(t, "sys", sysMsgs[0].Message)

	dialogs := OfType[DialogEvent](r)
	assert.Empty(t, dialogs)
}

func TestEventRecorder_RequireFirst(t *testing.T) {
	r := &EventRecorder{}
	r.Emit(SystemMessageEvent{Message: "hello"})
	r.Emit(NarrativeEvent{Text: "found it"})

	narr := RequireFirst[NarrativeEvent](t, r)
	assert.Equal(t, "found it", narr.Text)
}

func TestEventRecorder_AssertHasEvent(t *testing.T) {
	r := &EventRecorder{}
	r.Emit(NarrativeEvent{Text: "x"})

	AssertHasEvent[NarrativeEvent](t, r)
}

func TestEventRecorder_AssertNoEvent(t *testing.T) {
	r := &EventRecorder{}
	r.Emit(NarrativeEvent{Text: "x"})

	AssertNoEvent[DialogEvent](t, r)
}
