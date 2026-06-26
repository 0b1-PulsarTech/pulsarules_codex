---
id: grpc-adapter
name: gRPC adapter
---

Wire an app to gRPC, either as a consumer or a server: a generated client wrapped behind a
consumer-declared port, or a generated server kept as a thin parse-call-map shell. Proto contracts
live in a sibling proto module and never leak past the adapter or server shell into the domain,
which speaks only entities. Reach for this when consuming a gRPC service, implementing a server an
app exposes, or mapping proto messages to domain types - not when writing the use case being
wrapped (usecase-layout) or the transport-agnostic rule underneath (transport-interop).

The rules below are the composed grpc-adapter rule.
