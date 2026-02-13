package ch.rasc.methanol_server_demo.controller;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ThreadLocalRandom;
import java.util.concurrent.ConcurrentHashMap;
import java.util.zip.GZIPOutputStream;

import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;


import com.aayushatharva.brotli4j.encoder.Encoder;

import jakarta.servlet.http.HttpServletRequest;

@RestController
public class DemoController {

  private final Map<String, String> dataStore = new ConcurrentHashMap<>();

  // Simple GET returning JSON
  @GetMapping("/api/data")
  public Map<String, String> getData() {
    System.out.println("GET /api/data called");
    return Map.of("message", "Hello from GET", "status", "success");
  }

  // POST that accepts and returns JSON
  @PostMapping("/api/process")
  public ResponseEntity<?> processData(@RequestBody ProcessRequest request) {
    System.out.println("Received request: " + request);
    String fullName = request.firstName() + " " + request.lastName();
    this.dataStore.put("processed", fullName);
    return ResponseEntity.ok(Map.of("result", fullName, "status", "processed",
        "nameLength", fullName.length()));
  }

  // GET with GZIP compressed response
  @GetMapping("/api/compressed/gzip")
  public ResponseEntity<byte[]> getGzipContent(HttpServletRequest request)
      throws IOException {
    String response = "This is the response content for GZIP check";
    String acceptEncoding = request.getHeader("Accept-Encoding");
    System.out.println("Accept-Encoding: " + acceptEncoding);

    if (acceptEncoding != null && acceptEncoding.contains("gzip")) {
      ByteArrayOutputStream baos = new ByteArrayOutputStream();
      try (GZIPOutputStream gzipOut = new GZIPOutputStream(baos)) {
        gzipOut.write(response.getBytes(StandardCharsets.UTF_8));
      }
      return ResponseEntity.ok().header("Content-Encoding", "gzip")
          .contentType(MediaType.TEXT_PLAIN).body(baos.toByteArray());
    }
    return ResponseEntity.ok(response.getBytes(StandardCharsets.UTF_8));
  }

  // GET with Brotli compressed response
  @GetMapping("/api/compressed/brotli")
  public ResponseEntity<byte[]> getBrotliContent(HttpServletRequest request)
      throws IOException {
    String response = "This is the response content for Brotli check";
    String acceptEncoding = request.getHeader("Accept-Encoding");
    System.out.println("Accept-Encoding: " + acceptEncoding);

    if (acceptEncoding != null && acceptEncoding.contains("br")) {
      byte[] compressedResponse = Encoder.compress(response.getBytes(StandardCharsets.UTF_8));
      return ResponseEntity.ok().header("Content-Encoding", "br")
          .contentType(MediaType.TEXT_PLAIN).body(compressedResponse);
    }
    return ResponseEntity.ok(response.getBytes(StandardCharsets.UTF_8));
  }

  // API key protected endpoint
  @GetMapping("/api/secret")
  public ResponseEntity<String> handleSecretRequest(
      @RequestHeader(name = "X-API-Key", required = false) String apiKey) {
    if (!"secret".equals(apiKey)) {
      return ResponseEntity.status(HttpStatus.UNAUTHORIZED).body("Invalid API key");
    }
    return ResponseEntity.ok("the secret");
  }

  // POST with multipart file upload and additional parameters
  @PostMapping("/api/upload")
  public ResponseEntity<?> handleFileUpload(@RequestParam("file") MultipartFile file,
      @RequestParam String description, @RequestParam List<String> tags) {
    try {
      System.out.println("Received file: " + file.getOriginalFilename());
      System.out.println("Description: " + description);
      System.out.println("Tags: " + tags);
      
      byte[] bytes = file.getBytes();
      return ResponseEntity.ok(Map.of("filename", file.getOriginalFilename(), "size",
          bytes.length, "contentType", file.getContentType(), "description", description,
          "tags", tags, "tagCount", tags.size()));
    }
    catch (IOException e) {
      return ResponseEntity.badRequest().body(Map.of("error", "File processing failed"));
    }
  }

  // POST with x-www-form-urlencoded
  @PostMapping("/api/form")
  public ResponseEntity<?> handleFormSubmission(@RequestParam String name,
      @RequestParam String email) {
    System.out.println("Received name: " + name);
    System.out.println("Received email: " + email);
    return ResponseEntity.ok(
        Map.of("receivedName", name, "receivedEmail", email, "status", "form processed"));
  }

  // GET endpoint that fails around 80% of requests
  @GetMapping("/api/flaky")
  public ResponseEntity<?> flaky() {
    boolean shouldFail = ThreadLocalRandom.current().nextInt(10) < 8;
    if (shouldFail) {
      return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR)
          .body(Map.of("status", "error", "message", "Transient server failure"));
    }

    return ResponseEntity.ok(
        Map.of("status", "success", "message", "Flaky endpoint response"));
  }
}
