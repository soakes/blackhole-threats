package logging

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var ErrUnknownFormat = errors.New("unknown log format")

type OperatorFormatter struct{}

func NewFormatter(name string) (log.Formatter, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "logfmt":
		return OperatorFormatter{}, nil
	case "json":
		return &log.JSONFormatter{TimestampFormat: time.RFC3339}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownFormat, name)
	}
}

func (f OperatorFormatter) Format(entry *log.Entry) ([]byte, error) {
	var b bytes.Buffer

	timestamp := entry.Time
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	timestamp = timestamp.UTC()

	fmt.Fprintf(&b, "time=%s level=%s msg=%s",
		timestamp.Format(time.RFC3339),
		entry.Level.String(),
		formatValue(entry.Message),
	)

	if len(entry.Data) > 0 {
		keys := make([]string, 0, len(entry.Data))
		for key := range entry.Data {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			fmt.Fprintf(&b, " %s=%s", key, formatValue(entry.Data[key]))
		}
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func formatValue(value any) string {
	text := fmt.Sprint(value)
	if strings.ContainsAny(text, " \t\n\r\"'=") {
		return strconv.Quote(text)
	}

	return text
}
