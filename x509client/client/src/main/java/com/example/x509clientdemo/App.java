package com.example.x509clientdemo;

import java.io.FileInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpRequest.BodyPublishers;
import java.net.http.HttpResponse;
import java.net.http.HttpResponse.BodyHandlers;
import java.security.KeyStore;
import java.security.SecureRandom;
import java.time.Duration;
import java.util.Properties;

import javax.net.ssl.KeyManagerFactory;
import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManagerFactory;

import com.fasterxml.jackson.databind.ObjectMapper;

public class App {

  record UpdateRequest(String action, String value, long timestamp) {
  }

  private final Properties properties;
  private final String serverBaseUrl;
  private final String clientKeystorePath;
  private final String clientKeystorePassword;
  private final String truststorePath;
  private final String truststorePassword;
  private final int connectionTimeout;
  private final ObjectMapper objectMapper = new ObjectMapper();

  public static void main(String[] args) {
    try {
      App client = new App();
      client.runDemo();
    }
    catch (Exception e) {
      System.err.println("Error running client demo: " + e.getMessage());
      e.printStackTrace();
    }
  }

  public App() throws IOException {
    this.properties = loadProperties();
    this.serverBaseUrl = this.properties.getProperty("client.server.base-url");
    this.clientKeystorePath = this.properties.getProperty("client.ssl.keystore.path");
    this.clientKeystorePassword = this.properties
        .getProperty("client.ssl.keystore.password");
    this.truststorePath = this.properties.getProperty("client.ssl.truststore.path");
    this.truststorePassword = this.properties
        .getProperty("client.ssl.truststore.password");
    this.connectionTimeout = Integer
        .parseInt(this.properties.getProperty("client.connection.timeout"));
  }

  private static Properties loadProperties() throws IOException {
    Properties props = new Properties();
    try (InputStream input = App.class.getClassLoader()
        .getResourceAsStream("application.properties")) {
      if (input == null) {
        throw new IOException("Unable to find application.properties");
      }
      props.load(input);
    }
    return props;
  }

  public void runDemo() throws Exception {
    SSLContext sslContext = createSSLContext();
    try (HttpClient httpClient = HttpClient.newBuilder().sslContext(sslContext)
        .connectTimeout(Duration.ofSeconds(this.connectionTimeout)).build()) {

      testPublicEndpoint(httpClient);
      testSecureGetEndpoint(httpClient);
      testSecurePostEndpoint(httpClient);
    }

    SSLContext sslContextWithoutClientCertificate = createSSLContextWithoutClientCertificate();
    try (HttpClient httpClient = HttpClient.newBuilder()
        .sslContext(sslContextWithoutClientCertificate)
        .connectTimeout(Duration.ofSeconds(this.connectionTimeout)).build()) {
      testPublicEndpoint(httpClient);
    }
  }

  private SSLContext createSSLContext() throws Exception {
    // Client certificate
    KeyStore clientKeyStore = KeyStore.getInstance("PKCS12");
    try (FileInputStream fis = new FileInputStream(this.clientKeystorePath)) {
      clientKeyStore.load(fis, this.clientKeystorePassword.toCharArray());
    }

    KeyManagerFactory kmf = KeyManagerFactory
        .getInstance(KeyManagerFactory.getDefaultAlgorithm());
    kmf.init(clientKeyStore, this.clientKeystorePassword.toCharArray());

    // Trust store with CA certificate
    KeyStore trustStore = KeyStore.getInstance("PKCS12");
    try (FileInputStream fis = new FileInputStream(this.truststorePath)) {
      trustStore.load(fis, this.truststorePassword.toCharArray());
    }

    TrustManagerFactory tmf = TrustManagerFactory
        .getInstance(TrustManagerFactory.getDefaultAlgorithm());
    tmf.init(trustStore);

    SSLContext sslContext = SSLContext.getInstance("TLS");
    sslContext.init(kmf.getKeyManagers(), tmf.getTrustManagers(), new SecureRandom());

    return sslContext;
  }

  private SSLContext createSSLContextWithoutClientCertificate() throws Exception {
    // Trust store with CA certificates
    KeyStore trustStore = KeyStore.getInstance("PKCS12");
    try (FileInputStream fis = new FileInputStream(this.truststorePath)) {
      trustStore.load(fis, this.truststorePassword.toCharArray());
    }

    TrustManagerFactory tmf = TrustManagerFactory
        .getInstance(TrustManagerFactory.getDefaultAlgorithm());
    tmf.init(trustStore);

    SSLContext sslContext = SSLContext.getInstance("TLS");
    sslContext.init(null, tmf.getTrustManagers(), new SecureRandom());

    return sslContext;
  }

  private void testPublicEndpoint(HttpClient httpClient) {
    System.out.println("1. Testing public endpoint (no authentication required):");
    try {
      HttpRequest request = HttpRequest.newBuilder()
          .uri(URI.create(this.serverBaseUrl + "/public/health")).GET().build();

      HttpResponse<String> response = httpClient.send(request, BodyHandlers.ofString());

      System.out.println("   Status: " + response.statusCode());
      System.out.println("   Response: " + response.body());
      System.out.println();
    }
    catch (Exception e) {
      System.err.println("   Error: " + e);
      System.out.println();
    }
  }

  private void testSecureGetEndpoint(HttpClient httpClient) {
    System.out.println("2. Testing secure GET endpoint (with X.509 cert):");
    try {
      HttpRequest request = HttpRequest.newBuilder()
          .uri(URI.create(this.serverBaseUrl + "/secure/data"))
          .header("Accept", "application/json").GET().build();

      HttpResponse<String> response = httpClient.send(request, BodyHandlers.ofString());

      System.out.println("   Status: " + response.statusCode());
      System.out.println("   Response: " + response.body());
      System.out.println();
    }
    catch (Exception e) {
      System.err.println("   Error: " + e);
      System.out.println();
    }
  }

  private void testSecurePostEndpoint(HttpClient httpClient) {
    System.out.println("3. Testing secure POST endpoint (with X.509 cert):");
    try {
      UpdateRequest requestData = new UpdateRequest("update", "test data",
          System.currentTimeMillis());
      String jsonBody = this.objectMapper.writeValueAsString(requestData);

      HttpRequest request = HttpRequest.newBuilder()
          .uri(URI.create(this.serverBaseUrl + "/secure/update"))
          .header("Content-Type", "application/json").header("Accept", "application/json")
          .POST(BodyPublishers.ofString(jsonBody)).build();

      HttpResponse<String> response = httpClient.send(request, BodyHandlers.ofString());

      System.out.println("   Status: " + response.statusCode());
      System.out.println("   Response: " + response.body());
      System.out.println();
    }
    catch (Exception e) {
      System.err.println("   Error: " + e);
      System.out.println();
    }
  }

}
