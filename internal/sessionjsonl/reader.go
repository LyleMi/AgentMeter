package sessionjsonl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
)

type parsedRawRecord struct {
	lineNo      int
	line        string
	raw         rawRecord
	ts          time.Time
	payloadType string
	rawType     string
}

type rawRecordReader struct {
	path     string
	scanner  *bufio.Scanner
	lineNo   int
	record   parsedRawRecord
	warnings []string
}

func newRawRecordReader(path string, reader io.Reader) *rawRecordReader {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
	return &rawRecordReader{path: path, scanner: scanner}
}

func (r *rawRecordReader) Next() bool {
	for r.scanner.Scan() {
		r.lineNo++
		line := strings.TrimSpace(r.scanner.Text())
		if line == "" {
			continue
		}
		var raw rawRecord
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			r.warn(fmt.Sprintf("%s:%d malformed JSON: %v", filepath.Base(r.path), r.lineNo, err))
			continue
		}
		ts := recordTimestamp(raw)
		if ts.IsZero() {
			ts = timestampFromPayload(raw.Payload, "started_at")
		}
		if ts.IsZero() {
			ts = timestampFromPayload(raw.Payload, "completed_at")
		}
		if ts.IsZero() {
			r.warn(fmt.Sprintf("%s:%d missing timestamp", filepath.Base(r.path), r.lineNo))
			continue
		}
		payloadType := stringValue(raw.Payload, "type")
		rawType := payloadType
		if rawType == "" {
			rawType = raw.Type
		}
		r.record = parsedRawRecord{
			lineNo:      r.lineNo,
			line:        line,
			raw:         raw,
			ts:          ts,
			payloadType: payloadType,
			rawType:     rawType,
		}
		return true
	}
	return false
}

func (r *rawRecordReader) Record() parsedRawRecord {
	return r.record
}

func (r *rawRecordReader) Err() error {
	return r.scanner.Err()
}

func (r *rawRecordReader) Warnings() []string {
	return r.warnings
}

func (r *rawRecordReader) warn(message string) {
	r.warnings = append(r.warnings, message)
}
