package com.example.x509clientdemo.config;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.config.annotation.web.configurers.CsrfConfigurer;
import org.springframework.security.core.authority.AuthorityUtils;
import org.springframework.security.core.userdetails.User;
import org.springframework.security.core.userdetails.UserDetailsService;
import org.springframework.security.core.userdetails.UsernameNotFoundException;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.security.web.authentication.preauth.x509.SubjectX500PrincipalExtractor;

@Configuration
@EnableWebSecurity
public class SecurityConfig {

  @Bean
  SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
    http.authorizeHttpRequests(
        authz -> authz.requestMatchers("/api/public/**").permitAll()
            .requestMatchers("/api/secure/**").authenticated().anyRequest().permitAll())
        .x509(x509 -> x509.x509PrincipalExtractor(new SubjectX500PrincipalExtractor())
            .userDetailsService(userDetailsService()))
        .csrf(CsrfConfigurer::disable);

    return http.build();
  }

  @Bean
  UserDetailsService userDetailsService() {
    return cn -> {
      // For this demo, we'll accept only certificate with CN == "demo-client"
      if ("demo-client".equals(cn.toLowerCase())) {
        return new User(cn, "", AuthorityUtils.createAuthorityList("ROLE_USER"));
      }
      throw new UsernameNotFoundException("Certificate not authorized: " + cn);
    };
  }
}
