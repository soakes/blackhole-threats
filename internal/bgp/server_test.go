package bgp

import (
	"context"
	"net/netip"
	"slices"
	"testing"

	"github.com/soakes/blackhole-threats/internal/config"
	"github.com/soakes/blackhole-threats/internal/feed"
)

type stubFeedReader struct {
	results map[string]feed.ReadResult
}

func (s stubFeedReader) ReadMany(_ context.Context, sources ...string) feed.ReadResult {
	if len(sources) != 1 {
		return feed.ReadResult{}
	}
	return s.results[sources[0]]
}

func TestBuildRouteTableKeepsPreviousCommunityOnFeedFailure(t *testing.T) {
	t.Parallel()

	defaultCommunity := config.DefaultCommunity(64512)
	otherCommunity := config.Community(64512<<16 | 777)

	server := &Server{
		asn: 64512,
		feeds: stubFeedReader{
			results: map[string]feed.ReadResult{
				"https://feeds.example.net/default.txt": {
					FailedSources: []string{"https://feeds.example.net/default.txt"},
				},
				"https://feeds.example.net/other.txt": {
					Prefixes: []netip.Prefix{
						netip.MustParsePrefix("203.0.113.0/24"),
					},
					Count: 1,
				},
			},
		},
	}

	previous := map[string]routeState{
		"198.51.100.0/24": {
			prefix:      netip.MustParsePrefix("198.51.100.0/24"),
			communities: []config.Community{defaultCommunity},
		},
	}

	next := server.buildRouteTable(context.Background(), []config.Feed{
		{URL: "https://feeds.example.net/default.txt"},
		{URL: "https://feeds.example.net/other.txt", Community: otherCommunity},
	}, previous)

	if len(next) != 2 {
		t.Fatalf("buildRouteTable() len = %d, want %d", len(next), 2)
	}

	carried, ok := next["198.51.100.0/24"]
	if !ok {
		t.Fatal("buildRouteTable() missing carried-forward prefix")
	}
	if !slices.Equal(carried.communities, []config.Community{defaultCommunity}) {
		t.Fatalf("carried communities = %v, want %v", carried.communities, []config.Community{defaultCommunity})
	}

	fresh, ok := next["203.0.113.0/24"]
	if !ok {
		t.Fatal("buildRouteTable() missing refreshed prefix")
	}
	if !slices.Equal(fresh.communities, []config.Community{otherCommunity}) {
		t.Fatalf("fresh communities = %v, want %v", fresh.communities, []config.Community{otherCommunity})
	}
}

func TestRunOnceReturnsAfterInitialRefresh(t *testing.T) {
	t.Parallel()

	server := &Server{
		asn:      64512,
		routerID: "192.0.2.1",
		feeds: stubFeedReader{
			results: map[string]feed.ReadResult{},
		},
	}

	if err := server.Run(context.Background(), nil, 2, nil, true); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}
