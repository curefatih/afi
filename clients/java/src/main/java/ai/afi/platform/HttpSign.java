package ai.afi.platform;

import java.net.URI;
import java.nio.charset.StandardCharsets;
import java.security.KeyFactory;
import java.security.MessageDigest;
import java.security.PrivateKey;
import java.security.SecureRandom;
import java.security.Signature;
import java.security.spec.PKCS8EncodedKeySpec;
import java.time.Instant;
import java.util.Base64;
import java.util.HexFormat;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;

/**
 * RFC 9421 gateway request signing helpers for AFI signed-request auth.
 *
 * <p>Covers {@code @method}, {@code @path}, {@code @query}, and {@code content-digest} — the set
 * required by the AFI gateway.
 */
public final class HttpSign {
  public static final String SIGNATURE_NAME = "sig1";
  public static final List<String> REQUIRED_COMPONENTS =
      List.of("@method", "@path", "@query", "content-digest");

  private HttpSign() {}

  /** Build an RFC 9530 Content-Digest header value for sha-256. */
  public static String contentDigestSha256(byte[] body) {
    try {
      byte[] digest = MessageDigest.getInstance("SHA-256").digest(body == null ? new byte[0] : body);
      return "sha-256=:" + Base64.getEncoder().encodeToString(digest) + ":";
    } catch (Exception e) {
      throw new IllegalStateException("SHA-256 unavailable", e);
    }
  }

  /**
   * Return Content-Digest / Signature-Input / Signature headers for a gateway call.
   *
   * @param method HTTP method
   * @param url absolute URL or path (query included)
   * @param body request body bytes (may be empty)
   * @param privateKeyPem Ed25519 PKCS#8 PEM
   * @param keyId signing key id (required)
   */
  public static Map<String, String> signHeaders(
      String method,
      String url,
      byte[] body,
      byte[] privateKeyPem,
      String keyId) {
    return signHeaders(method, url, body, privateKeyPem, keyId, null, null);
  }

  public static Map<String, String> signHeaders(
      String method,
      String url,
      byte[] body,
      byte[] privateKeyPem,
      String keyId,
      String nonce,
      Long created) {
    if (keyId == null || keyId.isBlank()) {
      throw new IllegalArgumentException("keyId is required");
    }
    Objects.requireNonNull(method, "method");
    Objects.requireNonNull(url, "url");
    Objects.requireNonNull(privateKeyPem, "privateKeyPem");
    byte[] payload = body == null ? new byte[0] : body;
    PathAndQuery pq = pathAndQuery(url);
    long createdTs = created == null ? Instant.now().getEpochSecond() : created;
    String nonceVal = nonce == null || nonce.isBlank() ? randomNonce() : nonce;
    String digest = contentDigestSha256(payload);
    String sigParams =
        "(\"@method\" \"@path\" \"@query\" \"content-digest\")"
            + ";created="
            + createdTs
            + ";nonce=\""
            + nonceVal
            + "\";alg=\"ed25519\";keyid=\""
            + keyId
            + "\"";
    String sigBase =
        String.join(
            "\n",
            "\"@method\": " + method.toUpperCase(),
            "\"@path\": " + pq.path(),
            "\"@query\": " + pq.query(),
            "\"content-digest\": " + digest,
            "\"@signature-params\": " + sigParams);
    byte[] signature = signEd25519(privateKeyPem, sigBase.getBytes(StandardCharsets.UTF_8));
    Map<String, String> headers = new LinkedHashMap<>();
    headers.put("Content-Digest", digest);
    headers.put("Signature-Input", SIGNATURE_NAME + "=" + sigParams);
    headers.put(
        "Signature",
        SIGNATURE_NAME + "=:" + Base64.getEncoder().encodeToString(signature) + ":");
    return headers;
  }

  /** Copy {@code headers} and overlay RFC 9421 signing headers. */
  public static Map<String, String> mergeSignedHeaders(
      Map<String, String> headers,
      String method,
      String url,
      byte[] body,
      byte[] privateKeyPem,
      String keyId) {
    Map<String, String> out = new LinkedHashMap<>();
    if (headers != null) {
      out.putAll(headers);
    }
    out.putAll(signHeaders(method, url, body, privateKeyPem, keyId));
    return out;
  }

  /** Split path and RFC 9421 {@code @query} ({@code "?" + rawQuery}, even when empty). */
  public static PathAndQuery pathAndQuery(String urlOrPath) {
    URI uri = URI.create(urlOrPath);
    String path = uri.getRawPath();
    if (path == null || path.isEmpty()) {
      path = "/";
    }
    String rawQuery = uri.getRawQuery();
    String query = "?" + (rawQuery == null ? "" : rawQuery);
    return new PathAndQuery(path, query);
  }

  public record PathAndQuery(String path, String query) {}

  private static String randomNonce() {
    byte[] b = new byte[16];
    new SecureRandom().nextBytes(b);
    return HexFormat.of().formatHex(b);
  }

  private static byte[] signEd25519(byte[] privateKeyPem, byte[] message) {
    try {
      PrivateKey key = loadPrivateKey(privateKeyPem);
      Signature sig = Signature.getInstance("Ed25519");
      sig.initSign(key);
      sig.update(message);
      return sig.sign();
    } catch (Exception e) {
      throw new IllegalArgumentException("failed to sign with Ed25519 private key", e);
    }
  }

  static PrivateKey loadPrivateKey(byte[] privateKeyPem) throws Exception {
    String text = new String(privateKeyPem, StandardCharsets.US_ASCII);
    String b64 =
        text.replace("-----BEGIN PRIVATE KEY-----", "")
            .replace("-----END PRIVATE KEY-----", "")
            .replaceAll("\\s+", "");
    byte[] der = Base64.getDecoder().decode(b64);
    return KeyFactory.getInstance("Ed25519").generatePrivate(new PKCS8EncodedKeySpec(der));
  }
}
