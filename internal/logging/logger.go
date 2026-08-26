// Package logging provides a shared logger that writes structured lines
// to both an in-memory ring buffer (consumed by the log viewer window)
// and a log file on disk.
//
// The ring buffer keeps the most recent lines so the UI can display them
// without re-reading the file. The file logger appends every line for
// persistent troubleshooting.
package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"subrelay/internal/paths"
)

// DefaultRingSize is the default number of lines retained in memory.
const DefaultRingSize = 1000

// Line is a single log entry captured by the ring buffer.
type Line struct {
	Time   time.Time
	Level  string
	Text   string
}

// Logger is the shared application logger. It is safe for concurrent use.
type Logger struct {
	mu       sync.Mutex
	ring     []Line
	ringSize int
	head     int
	count    int
	file     *os.File
	subscribers map[chan Line]struct{}
}

// global is the process-wide logger instance.
var (
	global     *Logger
	globalOnce sync.Once
)

// Global returns the process-wide logger, creating it on first call.
//
// Returns:
//   - The singleton Logger instance.
//
// Errors:
//   - Panics when the log file cannot be opened, which is a fatal
//     startup condition.
func Global() *Logger {
	globalOnce.Do(func() {
		l, err := New(DefaultRingSize)
		if err != nil {
			panic(fmt.Sprintf("logging: open log file: %v", err))
		}
		global = l
	})
	return global
}

// New creates a Logger with the given ring buffer size and opens the log
// file for appending.
//
// Args:
//   - ringSize: maximum number of lines retained in the ring buffer.
//
// Returns:
//   - A pointer to the new Logger.
//
// Errors:
//   - Returns an error wrapping os.OpenFile when the log file cannot be
//     opened.
func New(ringSize int) (*Logger, error) {
	if ringSize <= 0 {
		ringSize = DefaultRingSize
	}
	path, err := paths.LogFile()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("logging: open file: %w", err)
	}
	return &Logger{
		ring:        make([]Line, ringSize),
		ringSize:    ringSize,
		file:        f,
		subscribers: make(map[chan Line]struct{}),
	}, nil
}

// Close releases the log file handle.
//
// Errors:
//   - Returns an error wrapping the file Close call.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

// Log appends a line at the given level with the formatted text. The
// line is written to the file, added to the ring buffer, and forwarded
// to all subscribers.
func (l *Logger) Log(level, format string, args ...any) {
	line := Line{
		Time:  time.Now(),
		Level: level,
		Text:  fmt.Sprintf(format, args...),
	}
	formatted := fmt.Sprintf("%s [%s] %s\n",
		line.Time.Format("2006-01-02 15:04:05"),
		level, line.Text)

	l.mu.Lock()
	l.appendRing(line)
	if l.file != nil {
		_, _ = l.file.WriteString(formatted)
	}
	subs := make([]chan Line, 0, len(l.subscribers))
	for ch := range l.subscribers {
		subs = append(subs, ch)
	}
	l.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- line:
		default:
			// Drop when the subscriber is slow to avoid blocking the logger.
		}
	}
}

// Info logs at INFO level.
func (l *Logger) Info(format string, args ...any) {
	l.Log("INFO", format, args...)
}

// Warn logs at WARN level.
func (l *Logger) Warn(format string, args ...any) {
	l.Log("WARN", format, args...)
}

// Error logs at ERROR level.
func (l *Logger) Error(format string, args ...any) {
	l.Log("ERROR", format, args...)
}

// appendRing adds a line to the ring buffer, overwriting the oldest entry
// when full. Caller must hold l.mu.
func (l *Logger) appendRing(line Line) {
	l.ring[l.head] = line
	l.head = (l.head + 1) % l.ringSize
	if l.count < l.ringSize {
		l.count++
	}
}

// Clear empties the in-memory ring buffer. The on-disk log file is left
// untouched so history remains available for troubleshooting.
func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.head = 0
	l.count = 0
}

// Lines returns a snapshot of the ring buffer contents in chronological
// order (oldest first).
func (l *Logger) Lines() []Line {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Line, 0, l.count)
	start := 0
	if l.count == l.ringSize {
		start = l.head
	}
	for i := 0; i < l.count; i++ {
		idx := (start + i) % l.ringSize
		out = append(out, l.ring[idx])
	}
	return out
}

// Subscribe registers a channel that receives every new log line. The
// caller must call Unsubscribe to stop delivery and release the channel.
func (l *Logger) Subscribe(ch chan Line) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.subscribers[ch] = struct{}{}
}

// Unsubscribe removes a previously registered channel.
func (l *Logger) Unsubscribe(ch chan Line) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.subscribers, ch)
}

// Write implements io.Writer so the logger can capture standard library
// log output. Each write is treated as an INFO line.
func (l *Logger) Write(p []byte) (int, error) {
	text := string(p)
	// Trim trailing newlines added by the standard log package.
	for len(text) > 0 && (text[len(text)-1] == '\n' || text[len(text)-1] == '\r') {
		text = text[:len(text)-1]
	}
	l.Info("%s", text)
	return len(p), nil
}

// Writer returns the logger as an io.Writer for use with standard
// libraries that accept a writer.
func (l *Logger) Writer() io.Writer { return l }
