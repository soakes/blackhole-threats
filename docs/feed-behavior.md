# Feed Behavior

This guide explains how `blackhole-threats` reads feeds, normalises their
contents, groups them by BGP community, and preserves last known-good routes
when a source fails.

See also:

- [Configuration Reference](config-reference.md)
- [Architecture](architecture.md)
- [Troubleshooting](troubleshooting.md)

## Retrieval Model

The runtime groups feeds by BGP community first, then reads all sources mapped
to the same community together.

That grouping defines the failure boundary:

- if every source in a community refresh succeeds, the daemon builds a fresh
  route set for that community
- if any source in that community refresh fails, the daemon carries forward the
  previously advertised routes for that community for the current cycle

This is why community design matters operationally. Do not group unrelated
feeds together unless they should share the same failure fate.

## Supported Source Types

The feed reader accepts:

- local file paths with no URI scheme
- `http://` URLs
- `https://` URLs

The default HTTP client timeout is `30s`.

Unsupported schemes, such as `ftp://`, are rejected during configuration
validation.

## Format Detection

The reader decides whether a source is JSON-based by checking:

- file extension
- HTTP `Content-Type`

JSON is detected for extensions:

- `.json`
- `.jsonl`
- `.ndjson`

JSON is also detected for media types such as:

- `application/json`
- `application/jsonl`
- `application/ndjson`
- `application/x-ndjson`
- `text/json`
- any content type ending in `+json`

If a source does not match those checks, it is parsed as plain text.

## Text Feed Parsing

Text parsing is intentionally forgiving.

The parser:

- ignores empty lines
- ignores lines starting with `#`, `;`, or `//`
- splits mixed-content lines on spaces, tabs, commas, and semicolons
- strips surrounding punctuation such as quotes, brackets, and angle brackets
- accepts either CIDR prefixes or individual IP addresses

Examples that work:

```text
192.0.2.0/24
198.51.100.10
prefix=203.0.113.0/24
"2001:db8:10::/48"
```

Individual IP addresses are converted into host prefixes:

- IPv4 host addresses become `/32`
- IPv6 host addresses become `/128`

## JSON Feed Parsing

JSON feeds support two layouts:

- top-level arrays
- JSON streams such as JSONL and NDJSON

Each JSON object is checked for the first usable value in these fields:

- `cidr`
- `prefix`
- `ip`
- `address`

Examples:

```json
{"cidr":"203.0.113.0/24"}
{"ip":"198.51.100.10"}
{"address":"2001:db8:20::1"}
```

As with text feeds, individual IP values are converted into host prefixes.

## Prefix Normalisation

Before routes are compared or published, the parser normalises input by:

- masking prefixes to their canonical network boundary
- discarding invalid values
- converting host IPs into host prefixes

This keeps route comparisons stable and avoids churn caused by differently
formatted but equivalent inputs.

## Prefix Summarisation

After all successful feed inputs are parsed, the reader summarises the prefix
set before passing it to the BGP layer.

The summariser:

1. masks every prefix
2. sorts the prefix list
3. removes duplicates
4. removes prefixes already contained inside a broader prefix
5. merges sibling prefixes into their parent where possible
6. repeats until the prefix set stops changing

Operational effect:

- less route churn
- fewer redundant advertisements
- more stable route-table diffs between refreshes

## Community Grouping and Failure Semantics

The daemon treats each community as a refresh unit.

If a community refresh fails:

- it logs `Keeping previous community routes after feed errors`
- it keeps the prior routes for that community
- it continues processing other communities
- it retries on the next timer tick or `SIGUSR1`

This behavior is conservative by design. It avoids withdrawing good routes just
because one source was temporarily unavailable or malformed.

## Choosing Communities Well

Use separate communities when:

- feeds come from unrelated providers
- feeds have different trust levels
- feeds should fail independently
- routers need different downstream policy handling

Use the same community when:

- the feeds represent one policy bucket
- shared carry-forward behavior is acceptable
- the downstream router policy should treat the sources identically

## Logs to Watch

Useful feed lifecycle logs include:

- `Parsed threat feed`
- `Prepared community routes`
- `Failed to read threat feed`
- `Keeping previous community routes after feed errors`
- `Refresh completed`

The most useful fields are:

- `source`
- `community`
- `feed_count`
- `failed_feed_count`
- `failed_feeds`
- `prefixes_read`
- `prefixes_summarized`
- `carried_forward`

## Practical Advice

- Prefer stable URLs and predictable content types.
- Keep experimental feeds in their own community.
- Expect route counts to drop after summarisation when sibling or contained
  prefixes collapse.
- Treat persistent carry-forward logs as an input-quality incident, not as a
  harmless warning to ignore indefinitely.
