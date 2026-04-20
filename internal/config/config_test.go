package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/osrg/gobgp/v3/pkg/config/oc"
)

func TestCommunityRoundTrip(t *testing.T) {
	t.Parallel()

	var community Community
	if err := community.UnmarshalText([]byte("64512:666")); err != nil {
		t.Fatalf("UnmarshalText() error = %v", err)
	}

	if got := community.String(); got != "64512:666" {
		t.Fatalf("String() = %q, want %q", got, "64512:666")
	}
}

func TestCommunityRejectsMalformedValue(t *testing.T) {
	t.Parallel()

	var community Community
	err := community.UnmarshalText([]byte("64512:not-a-number"))
	if !errors.Is(err, ErrMalformedCommunity) {
		t.Fatalf("UnmarshalText() error = %v, want ErrMalformedCommunity", err)
	}
}

func TestFileValidateAcceptsMinimalConfig(t *testing.T) {
	t.Parallel()

	cfg := File{
		GoBGP: oc.BgpConfigSet{
			Global: oc.Global{
				Config: oc.GlobalConfig{
					As:       64512,
					RouterId: "192.0.2.1",
				},
			},
		},
		Feeds: []Feed{
			{URL: "https://example.com/feed.txt"},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestFileValidateRejectsInvalidRouterID(t *testing.T) {
	t.Parallel()

	cfg := File{
		GoBGP: oc.BgpConfigSet{
			Global: oc.Global{
				Config: oc.GlobalConfig{
					As:       64512,
					RouterId: "not-an-ip",
				},
			},
		},
	}

	err := cfg.Validate()
	if !errors.Is(err, ErrInvalidRouterID) {
		t.Fatalf("Validate() error = %v, want ErrInvalidRouterID", err)
	}
}

func TestValidateFeedsRejectsMissingFeedURL(t *testing.T) {
	t.Parallel()

	err := ValidateFeeds([]Feed{{URL: ""}})
	if !errors.Is(err, ErrMissingFeedURL) {
		t.Fatalf("ValidateFeeds() error = %v, want ErrMissingFeedURL", err)
	}
}

func TestValidateFeedsRejectsUnsupportedFeedScheme(t *testing.T) {
	t.Parallel()

	err := ValidateFeeds([]Feed{{URL: "ftp://example.com/feed.txt"}})
	if !errors.Is(err, ErrUnsupportedFeedScheme) {
		t.Fatalf("ValidateFeeds() error = %v, want ErrUnsupportedFeedScheme", err)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "blackhole-threats.yaml")
	content := []byte(`
gobgp:
  global:
    config:
      as: 64512
      routerid: "192.0.2.1"
unexpected: true
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want unknown-key parse failure")
	}
}
