# Configuration Reference

This guide documents the configuration surface that `blackhole-threats` owns
directly, plus the runtime flags that shape startup behavior.

See also:

- [Operations Guide](operations.md)
- [Feed Behavior](feed-behavior.md)
- [Deployment Examples](deployment-examples.md)

## Schema Overview

The YAML file has two top-level sections:

- `gobgp`
- `feeds`

The decoder runs in strict known-field mode, so unknown YAML keys are rejected
at load time rather than ignored silently.

## Top-Level Keys

### `gobgp`

Type: upstream GoBGP `oc.BgpConfigSet`

This project passes the `gobgp` block into the embedded GoBGP server largely
as-is. The repo does not redefine the entire GoBGP schema in local code, but
it does provide a small per-neighbor port shorthand and enforce two startup
requirements:

- `gobgp.global.config.as` must be non-zero
- `gobgp.global.config.routerid` must be a valid IPv4 address

Minimal shape:

```yaml
gobgp:
  global:
    config:
      as: 64520
      routerid: "198.51.100.10"
  neighbors:
    - config:
        neighboraddress: "198.51.100.1"
        peeras: 64520
```

Operational notes:

- The router ID must be IPv4 even if you also peer over IPv6.
- BGP port settings are optional. If omitted, GoBGP uses standard BGP port
  `179`.
- `gobgp.global.config.port` optionally changes the local BGP listen port.
- `gobgp.neighbors[].config.port` optionally changes the remote TCP port used
  when dialing that peer.
- The neighbor `config.port` shorthand maps to GoBGP
  `transport.config.remoteport`; raw `remoteport` and `remote-port` spellings
  are also accepted under the transport block for this field.

Example with separate local and remote lab ports:

```yaml
gobgp:
  global:
    config:
      as: 64520
      routerid: "198.51.100.10"
      port: 1179
  neighbors:
    - config:
        neighboraddress: "198.51.100.1"
        peeras: 64520
        port: 2179
```

### `feeds`

Type: array of feed objects

Each element supports:

- `url`
- `community`

Example:

```yaml
feeds:
  - url: https://team-cymru.org/Services/Bogons/fullbogons-ipv4.txt
    community: 64520:1101
  - url: /var/lib/blackhole-threats/local-blocklist.txt
    community: 64520:1102
```

## Feed Object

### `url`

Type: string

Required: yes

Supported forms:

- local file path with no URI scheme
- `http://...`
- `https://...`

Rejected:

- empty values
- unsupported schemes such as `ftp://`

Validation is performed both for YAML feeds and for any extra `-feed` arguments
provided on the command line.

### `community`

Type: BGP community written as `<upper>:<lower>`

Required: no

Rules:

- both components must be unsigned 16-bit integers
- values must fit within `0-65535`
- malformed values fail startup validation

If omitted, the daemon assigns the default community:

```text
<local ASN>:666
```

For example, local ASN `64520` produces default community `64520:666`.

## Minimal Working Configuration

```yaml
gobgp:
  global:
    config:
      as: 64520
      routerid: "198.51.100.10"
      port: 1179
  neighbors:
    - config:
        neighboraddress: "198.51.100.1"
        peeras: 64520
        port: 1179

feeds:
  - url: https://team-cymru.org/Services/Bogons/fullbogons-ipv4.txt
```

This example is suitable for validation and one-shot smoke tests. Because the
feed omits `community`, the route set will use the default community
`64520:666`. The neighbor `port` line is optional and should only be set when
the peer listens on a non-standard BGP port.

## Full Reference Example

The repo’s fuller example lives at:

- [`examples/blackhole-threats.yaml`](../examples/blackhole-threats.yaml)

That file includes:

- multiple IPv4 neighbors
- multiple IPv6 neighbors
- multiple feed sources
- explicit per-feed communities

## Command-Line Flags

These flags shape runtime behavior independently from the YAML.

### `-conf`

Path to the configuration file.

Default:

```text
blackhole-threats.yaml
```

### `-check-config`

Validate the configuration and exit without starting GoBGP.

### `-feed`

Append an additional feed URL without editing the YAML.

This flag can be used multiple times:

```bash
./dist/blackhole-threats \
  -conf /etc/blackhole-threats.yaml \
  -feed https://www.spamhaus.org/drop/drop.txt \
  -feed /var/lib/blackhole-threats/local.txt
```

Important limitation:

- `-feed` adds only the URL
- it does not provide a way to attach a custom BGP community
- extra feeds therefore use the default community for the local ASN

### `-once`

Run one refresh cycle and exit.

### `-refresh-rate`

Set the refresh interval.

Default:

```text
2h0m0s
```

The value must be greater than zero.

### `-log-format`

Supported values:

- `logfmt`
- `json`

### `-log-level`

Supported values:

- `panic`
- `fatal`
- `error`
- `warn`
- `info`
- `debug`
- `trace`

### `-debug`

Shortcut for debug-level logging. If set, it overrides `-log-level` and forces
the effective level to `debug`.

### `-version`

Print build metadata and exit.

## Validation Behavior

Startup validation fails for:

- unreadable config files
- malformed YAML
- unknown YAML keys
- local ASN `0`
- invalid or non-IPv4 router IDs
- invalid or conflicting neighbor endpoint ports
- empty feed URLs
- unsupported feed schemes
- malformed communities
- refresh intervals less than or equal to zero

## Recommended Editing Workflow

1. Edit the YAML file.
2. Run `-check-config`.
3. Run `-once` if the change affects feeds, route count, or policy behavior.
4. Restart the daemon to apply the changed YAML.

Use [Operations Guide](operations.md) for the end-to-end operator workflow.
