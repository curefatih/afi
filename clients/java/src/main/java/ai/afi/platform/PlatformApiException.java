package ai.afi.platform;

/** Error from the AFI Platform HTTP API. */
public final class PlatformApiException extends RuntimeException {
  private final int status;
  private final Object body;

  public PlatformApiException(String message, int status) {
    this(message, status, null);
  }

  public PlatformApiException(String message, int status, Object body) {
    super(message);
    this.status = status;
    this.body = body;
  }

  public int status() {
    return status;
  }

  public Object body() {
    return body;
  }
}
