# ai.afi:platform-client

Thin Java client for the AFI control plane (`/api/v1/platform`).

Distributed via **GitHub Packages**:
https://github.com/curefatih/afi/packages

```bash
cd clients/java && mvn test
```

### Install (Maven)

Add the GitHub Packages repository and dependency (needs a PAT with `read:packages`):

```xml
<repositories>
  <repository>
    <id>github</id>
    <url>https://maven.pkg.github.com/curefatih/afi</url>
  </repository>
</repositories>

<dependency>
  <groupId>ai.afi</groupId>
  <artifactId>platform-client</artifactId>
  <version>1.0.0</version>
</dependency>
```

`~/.m2/settings.xml`:

```xml
<servers>
  <server>
    <id>github</id>
    <username>YOUR_GITHUB_USERNAME</username>
    <password>YOUR_PAT_WITH_READ_PACKAGES</password>
  </server>
</servers>
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
