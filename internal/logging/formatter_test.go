package logging

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestOperatorFormatterFormatsFieldsLikeOperatorOutput(t *testing.T) {
	formatter := OperatorFormatter{}

	entry := &log.Entry{
		Time:    time.Date(2026, time.April, 19, 20, 30, 45, 0, time.UTC),
		Level:   log.InfoLevel,
		Message: "Refresh complete",
		Data: log.Fields{
			"withdrawn": 0,
			"announced": 2,
			"note":      "feed retry",
		},
	}

	got, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	want := "time=2026-04-19T20:30:45Z level=info msg=\"Refresh complete\" announced=2 note=\"feed retry\" withdrawn=0\n"
	if string(got) != want {
		t.Fatalf("unexpected formatter output:\nwant: %q\ngot:  %q", want, string(got))
	}
}

func TestNewFormatterJSON(t *testing.T) {
	formatter, err := NewFormatter("json")
	if err != nil {
		t.Fatalf("NewFormatter returned error: %v", err)
	}

	entry := &log.Entry{
		Time:    time.Date(2026, time.April, 20, 0, 0, 28, 0, time.UTC),
		Level:   log.InfoLevel,
		Message: "Starting blackhole-threats",
		Data: log.Fields{
			"tag_version": "v1.0.0",
		},
	}

	got, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	if payload["msg"] != "Starting blackhole-threats" {
		t.Fatalf("msg = %v, want %q", payload["msg"], "Starting blackhole-threats")
	}
	if payload["level"] != "info" {
		t.Fatalf("level = %v, want %q", payload["level"], "info")
	}
	if payload["tag_version"] != "v1.0.0" {
		t.Fatalf("tag_version = %v, want %q", payload["tag_version"], "v1.0.0")
	}
}

func TestNewFormatterRejectsUnknownFormat(t *testing.T) {
	_, err := NewFormatter("yaml")
	if !errors.Is(err, ErrUnknownFormat) {
		t.Fatalf("NewFormatter error = %v, want ErrUnknownFormat", err)
	}
}
