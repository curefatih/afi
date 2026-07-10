# AFI - AI Gateway

Welcome to your local cloud-native AI gateway environment! This orchestration proxy is a high-performance, enterprise-grade access point designed to securely manage, route, audit, and intercept multi-tenant traffic hitting upstream LLM providers.


---

## System Architecture Overview

The gateway serves as a secure compliance buffer between your internal developer environments and public model endpoints.

```
 [ Your Apps / Clients ] 
          │
          ▼
 ┌────────────────────────────────────────────────────────┐
 │            LOCAL HEXAGONAL AI GATEWAY ENGINE           │
 │                                                        │
 │  🔒 1. Token Auth & Project Isolation Verification     │
 │  🪝 2. PII / Credit Card Masking JavaScript Sandbox    │
 │  🚦 3. Dynamic Rule Matrix Model Routing Router        │
 │  💰 4. Strict Synchronous Streaming Budget Tracker     │
 └────────────────────────────────────────────────────────┘
          │
          ▼
 [ Upstream Providers: OpenAI / Anthropic / Local Models ]

```

---

## Key Operational Features

* **🪝 Sandboxed Plugin Hooks Pipeline:** Inject JavaScript hooks natively (`onRequest`, `onBeforeUpstreamCall`, `onResponseChunk`) to scrub secrets, inject standard corporate guardrails, or alter streaming responses dynamically.
* **📈 Air-Tight SSE Streaming Telemetry:** Seamless interception of trailing vendor statistics frames ensures budget consumption limits and token-usage matrices are tracked perfectly on every connection closure.
* **🚦 Lightweight Condition Routing:** Zero-dependency wildcard or static route mappings configured natively via a decoupled memory state layer.

---
