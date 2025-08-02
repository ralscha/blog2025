package ch.rasc.methanoldemo;

import java.nio.file.Path;
import java.time.Duration;

import com.github.mizosoft.methanol.HttpCache;
import com.github.mizosoft.methanol.Methanol;
import com.github.mizosoft.methanol.store.redis.RedisStorageExtension;

import io.lettuce.core.RedisURI;

public class Demo2 {
  public static void main(String[] args) {
    HttpCache cache = HttpCache.newBuilder()
        .cacheOnDisk(Path.of(".cache"), (long) (100 * 1024 * 1024)).build();

    Methanol.Builder builder = Methanol.newBuilder().userAgent("MyCustomUserAgent/1.0")
        .baseUri("https://api.github.com").defaultHeader("Accept", "application/json")
        .requestTimeout(Duration.ofSeconds(20)).headersTimeout(Duration.ofSeconds(5))
        .connectTimeout(Duration.ofSeconds(10)).readTimeout(Duration.ofSeconds(5))
        .cache(cache).interceptor(new LoggingInterceptor());

    Methanol client = builder.build();
  }
  
  public static void disableDecompression() {
    Methanol client = Methanol.newBuilder().autoAcceptEncoding(false).build();
  }

  public static void fileCache() {
    HttpCache cache = HttpCache.newBuilder()
        .cacheOnDisk(Path.of(".cache"), (long) (100 * 1024 * 1024)).build();
    Methanol client = Methanol.newBuilder().cache(cache).build();
  }
  
  public static void memoryCache() {
    HttpCache cache = HttpCache.newBuilder().cacheOnMemory((long) (100 * 1024 * 1024))
        .build();
    Methanol client = Methanol.newBuilder().cache(cache).build();
  }
  
  public static void redisCache() {
    RedisURI redisUri = RedisURI.create("redis://localhost:6379");
    HttpCache cache = HttpCache.newBuilder()
        .cacheOn(RedisStorageExtension.newBuilder().standalone(redisUri).build()).build();
    Methanol client = Methanol.newBuilder().cache(cache).build();
  }

}
