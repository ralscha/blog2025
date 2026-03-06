package ch.rasc.springgrpc.client.service;

import ch.rasc.springgrpc.proto.IotAnomalyServiceGrpc;
import java.util.List;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.grpc.client.ChannelBuilderOptions;
import org.springframework.grpc.client.GrpcChannelFactory;
import org.springframework.grpc.client.interceptor.security.BasicAuthenticationInterceptor;

@Configuration
public class ClientGrpcConfig {

  @Value("${app.grpc.username}")
  private String username;

  @Value("${app.grpc.password}")
  private String password;

  @Bean
  IotAnomalyServiceGrpc.IotAnomalyServiceBlockingStub anomalyBlockingStub(GrpcChannelFactory channels) {
    return IotAnomalyServiceGrpc.newBlockingStub(channels.createChannel("anomaly-server", channelOptions()));
  }

  @Bean
  IotAnomalyServiceGrpc.IotAnomalyServiceStub anomalyAsyncStub(GrpcChannelFactory channels) {
    return IotAnomalyServiceGrpc.newStub(channels.createChannel("anomaly-server", channelOptions()));
  }

  private ChannelBuilderOptions channelOptions() {
    return ChannelBuilderOptions.defaults()
        .withInterceptors(List.of(new BasicAuthenticationInterceptor(this.username, this.password)));
  }
}
