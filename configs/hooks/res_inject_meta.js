if (payload.choices && payload.choices.length > 0) {
  // 1. Audit assistant's generated messages
  for (let i = 0; i < payload.choices.length; i++) {
    let choice = payload.choices[i];

    if (choice.message) {
      // Tag the message metadata so downstream microservices know it passed through your gateway
      choice.message.name = "gateway_verified_agent";
    }
  }

  // 2. Inject corporate warning notes directly inside the response if the upstream model wasn't the ideal tier
  if (payload.model && !payload.model.includes("gpt-4o")) {
    let primaryChoiceText = payload.choices[0].message.parts[0].text;

    // Append an institutional watermark note to the end of the content string
    primaryChoiceText.text +=
      "\n\n[System Notice: This response was fulfilled via a legacy fallback model tier]";
  }
}
