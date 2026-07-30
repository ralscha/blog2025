package ch.rasc.springgrpc.client.service;

import java.util.List;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.grpc.client.ChannelBuilderOptions;
import org.springframework.grpc.client.GrpcChannelFactory;
import org.springframework.grpc.client.interceptor.security.BasicAuthenticationInterceptor;

import ch.rasc.springgrpc.proto.IotAnomalyServiceGrpc;

@Configuration
public class ClientGrpcConfig {

  @Value("${app.grpc.target:localhost:9090}")
  private String target;

  @Value("${app.grpc.username}")
  private String username;

  @Value("${app.grpc.password}")
  private String password;

  @Bean
  IotAnomalyServiceGrpc.IotAnomalyServiceBlockingStub anomalyBlockingStub(GrpcChannelFactory channels) {
    return IotAnomalyServiceGrpc.newBlockingStub(channels.createChannel(this.target, channelOptions()));
  }

  @Bean
  IotAnomalyServiceGrpc.IotAnomalyServiceStub anomalyAsyncStub(GrpcChannelFactory channels) {
    return IotAnomalyServiceGrpc.newStub(channels.createChannel(this.target, channelOptions()));
  }

  private ChannelBuilderOptions channelOptions() {
    return ChannelBuilderOptions.defaults()
        .withInterceptors(List.of(new BasicAuthenticationInterceptor(this.username, this.password)));
  }
}
