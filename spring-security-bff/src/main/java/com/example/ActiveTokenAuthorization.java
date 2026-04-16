package com.example;

import org.springframework.security.core.Authentication;
import org.springframework.security.oauth2.client.OAuth2AuthorizeRequest;
import org.springframework.security.oauth2.client.OAuth2AuthorizedClient;
import org.springframework.security.oauth2.client.authentication.OAuth2AuthenticationToken;
import org.springframework.security.oauth2.client.web.DefaultOAuth2AuthorizedClientManager;
import org.springframework.security.oauth2.core.OAuth2AuthorizationException;
import org.springframework.stereotype.Component;
import org.springframework.web.context.request.RequestAttributes;
import org.springframework.web.context.request.RequestContextHolder;
import org.springframework.web.context.request.ServletRequestAttributes;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

@Component("activeTokenAuthorization")
public class ActiveTokenAuthorization {

	static final String TOKEN_INACTIVE_ATTRIBUTE = ActiveTokenAuthorization.class.getName() + ".TOKEN_INACTIVE";

	private final DefaultOAuth2AuthorizedClientManager authorizedClientManager;

	private final TokenValidationService tokenValidationService;

	public ActiveTokenAuthorization(DefaultOAuth2AuthorizedClientManager authorizedClientManager,
			TokenValidationService tokenValidationService) {
		this.authorizedClientManager = authorizedClientManager;
		this.tokenValidationService = tokenValidationService;
	}

	public boolean hasActiveToken(Authentication authentication) {
		if (!(authentication instanceof OAuth2AuthenticationToken oauth2AuthenticationToken)) {
			return false;
		}

		ServletRequestAttributes requestAttributes = currentRequestAttributes();
		if (requestAttributes == null) {
			return false;
		}

		OAuth2AuthorizedClient authorizedClient;
		try {
			authorizedClient = this.authorizedClientManager.authorize(OAuth2AuthorizeRequest
				.withClientRegistrationId(oauth2AuthenticationToken.getAuthorizedClientRegistrationId())
				.principal(oauth2AuthenticationToken)
				.attribute(HttpServletRequest.class.getName(), requestAttributes.getRequest())
				.attribute(HttpServletResponse.class.getName(), requestAttributes.getResponse())
				.build());
		}
		catch (OAuth2AuthorizationException ex) {
			requestAttributes.getRequest().setAttribute(TOKEN_INACTIVE_ATTRIBUTE, Boolean.TRUE);
			return false;
		}

		boolean active = this.tokenValidationService.isActive(authorizedClient);
		if (!active) {
			requestAttributes.getRequest().setAttribute(TOKEN_INACTIVE_ATTRIBUTE, Boolean.TRUE);
		}
		return active;
	}

	private ServletRequestAttributes currentRequestAttributes() {
		RequestAttributes requestAttributes = RequestContextHolder.getRequestAttributes();
		if (requestAttributes instanceof ServletRequestAttributes servletRequestAttributes) {
			return servletRequestAttributes;
		}
		return null;
	}

}