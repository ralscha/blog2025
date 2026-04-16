package com.example;

import java.util.LinkedHashMap;
import java.util.Map;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.core.Authentication;
import org.springframework.security.oauth2.client.OAuth2AuthorizeRequest;
import org.springframework.security.oauth2.client.OAuth2AuthorizedClient;
import org.springframework.security.oauth2.client.authentication.OAuth2AuthenticationToken;
import org.springframework.security.oauth2.client.web.DefaultOAuth2AuthorizedClientManager;
import org.springframework.security.oauth2.core.oidc.user.OidcUser;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

@RestController
@RequestMapping("/api")
public class BffController {

	private final DefaultOAuth2AuthorizedClientManager authorizedClientManager;

	private final OidcLogoutService oidcLogoutService;

	private final TokenValidationService tokenValidationService;

	private final String registrationId;

	public BffController(DefaultOAuth2AuthorizedClientManager authorizedClientManager,
			OidcLogoutService oidcLogoutService, TokenValidationService tokenValidationService,
			@Value("${app.security.registration-id}") String registrationId) {
		this.authorizedClientManager = authorizedClientManager;
		this.oidcLogoutService = oidcLogoutService;
		this.tokenValidationService = tokenValidationService;
		this.registrationId = registrationId;
	}

	@GetMapping("/me")
	public Map<String, Object> me(Authentication authentication, HttpServletRequest request,
			HttpServletResponse response) {
		if (!(authentication instanceof OAuth2AuthenticationToken oauth2AuthenticationToken)) {
			return Map.of("authenticated", false, "loginUrl", "/oauth2/authorization/" + this.registrationId);
		}

		OAuth2AuthorizedClient authorizedClient = authorizeClient(oauth2AuthenticationToken, request, response);
		OidcUser oidcUser = (OidcUser) oauth2AuthenticationToken.getPrincipal();

		Map<String, Object> state = new LinkedHashMap<>();
		state.put("authenticated", true);
		state.put("tokenActive", this.tokenValidationService.isActive(authorizedClient));
		state.put("name", oidcUser.getFullName() != null ? oidcUser.getFullName() : oidcUser.getPreferredUsername());
		state.put("email", oidcUser.getEmail());
		state.put("subject", oidcUser.getSubject());
		state.put("loginUrl", "/oauth2/authorization/" + this.registrationId);

		return state;
	}

	@PostMapping("/logout")
	public Map<String, Object> logout(Authentication authentication, HttpServletRequest request,
			HttpServletResponse response) {
		return this.oidcLogoutService.logout(authentication, request, response);
	}

	private OAuth2AuthorizedClient authorizeClient(OAuth2AuthenticationToken authentication, HttpServletRequest request,
			HttpServletResponse response) {
		return this.authorizedClientManager.authorize(
				OAuth2AuthorizeRequest.withClientRegistrationId(authentication.getAuthorizedClientRegistrationId())
					.principal(authentication)
					.attribute(HttpServletRequest.class.getName(), request)
					.attribute(HttpServletResponse.class.getName(), response)
					.build());
	}

}