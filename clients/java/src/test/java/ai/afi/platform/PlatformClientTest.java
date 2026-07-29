package ai.afi.platform;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import com.fasterxml.jackson.databind.JsonNode;
import java.net.URI;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicReference;
import org.junit.jupiter.api.Test;

class PlatformClientTest {

  @Test
  void meSendsBearer() {
    PlatformClient client =
        PlatformClient.withExecutor(
            "http://cp.test",
            () -> "tok",
            (method, uri, headers, body) -> {
              assertEquals("GET", method);
              assertEquals("/api/v1/platform/auth/me", uri.getPath());
              assertEquals("Bearer tok", headers.get("Authorization"));
              return new PlatformClient.RawResponse(
                  200, "{\"id\":\"u1\",\"name\":\"A\",\"email\":\"a@b.c\",\"role\":\"user\"}");
            });
    JsonNode me = client.me();
    assertEquals("u1", me.path("id").asText());
  }

  @Test
  void errorEnvelope() {
    PlatformClient client =
        PlatformClient.withExecutor(
            "http://cp.test",
            () -> "tok",
            (method, uri, headers, body) ->
                new PlatformClient.RawResponse(403, "{\"error\":\"nope\"}"));
    PlatformApiException ex =
        assertThrows(PlatformApiException.class, client::listOrganizations);
    assertEquals(403, ex.status());
    assertEquals("nope", ex.getMessage());
  }

  @Test
  void loginNoAuth() {
    AtomicReference<String> auth = new AtomicReference<>();
    PlatformClient client =
        PlatformClient.withExecutor(
            "http://cp.test",
            null,
            (method, uri, headers, body) -> {
              auth.set(headers.get("Authorization"));
              assertEquals("/api/v1/platform/auth/login", uri.getPath());
              return new PlatformClient.RawResponse(200, "{\"token\":\"jwt\"}");
            });
    assertEquals("jwt", client.login("a@b.c", "x").path("token").asText());
    assertNull(auth.get());
  }

  @Test
  void registerAndResetNoAuth() {
    List<String> paths = new ArrayList<>();
    PlatformClient client =
        PlatformClient.withExecutor(
            "http://cp.test",
            null,
            (method, uri, headers, body) -> {
              paths.add(uri.getPath());
              assertNull(headers.get("Authorization"));
              String path = uri.getPath();
              if (path.endsWith("/features")) {
                return new PlatformClient.RawResponse(
                    200, "{\"signup_enabled\":true,\"password_reset_enabled\":true}");
              }
              if (path.endsWith("/register")) {
                return new PlatformClient.RawResponse(
                    201,
                    "{\"token\":\"jwt\",\"user\":{\"id\":\"u1\",\"email\":\"a@b.c\",\"name\":\"A\",\"role\":\"member\"}}");
              }
              if (path.endsWith("/password-reset")) {
                return new PlatformClient.RawResponse(200, "{\"ok\":true}");
              }
              return new PlatformClient.RawResponse(
                  200,
                  "{\"token\":\"jwt2\",\"user\":{\"id\":\"u1\",\"email\":\"a@b.c\",\"name\":\"A\",\"role\":\"member\"}}");
            });
    assertTrue(client.authFeatures().path("signup_enabled").asBoolean());
    assertEquals("jwt", client.register("a@b.c", "A", "password1").path("token").asText());
    assertTrue(client.requestPasswordReset("a@b.c").path("ok").asBoolean());
    assertEquals("jwt2", client.confirmPasswordReset("tok", "password2").path("token").asText());
    assertTrue(paths.contains("/api/v1/platform/auth/features"));
  }

  @Test
  void missingToken() {
    PlatformClient client =
        PlatformClient.withExecutor(
            "http://cp.test",
            () -> null,
            (method, uri, headers, body) -> new PlatformClient.RawResponse(200, "{}"));
    PlatformApiException ex = assertThrows(PlatformApiException.class, client::me);
    assertEquals(401, ex.status());
    assertEquals("missing access token", ex.getMessage());
  }

  @Test
  void queryEncoding() throws Exception {
    AtomicReference<URI> seen = new AtomicReference<>();
    PlatformClient client =
        PlatformClient.withExecutor(
            "http://cp.test",
            () -> "tok",
            (method, uri, headers, body) -> {
              seen.set(uri);
              return new PlatformClient.RawResponse(200, "[]");
            });
    client.listUsage("org1", Map.of("limit", 10));
    assertTrue(seen.get().toString().contains("limit=10"));
    assertEquals("/api/v1/platform/organizations/org1/usage", seen.get().getPath());
  }
}
