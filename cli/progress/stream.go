// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package progress

import (
	"encoding/json"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// StreamEnvVar names the destination for the machine-readable scan progress
// stream. It is read by the scanner when it builds its progress reporter.
//
// A parent process that runs `cnspec scan` as a child sets it to either:
//
//	/path/to/file    an absolute or relative path. A named pipe (FIFO) works
//	                 and is usually what a live consumer wants; a regular file
//	                 is truncated on open.
//	fd:3             a file descriptor the parent passed down, e.g. via
//	                 exec.Cmd.ExtraFiles (the first extra file is fd 3).
//
// It is deliberately an environment variable rather than a flag: the whole
// point is that a supervising process can turn the stream on for a child it
// spawns without having to reach into how that child's command line is built,
// and env is what a child already inherits. It also keeps the stream orthogonal
// to `-o`/`--output-target`, which describe the *report*, not progress.
const StreamEnvVar = "MONDOO_PROGRESS_STREAM"

// StreamVersion is the schema version reported by the scan_start event.
// Bump it when the meaning of an existing field changes; adding a new field
// or a new event type is backwards compatible and does not need a bump.
const StreamVersion = 1

// Event names emitted on the stream.
const (
	EventScanStart     = "scan_start"
	EventDiscovered    = "discovered"
	EventFiltered      = "filtered"
	EventAssetAdded    = "asset_added"
	EventAssetProgress = "asset_progress"
	EventAssetScore    = "asset_score"
	EventAssetDone     = "asset_done"
	EventScanDone      = "scan_done"
)

// Terminal states reported by an asset_done event.
const (
	StateCompleted     = "completed"
	StateErrored       = "errored"
	StateNotApplicable = "not_applicable"
)

// StreamEvent is one line of the NDJSON progress stream.
//
// Every line is a complete JSON object followed by a single newline, so a
// consumer can read it with a line scanner as the scan runs. Fields are
// omitted when empty; a consumer must read an absent field as the zero value
// (0 for numbers, "" for strings) rather than as "unknown".
//
//	scan_start      version
//	discovered      count, discovered   (count = this batch, discovered = running total)
//	filtered        count, filtered
//	asset_added     index, name, platform, num
//	asset_progress  index, num, percent (0..1, emitted when the whole percent changes)
//	asset_score     index, num, score
//	asset_done      index, num, state, score, done, total, duration_ms
//	scan_done       total, done, completed, errored, not_applicable,
//	                discovered, filtered, duration_ms
//
// `index` is the asset's primary platform ID — the same key the scanner uses
// internally — and `num` is its 1-based position in the order assets were
// announced. `total` is the number of assets known at the time of the event;
// assets are discovered while the scan runs, so it can grow.
type StreamEvent struct {
	Seq   uint64 `json:"seq"`
	Time  string `json:"time"`
	Event string `json:"event"`

	Version int `json:"version,omitempty"`

	Index    string  `json:"index,omitempty"`
	Name     string  `json:"name,omitempty"`
	Platform string  `json:"platform,omitempty"`
	Num      int     `json:"num,omitempty"`
	Percent  float64 `json:"percent,omitempty"`
	Score    string  `json:"score,omitempty"`
	State    string  `json:"state,omitempty"`

	Count         int `json:"count,omitempty"`
	Total         int `json:"total,omitempty"`
	Done          int `json:"done,omitempty"`
	Completed     int `json:"completed,omitempty"`
	Errored       int `json:"errored,omitempty"`
	NotApplicable int `json:"not_applicable,omitempty"`
	Discovered    int `json:"discovered,omitempty"`
	Filtered      int `json:"filtered,omitempty"`

	DurationMs int64 `json:"duration_ms,omitempty"`
}

type streamTask struct {
	num         int
	startedAt   time.Time
	lastPercent int // whole percent last emitted, -1 before the first progress event
	done        bool
}

// streamProgress is a MultiProgress that writes NDJSON events instead of
// drawing a terminal UI. It is safe for concurrent use: the scanner runs a
// pipeline with configurable parallelism, so every method can be called from
// a different goroutine at the same time.
type streamProgress struct {
	mu     sync.Mutex
	out    io.Writer
	closer io.Closer
	now    func() time.Time

	seq       uint64
	startedAt time.Time
	order     []string
	tasks     map[string]*streamTask

	discovered    int
	filtered      int
	completed     int
	errored       int
	notApplicable int

	closed   bool
	writeErr error
}

// NewStream opens the progress stream target described by `target` (see
// StreamEnvVar for the accepted forms) and returns a MultiProgress that emits
// NDJSON to it.
func NewStream(target string) (*streamProgress, error) {
	w, err := openStreamTarget(target)
	if err != nil {
		return nil, err
	}
	s := NewStreamWriter(w)
	s.closer = w
	return s, nil
}

// NewStreamWriter returns an NDJSON progress reporter writing to w. The writer
// is not closed by Close; use NewStream when the stream owns its destination.
func NewStreamWriter(w io.Writer) *streamProgress {
	s := &streamProgress{
		out:   w,
		now:   time.Now,
		tasks: map[string]*streamTask{},
	}
	s.startedAt = s.now()
	// The scan_start event is written here rather than in Open because the
	// scanner calls Open from its own goroutine, which would race with the
	// first AddTask and could put scan_start in the middle of the stream.
	s.emit(StreamEvent{Event: EventScanStart, Version: StreamVersion})
	return s
}

func openStreamTarget(target string) (*os.File, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, errors.New("empty progress stream target")
	}

	if rest, ok := strings.CutPrefix(target, "fd:"); ok {
		fd, err := strconv.Atoi(rest)
		if err != nil || fd < 0 {
			return nil, errors.Newf("invalid progress stream file descriptor %q", target)
		}
		return os.NewFile(uintptr(fd), target), nil
	}

	// Not buffered on purpose: a consumer reads this while the scan runs, so
	// every event has to reach the destination as it happens.
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open progress stream %q", target)
	}
	return f, nil
}

// emit serializes and writes one event. The whole line is handed to a single
// Write call while the lock is held, so events never interleave and a line is
// never written in halves.
func (s *streamProgress) emit(e StreamEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emitLocked(e)
}

func (s *streamProgress) emitLocked(e StreamEvent) {
	if s.closed || s.out == nil || s.writeErr != nil {
		return
	}

	s.seq++
	e.Seq = s.seq
	e.Time = s.now().UTC().Format(time.RFC3339Nano)

	line, err := json.Marshal(&e)
	if err != nil {
		s.fail(errors.Wrap(err, "failed to encode progress event"))
		return
	}
	line = append(line, '\n')

	if _, err := s.out.Write(line); err != nil {
		s.fail(errors.Wrap(err, "failed to write progress event"))
	}
}

// fail records the first write failure and stops emitting. A broken progress
// stream — the consumer went away, the disk filled up — must never take the
// scan down with it.
func (s *streamProgress) fail(err error) {
	s.writeErr = err
	log.Warn().Err(err).Msg("progress stream disabled")
}

// taskLocked returns the tracked task for index, announcing it first if the
// scanner reports on an asset it never added. Every index that appears on the
// stream has an asset_added event before it.
func (s *streamProgress) taskLocked(index string, asset *inventory.Asset) *streamTask {
	if t, ok := s.tasks[index]; ok {
		return t
	}

	s.order = append(s.order, index)
	t := &streamTask{num: len(s.order), lastPercent: -1}
	s.tasks[index] = t

	added := StreamEvent{Event: EventAssetAdded, Index: index, Num: t.num}
	if asset != nil {
		added.Name = asset.Name
		if asset.Platform != nil {
			added.Platform = asset.Platform.Name
		}
	}
	s.emitLocked(added)
	return t
}

func (s *streamProgress) Open() error {
	// Nothing to do: the destination is opened by NewStream and scan_start is
	// already on the wire. The scanner runs Open in a goroutine and waits for
	// it after Close, so returning immediately is correct here.
	return nil
}

func (s *streamProgress) Discovered(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discovered += count
	s.emitLocked(StreamEvent{Event: EventDiscovered, Count: count, Discovered: s.discovered})
}

func (s *streamProgress) Filtered(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filtered += count
	s.emitLocked(StreamEvent{Event: EventFiltered, Count: count, Filtered: s.filtered})
}

func (s *streamProgress) AddTask(index string, asset *inventory.Asset) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.taskLocked(index, asset)
}

func (s *streamProgress) OnProgress(index string, percent float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	percent = min(max(percent, 0), 1)
	t := s.taskLocked(index, nil)
	if t.startedAt.IsZero() {
		t.startedAt = s.now()
	}

	// The executor reports progress per query, which on a large policy is
	// thousands of calls per asset. One event per whole percent keeps the
	// stream readable without losing anything a display could show.
	whole := int(percent * 100)
	if whole == t.lastPercent {
		return
	}
	t.lastPercent = whole

	// Rounded: the executor's raw ratio carries 17 digits of precision that no
	// display can use and that only makes the line harder to read.
	rounded := math.Round(percent*10000) / 10000
	s.emitLocked(StreamEvent{Event: EventAssetProgress, Index: index, Num: t.num, Percent: rounded})
}

func (s *streamProgress) Score(index string, score string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.taskLocked(index, nil)
	s.emitLocked(StreamEvent{Event: EventAssetScore, Index: index, Num: t.num, Score: score})
}

func (s *streamProgress) Errored(index string) { s.finish(index, StateErrored) }

func (s *streamProgress) NotApplicable(index string) { s.finish(index, StateNotApplicable) }

func (s *streamProgress) Completed(index string) { s.finish(index, StateCompleted) }

func (s *streamProgress) finish(index string, state string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.taskLocked(index, nil)
	if t.done {
		return
	}
	t.done = true

	switch state {
	case StateErrored:
		s.errored++
	case StateNotApplicable:
		s.notApplicable++
	default:
		s.completed++
	}

	e := StreamEvent{
		Event: EventAssetDone,
		Index: index,
		Num:   t.num,
		State: state,
		Done:  s.completed + s.errored + s.notApplicable,
		Total: len(s.order),
	}
	if !t.startedAt.IsZero() {
		e.DurationMs = s.now().Sub(t.startedAt).Milliseconds()
	}
	s.emitLocked(e)
}

// Close writes the summary event and releases the destination. The scanner
// calls it both explicitly and from a defer, so it has to be idempotent.
func (s *streamProgress) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	s.emitLocked(StreamEvent{
		Event:         EventScanDone,
		Total:         len(s.order),
		Done:          s.completed + s.errored + s.notApplicable,
		Completed:     s.completed,
		Errored:       s.errored,
		NotApplicable: s.notApplicable,
		Discovered:    s.discovered,
		Filtered:      s.filtered,
		DurationMs:    s.now().Sub(s.startedAt).Milliseconds(),
	})

	s.closed = true
	if s.closer != nil {
		if err := s.closer.Close(); err != nil {
			log.Warn().Err(err).Msg("failed to close progress stream")
		}
	}
}
