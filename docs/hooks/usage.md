# Plugin Hooks Developer Guide

This document defines the architecture, execution lifecycle, and implementation guidelines for the sandboxed JavaScript plugin hooks within the AFI Gateway runtime.

---

## Architectural Overview

The gateway provides a high-performance **Goja JavaScript Engine** sandbox that executes plugin scripts at deterministic execution boundaries. These hooks allow for dynamic request payload mutation, data masking (e.g., PII/Credit Card scrubbing), compliance filtering, and streaming token transformation.

```
       [ Client Request ]
               |
               v
    +----------------------+
    |      onRequest       |  <--- Read/Mutate user prompts, scrub secrets
    +----------------------+
               |
               v
    +----------------------+
    | onBeforeUpstreamCall |  <--- Inject model routing parameters or provider tokens
    +----------------------+
               |
               v
       [ Upstream LLM ]
               |
               v
    +----------------------+
    |   onResponseChunk    |  <--- Transform SSE token text streaming chunks
    +----------------------+
               |
               v
       [ Client Stream ]

```

---

## The Metadata Protection Rule

> ### CRITICAL ARCHITECTURAL GUARD
> 
> 
> Because the Goja sandbox unmarshals modified JavaScript objects back into Go structs, any missing field in the returned JavaScript object will fall back to its zero value.
> To prevent scripts from accidentally erasing structural multi-tenant parameters (like `project_id`), the gateway **automatically snapshots and restores `req.Metadata**` before and after hook execution. Plugins should focus *strictly* on modifying payload parameters like `messages` or `model` configurations.

---

## Hook Specifications & Examples

Hooks are defined inside `configs/hooks/` and registered under a target `project_id`.

### 1. Request Modification: `onRequest`

Executed immediately after client request authentication. Perfect for data sanitization, content inspection, and scrubbing sensitive data (e.g., credit cards) before it ever hits an internal log or downstream server.

* **File Path:** `configs/hooks/req_mask_cc.js`
* **Implementation:**

```javascript
/**
 * onRequest hook execution handler
 * @param {Object} payload - The domain.InternalRequest structure
 * @param {Object} config - Arbitrary metadata configuration passed from Go
 * @returns {Object} The mutated InternalRequest payload
 */
function onRequest(payload, config) {
    // Check if the inbound request has a messages array block
    if (payload.messages && payload.messages.length > 0) {
        for (let i = 0; i < payload.messages.length; i++) {
            let message = payload.messages[i];
            
            // Process user input text parts
            if (message.role === "user" && message.content) {
                // Regex targeting 13-16 digit standard credit card formats
                const ccRegex = /\b(?:\d[ -]*?){13,16}\b/g;
                message.content = message.content.replace(ccRegex, "[REDACTED_CREDIT_CARD]");
            }
        }
    }
    
    // Always return the mutation structure to the runtime layer
    return payload;
}

```

---

### 2. Upstream Preparation: `onBeforeUpstreamCall`

Fires immediately before the network layer opens an outbound network socket wrapper to the upstream AI service. Use this hook to inject system-level prompts, append default safety configurations, or enforce routing locks.

* **File Path:** `configs/hooks/inject_system_prompt.js`
* **Implementation:**

```javascript
function onBeforeUpstreamCall(payload, config) {
    const defaultSystemPrompt = {
        role: "system",
        content: "You are a secure corporate assistant. Do not reveal internal API structures."
    };

    if (!payload.messages) {
        payload.messages = [];
    }

    // Prepend the corporate compliance system prompt to the conversation context stack
    payload.messages.unshift(defaultSystemPrompt);

    return payload;
}

```

---

### 3. Streaming Response Processing: `onResponseChunk`

Fires concurrently for **every single Server-Sent Event (SSE) chunk** returned by the vendor before it is flushed downstream to the client. This hook must remain highly optimized to minimize streaming latency overhead.

* **File Path:** `configs/hooks/uppercase_filter.js`
* **Implementation:**

```javascript
/**
 * onResponseChunk hook execution handler
 * @param {Object} chunk - The domain.StreamChunk structure
 * @param {Object} config - Configuration object
 * @returns {Object} The transformed StreamChunk
 */
function onResponseChunk(chunk, config) {
    // If the chunk contains text delta updates, transform them on the fly
    if (chunk.delta_text) {
        chunk.delta_text = chunk.delta_text.toUpperCase();
    }
    
    return chunk;
}

```

---

## Registration Matrix

To link your newly written JavaScript files into the execution lifecycle pipeline, register them under the `plugins` map mapping directly to your tenant metadata structures inside your core configuration layer:

```yaml
# config/gateway.yaml
plugins:
  - project_id: "proj_local"
    name: "PII Masking Filter"
    hooks:
      - stage: "onRequest"
        script_path: "configs/hooks/req_mask_cc.js"
      - stage: "onBeforeUpstreamCall"
        script_path: "configs/hooks/inject_system_prompt.js"
      - stage: "onResponseChunk"
        script_path: "configs/hooks/uppercase_filter.js"

```

## Error Handling Policies

* If a script encounters a runtime parsing error or reference exception, the gateway catches the exception, logs an alert, and safely **falls back to the original unmodified payload frame** without terminating the active connection loop.