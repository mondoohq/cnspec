// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package progress

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// keepFilesAlive holds files whose descriptor was handed off to a stream, so
// that the descriptor is closed exactly once — by the stream.
var keepFilesAlive []*os.File

// lineWriter records every individual Write call so a test can prove the
// stream never emits a partial line.
type lineWriter struct {
	mu     sync.Mutex
	writes []string
}

func (l *lineWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writes = append(l.writes, string(p))
	return len(p), nil
}

func (l *lineWriter) all() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string{}, l.writes...)
}

func parseEvents(t *testing.T, raw string) []StreamEvent {
	t.Helper()

	var events []StreamEvent
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var e StreamEvent
		require.NoErrorf(t, json.Unmarshal([]byte(line), &e), "unparseable line: %q", line)
		events = append(events, e)
	}
	require.NoError(t, scanner.Err())
	return events
}

func eventNames(events []StreamEvent) []string {
	names := make([]string, len(events))
	for i, e := range events {
		names[i] = e.Event
	}
	return names
}

func TestStreamSingleAsset(t *testing.T) {
	w := &lineWriter{}
	s := NewStreamWriter(w)

	s.Discovered(3)
	s.Filtered(1)
	s.AddTask("id-1", testAsset("test1", "linux"))
	s.OnProgress("id-1", 0.0)
	s.OnProgress("id-1", 0.5)
	s.OnProgress("id-1", 0.503) // same whole percent, must not emit again
	s.OnProgress("id-1", 1.0)
	s.Score("id-1", "A")
	s.Completed("id-1")
	s.Close()

	events := parseEvents(t, strings.Join(w.all(), ""))
	assert.Equal(t, []string{
		EventScanStart,
		EventDiscovered,
		EventFiltered,
		EventAssetAdded,
		EventAssetProgress, // 0%
		EventAssetProgress, // 50%
		EventAssetProgress, // 100%
		EventAssetScore,
		EventAssetDone,
		EventScanDone,
	}, eventNames(events))

	// seq is monotonic and gapless, and every event is timestamped.
	for i, e := range events {
		assert.Equal(t, uint64(i+1), e.Seq)
		assert.NotEmpty(t, e.Time)
	}

	assert.Equal(t, StreamVersion, events[0].Version)
	assert.Equal(t, 3, events[1].Count)
	assert.Equal(t, 3, events[1].Discovered)
	assert.Equal(t, 1, events[2].Filtered)

	added := events[3]
	assert.Equal(t, "id-1", added.Index)
	assert.Equal(t, "test1", added.Name)
	assert.Equal(t, "linux", added.Platform)
	assert.Equal(t, 1, added.Num)

	assert.Equal(t, 0.5, events[5].Percent)
	assert.Equal(t, 1.0, events[6].Percent)
	assert.Equal(t, "A", events[7].Score)

	done := events[8]
	assert.Equal(t, StateCompleted, done.State)
	assert.Equal(t, 1, done.Num)
	assert.Equal(t, 1, done.Done)
	assert.Equal(t, 1, done.Total)

	summary := events[9]
	assert.Equal(t, EventScanDone, summary.Event)
	assert.Equal(t, 1, summary.Total)
	assert.Equal(t, 1, summary.Completed)
	assert.Equal(t, 0, summary.Errored)
	assert.Equal(t, 3, summary.Discovered)
	assert.Equal(t, 1, summary.Filtered)
}

func TestStreamTerminalStates(t *testing.T) {
	w := &lineWriter{}
	s := NewStreamWriter(w)

	s.AddTask("ok", testAsset("ok", "linux"))
	s.AddTask("bad", testAsset("bad", "linux"))
	s.AddTask("na", testAsset("na", ""))

	s.Completed("ok")
	s.Errored("bad")
	s.NotApplicable("na")
	s.Completed("ok") // duplicate, must not double count
	s.Close()

	events := parseEvents(t, strings.Join(w.all(), ""))

	var states []string
	var summary StreamEvent
	for _, e := range events {
		switch e.Event {
		case EventAssetDone:
			states = append(states, e.State)
		case EventScanDone:
			summary = e
		}
	}

	assert.Equal(t, []string{StateCompleted, StateErrored, StateNotApplicable}, states)
	assert.Equal(t, 3, summary.Total)
	assert.Equal(t, 3, summary.Done)
	assert.Equal(t, 1, summary.Completed)
	assert.Equal(t, 1, summary.Errored)
	assert.Equal(t, 1, summary.NotApplicable)
}

// An asset the scanner never announced still gets an asset_added event before
// anything else references it, so the stream stays self-describing.
func TestStreamAnnouncesUnknownAssets(t *testing.T) {
	w := &lineWriter{}
	s := NewStreamWriter(w)

	s.OnProgress("ghost", 0.25)
	s.Completed("ghost")
	s.Close()

	events := parseEvents(t, strings.Join(w.all(), ""))
	assert.Equal(t, []string{
		EventScanStart, EventAssetAdded, EventAssetProgress, EventAssetDone, EventScanDone,
	}, eventNames(events))
	assert.Equal(t, "ghost", events[1].Index)
	assert.Equal(t, 1, events[1].Num)
}

func TestStreamCloseIsIdempotent(t *testing.T) {
	w := &lineWriter{}
	s := NewStreamWriter(w)

	s.AddTask("id-1", testAsset("test1", "linux"))
	s.Close()
	s.Close()

	// Events after Close are dropped rather than trailing the summary.
	s.AddTask("id-2", testAsset("test2", "linux"))
	s.Completed("id-2")

	events := parseEvents(t, strings.Join(w.all(), ""))
	assert.Equal(t, []string{EventScanStart, EventAssetAdded, EventScanDone}, eventNames(events))
}

// The scanner runs its pipeline with configurable parallelism, so every method
// can be called from a different goroutine at once. Run with -race.
func TestStreamConcurrentWriters(t *testing.T) {
	w := &lineWriter{}
	s := NewStreamWriter(w)

	const assets = 16
	var wg sync.WaitGroup
	for i := range assets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			index := fmt.Sprintf("asset-%d", i)
			s.AddTask(index, testAsset(index, "linux"))
			for p := range 10 {
				s.OnProgress(index, float64(p)/10)
			}
			s.Score(index, "A")
			if i%4 == 0 {
				s.Errored(index)
			} else {
				s.Completed(index)
			}
			s.Discovered(1)
		}()
	}
	wg.Wait()
	s.Close()

	// Every Write call is exactly one complete line: a consumer reading the
	// stream can never see two events spliced together or half of one.
	writes := w.all()
	for _, line := range writes {
		assert.Truef(t, strings.HasSuffix(line, "\n"), "write did not end a line: %q", line)
		assert.Equalf(t, 1, strings.Count(line, "\n"), "write held more than one line: %q", line)
	}

	events := parseEvents(t, strings.Join(writes, ""))
	require.Len(t, events, len(writes))
	for i, e := range events {
		assert.Equal(t, uint64(i+1), e.Seq)
	}

	summary := events[len(events)-1]
	assert.Equal(t, EventScanDone, summary.Event)
	assert.Equal(t, assets, summary.Total)
	assert.Equal(t, assets, summary.Done)
	assert.Equal(t, assets/4, summary.Errored)
	assert.Equal(t, assets-assets/4, summary.Completed)
	assert.Equal(t, assets, summary.Discovered)

	// Each asset was announced exactly once, with a unique 1-based num.
	nums := map[int]string{}
	for _, e := range events {
		if e.Event != EventAssetAdded {
			continue
		}
		_, dup := nums[e.Num]
		assert.Falsef(t, dup, "num %d used twice", e.Num)
		nums[e.Num] = e.Index
	}
	assert.Len(t, nums, assets)
}

func TestStreamToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.ndjson")

	s, err := NewStream(path)
	require.NoError(t, err)

	s.AddTask("id-1", testAsset("test1", "linux"))
	s.Completed("id-1")
	s.Close()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, []string{
		EventScanStart, EventAssetAdded, EventAssetDone, EventScanDone,
	}, eventNames(parseEvents(t, string(raw))))
}

// A file descriptor handed down by a parent process (exec.Cmd.ExtraFiles) is
// the form the launcher uses: no temp file to clean up, and the reader sees
// EOF when the child exits.
func TestStreamToFileDescriptor(t *testing.T) {
	r, wf, err := os.Pipe()
	require.NoError(t, err)
	defer func() { _ = r.Close() }()

	// Ownership of the descriptor moves to the stream, which closes it. The
	// pipe's own *os.File must therefore outlive the test: if it were
	// collected, its cleanup would close the same descriptor a second time,
	// and by then the number can already belong to something else.
	fd := wf.Fd()
	keepFilesAlive = append(keepFilesAlive, wf)

	s, err := NewStream(fmt.Sprintf("fd:%d", fd))
	require.NoError(t, err)

	lines := make(chan []StreamEvent, 1)
	go func() {
		var events []StreamEvent
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			var e StreamEvent
			if err := json.Unmarshal(scanner.Bytes(), &e); err == nil {
				events = append(events, e)
			}
		}
		lines <- events
	}()

	s.AddTask("id-1", testAsset("test1", "linux"))
	s.Completed("id-1")
	s.Close() // closes the write end, so the reader sees EOF

	assert.Equal(t, []string{
		EventScanStart, EventAssetAdded, EventAssetDone, EventScanDone,
	}, eventNames(<-lines))
}

func TestStreamTargetErrors(t *testing.T) {
	_, err := NewStream("")
	assert.Error(t, err)

	_, err = NewStream("fd:not-a-number")
	assert.Error(t, err)

	_, err = NewStream(filepath.Join(t.TempDir(), "missing-dir", "progress.ndjson"))
	assert.Error(t, err)
}

// A destination that goes away mid-scan must not take the scan down with it.
func TestStreamSurvivesABrokenDestination(t *testing.T) {
	w := &failingWriter{}
	s := NewStreamWriter(w)

	assert.NotPanics(t, func() {
		s.AddTask("id-1", testAsset("test1", "linux"))
		s.OnProgress("id-1", 0.5)
		s.Completed("id-1")
		s.Close()
	})
	assert.Equal(t, 1, w.calls, "writing stopped after the first failure")
	assert.Error(t, s.writeErr)
}

type failingWriter struct{ calls int }

func (f *failingWriter) Write(p []byte) (int, error) {
	f.calls++
	return 0, errors.New("destination is gone")
}
