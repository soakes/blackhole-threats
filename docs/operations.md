# Operations Guide

This guide covers the operator workflow for validating, starting, refreshing,
and maintaining `blackhole-threats` in production.

See also:

- [Configuration Reference](config-reference.md)
- [Deployment Examples](deployment-examples.md)
- [Troubleshooting](troubleshooting.md)

## At a Glance

The service follows a conservative runtime pattern:

1. Load and validate YAML.
2. Start the embedded GoBGP server.
3. Fetch and parse the configured feeds.
4. Build the desired route table.
5. Withdraw stale routes and announce fresh ones.
6. Repeat on the refresh timer, or immediately when sent `SIGUSR1`.

Two operational details matter most:

- A manual refresh re-reads feeds, but it does not reload the YAML
  configuration.
- If a feed group fails, the daemon keeps the last known-good routes for that
  BGP community until a later refresh succeeds.

## First Deployment Checklist

1. Copy [`examples/blackhole-threats.yaml`](../examples/blackhole-threats.yaml)
   into your deployment path.
2. Replace the local ASN, router ID, neighbors, and feed communities with real
   values for your network.
3. Validate the file before starting BGP:

   ```bash
   ./dist/blackhole-threats -conf /path/to/blackhole-threats.yaml -check-config
   ```

4. Run a one-shot smoke test with an unprivileged local BGP port such as
   `1179`. Port settings are optional; set `gobgp.global.config.port` for the
   local listener and only set `gobgp.neighbors[].config.port` for peer
   endpoints that are not using `179`.
5. Confirm the startup logs show the expected `local_as`, `router_id`,
   `peer_count`, and `default_community`.
6. Only then bind to production port `179` and enable long-running service
   management.

## Validation Modes

### Validation-only

Use `-check-config` when you want to validate the config file and exit before
starting GoBGP:

```bash
./dist/blackhole-threats -conf /path/to/blackhole-threats.yaml -check-config
```

This is the safest first command after editing YAML.

### One-shot refresh

Use `-once` when you want a full route-table build and reconciliation cycle
without leaving the daemon running:

```bash
./dist/blackhole-threats -conf /path/to/blackhole-threats.yaml -once
```

This is the best smoke test for:

- parser changes
- feed reachability changes
- policy changes that affect route count
- staged configuration changes before enabling automatic restarts

## Starting the Service

### Source build

```bash
make build
./dist/blackhole-threats -conf /path/to/blackhole-threats.yaml
```

### Debian package

```bash
sudo systemctl enable --now blackhole-threats
sudo journalctl -u blackhole-threats -f
```

The package uses:

- `/etc/blackhole-threats.yaml` for the default config path
- `/etc/default/blackhole-threats` for additional runtime options
- `ExecReload=/bin/kill -USR1 $MAINPID` for immediate feed refresh

### Container

```bash
docker run -d \
  -p 179:179 \
  -v "$PWD/config:/config" \
  --name blackhole-threats \
  ghcr.io/soakes/blackhole-threats:latest
```

The container creates `/config/blackhole-threats.yaml` automatically on first
boot if the default config path does not exist.

## Runtime Controls

### Immediate feed refresh

Trigger an immediate refresh when you want to re-read feeds without waiting for
the next timer tick.

Source install:

```bash
kill -USR1 <pid>
```

Debian package:

```bash
sudo systemctl reload blackhole-threats
```

Container:

```bash
docker kill --signal=SIGUSR1 blackhole-threats
```

This refresh path re-runs feed retrieval and route reconciliation. It does not
reload the YAML configuration or command-line flags.

### Configuration changes

Any change to the YAML file or startup flags requires a process restart, not a
reload.

Debian package:

```bash
sudo systemctl restart blackhole-threats
```

Container:

```bash
docker restart blackhole-threats
```

Source build:

Stop and start the process again with the updated flags or config path.

## What Healthy Operation Looks Like

Healthy startup logs include:

- `tag_version`, `commit`, and `build_date`
- `config_path`
- `run_mode`
- `refresh_interval`
- `configured_feeds`, `cli_feeds`, and `total_feeds`
- `peer_count`
- `local_as`
- `router_id`
- `default_community`

Healthy steady-state refresh logs include:

- `Prepared community routes`
- `Refresh completed`
- `announced`
- `withdrawn`
- `active_routes`
- `duration_ms`

## Day-2 Changes

### Add or remove feeds

1. Update the YAML file.
2. Run `-check-config`.
3. Run `-once` if the change is operationally significant.
4. Restart the service to load the changed YAML.

### Change communities

Treat BGP community changes as routing policy changes, not cosmetic edits. A
changed community can force route withdrawal and re-announcement.

Recommended path:

1. Validate the YAML.
2. Run `-once` against a lab or unprivileged port if possible.
3. Confirm the router-side filter policy still matches the intended
   communities.
4. Restart the production service.

### Change refresh interval

Use a shorter interval when validating or investigating a feed, then return to
the normal cadence once the issue is understood. The repo default is `2h`.

## Operational Safety Notes

- Keep feeds that should fail independently in separate communities.
- Avoid grouping unrelated feeds under one community unless the shared
  carry-forward behavior is intentional.
- Use `-once` before first binding to BGP port `179`, and verify any
  per-neighbor `config.port` overrides before production rollout.
- Prefer a restart after config edits; `SIGUSR1` is for feed refresh only.
- Watch route counts after major feed changes to catch accidental
  over-summarisation or unexpected source content.

## Escalation Triggers

Pause and investigate if you see any of the following:

- repeated `Failed to read threat feed` logs for the same source
- repeated `Keeping previous community routes after feed errors` warnings
- sudden large drops in `active_routes`
- route counts that jump unexpectedly after a feed change
- startup failures caused by router ID, ASN, or YAML validation errors

For concrete recovery steps, use [Troubleshooting](troubleshooting.md).
