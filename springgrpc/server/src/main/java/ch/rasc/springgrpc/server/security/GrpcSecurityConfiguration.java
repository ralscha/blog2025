package ch.rasc.springgrpc.server.security;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.grpc.server.GlobalServerInterceptor;
import org.springframework.grpc.server.security.AuthenticationProcessInterceptor;
import org.springframework.grpc.server.security.GrpcSecurity;
import org.springframework.security.core.userdetails.User;
import org.springframework.security.core.userdetails.UserDetailsService;
import org.springframework.security.provisioning.InMemoryUserDetailsManager;

import static org.springframework.security.config.Customizer.withDefaults;

@Configuration
public class GrpcSecurityConfiguration {

  @Bean
  UserDetailsService userDetailsService() {
    return new InMemoryUserDetailsManager(
        User.withUsername("iot-client")
            .password("{noop}iot-secret")
            .roles("USER")
            .build());
  }

  @Bean
  @GlobalServerInterceptor
  AuthenticationProcessInterceptor grpcAuthenticationInterceptor(GrpcSecurity grpc) throws Exception {
    return grpc
        .authorizeRequests(requests -> requests
            .methods("grpc.*/*").permitAll()
            .allRequests().authenticated())
        .httpBasic(withDefaults())
        .build();
  }
}
