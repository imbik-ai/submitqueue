# Queue configuration

Queue configuration is deployment policy, separate from mutable queue state in storage. The store interface supplies immutable, fully resolved settings by queue name; it does not imply database persistence.

## Stage ownership

Configuration is grouped by stage: process, build signal, build analysis, and issue tracking. Each stage owns its resolved entity, its validation, and its optional overrides in stage-specific files. A stage's override application copies the supplied baseline and replaces only explicitly supplied fields. The aggregate composes stages; it does not inspect their individual settings.

The process stage owns concurrency and admission waits. The other current stage settings control build-status polling, build-issue analysis polling, and issue-tracking refresh and recovery waits. Stage configuration can contain both timing and non-timing settings. Immediate batch continuation and consumer error-retry policies remain separate.

## Resolution

Hosts supply a complete deployment baseline and ordered per-queue override layers. The static store requires a valid deployment baseline, applies every layer for each named queue, then validates the final queue configuration. Partial override layers are not validated in isolation. Nil fields inherit; explicit values follow the owning stage's constraints. The current concurrency limits and millisecond intervals must be positive.

The static store rejects duplicate names and overrides for unknown queues, and returns independent configuration values. There is no permissive fallback for unregistered queues.

## Host integration

The shared library does not parse configuration files or supply hardcoded operational defaults. Hosts own loading and dependency injection and can use different configuration frameworks. The reference host embeds a service baseline and optionally layers another YAML file over it. Queue-specific layers take precedence over deployment-wide settings. Changes to startup configuration require a restart.

Adding a setting to an existing stage requires updating that stage's entity, override application, validation, and host adapter. Required values must be supplied by the host baseline. It does not require adding field-level logic to the aggregate or store.
