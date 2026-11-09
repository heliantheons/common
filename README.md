# Common

Shared Go packages used by the Helios services. This repository is for code that is independent of a product domain: configuration, database and Redis clients, logging, pagination, patch handling, mail, throttling, and small infrastructure helpers.

The platform-facing packages are intentionally flat:

- `log`: independent `log/slog` JSON loggers with trace/span correlation.
- `metric`: independent Prometheus registries and safe custom metric helpers.

Applications write logs to stdout and expose Prometheus metrics. They do not
configure an OTLP endpoint; Alloy/Beyla owns collection and trace export.

Authentication-specific code belongs in [`aegis-go`](https://github.com/heliantheon/aegis-go). Hermes gRPC contracts live in [`hermes`](https://github.com/heliantheon/hermes) under `proto/v1`.

## Mail transport

The SMTP pool verifies the configured server hostname for STARTTLS and implicit
TLS, with TLS 1.2 as the minimum version. Certificate verification remains
enabled; SMTP startup verification and actual pooled delivery use the same
server identity.

## Install

```bash
go get github.com/heliantheon/common/pagination
```

Import the package you need directly:

```go
import "github.com/heliantheon/common/pagination"
```

## Development

```bash
make test
make lint
make tidy
```

The API query conventions implemented by `filter`, `pagination`, and `patch` are documented in [`docs/api-query-design.md`](docs/api-query-design.md).

The logging, metrics, and tracing contracts are documented in [`docs/observability.md`](docs/observability.md).

## Package policy

- Packages must not depend on a Helios service or a product domain.
- Prefer a small package with an explicit API over a general-purpose utility bucket.
- Breaking changes require all known consumers to be migrated first.
