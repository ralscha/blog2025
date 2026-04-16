package com.example;

import java.time.Instant;
import java.util.Map;

import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.security.oauth2.client.authentication.OAuth2AuthenticationToken;
import org.springframework.security.oauth2.core.oidc.user.OidcUser;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/protected")
@RequireActiveToken
public class ProtectedController {

	@GetMapping
	public ResponseEntity<Map<String, Object>> protectedResource(Authentication authentication) {
		OAuth2AuthenticationToken oauth2AuthenticationToken = (OAuth2AuthenticationToken) authentication;
		OidcUser oidcUser = (OidcUser) oauth2AuthenticationToken.getPrincipal();
		return ResponseEntity.ok(Map.of("message",
				"Protected resource returned by the BFF without any bearer token from the browser.", "subject",
				oidcUser.getSubject(), "email", oidcUser.getEmail(), "servedAt", Instant.now().toString()));
	}

}