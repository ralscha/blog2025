package com.example;

import java.time.Instant;

import org.springframework.security.oauth2.client.OAuth2AuthorizedClient;
import org.springframework.security.oauth2.core.OAuth2AccessToken;
import org.springframework.security.oauth2.core.OAuth2AuthenticationException;
import org.springframework.security.oauth2.server.resource.introspection.OpaqueTokenIntrospector;
import org.springframework.stereotype.Service;

@Service
public class TokenValidationService {

	private final OpaqueTokenIntrospector opaqueTokenIntrospector;

	public TokenValidationService(OpaqueTokenIntrospector opaqueTokenIntrospector) {
		this.opaqueTokenIntrospector = opaqueTokenIntrospector;
	}

	public boolean isActive(OAuth2AuthorizedClient authorizedClient) {
		if (authorizedClient == null) {
			return false;
		}

		OAuth2AccessToken accessToken = authorizedClient.getAccessToken();
		if (accessToken == null) {
			return false;
		}

		Instant expiresAt = accessToken.getExpiresAt();
		if (expiresAt != null && expiresAt.isBefore(Instant.now().plusSeconds(5))) {
			return false;
		}

		try {
			this.opaqueTokenIntrospector.introspect(accessToken.getTokenValue());
			return true;
		}
		catch (OAuth2AuthenticationException ex) {
			return false;
		}
	}

}