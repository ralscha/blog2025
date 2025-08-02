package ch.rasc.methanoldemo;

import java.io.IOException;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.concurrent.CompletableFuture;

import com.github.mizosoft.methanol.Methanol.Interceptor;

public class LoggingInterceptor implements Interceptor {

  @Override
  public <T> HttpResponse<T> intercept(HttpRequest request, Chain<T> chain)
      throws IOException, InterruptedException {

    System.out.println("Request Method: " + request.method());
    System.out.println("Request URI: " + request.uri());
    System.out.println("Request Headers: " + request.headers());

    return chain.forward(request);
  }

  @Override
  public <T> CompletableFuture<HttpResponse<T>> interceptAsync(HttpRequest request,
      Chain<T> chain) {

    System.out.println("Async Request Method: " + request.method());
    System.out.println("Async Request URI: " + request.uri());
    System.out.println("Async Request Headers: " + request.headers());

    return chain.forwardAsync(request).thenApply(response -> {
      System.out.println("Async Response Status Code: " + response.statusCode());
      return response;
    });

  }

}
