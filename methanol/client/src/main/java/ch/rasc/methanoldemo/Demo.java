package ch.rasc.methanoldemo;

import java.io.IOException;
import java.net.http.HttpResponse;
import java.net.http.HttpResponse.BodyHandlers;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import java.util.Map;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.mizosoft.methanol.AdapterCodec;
import com.github.mizosoft.methanol.FormBodyPublisher;
import com.github.mizosoft.methanol.MediaType;
import com.github.mizosoft.methanol.Methanol;
import com.github.mizosoft.methanol.MultipartBodyPublisher;
import com.github.mizosoft.methanol.MutableRequest;
import com.github.mizosoft.methanol.adapter.jackson.JacksonAdapterFactory;

public class Demo {
  private static final ObjectMapper objectMapper = new ObjectMapper();

  private static final Methanol client = Methanol.newBuilder()
      .baseUri("http://localhost:8080")
      .adapterCodec(AdapterCodec.newBuilder().basic()
          .encoder(JacksonAdapterFactory.createJsonEncoder(objectMapper))
          .decoder(JacksonAdapterFactory.createJsonDecoder(objectMapper)).build())
      .cookieHandler(new java.net.CookieManager()).connectTimeout(Duration.ofSeconds(10))
      .readTimeout(Duration.ofSeconds(20)).build();

  @JsonIgnoreProperties(ignoreUnknown = true)
  public record DataResponse(String message, String status) {
  }

  @JsonIgnoreProperties(ignoreUnknown = true)
  public record ProcessRequest(String firstName, String lastName) {
  }

  public static void main(String[] args) throws IOException, InterruptedException {
    getDataDemoString();
    getDataDemo();
    processDataDemo();
    getGzipContentDemo();
    getBrotliContentDemo();
    secretRequestDemo();
    fileUploadDemo();
    formSubmissionDemo();
    timeoutConfigDemo();
    cookieManagementDemo();
  }

  // GET /api/data - String
  private static void getDataDemoString() throws IOException, InterruptedException {
    MutableRequest request = MutableRequest.GET("/api/data").header("X-Request-ID",
        "12345");
    HttpResponse<String> response = client.send(request, BodyHandlers.ofString());
    System.out.println("GET /api/data:\n" + response.body());
  }

  // GET /api/data - JSON response mapped to object
  private static void getDataDemo() throws IOException, InterruptedException {
    HttpResponse<DataResponse> response = client.send(MutableRequest.GET("/api/data"),
        DataResponse.class);
    DataResponse data = response.body();
    System.out.println("GET /api/data:");
    System.out.println("Message: " + data.message());
    System.out.println("Status: " + data.status());
  }

  // POST /api/process - Object mapping with Jackson
  private static void processDataDemo() throws IOException, InterruptedException {
    ProcessRequest requestBody = new ProcessRequest("John", "Doe");
    HttpResponse<Map> response = client.send(
        MutableRequest.POST("/api/process", requestBody, MediaType.APPLICATION_JSON),
        Map.class);

    Map<String, Object> result = response.body();
    System.out.println("\nPOST /api/process:");
    System.out.println("Full Name: " + result.get("result"));
    System.out.println("Status: " + result.get("status"));
    System.out.println("Name Length: " + result.get("nameLength"));
  }

  // GET /api/compressed/gzip - Handle compressed response
  private static void getGzipContentDemo() throws IOException, InterruptedException {
    HttpResponse<String> response = client.send(
        MutableRequest.GET("/api/compressed/gzip").header("Accept-Encoding", "gzip"),
        BodyHandlers.ofString());
    System.out.println("\nGET /gzip:\n" + response.body());
  }

  // GET /api/compressed/brotli - Handle compressed response
  private static void getBrotliContentDemo() throws IOException, InterruptedException {
    HttpResponse<String> response = client
        .send(MutableRequest.GET("/api/compressed/brotli"), BodyHandlers.ofString());
    System.out.println("\nGET /brotli:\n" + response.body());
  }

  // GET /api/secret - API key protected
  private static void secretRequestDemo() throws IOException, InterruptedException {
    HttpResponse<String> response = client.send(
        MutableRequest.GET("/api/secret").header("X-API-Key", "secret"),
        BodyHandlers.ofString());
    System.out.println("\nGET /secret:\n" + response.body());
  }

  // POST /api/upload - Multipart file upload
  private static void fileUploadDemo() throws IOException, InterruptedException {
    Path tempFile = Files.createTempFile("upload", ".txt");
    Files.writeString(tempFile, "Test file content");

    MultipartBodyPublisher multipartBody = MultipartBodyPublisher.newBuilder()
        .filePart("file", tempFile).textPart("description", "Demo upload")
        .textPart("tags", "demo").textPart("tags", "test").build();

    HttpResponse<String> response = client
        .send(MutableRequest.POST("/api/upload", multipartBody), BodyHandlers.ofString());
    System.out.println("\nPOST /upload:\n" + response.body());
  }

  // POST /api/form - Form-urlencoded
  private static void formSubmissionDemo() throws IOException, InterruptedException {
    FormBodyPublisher.Builder builder = FormBodyPublisher.newBuilder();
    builder.query("name", "Jane Doe");
    builder.query("email", "jane@example.com").build();

    HttpResponse<String> response = client
        .send(MutableRequest.POST("/api/form", builder.build()), BodyHandlers.ofString());
    System.out.println("\nPOST /form:\n" + response.body());
  }

  // Timeout configuration demo
  private static void timeoutConfigDemo() throws InterruptedException {
    try (Methanol timeoutClient = Methanol.newBuilder().baseUri("http://localhost:8080")
        .connectTimeout(java.time.Duration.ofMillis(500))
        .readTimeout(java.time.Duration.ofMillis(1000)).build()) {

      try {
        timeoutClient.send(MutableRequest.GET("/api/delay"), BodyHandlers.ofString());
      }
      catch (IOException e) {
        System.out.println("\nTimeout error: " + e.getMessage());
      }
    }
  }

  // Cookie management demo
  private static void cookieManagementDemo() throws IOException, InterruptedException {
    // First request to set cookie
    client.send(MutableRequest.GET("/api/cookie/set"), BodyHandlers.discarding());

    // Subsequent request that sends cookie
    var response = client.send(MutableRequest.GET("/api/cookie/get"),
        BodyHandlers.ofString());
    System.out.println("\nCookie value: " + response.body());
  }
}
