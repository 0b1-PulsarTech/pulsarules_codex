---
id: grpc-adapter
name: gRPC adapter
description: Wrap a generated proto client behind a consumer-declared port; the server is a thin parse-call-map shell embedding Unimplemented...Server; proto never leaks into the domain.
tags:
    - go
    - transport
    - grpc
dependencies:
    - grpc
    - transport
---

# gRPC adapter

> Wrap a generated proto client behind a consumer-declared port (adapter in `internal/infra/grpc/`);
> the gRPC server is a thin parse-call-map shell embedding `Unimplemented...Server`; proto types
> never leak into the domain.

Reference tools: a sibling proto module; `google.golang.org/grpc`.

{{define "when"}}
- Consuming a gRPC service.
- Implementing a gRPC server an app exposes.
- Mapping proto messages to domain entities.
{{end}}

{{define "recipe"}}
Consumer port (domain types only):

```go
// internal/domain/usecases/greetmessage/port.go
type Greeter interface {
    Greet(ctx context.Context, name string) (entities.Greeting, error)
}
```

Client adapter:

```go
// internal/infra/grpc/greetgrpc/adapter.go
type Adapter struct{ client foov1grpc.GreeterServiceClient }

func New(conn *grpc.ClientConn) *Adapter {
    return &Adapter{client: foov1grpc.NewGreeterServiceClient(conn)}
}

var _ greetmessage.Greeter = (*Adapter)(nil)

func (a *Adapter) Greet(ctx context.Context, name string) (entities.Greeting, error) {
    resp, err := a.client.Greet(ctx, &foov1.GreetRequest{Name: name})
    if err != nil {
        return entities.Greeting{}, fmt.Errorf("greetgrpc.Greet: %w", err)
    }
    return toGreeting(resp), nil
}
```

Server (thin shell, logic in the use case):

```go
// internal/transport/grpc/greeterservice/server.go
type Server struct {
    foov1grpc.UnimplementedGreeterServiceServer
    uc greet.UseCase
}

func (s *Server) Greet(ctx context.Context, req *foov1.GreetRequest) (*foov1.GreetResponse, error) {
    // parse -> call use case -> map entities to proto -> respond
}
```

DI: register the gRPC client connection as a singleton; the adapter as a singleton (or factory if
request-scoped). The use case depends only on the port.
{{end}}

<!-- No forbidden/validation blocks here on purpose: every line this pattern used to carry was a
     verbatim subset of [[grpc]]'s own, and the two only ever render into the same skill, so the
     reader met each obligation twice. The rule owns them; this file owns the recipe. -->

