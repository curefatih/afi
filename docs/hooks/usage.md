# Plugins / hooks

Historically AFI experimented with in-process Goja JavaScript hooks (`onRequest`, `onBeforeUpstreamCall`, `onResponseChunk`).

The target architecture uses **extension points** instead:

* gRPC extensions — providers, auth, secrets, notifications
* WASM extensions — prompt/header mutation, PII masking, enrichment

The current vertical slice ships **no runtime hooks**. Provider calls go through the built-in OpenAI adapter in the data plane.

When the extension runtime lands, this page will document capability contracts and registration.
