function onRequest(ctx) {
  ctx.metadata.processedAt = new Date().toISOString();
  ctx.headers["x-afi-gateway"] = "true";
  return ctx;
}

function onBeforeUpstream(ctx) {
  return ctx;
}

function onResponse(ctx) {
  return ctx;
}
