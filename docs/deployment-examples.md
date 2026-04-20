# Deployment Examples

This guide provides concrete deployment patterns for common environments.

See also:

- [Operations Guide](operations.md)
- [Configuration Reference](config-reference.md)
- [Troubleshooting](troubleshooting.md)

## Source Build Smoke Test

Use this pattern when validating a config locally before introducing systemd,
containers, or privileged BGP ports.

### Example config

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

feeds:
  - url: https://team-cymru.org/Services/Bogons/fullbogons-ipv4.txt
    community: 64520:1101
```

### Commands

```bash
make build
./dist/blackhole-threats -conf ./blackhole-threats.yaml -check-config
./dist/blackhole-threats -conf ./blackhole-threats.yaml -once -log-level debug
```

Use this when you want to prove:

- the YAML is valid
- feeds are reachable
- the route table can be built
- the process can complete one full cycle cleanly

## Debian Package Deployment

Use this path for a conventional long-running Linux service with systemd and
journald.

### Install from the signed APT repository

```bash
sudo install -d -m 0755 /etc/apt/keyrings
curl -fsSL https://soakes.github.io/blackhole-threats/blackhole-threats-archive-keyring.gpg \
  | sudo tee /etc/apt/keyrings/blackhole-threats-archive-keyring.gpg >/dev/null

sudo tee /etc/apt/sources.list.d/blackhole-threats.sources >/dev/null <<'EOF'
Types: deb deb-src
URIs: https://soakes.github.io/blackhole-threats/
Suites: stable
Components: main
Signed-By: /etc/apt/keyrings/blackhole-threats-archive-keyring.gpg
EOF

sudo apt update
sudo apt install blackhole-threats
```

### Validate before enabling startup

```bash
sudo /usr/sbin/blackhole-threats -conf /etc/blackhole-threats.yaml -check-config
sudo systemctl start blackhole-threats
sudo journalctl -u blackhole-threats -f
```

### Useful package-specific commands

Immediate feed refresh:

```bash
sudo systemctl reload blackhole-threats
```

Full restart after config changes:

```bash
sudo systemctl restart blackhole-threats
```

Inspect defaults:

```bash
sudo editor /etc/default/blackhole-threats
```

## Container Deployment

Use this path when you want an immutable runtime with a mounted config
directory.

### Basic run

```bash
docker pull ghcr.io/soakes/blackhole-threats:latest
docker run -d \
  -p 179:179 \
  -v "$PWD/config:/config" \
  --name blackhole-threats \
  ghcr.io/soakes/blackhole-threats:latest
```

On first boot, the container will create `/config/blackhole-threats.yaml` if it
does not already exist.

### Add runtime flags

```bash
docker run -d \
  -p 179:179 \
  -v "$PWD/config:/config" \
  -e BLACKHOLE_THREATS_EXTRA_OPTS="-log-format json -log-level debug -refresh-rate 15m" \
  --name blackhole-threats \
  ghcr.io/soakes/blackhole-threats:latest
```

### Use an alternate config path

```bash
docker run -d \
  -p 179:179 \
  -v "$PWD/config:/config" \
  -e BLACKHOLE_THREATS_CONF="/config/custom.yaml" \
  --name blackhole-threats \
  ghcr.io/soakes/blackhole-threats:latest
```

### Trigger a manual refresh

```bash
docker kill --signal=SIGUSR1 blackhole-threats
```

### Restart after config edits

```bash
docker restart blackhole-threats
```

## Local Feed File Example

Use a local file when testing parser behavior or staging a short custom list.

`local-feed.txt`:

```text
# Local blocklist
198.51.100.10
203.0.113.0/24
2001:db8:10::/48
```

Config snippet:

```yaml
feeds:
  - url: ./local-feed.txt
    community: 64520:1200
```

Validation:

```bash
./dist/blackhole-threats -conf ./blackhole-threats.yaml -once
```

## Safe Rollout Pattern

This pattern works well regardless of packaging method:

1. Edit the config.
2. Run `-check-config`.
3. Run `-once` in a safe environment if the change is significant.
4. Start or restart the long-running service.
5. Confirm the first `Refresh completed` log line.

If the change only affects feed contents and not YAML, use a manual refresh
signal instead of a restart.
