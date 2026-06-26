---
id: external-provider
name: External provider (Strategy + Adapter)
description: An external integration is a Strategy port; the concrete HTTP impl lives in a per-provider package with a fixed file set; DTOs unexported; mapping in mapper.go; bootstrap switches on the code.
tags:
    - go
    - integration
dependencies:
    - http-clients
    - dependency-injection
composes:
    - retry-backoff
---

# External provider (Strategy + Adapter)

> An external integration is a `Strategy` port; the concrete HTTP implementation lives under
> `internal/infra/fetchers/<name>fetch/` (or `internal/infra/providers/<name>/`) with a fixed file
> set; DTOs are unexported; mapping happens in `mapper.go`; the bootstrap is the only place that
> switches on the provider code.

Reference tools: the HTTP gateway pattern ([[http-clients]]); `remy` DI; an ID generator (`idgen`).

{{define "when"}}
- Adding a new external HTTP fetcher/provider/integration.
- Onboarding a Strategy implementation behind a port.
{{end}}

{{define "recipe"}}
Package layout:

```
internal/infra/fetchers/<name>fetch/
├── fetcher.go    # Fetcher struct + Code()/DisplayName()/Sync methods
├── client.go     # HTTP calls via httpdatasource.Client, cache-aware
├── url.go        # endpoint builders (URLs come from config)
├── dto.go        # package-local API DTOs; never exported
├── mapper.go     # dto -> domain SyncDelta
├── di.go         # Register(inj, baseURL, lang)
└── doc.go        # package comment + endpoint list
```

Implement the Strategy port:

```go
type Fetcher struct {
    http    httpdatasource.Client
    cache   cachestore.Cache
    baseURL string
    lang    string
}

func New(http httpdatasource.Client, cache cachestore.Cache, baseURL, lang string) *Fetcher {
    return &Fetcher{http: http, cache: cache, baseURL: baseURL, lang: lang}
}

func (f *Fetcher) Code() domain.ProviderID { return "<name>" }

func (f *Fetcher) SyncByDate(ctx context.Context, day time.Time) (domain.SyncDelta, error) {
    raw, err := f.fetch(ctx, scheduleURL(f.baseURL, day, f.lang))
    if err != nil {
        return domain.SyncDelta{}, err
    }
    var dto apiResponse
    if err := json.Unmarshal(raw, &dto); err != nil {
        return domain.SyncDelta{}, fmt.Errorf("<name>fetch: decode: %w", err)
    }
    return mapResponse(dto), nil
}
```

DI registers both the concrete and the Strategy binding the set consumes:

```go
func Register(inj remy.Injector, baseURL, lang string) {
    remy.RegisterConstructorArgs2(inj, remy.Factory[*Fetcher],
        func(http httpdatasource.Client, cache cachestore.Cache) *Fetcher {
            return New(http, cache, baseURL, lang)
        })
    remy.RegisterConstructorArgs1(inj, remy.Factory[domain.Strategy],
        func(f *Fetcher) domain.Strategy { return f })
}
```

Wire in bootstrap (the only switchboard):

```go
for _, pc := range conf.Providers {
    if !pc.Enabled { continue }
    switch pc.Code {
    case domain.ProviderFoo:
        foofetch.Register(inj, pc.BaseURL, pc.Language)
    }
}
```

Add config:

```toml
[[providers]]
code     = "foo"
enabled  = false
base_url = "https://api.example.com"
language = "en"
```
{{end}}

{{define "forbidden"}}
- Hardcoded URLs or language outside `url.go`/config.
- Exported DTOs; mapping outside `mapper.go`.
- A `switch code` inside the fetcher package (bootstrap owns it).
- HTTP calls outside the gateway.
{{end}}

{{define "validation"}}
- [ ] Fixed file set; DTOs unexported; mapping in `mapper.go`; `doc.go` lists endpoints.
- [ ] DI registers concrete + Strategy bindings; bootstrap switches on code.
- [ ] `mapper.go` has table-driven tests (happy path + one edge case).
- [ ] No hardcoded URL/language; HTTP only via the gateway.
{{end}}
