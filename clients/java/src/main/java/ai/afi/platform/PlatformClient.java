package ai.afi.platform;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.function.Supplier;
import java.util.stream.Collectors;

/**
 * Thin synchronous HTTP client for {@code /api/v1/platform/*}.
 *
 * <pre>{@code
 * PlatformClient client = new PlatformClient("http://localhost:8081");
 * String token = client.login("admin@example.com", "secret").path("token").asText();
 * client = new PlatformClient("http://localhost:8081", () -> token);
 * }</pre>
 */
public final class PlatformClient implements AutoCloseable {
  private static final ObjectMapper MAPPER = new ObjectMapper();
  private static final TypeReference<List<JsonNode>> LIST_OF_NODE = new TypeReference<>() {};

  private final String baseUrl;
  private final Supplier<String> tokenGetter;
  private final HttpClient http;
  private final Duration timeout;
  private final RequestExecutor executor;
  private final boolean ownsHttp;

  /** Functional escape hatch for tests. */
  @FunctionalInterface
  public interface RequestExecutor {
    RawResponse execute(String method, URI uri, Map<String, String> headers, byte[] body)
        throws IOException, InterruptedException;
  }

  /** Minimal HTTP response used by {@link RequestExecutor}. */
  public record RawResponse(int status, String body) {}

  public PlatformClient(String baseUrl) {
    this(baseUrl, null, null, Duration.ofSeconds(30), null);
  }

  public PlatformClient(String baseUrl, Supplier<String> tokenGetter) {
    this(baseUrl, tokenGetter, null, Duration.ofSeconds(30), null);
  }

  public PlatformClient(
      String baseUrl, Supplier<String> tokenGetter, HttpClient http, Duration timeout) {
    this(baseUrl, tokenGetter, http, timeout, null);
  }

  PlatformClient(
      String baseUrl,
      Supplier<String> tokenGetter,
      HttpClient http,
      Duration timeout,
      RequestExecutor executor) {
    Objects.requireNonNull(baseUrl, "baseUrl");
    String normalized = baseUrl.endsWith("/") ? baseUrl.substring(0, baseUrl.length() - 1) : baseUrl;
    this.baseUrl = normalized;
    this.tokenGetter = tokenGetter;
    this.timeout = timeout == null ? Duration.ofSeconds(30) : timeout;
    this.ownsHttp = http == null && executor == null;
    this.http =
        http != null
            ? http
            : HttpClient.newBuilder().connectTimeout(this.timeout).build();
    this.executor = executor != null ? executor : this::defaultExecute;
  }

  /** Test constructor with a custom transport. */
  public static PlatformClient withExecutor(
      String baseUrl, Supplier<String> tokenGetter, RequestExecutor executor) {
    return new PlatformClient(baseUrl, tokenGetter, null, Duration.ofSeconds(30), executor);
  }

  @Override
  public void close() {
    // java.net.http.HttpClient does not need explicit close on modern JDKs.
  }

  public JsonNode request(
      String method, String path, Object body, boolean auth, Map<String, ?> query) {
    try {
      Map<String, String> headers = new LinkedHashMap<>();
      byte[] bodyBytes = null;
      if (body != null) {
        headers.put("Content-Type", "application/json");
        bodyBytes = MAPPER.writeValueAsBytes(body);
      }
      if (auth) {
        String token = tokenGetter == null ? null : tokenGetter.get();
        if (token == null || token.isBlank()) {
          throw new PlatformApiException("missing access token", 401);
        }
        headers.put("Authorization", "Bearer " + token);
      }
      URI uri = buildUri(path, query);
      RawResponse res = executor.execute(method, uri, headers, bodyBytes);
      if (res.status() == 204) {
        return null;
      }
      JsonNode parsed = null;
      if (res.body() != null && !res.body().isBlank()) {
        try {
          parsed = MAPPER.readTree(res.body());
        } catch (IOException ignored) {
          // leave parsed null; error path uses raw text via body field
        }
      }
      if (res.status() >= 400) {
        String message = "request failed";
        if (parsed != null && parsed.path("error").isTextual()) {
          message = parsed.get("error").asText();
        }
        Object errBody = parsed != null ? parsed : res.body();
        throw new PlatformApiException(message, res.status(), errBody);
      }
      return parsed;
    } catch (PlatformApiException e) {
      throw e;
    } catch (InterruptedException e) {
      Thread.currentThread().interrupt();
      throw new PlatformApiException("request interrupted", 0, e.getMessage());
    } catch (IOException e) {
      throw new PlatformApiException("request failed: " + e.getMessage(), 0, e.getMessage());
    }
  }

  public JsonNode request(String method, String path) {
    return request(method, path, null, true, null);
  }

  public JsonNode healthz() {
    return request("GET", "/healthz", null, false, null);
  }

  public JsonNode login(String email, String password) {
    return request(
        "POST",
        "/api/v1/platform/auth/login",
        Map.of("email", email, "password", password),
        false,
        null);
  }

  public JsonNode authFeatures() {
    return request("GET", "/api/v1/platform/auth/features", null, false, null);
  }

  public JsonNode register(String email, String name, String password) {
    return request(
        "POST",
        "/api/v1/platform/auth/register",
        Map.of("email", email, "name", name, "password", password),
        false,
        null);
  }

  public JsonNode requestPasswordReset(String email) {
    return request(
        "POST",
        "/api/v1/platform/auth/password-reset",
        Map.of("email", email),
        false,
        null);
  }

  public JsonNode confirmPasswordReset(String token, String password) {
    return request(
        "POST",
        "/api/v1/platform/auth/password-reset/" + encodePath(token),
        Map.of("password", password),
        false,
        null);
  }

  public JsonNode me() {
    return request("GET", "/api/v1/platform/auth/me");
  }

  public List<JsonNode> listOrganizations() {
    return asList(request("GET", "/api/v1/platform/organizations"));
  }

  public JsonNode createOrganization(String name) {
    return request("POST", "/api/v1/platform/organizations", Map.of("name", name), true, null);
  }

  public List<JsonNode> listOrgKeys(String orgId) {
    return asList(
        request("GET", "/api/v1/platform/organizations/" + encodePath(orgId) + "/keys"));
  }

  public JsonNode createOrgKey(String orgId, Map<String, ?> body) {
    return request(
        "POST",
        "/api/v1/platform/organizations/" + encodePath(orgId) + "/keys",
        body,
        true,
        null);
  }

  public List<JsonNode> listEnvironments(String orgId, String projectId) {
    return asList(
        request(
            "GET",
            "/api/v1/platform/organizations/"
                + encodePath(orgId)
                + "/projects/"
                + encodePath(projectId)
                + "/environments"));
  }

  public JsonNode createEnvironment(String orgId, String projectId, String name, String slug) {
    return request(
        "POST",
        "/api/v1/platform/organizations/"
            + encodePath(orgId)
            + "/projects/"
            + encodePath(projectId)
            + "/environments",
        Map.of("name", name, "slug", slug),
        true,
        null);
  }

  public void deleteEnvironment(String environmentId) {
    request("DELETE", "/api/v1/platform/environments/" + encodePath(environmentId));
  }

  public List<JsonNode> listProviders(String orgId) {
    return asList(
        request("GET", "/api/v1/platform/organizations/" + encodePath(orgId) + "/providers"));
  }

  public List<JsonNode> listRoutes(String orgId) {
    return asList(
        request("GET", "/api/v1/platform/organizations/" + encodePath(orgId) + "/routes"));
  }

  public List<JsonNode> listUsage(String orgId, Map<String, ?> query) {
    return asList(
        request(
            "GET",
            "/api/v1/platform/organizations/" + encodePath(orgId) + "/usage",
            null,
            true,
            query));
  }

  public List<JsonNode> listAudit(String orgId, Map<String, ?> query) {
    return asList(
        request(
            "GET",
            "/api/v1/platform/organizations/" + encodePath(orgId) + "/audit",
            null,
            true,
            query));
  }

  private RawResponse defaultExecute(
      String method, URI uri, Map<String, String> headers, byte[] body)
      throws IOException, InterruptedException {
    HttpRequest.Builder b =
        HttpRequest.newBuilder(uri).timeout(timeout).method(
            method,
            body == null
                ? HttpRequest.BodyPublishers.noBody()
                : HttpRequest.BodyPublishers.ofByteArray(body));
    headers.forEach(b::header);
    HttpResponse<String> res = http.send(b.build(), HttpResponse.BodyHandlers.ofString());
    return new RawResponse(res.statusCode(), res.body());
  }

  private URI buildUri(String path, Map<String, ?> query) {
    String p = path.startsWith("/") ? path : "/" + path;
    StringBuilder sb = new StringBuilder(baseUrl).append(p);
    if (query != null && !query.isEmpty()) {
      String qs =
          query.entrySet().stream()
              .filter(e -> e.getValue() != null)
              .map(
                  e ->
                      URLEncoder.encode(e.getKey(), StandardCharsets.UTF_8)
                          + "="
                          + URLEncoder.encode(String.valueOf(e.getValue()), StandardCharsets.UTF_8))
              .collect(Collectors.joining("&"));
      if (!qs.isEmpty()) {
        sb.append('?').append(qs);
      }
    }
    return URI.create(sb.toString());
  }

  private static String encodePath(String segment) {
    return URLEncoder.encode(segment, StandardCharsets.UTF_8).replace("+", "%20");
  }

  private static List<JsonNode> asList(JsonNode node) {
    if (node == null || node.isNull()) {
      return List.of();
    }
    if (!node.isArray()) {
      throw new PlatformApiException("expected JSON array", 0, node);
    }
    try {
      return MAPPER.convertValue(node, LIST_OF_NODE);
    } catch (IllegalArgumentException e) {
      throw new PlatformApiException("expected JSON array", 0, node);
    }
  }
}
