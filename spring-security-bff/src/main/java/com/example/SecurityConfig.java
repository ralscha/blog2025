package com.example;

import java.io.IOException;
import java.util.Set;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.MediaType;
import org.springframework.security.config.Customizer;
import org.springframework.security.config.annotation.method.configuration.EnableMethodSecurity;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configurers.CsrfConfigurer;
import org.springframework.security.oauth2.client.OAuth2AuthorizedClientProvider;
import org.springframework.security.oauth2.client.OAuth2AuthorizedClientProviderBuilder;
import org.springframework.security.oauth2.client.registration.ClientRegistration;
import org.springframework.security.oauth2.client.registration.ClientRegistrationRepository;
import org.springframework.security.oauth2.client.web.DefaultOAuth2AuthorizationRequestResolver;
import org.springframework.security.oauth2.client.web.DefaultOAuth2AuthorizedClientManager;
import org.springframework.security.oauth2.client.web.OAuth2AuthorizationRequestCustomizers;
import org.springframework.security.oauth2.client.web.OAuth2AuthorizationRequestRedirectFilter;
import org.springframework.security.oauth2.client.web.OAuth2AuthorizationRequestResolver;
import org.springframework.security.oauth2.client.web.OAuth2AuthorizedClientRepository;
import org.springframework.security.oauth2.server.resource.introspection.OpaqueTokenIntrospector;
import org.springframework.security.oauth2.server.resource.introspection.SpringOpaqueTokenIntrospector;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.security.web.authentication.www.BasicAuthenticationFilter;
import org.springframework.web.client.RestClient;
import org.springframework.web.filter.OncePerRequestFilter;

import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

@Configuration
@EnableMethodSecurity
public class SecurityConfig {

	@Bean
	RestClient.Builder restClientBuilder() {
		return RestClient.builder();
	}

	@Bean
	SecurityFilterChain securityFilterChain(HttpSecurity http,
			OAuth2AuthorizationRequestResolver authorizationRequestResolver) throws Exception {
		http.authorizeHttpRequests(authorize -> authorize
			.requestMatchers("/", "/index.html", "/signed-out.html", "/app.css", "/app.js", "/error", "/api/me",
					"/api/logout")
			.permitAll()
			.requestMatchers("/api/protected", "/api/protected/**")
			.authenticated()
			.anyRequest()
			.authenticated())
			.oauth2Login(oauth2 -> oauth2
				.authorizationEndpoint(endpoint -> endpoint.authorizationRequestResolver(authorizationRequestResolver))
				.defaultSuccessUrl("/", true))
			.oauth2Client(Customizer.withDefaults())
			.exceptionHandling(
					exceptions -> exceptions.accessDeniedHandler((request, response, accessDeniedException) -> {
						if (Boolean.TRUE
							.equals(request.getAttribute(ActiveTokenAuthorization.TOKEN_INACTIVE_ATTRIBUTE))) {
							response.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
							response.setContentType(MediaType.APPLICATION_JSON_VALUE);
							response.getWriter()
								.write("{\"error\":\"token_inactive\",\"message\":\"The backend session exists, but the access token is no longer active.\"}");
							return;
						}
						response.sendError(HttpServletResponse.SC_FORBIDDEN, accessDeniedException.getMessage());
					}))
			.headers(headers -> headers.contentSecurityPolicy(csp -> csp.policyDirectives(
					"default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'")))
			.csrf(CsrfConfigurer::disable)
			.addFilterBefore(new FetchMetadataProtectionFilter(), BasicAuthenticationFilter.class);

		return http.build();
	}

	@Bean
	DefaultOAuth2AuthorizedClientManager authorizedClientManager(
			ClientRegistrationRepository clientRegistrationRepository,
			OAuth2AuthorizedClientRepository authorizedClientRepository) {
		OAuth2AuthorizedClientProvider authorizedClientProvider = OAuth2AuthorizedClientProviderBuilder.builder()
			.authorizationCode()
			.refreshToken()
			.build();

		DefaultOAuth2AuthorizedClientManager authorizedClientManager = new DefaultOAuth2AuthorizedClientManager(
				clientRegistrationRepository, authorizedClientRepository);
		authorizedClientManager.setAuthorizedClientProvider(authorizedClientProvider);
		return authorizedClientManager;
	}

	@Bean
	OAuth2AuthorizationRequestResolver authorizationRequestResolver(
			ClientRegistrationRepository clientRegistrationRepository) {
		DefaultOAuth2AuthorizationRequestResolver resolver = new DefaultOAuth2AuthorizationRequestResolver(
				clientRegistrationRepository,
				OAuth2AuthorizationRequestRedirectFilter.DEFAULT_AUTHORIZATION_REQUEST_BASE_URI);
		resolver.setAuthorizationRequestCustomizer(OAuth2AuthorizationRequestCustomizers.withPkce());
		return resolver;
	}

	@Bean
	OpaqueTokenIntrospector opaqueTokenIntrospector(ClientRegistrationRepository clientRegistrationRepository,
			@Value("${app.security.registration-id}") String registrationId) {
		ClientRegistration clientRegistration = clientRegistrationRepository.findByRegistrationId(registrationId);
		if (clientRegistration == null) {
			throw new IllegalStateException("Missing OAuth2 client registration: " + registrationId);
		}

		Object introspectionEndpoint = clientRegistration.getProviderDetails()
			.getConfigurationMetadata()
			.get("introspection_endpoint");

		if (!(introspectionEndpoint instanceof String endpoint) || endpoint.isBlank()) {
			throw new IllegalStateException("Authelia discovery metadata does not expose an introspection endpoint");
		}

		return SpringOpaqueTokenIntrospector.withIntrospectionUri(endpoint)
			.clientId(clientRegistration.getClientId())
			.clientSecret(clientRegistration.getClientSecret())
			.build();
	}

	private static final class FetchMetadataProtectionFilter extends OncePerRequestFilter {

		private static final Set<String> SAFE_METHODS = Set.of("GET", "HEAD", "OPTIONS", "TRACE");

		@Override
		protected boolean shouldNotFilter(HttpServletRequest request) {
			return SAFE_METHODS.contains(request.getMethod());
		}

		@Override
		protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response,
				FilterChain filterChain) throws ServletException, IOException {
			String fetchSite = request.getHeader("Sec-Fetch-Site");
			if (fetchSite == null || (!"same-origin".equals(fetchSite) && !"same-site".equals(fetchSite))) {
				response.sendError(HttpServletResponse.SC_FORBIDDEN, "Cross-site request rejected");
				return;
			}
			filterChain.doFilter(request, response);
		}

	}

}