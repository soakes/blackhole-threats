# Documentation

This directory holds the longer-form operator and contributor guides that sit
next to the main [README](../README.md).

## Guides

| Document | Audience | Purpose |
| --- | --- | --- |
| [architecture.md](architecture.md) | Contributors and operators | Package boundaries, runtime flow, and design goals |
| [operations.md](operations.md) | Operators | Deployment checklist, runtime controls, and day-2 workflow |
| [config-reference.md](config-reference.md) | Operators | Repo-owned YAML schema, validation rules, and CLI interaction |
| [feed-behavior.md](feed-behavior.md) | Operators and contributors | Feed retrieval, parsing, grouping, and summarisation behavior |
| [deployment-examples.md](deployment-examples.md) | Operators | Concrete source, Debian, and container deployment patterns |
| [release-and-publishing.md](release-and-publishing.md) | Maintainers and contributors | Release workflow, publish pipeline, and automation rules |
| [troubleshooting.md](troubleshooting.md) | Operators and maintainers | Focused troubleshooting and recovery playbooks |

## Suggested Reading Order

If you are deploying the service for the first time:

1. Start with [operations.md](operations.md).
2. Keep [config-reference.md](config-reference.md) open while editing YAML.
3. Use [deployment-examples.md](deployment-examples.md) for the path that
   matches your environment.
4. Jump to [troubleshooting.md](troubleshooting.md) if the first validation or
   first BGP session does not behave as expected.

If you are changing the code or release behavior:

1. Read [architecture.md](architecture.md).
2. Read [feed-behavior.md](feed-behavior.md) before touching parser or route
   lifecycle code.
3. Read [release-and-publishing.md](release-and-publishing.md) before changing
   workflows, packaging, or release automation.
