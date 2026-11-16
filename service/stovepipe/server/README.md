# Stovepipe service configuration

The server embeds `config/base.yaml` as its operational baseline and accepts an optional `-config` YAML file layered over it. Omitting the optional file still loads the baseline; there are no hidden Go fallback values. Invalid configuration fails startup before consumers start. Effective settings are logged at startup; updates require a restart.

## Stage settings

The `queueconfig.stages` section contains typed stage blocks. `process` owns `maxConcurrent` and `gateWaitDelay`. `buildSignal` owns `acceptedPollDelay` and `runningPollDelay`; `buildAnalysis` owns `pollDelay`; `issueTrack` owns `refreshDelay` and `recoveryWaitDelay`. Timing and non-timing settings belong to the stage that owns their behavior.

Each stage has its own YAML adapter, shared override application, and validation. The shared library uses integer milliseconds; host adapters accept duration strings such as `2m`, `30s`, or `250ms`, restricted to whole milliseconds. Partial layers are merged before semantic validation. Omitted fields inherit; explicit zero is not omission. The current limits and intervals must be positive. Unknown fields, duplicate keys, and multiple YAML documents are rejected.

## Layering and queues

Deployment-wide stage settings from the optional file override the service baseline. Entries under `queueconfig.queues`, keyed by queue name, contain their own `stages` blocks and override deployment settings. When multiple files specify the same queue, its fields are merged rather than replacing its entire configuration. Queue layers retain file order and are applied after deployment-wide settings.

Every queue override must name a queue in `MQ_TENANTS`; queues without overrides inherit the deployment configuration. Queue registration is separate from stage settings. The static store only serves registered queues.

The baseline retains a concurrency limit of one, a five-second admission wait, five-second accepted-build polling, two-second running-build polling, two-minute build analysis polling, and thirty-second issue refresh and recovery waits. These values are defined only in the baseline YAML.
