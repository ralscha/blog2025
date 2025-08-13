package com.example.x509clientdemo.controller;

import java.security.Principal;
import java.util.Collection;
import java.util.Map;

import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api")
public class SecureController {

  record HealthResponse(String status, String message) {
  }

  @GetMapping("/public/health")
  public ResponseEntity<HealthResponse> health() {
    var response = new HealthResponse("UP",
        "Public endpoint - no authentication required");
    return ResponseEntity.ok(response);
  }

  record UserData(String userId, double balance, String lastLogin) {
  }

  record SecureDataResponse(String message, String clientCertificate,
      Collection<? extends GrantedAuthority> authorities, boolean authenticated,
      long timestamp, UserData data) {
  }

  @GetMapping("/secure/data")
  public ResponseEntity<SecureDataResponse> getSecureData(Authentication authentication,
      Principal principal) {

    var userData = new UserData("12345", 1500.75, "2025-08-12T10:30:00Z");

    var response = new SecureDataResponse("Access granted to secure endpoint",
        principal.getName(), authentication.getAuthorities(),
        authentication.isAuthenticated(), System.currentTimeMillis(), userData);

    return ResponseEntity.ok(response);
  }

  record UpdateDataResponse(String message, String clientCertificate,
      Map<String, Object> receivedData, long timestamp) {
  }

  @PostMapping("/secure/update")
  public ResponseEntity<UpdateDataResponse> updateData(
      @RequestBody Map<String, Object> requestData, Principal principal) {

    var response = new UpdateDataResponse("Data updated successfully",
        principal.getName(), requestData, System.currentTimeMillis());

    return ResponseEntity.ok(response);
  }
}
