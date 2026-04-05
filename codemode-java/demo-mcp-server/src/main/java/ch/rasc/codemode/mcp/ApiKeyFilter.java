package ch.rasc.codemode.mcp;

import java.io.IOException;

import jakarta.servlet.Filter;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.ServletRequest;
import jakarta.servlet.ServletResponse;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

public class ApiKeyFilter implements Filter {

	private static final String API_KEY_HEADER = "X-api-key";

	private final String expectedApiKey;

	public ApiKeyFilter(String expectedApiKey) {
		this.expectedApiKey = expectedApiKey;
	}

	@Override
	public void doFilter(ServletRequest request, ServletResponse response, FilterChain chain)
			throws IOException, ServletException {

		if (this.expectedApiKey == null || this.expectedApiKey.isBlank()) {
			chain.doFilter(request, response);
			return;
		}

		HttpServletRequest httpRequest = (HttpServletRequest) request;
		String apiKey = httpRequest.getHeader(API_KEY_HEADER);

		if (this.expectedApiKey.equals(apiKey)) {
			chain.doFilter(request, response);
			return;
		}

		HttpServletResponse httpResponse = (HttpServletResponse) response;
		httpResponse.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
		httpResponse.setContentType("application/json");
		httpResponse.getWriter().write("{\"error\":\"Unauthorized\"}");
	}

}