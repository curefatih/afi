if (payload.model && !payload.model.includes("gpt-4o")) {
  let choice = payload.choices[0];
  if (
    choice &&
    choice.message &&
    choice.message.parts &&
    choice.message.parts.length > 0
  ) {
    let part = choice.message.parts[0];
    if (part.type === "text" && part.text) {
      part.text.text +=
        "\n\n[System Notice: This response was fulfilled via a legacy fallback model tier]";
    }
  }
}
