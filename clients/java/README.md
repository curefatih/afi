# ai.afi:platform-client

Thin Java client for the AFI control plane (`/api/v1/platform`).

```bash
cd clients/java && mvn test
```

```xml
<dependency>
  <groupId>ai.afi</groupId>
  <artifactId>platform-client</artifactId>
  <version>1.0.0</version>
</dependency>
```

```java
import ai.afi.platform.PlatformClient;

try (PlatformClient client = new PlatformClient("http://localhost:8081")) {
  String token = client.login("admin@example.com", "secret").path("token").asText();
  PlatformClient authed = new PlatformClient("http://localhost:8081", () -> token);
  System.out.println(authed.listOrganizations());
}
```

Contract: [`../../api/openapi/platform.openapi.yaml`](../../api/openapi/platform.openapi.yaml).

## Gateway signed requests

```java
import ai.afi.platform.HttpSign;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.HashMap;
import java.util.Map;

byte[] pem = Files.readAllBytes(Path.of("signer.pem"));
Map<String, String> headers = new HashMap<>();
headers.put("Content-Type", "application/json");
headers.putAll(
    HttpSign.signHeaders(
        "POST",
        "http://localhost:8080/v1/chat/completions",
        body,
        pem,
        "local-signer"));
```
