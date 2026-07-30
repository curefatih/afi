package ai.afi.platform;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.nio.charset.StandardCharsets;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.Signature;
import java.util.Base64;
import java.util.Map;
import org.junit.jupiter.api.Test;

class HttpSignTest {

  private static byte[] pemPrivate(KeyPair pair) {
    String b64 = Base64.getMimeEncoder(64, new byte[] {'\n'}).encodeToString(pair.getPrivate().getEncoded());
    return ("-----BEGIN PRIVATE KEY-----\n" + b64 + "\n-----END PRIVATE KEY-----\n")
        .getBytes(StandardCharsets.US_ASCII);
  }

  private static KeyPair ed25519() throws Exception {
    return KeyPairGenerator.getInstance("Ed25519").generateKeyPair();
  }

  @Test
  void contentDigestShape() {
    String d = HttpSign.contentDigestSha256("hello".getBytes(StandardCharsets.UTF_8));
    assertTrue(d.startsWith("sha-256=:"));
    assertTrue(d.endsWith(":"));
  }

  @Test
  void signHeadersRoundtripVerifyBase() throws Exception {
    KeyPair pair = ed25519();
    byte[] pem = pemPrivate(pair);
    byte[] body = "{\"ping\":true}".getBytes(StandardCharsets.UTF_8);
    Map<String, String> headers =
        HttpSign.signHeaders(
            "POST",
            "http://localhost:8080/v1/chat/completions",
            body,
            pem,
            "local-signer",
            "n1",
            1_700_000_000L);
    assertTrue(headers.containsKey("Content-Digest"));
    assertTrue(headers.get("Signature-Input").startsWith("sig1="));
    assertTrue(headers.get("Signature").startsWith("sig1=:"));
    for (String c : HttpSign.REQUIRED_COMPONENTS) {
      assertTrue(headers.get("Signature-Input").contains(c));
    }

    String digest = headers.get("Content-Digest");
    String sigParams = headers.get("Signature-Input").substring("sig1=".length());
    String sigBase =
        String.join(
            "\n",
            "\"@method\": POST",
            "\"@path\": /v1/chat/completions",
            "\"@query\": ?",
            "\"content-digest\": " + digest,
            "\"@signature-params\": " + sigParams);
    String sigB64 =
        headers
            .get("Signature")
            .substring("sig1=:".length(), headers.get("Signature").length() - 1);
    byte[] raw = Base64.getDecoder().decode(sigB64);
    Signature verifier = Signature.getInstance("Ed25519");
    verifier.initVerify(pair.getPublic());
    verifier.update(sigBase.getBytes(StandardCharsets.UTF_8));
    assertTrue(verifier.verify(raw));
  }

  @Test
  void signHeadersWithQuery() throws Exception {
    KeyPair pair = ed25519();
    byte[] pem = pemPrivate(pair);
    Map<String, String> headers =
        HttpSign.signHeaders("GET", "/v1/models?foo=1", new byte[0], pem, "kid", "n", 1L);
    assertTrue(headers.get("Signature-Input").contains("@query"));
    String digest = headers.get("Content-Digest");
    String sigParams = headers.get("Signature-Input").substring("sig1=".length());
    String sigBase =
        String.join(
            "\n",
            "\"@method\": GET",
            "\"@path\": /v1/models",
            "\"@query\": ?foo=1",
            "\"content-digest\": " + digest,
            "\"@signature-params\": " + sigParams);
    String sigB64 =
        headers
            .get("Signature")
            .substring("sig1=:".length(), headers.get("Signature").length() - 1);
    Signature verifier = Signature.getInstance("Ed25519");
    verifier.initVerify(pair.getPublic());
    verifier.update(sigBase.getBytes(StandardCharsets.UTF_8));
    assertTrue(verifier.verify(Base64.getDecoder().decode(sigB64)));
  }

  @Test
  void pathAndQuery() {
    HttpSign.PathAndQuery abs = HttpSign.pathAndQuery("http://x/v1/chat?a=1");
    assertEquals("/v1/chat", abs.path());
    assertEquals("?a=1", abs.query());
    HttpSign.PathAndQuery empty = HttpSign.pathAndQuery("/v1/models");
    assertEquals("/v1/models", empty.path());
    assertEquals("?", empty.query());
  }

  @Test
  void loadRoundtripKeyFactory() throws Exception {
    KeyPair pair = ed25519();
    byte[] pem = pemPrivate(pair);
    var priv = HttpSign.loadPrivateKey(pem);
    assertTrue(priv.getAlgorithm().equals("Ed25519") || priv.getAlgorithm().equals("EdDSA"));
  }
}
