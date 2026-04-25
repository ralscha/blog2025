package ch.rasc.timefold.demo.web;

import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.Objects;

import io.fusionauth.http.HTTPMethod;
import io.fusionauth.http.server.HTTPHandler;
import io.fusionauth.http.server.HTTPListenerConfiguration;
import io.fusionauth.http.server.HTTPRequest;
import io.fusionauth.http.server.HTTPResponse;
import io.fusionauth.http.server.HTTPServer;

public final class FestivalScheduleServer {

  private final HTTPServer server;
  private final byte[] scheduleJson;
  private final byte[] indexHtml;

  public FestivalScheduleServer(int port, ScheduleResponse response) throws IOException {
    this.scheduleJson = ScheduleJsonWriter.toJson(response).getBytes(StandardCharsets.UTF_8);
    this.indexHtml = readClasspathResource("/web/index.html");
    this.server = new HTTPServer().withHandler(this::handleRequest).withListener(new HTTPListenerConfiguration(port));
  }

  public void start() {
    this.server.start();
  }

  public void stop() {
    this.server.close();
  }

  private void handleRequest(HTTPRequest request, HTTPResponse response) throws IOException {
    if (request.getMethod() != HTTPMethod.GET) {
      write(response, 405, "text/plain; charset=utf-8", "Method not allowed".getBytes(StandardCharsets.UTF_8));
      return;
    }

    switch (Objects.requireNonNullElse(request.getPath(), "/")) {
      case "/api/schedule" -> write(response, 200, "application/json; charset=utf-8", this.scheduleJson);
      case "/", "/index.html" -> write(response, 200, "text/html; charset=utf-8", this.indexHtml);
      default -> write(response, 404, "text/plain; charset=utf-8", "Not found".getBytes(StandardCharsets.UTF_8));
    }
  }

  private static void write(HTTPResponse response, int status, String contentType, byte[] body) throws IOException {
    response.setStatus(status);
    response.setContentType(contentType);
    response.setContentLength(body.length);
    try (var outputStream = response.getOutputStream()) {
      outputStream.write(body);
    }
  }

  private static byte[] readClasspathResource(String path) throws IOException {
    try (InputStream inputStream = FestivalScheduleServer.class.getResourceAsStream(path)) {
      if (inputStream == null) {
        throw new IOException("Missing classpath resource: " + path);
      }
      return inputStream.readAllBytes();
    }
  }
}