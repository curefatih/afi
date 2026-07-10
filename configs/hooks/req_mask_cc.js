// config/hooks/req_mask_cc.js

// Loop through and mutate in place (this preserves metadata perfectly)
if (payload.messages && payload.messages.length > 0) {
  for (let i = 0; i < payload.messages.length; i++) {
    let message = payload.messages[i];
    if (message.role === "user" && message.parts) {
      for (let j = 0; j < message.parts.length; j++) {
        let part = message.parts[j];
        if (part.type === "text" && part.text && part.text.text) {
          const ccRegex = /\b(?:\d[ -]*?){13,16}\b/g;
          part.text.text = part.text.text.replace(
            ccRegex,
            "[REDACTED_CREDIT_CARD]",
          );
        }
      }
    }
  }
}

// DO NOT reassign payload = { messages: [...] } manually,
// as that drops payload.metadata on the floor.
