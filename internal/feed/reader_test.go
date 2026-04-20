package feed

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleJSONLFeed = `{"cidr":"203.0.113.0/24","rir":"TEST","sblid":"SBL1"}
{"cidr":"2001:db8::/32","rir":"TEST","sblid":"SBL2"}
{"generated_at":"2026-04-19T00:00:00Z"}`

const sampleJSONArrayFeed = `[
  {"cidr":"203.0.113.0/24","rir":"TEST","sblid":"SBL1"},
  {"cidr":"2001:db8::/32","rir":"TEST","sblid":"SBL2"},
  {"generated_at":"2026-04-19T00:00:00Z"}
]`

func TestReadFromJSON(t *testing.T) {
	t.Parallel()

	prefixes, count, err := readFromJSON(strings.NewReader(sampleJSONLFeed))
	if err != nil {
		t.Fatalf("readFromJSON() error = %v", err)
	}

	if count != 2 {
		t.Fatalf("readFromJSON() count = %d, want %d", count, 2)
	}

	want := []string{"203.0.113.0/24", "2001:db8::/32"}
	if got := prefixesToStrings(prefixes); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("readFromJSON() nets = %v, want %v", got, want)
	}
}

func TestReaderReadsJSONLFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	feedFile := filepath.Join(dir, "feed.jsonl")
	if err := os.WriteFile(feedFile, []byte(sampleJSONLFeed), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	reader := Reader{}
	_, count, err := reader.readOne(context.Background(), feedFile)
	if err != nil {
		t.Fatalf("readOne() error = %v", err)
	}

	if count != 2 {
		t.Fatalf("readOne() count = %d, want %d", count, 2)
	}
}

func TestReaderReadsNDJSONOverHTTP(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = io.WriteString(w, sampleJSONLFeed)
	}))
	defer srv.Close()

	reader := Reader{Client: srv.Client()}
	_, count, err := reader.readOne(context.Background(), srv.URL+"/feed.txt")
	if err != nil {
		t.Fatalf("readOne() error = %v", err)
	}

	if count != 2 {
		t.Fatalf("readOne() count = %d, want %d", count, 2)
	}
}

func TestReadFromJSONArray(t *testing.T) {
	t.Parallel()

	prefixes, count, err := readFromJSON(strings.NewReader(sampleJSONArrayFeed))
	if err != nil {
		t.Fatalf("readFromJSON() error = %v", err)
	}

	if count != 2 {
		t.Fatalf("readFromJSON() count = %d, want %d", count, 2)
	}

	want := []string{"203.0.113.0/24", "2001:db8::/32"}
	if got := prefixesToStrings(prefixes); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("readFromJSON() nets = %v, want %v", got, want)
	}
}

func TestReaderReadManyTracksFailedSources(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	goodFeed := filepath.Join(dir, "good.txt")
	if err := os.WriteFile(goodFeed, []byte("198.51.100.0/24\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := Reader{}.ReadMany(context.Background(), goodFeed, filepath.Join(dir, "missing.txt"))

	if result.Count != 1 {
		t.Fatalf("ReadMany() count = %d, want %d", result.Count, 1)
	}
	if len(result.Prefixes) != 1 || result.Prefixes[0].String() != "198.51.100.0/24" {
		t.Fatalf("ReadMany() prefixes = %v, want [198.51.100.0/24]", prefixesToStrings(result.Prefixes))
	}
	if len(result.FailedSources) != 1 {
		t.Fatalf("ReadMany() failed sources = %v, want 1 failure", result.FailedSources)
	}
}

func TestReadFromTextAcceptsIPsAndCIDRs(t *testing.T) {
	t.Parallel()

	text := strings.NewReader("203.0.113.10\n# comment\n198.51.100.0/24\n")
	prefixes, count, err := readFromText(text)
	if err != nil {
		t.Fatalf("readFromText() error = %v", err)
	}

	if count != 2 {
		t.Fatalf("readFromText() count = %d, want %d", count, 2)
	}

	want := []string{"203.0.113.10/32", "198.51.100.0/24"}
	if got := prefixesToStrings(prefixes); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("readFromText() prefixes = %v, want %v", got, want)
	}
}

func TestSummarizePrefixesMergesSiblings(t *testing.T) {
	t.Parallel()

	prefixes := []netip.Prefix{
		netip.MustParsePrefix("198.51.100.0/25"),
		netip.MustParsePrefix("198.51.100.128/25"),
		netip.MustParsePrefix("2001:db8::/33"),
		netip.MustParsePrefix("2001:db8:8000::/33"),
	}

	want := []string{"198.51.100.0/24", "2001:db8::/32"}
	if got := prefixesToStrings(summarizePrefixes(prefixes)); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("summarizePrefixes() = %v, want %v", got, want)
	}
}

func prefixesToStrings(prefixes []netip.Prefix) []string {
	values := make([]string, len(prefixes))
	for i, prefix := range prefixes {
		values[i] = prefix.String()
	}
	return values
}
