package ch.rasc.methanol_server_demo;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

import com.aayushatharva.brotli4j.Brotli4jLoader;

@SpringBootApplication
public class Application {

  public static void main(String[] args) {
    Brotli4jLoader.ensureAvailability();
    SpringApplication.run(Application.class, args);
  }

}
