# Common observability contracts

## Logging

Create one `log.New` logger in the composition root and inject it. Use the
context-aware `slog` methods so an existing span context is emitted as
`trace_id` and `span_id`. The package always writes JSON to stdout and never
pushes logs over OTLP.

## Metrics

Each process creates its own `metric.Registry` and exposes `Handler()` at
`/metrics`. HTTP/gRPC RED metrics come from platform eBPF instrumentation;
services use this package for bounded business counters and histograms. Raw
paths, request IDs, trace IDs, user IDs, email addresses, and URLs are rejected
as labels.

## Event delivery

CloudEvents are transported by the platform-managed Knative Broker. Services
publish over HTTP and expose HTTP subscribers; broker durability, routing,
retry, and dead-letter behavior are declared as Kubernetes resources.
Business event schemas stay in their owning service. In particular, the Chaos
mail delivery event lives under `chaos/internal/mail`, not in this module.

## Distributed tracing

Heliantheon Go services use OpenTelemetry for distributed traces and keep
application logs in structured JSON. The shared `observability` package owns
the transport-neutral setup used by every service.

## Runtime configuration

Tracing is enabled when either `OTEL_EXPORTER_OTLP_ENDPOINT` or
`OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` is set. Services export OTLP over gRPC and
honor the standard exporter variables supported by the OpenTelemetry Go
exporter. Set `OTEL_TRACES_EXPORTER=none` to disable exporting explicitly.

The default root sampler is `parentbased_traceidratio` with a ratio of `0.25`.
Override it with `OTEL_TRACES_SAMPLER` and `OTEL_TRACES_SAMPLER_ARG`.

## HTTP and gRPC propagation

HTTP uses the W3C `traceparent`, `tracestate`, and `baggage` headers. gRPC uses
the same propagator through metadata. A valid incoming parent is continued by
the service, and each traced HTTP response includes `X-Trace-ID` so clients can
report the backend trace without exporting browser spans.

`traceparent` and `X-Trace-ID` are correlation data only. They must never be
used for authentication, authorization, rate-limit identity, or tenancy.

Health probes are excluded from application traces and structured access logs.
Metrics remain the signal for health and availability monitoring.

## Database tracing

`database.Connect` installs the GORM OpenTelemetry plugin and creates client
spans for create, query, update, delete, row, and raw operations. Query values
are excluded from span attributes, and database pool metrics remain owned by
the metrics pipeline rather than the tracing plugin.

Every request-scoped query must use `db.WithContext(ctx)`. Start transactions
from that contextual database handle so all statements inherit the incoming
HTTP or gRPC trace. Background jobs must supply their own bounded context;
using a bare shared `*gorm.DB` creates an unrelated root span.

## Log correlation

Use `logger.WithContext(ctx)` for application events that need trace
correlation. It adds `trace_id` and `span_id` only when the context contains a
valid span. The shared HTTP access logger applies this automatically.

Do not add request bodies, authorization headers, cookies, OAuth codes, email
addresses, or database query parameters to trace attributes or logs.
