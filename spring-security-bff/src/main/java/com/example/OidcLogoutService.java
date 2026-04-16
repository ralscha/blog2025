package com.example;

import java.util.Map;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseCookie;
import org.springframework.security.core.Authentication;
import org.springframework.security.oauth2.client.OAuth2AuthorizedClient;
import org.springframework.security.oauth2.client.authentication.OAuth2AuthenticationToken;
import org.springframework.security.oauth2.client.registration.ClientRegistration;
import org.springframework.security.oauth2.client.registration.ClientRegistrationRepository;
import org.springframework.security.oauth2.client.web.OAuth2AuthorizedClientRepository;
import org.springframework.security.oauth2.core.ClientAuthenticationMethod;
import org.springframework.security.oauth2.core.oidc.user.OidcUser;
import org.springframework.security.web.authentication.logout.SecurityContextLogoutHandler;
import org.springframework.stereotype.Service;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import org.springframework.util.StringUtils;
import org.springframework.web.client.RestClient;
import org.springframework.web.client.RestClientException;
import org.springframework.web.client.RestClientResponseException;
import org.springframework.web.servlet.support.ServletUriComponentsBuilder;
import org.springframework.web.util.UriComponentsBuilder;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

@Service
public class OidcLogoutService {

	private static final Logger log = LoggerFactory.getLogger(OidcLogoutService.class);

	private final ClientRegistrationRepository clientRegistrationRepository;

	private final OAuth2AuthorizedClientRepository authorizedClientRepository;

	private final RestClient restClient;

	private final String sessionCookieName;

	private final SecurityContextLogoutHandler logoutHandler = new SecurityContextLogoutHandler();

	public OidcLogoutService(ClientRegistrationRepository clientRegistrationRepository,
			OAuth2AuthorizedClientRepository authorizedClientRepository, RestClient.Builder restClientBuilder,
			@Value("${server.servlet.session.cookie.name:JSESSIONID}") String sessionCookieName) {
		this.clientRegistrationRepository = clientRegistrationRepository;
		this.authorizedClientRepository = authorizedClientRepository;
		this.restClient = restClientBuilder.build();
		this.sessionCookieName = sessionCookieName;
		this.logoutHandler.setInvalidateHttpSession(true);
		this.logoutHandler.setClearAuthentication(true);
	}

	public Map<String, Object> logout(Authentication authentication, HttpServletRequest request,
			HttpServletResponse response) {
		String redirectUrl = buildSignedOutUrl(request);
		boolean providerLogout = false;

		if (authentication instanceof OAuth2AuthenticationToken oauth2AuthenticationToken) {
			String registrationId = oauth2AuthenticationToken.getAuthorizedClientRegistrationId();
			ClientRegistration clientRegistration = this.clientRegistrationRepository
				.findByRegistrationId(registrationId);
			OAuth2AuthorizedClient authorizedClient = this.authorizedClientRepository
				.loadAuthorizedClient(registrationId, oauth2AuthenticationToken, request);

			revokeTokensIfSupported(clientRegistration, authorizedClient);
			this.authorizedClientRepository.removeAuthorizedClient(registrationId, oauth2AuthenticationToken, request,
					response);

			String providerLogoutUrl = buildProviderLogoutUrl(request, clientRegistration, oauth2AuthenticationToken);
			if (providerLogoutUrl != null) {
				redirectUrl = providerLogoutUrl;
				providerLogout = true;
			}
		}

		this.logoutHandler.logout(request, response, authentication);
		expireSessionCookie(request, response);

		return Map.of("redirectUrl", redirectUrl, "providerLogout", providerLogout);
	}

	private void revokeTokensIfSupported(ClientRegistration clientRegistration,
			OAuth2AuthorizedClient authorizedClient) {
		if (clientRegistration == null || authorizedClient == null) {
			return;
		}

		String revocationEndpoint = getConfigurationValue(clientRegistration, "revocation_endpoint");
		if (!StringUtils.hasText(revocationEndpoint)) {
			return;
		}

		var refreshToken = authorizedClient.getRefreshToken();
		if (refreshToken != null) {
			revokeToken(revocationEndpoint, clientRegistration, refreshToken.getTokenValue(), "refresh_token");
		}

		var accessToken = authorizedClient.getAccessToken();
		if (accessToken != null) {
			revokeToken(revocationEndpoint, clientRegistration, accessToken.getTokenValue(), "access_token");
		}
	}

	private void revokeToken(String revocationEndpoint, ClientRegistration clientRegistration, String tokenValue,
			String tokenTypeHint) {
		if (!StringUtils.hasText(tokenValue)) {
			return;
		}

		MultiValueMap<String, String> form = new LinkedMultiValueMap<>();
		form.add("token", tokenValue);
		form.add("token_type_hint", tokenTypeHint);
		appendClientAuthentication(form, clientRegistration);

		try {
			this.restClient.post()
				.uri(revocationEndpoint)
				.contentType(MediaType.APPLICATION_FORM_URLENCODED)
				.headers(headers -> applyClientAuthentication(headers, clientRegistration))
				.body(form)
				.retrieve()
				.toBodilessEntity();
		}
		catch (RestClientResponseException ex) {
			log.warn("Token revocation failed with status {} for {}", ex.getStatusCode(), tokenTypeHint);
		}
		catch (RestClientException ex) {
			log.warn("Token revocation failed for {}", tokenTypeHint, ex);
		}
	}

	private void appendClientAuthentication(MultiValueMap<String, String> form, ClientRegistration clientRegistration) {
		ClientAuthenticationMethod authenticationMethod = clientRegistration.getClientAuthenticationMethod();
		if (ClientAuthenticationMethod.CLIENT_SECRET_POST.equals(authenticationMethod)) {
			form.add("client_id", clientRegistration.getClientId());
			form.add("client_secret", clientRegistration.getClientSecret());
		}
		else if (ClientAuthenticationMethod.NONE.equals(authenticationMethod)) {
			form.add("client_id", clientRegistration.getClientId());
		}
	}

	private void applyClientAuthentication(HttpHeaders headers, ClientRegistration clientRegistration) {
		ClientAuthenticationMethod authenticationMethod = clientRegistration.getClientAuthenticationMethod();
		if (ClientAuthenticationMethod.CLIENT_SECRET_BASIC.equals(authenticationMethod)) {
			headers.setBasicAuth(clientRegistration.getClientId(), clientRegistration.getClientSecret());
		}
	}

	private String buildProviderLogoutUrl(HttpServletRequest request, ClientRegistration clientRegistration,
			OAuth2AuthenticationToken authentication) {
		if (clientRegistration == null || !(authentication.getPrincipal() instanceof OidcUser oidcUser)
				|| oidcUser.getIdToken() == null) {
			return null;
		}

		String endSessionEndpoint = getConfigurationValue(clientRegistration, "end_session_endpoint");
		if (!StringUtils.hasText(endSessionEndpoint)) {
			return null;
		}

		return UriComponentsBuilder.fromUriString(endSessionEndpoint)
			.queryParam("id_token_hint", oidcUser.getIdToken().getTokenValue())
			.queryParam("post_logout_redirect_uri", buildSignedOutUrl(request))
			.build()
			.encode()
			.toUriString();
	}

	private String buildSignedOutUrl(HttpServletRequest request) {
		return ServletUriComponentsBuilder.fromRequestUri(request)
			.replacePath(request.getContextPath())
			.replaceQuery(null)
			.path("/signed-out.html")
			.build()
			.toUriString();
	}

	private String getConfigurationValue(ClientRegistration clientRegistration, String key) {
		Object value = clientRegistration.getProviderDetails().getConfigurationMetadata().get(key);
		if (value instanceof String stringValue && StringUtils.hasText(stringValue)) {
			return stringValue;
		}
		return null;
	}

	private void expireSessionCookie(HttpServletRequest request, HttpServletResponse response) {
		ResponseCookie expiredCookie = ResponseCookie.from(this.sessionCookieName, "")
			.httpOnly(true)
			.secure(request.isSecure())
			.path("/")
			.maxAge(0)
			.sameSite("Lax")
			.build();
		response.addHeader(HttpHeaders.SET_COOKIE, expiredCookie.toString());
	}

}