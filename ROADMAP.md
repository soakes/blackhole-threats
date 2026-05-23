# Roadmap

This roadmap describes the current direction of the public project. It is not a
promise of delivery dates; priorities may change when security, packaging, or
operator-impacting issues appear.

## Current Priorities

- Keep source, container, Debian, and signed APT release paths reliable.
- Preserve predictable RTBH route announcement and withdrawal behavior.
- Improve test coverage around feed parsing, failed refreshes, and route-table
  reconciliation.
- Keep operator documentation accurate for common deployment patterns.
- Continue supply-chain hardening for Go modules, GitHub Actions, container
  bases, and packaging inputs.

## Near-Term Ideas

- Add more real-world feed examples and parser fixtures.
- Expand RouterOS and common BGP policy examples.
- Improve troubleshooting guidance for BGP session establishment and route
  visibility.
- Add clearer release-health checks around published artifacts.
- Improve public project-health metadata for contributors, operators, and
  open source support programs.

## Out Of Scope For Now

- A hosted SaaS version of the daemon.
- A separate registry or package archive outside the existing GitHub/GHCR/APT
  publication model.
- Runtime behavior that depends on a proprietary service.
- Broad plugin systems or large shared utility layers without a concrete
  operator need.
