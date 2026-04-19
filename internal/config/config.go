package config

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"gopkg.in/yaml.v3"
)

var ErrMalformedCommunity = errors.New("malformed community")
var ErrInvalidLocalASN = errors.New("invalid local ASN")
var ErrInvalidRouterID = errors.New("invalid router ID")
var ErrMissingFeedURL = errors.New("missing feed URL")
var ErrUnsupportedFeedScheme = errors.New("unsupported feed scheme")

type Community uint32

func (c Community) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "%d:%d", uint16(c>>16), uint16(c)), nil
}

func (c Community) String() string {
	s, _ := c.MarshalText()
	return string(s)
}

func (c *Community) UnmarshalText(text []byte) error {
	parts := strings.Split(string(text), ":")
	if len(parts) != 2 {
		return ErrMalformedCommunity
	}

	upper, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return fmt.Errorf("%w: upper %s", ErrMalformedCommunity, err)
	}
	lower, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return fmt.Errorf("%w: lower %s", ErrMalformedCommunity, err)
	}

	*c = Community(upper<<16 | lower)
	return nil
}

func (c Community) Uint32() uint32 {
	return uint32(c)
}

type Feed struct {
	URL       string    `yaml:"url"`
	Community Community `yaml:"community"`
}

func (f Feed) String() string {
	if f.Community == 0 {
		return f.URL
	}

	return fmt.Sprintf("%s[%s]", f.URL, f.Community)
}

type File struct {
	GoBGP oc.BgpConfigSet `yaml:"gobgp"`
	Feeds []Feed          `yaml:"feeds"`
}

type FeedList []Feed

var _ flag.Value = (*FeedList)(nil)

func (f *FeedList) String() string {
	values := make([]string, len(*f))
	for i := range len(*f) {
		values[i] = (*f)[i].String()
	}

	return "[" + strings.Join(values, " ") + "]"
}

func (f *FeedList) Set(value string) error {
	*f = append(*f, Feed{URL: value})
	return nil
}

func Load(path string) (File, error) {
	var cfg File

	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read config %q: %w", path, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return File{}, fmt.Errorf("parse config %q: %w", path, err)
	}

	return cfg, nil
}

func (cfg File) Validate() error {
	if cfg.GoBGP.Global.Config.As == 0 {
		return ErrInvalidLocalASN
	}

	routerID, err := netip.ParseAddr(cfg.GoBGP.Global.Config.RouterId)
	if err != nil || !routerID.Is4() {
		return fmt.Errorf("%w: %q", ErrInvalidRouterID, cfg.GoBGP.Global.Config.RouterId)
	}

	for i, feed := range cfg.Feeds {
		if err := ValidateFeed(feed); err != nil {
			return fmt.Errorf("feed %d: %w", i+1, err)
		}
	}

	return nil
}

func ValidateFeed(feed Feed) error {
	source := strings.TrimSpace(feed.URL)
	if source == "" {
		return ErrMissingFeedURL
	}

	parsed, err := url.Parse(source)
	if err != nil {
		return fmt.Errorf("parse feed %q: %w", source, err)
	}

	switch parsed.Scheme {
	case "", "http", "https":
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedFeedScheme, parsed.Scheme)
	}
}

func ValidateFeeds(feeds []Feed) error {
	for i, feed := range feeds {
		if err := ValidateFeed(feed); err != nil {
			return fmt.Errorf("feed %d: %w", i+1, err)
		}
	}

	return nil
}

func DefaultCommunity(asn uint32) Community {
	return Community(asn<<16 | 666)
}
