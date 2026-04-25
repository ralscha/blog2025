package ch.rasc.timefold.demo.web;

import static org.assertj.core.api.Assertions.assertThat;

import java.net.ServerSocket;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.List;

import org.junit.jupiter.api.Test;

class FestivalScheduleServerTest {

  @Test
  void servesIndexAndScheduleEndpoints() throws Exception {
    ScheduleResponse response = new ScheduleResponse("0hard/0soft", 0, 0, List.of(), List.of(), List.of());
    int port = findOpenPort();

    FestivalScheduleServer server = new FestivalScheduleServer(port, response);
    HttpClient client = HttpClient.newHttpClient();
    try {
      server.start();

      HttpResponse<String> indexResponse = client.send(
          HttpRequest.newBuilder(URI.create(rootUri(port))).GET().timeout(Duration.ofSeconds(5)).build(),
          HttpResponse.BodyHandlers.ofString());
      assertThat(indexResponse.statusCode()).isEqualTo(200);
      assertThat(indexResponse.headers().firstValue("content-type"))
          .hasValueSatisfying(value -> assertThat(value).contains("text/html"));

      HttpResponse<String> scheduleResponse = client.send(
          HttpRequest.newBuilder(URI.create(apiUri(port))).GET().timeout(Duration.ofSeconds(5)).build(),
          HttpResponse.BodyHandlers.ofString());
      assertThat(scheduleResponse.statusCode()).isEqualTo(200);
      assertThat(scheduleResponse.headers().firstValue("content-type"))
          .hasValueSatisfying(value -> assertThat(value).contains("application/json"));
      assertThat(scheduleResponse.body()).contains("\"score\":\"0hard/0soft\"");

      HttpResponse<String> notFoundResponse = client.send(
          HttpRequest.newBuilder(URI.create(rootUri(port) + "missing")).GET().timeout(Duration.ofSeconds(5)).build(),
          HttpResponse.BodyHandlers.ofString());
      assertThat(notFoundResponse.statusCode()).isEqualTo(404);

      HttpResponse<String> methodNotAllowedResponse = client.send(HttpRequest.newBuilder(URI.create(apiUri(port)))
          .POST(HttpRequest.BodyPublishers.noBody()).timeout(Duration.ofSeconds(5)).build(),
          HttpResponse.BodyHandlers.ofString());
      assertThat(methodNotAllowedResponse.statusCode()).isEqualTo(405);
    } finally {
      server.stop();
    }
  }

  private static String apiUri(int port) {
    return rootUri(port) + "api/schedule";
  }

  private static int findOpenPort() throws Exception {
    try (ServerSocket socket = new ServerSocket(0)) {
      return socket.getLocalPort();
    }
  }

  private static String rootUri(int port) {
    return "http://127.0.0.1:" + port + "/";
  }
}