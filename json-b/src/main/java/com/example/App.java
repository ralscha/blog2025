package com.example;

import jakarta.json.bind.Jsonb;
import jakarta.json.bind.JsonbBuilder;
import jakarta.json.bind.JsonbConfig;
import jakarta.json.bind.JsonbException;
import jakarta.json.bind.config.PropertyNamingStrategy;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.Arrays;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

public class App {

  public static void main(String[] args) {
    Jsonb jsonb = JsonbBuilder.create();

    Jsonb jsonb2 = null;

    try {
      Person person =
          new Person("John Doe", 30, "john@example.com", "secret", LocalDate.now(), null, null);
      String json = jsonb.toJson(person);
      System.out.println(json);
      // Output:
      // {"full_name":"John Doe","email":"john@example.com","age":30,"createdDate":"2025-09-10"}

      Person deserializedPerson = jsonb.fromJson(json, Person.class);
      System.out.println(deserializedPerson.name());
      // Output: John Doe

      Map<String, String> metadata = new HashMap<>();
      metadata.put("department", "IT");
      metadata.put("level", "Senior");
      Person annotatedPerson =
          new Person(
              "Jane Smith",
              25,
              "jane@example.com",
              "password",
              LocalDate.now(),
              Arrays.asList("reading", "coding", "gaming"),
              metadata);

      String annotatedJson = jsonb.toJson(annotatedPerson);
      System.out.println(annotatedJson);
      // Output:
      // {"full_name":"Jane Smith","email":"jane@example.com","age":25,"createdDate":"2025-09-10",
      // "hobbies":["reading","coding","gaming"],"metadata":{"level":"Senior","department":"IT"}}

      Person deserializedAnnotatedPerson = jsonb.fromJson(annotatedJson, Person.class);
      System.out.println(
          deserializedAnnotatedPerson.name()
              + ", Hobbies: "
              + deserializedAnnotatedPerson.hobbies()
              + "\n");
      // Output: Jane Smith, Hobbies: [reading, coding, gaming]

      Event event =
          new Event("Tech Conference", LocalDateTime.now(), "Annual developer conference");
      String eventJson = jsonb.toJson(event);
      System.out.println(eventJson);
      // Output:
      // {"description":"Annual developer
      // conference","eventDate":"2025-09-10T08:09:51.1675905","title":"Tech Conference"}

      Event deserializedEvent = jsonb.fromJson(eventJson, Event.class);
      System.out.println(deserializedEvent.eventDate());
      // Output: 2025-09-10T08:09:51.167590500

      List<Animal> animals =
          Arrays.asList(new Dog("Buddy", "Golden Retriever"), new Cat("Whiskers", true));

      String animalsJson = jsonb.toJson(animals);
      System.out.println(animalsJson);
      // Output:
      // [{"@type":"dog","name":"Buddy","breed":"Golden
      // Retriever"},{"@type":"cat","name":"Whiskers","indoor":true}]

      Animal[] deserializedAnimals = jsonb.fromJson(animalsJson, Animal[].class);
      for (Animal animal : deserializedAnimals) {
        System.out.println(animal.getName() + " says: " + animal.makeSound());
      }
      // Output:
      // Buddy says: Woof!
      // Whiskers says: Meow!

      List<Person> people =
          Arrays.asList(
              new Person("Alice", 28, "alice@test.com", "pass", LocalDate.now(), null, null),
              new Person("Bob", 35, "bob@test.com", "pass", LocalDate.now(), null, null));
      Set<String> tags = new HashSet<>(Arrays.asList("java", "json", "tutorial"));
      Map<String, Integer> scores = new HashMap<>();
      scores.put("test1", 95);
      scores.put("test2", 87);

      DataContainer container = new DataContainer(people, tags, scores);
      String containerJson = jsonb.toJson(container);
      System.out.println(containerJson);
      // Output:
      // {"people":[{"full_name":"Alice","email":"alice@test.com","age":28,"createdDate":"2025-09-10"},
      // {"full_name":"Bob","email":"bob@test.com","age":35,"createdDate":"2025-09-10"}],
      // "scores":{"test2":87,"test1":95},"tags":["java","json","tutorial"]}

      DataContainer deserializedContainer = jsonb.fromJson(containerJson, DataContainer.class);
      System.out.println(deserializedContainer.people().size());
      // Output: 2
      System.out.println(deserializedContainer.people().get(0).name());
      // Output: Alice
      System.out.println(deserializedContainer.tags());
      // Output: [java, json, tutorial]
      System.out.println(deserializedContainer.scores());
      // Output: {test2=87, test1=95}

      JsonbConfig config = new JsonbConfig().withNullValues(true).withFormatting(true);

      Jsonb configuredJsonb = JsonbBuilder.create(config);

      Person personWithNulls = new Person("Test User", 0, null, null, null, null, null);
      String formattedJson = configuredJsonb.toJson(personWithNulls);
      System.out.println(formattedJson);
      // Output:
      // {
      //   "full_name": "Test User",
      //   "email": null,
      //   "age": 0,
      //   "createdDate": null,
      //   "hobbies": null,
      //   "metadata": null
      // }

      Person deserializedPersonWithNulls = configuredJsonb.fromJson(formattedJson, Person.class);
      System.out.println(
          deserializedPersonWithNulls.name() + ", Email: " + deserializedPersonWithNulls.email());
      // Output: Test User, Email: null

      TestNillable testNillable1 = new TestNillable(null, null);
      json = jsonb.toJson(testNillable1);
      System.out.println(json);
      // Output: {"nillableField":null}

      JsonbConfig config2 = new JsonbConfig().withNullValues(true);
      jsonb2 = JsonbBuilder.create(config2);
      TestNillable test = new TestNillable(null, null);
      String nillableJson = jsonb2.toJson(test);
      System.out.println(nillableJson);
      // Output: {"nillableField":null,"nonNillableField":null}

      String numberJson = "42";
      int number = jsonb.fromJson(numberJson, Integer.class);
      System.out.println(number);
      // Output: 42

      String booleanJson = "true";
      boolean bool = jsonb.fromJson(booleanJson, Boolean.class);
      System.out.println(bool);
      // Output: true

      String arrayJson = "[\"apple\", \"banana\", \"cherry\"]";
      String[] fruitsArray = jsonb.fromJson(arrayJson, String[].class);
      List<String> fruits = Arrays.asList(fruitsArray);
      System.out.println(fruits);
      // Output: [apple, banana, cherry]

      String invalidJson = "{\"name\": \"John\", \"age\": \"not-a-number\"}";
      try {
        jsonb.fromJson(invalidJson, Person.class);
        System.out.println("This shouldn't print");
      } catch (JsonbException e) {
        System.out.println("Caught deserialization error: " + e.getMessage());
      }
      // Output: Caught deserialization error: Unable to deserialize property 'age' because of:
      // Error deserialize JSON value into type: int.

      byte[] testBytes = {(byte) 0x3E, (byte) 0x3F, (byte) 0xFE, (byte) 0xFF};
      BinaryData binaryData = new BinaryData("test.bin", testBytes);

      JsonbConfig base64Config =
          new JsonbConfig()
              .withBinaryDataStrategy(jakarta.json.bind.config.BinaryDataStrategy.BASE_64);
      Jsonb base64Jsonb = JsonbBuilder.create(base64Config);

      String base64Json = base64Jsonb.toJson(binaryData);
      System.out.println(base64Json);
      // Output: {"content":"Pj/+/w==","filename":"test.bin"}

      BinaryData deserializedBase64 = base64Jsonb.fromJson(base64Json, BinaryData.class);
      System.out.println(deserializedBase64);
      // Output: BinaryData[filename=test.bin, content=[B@43e7f6c3]

      JsonbConfig base64UrlConfig =
          new JsonbConfig()
              .withBinaryDataStrategy(jakarta.json.bind.config.BinaryDataStrategy.BASE_64_URL);
      Jsonb base64UrlJsonb = JsonbBuilder.create(base64UrlConfig);

      String base64UrlJson = base64UrlJsonb.toJson(binaryData);
      System.out.println(base64UrlJson);
      // Output: {"content":"Pj_-_w==","filename":"test.bin"}

      BinaryData deserializedBase64Url = base64UrlJsonb.fromJson(base64UrlJson, BinaryData.class);
      System.out.println(deserializedBase64Url);
      // Output: BinaryData[filename=test.bin, content=[B@7df7dde0]

      JsonbConfig byteConfig =
          new JsonbConfig()
              .withBinaryDataStrategy(jakarta.json.bind.config.BinaryDataStrategy.BYTE);
      Jsonb byteJsonb = JsonbBuilder.create(byteConfig);

      String byteJson = byteJsonb.toJson(binaryData);
      System.out.println(byteJson);
      // Output: {"content":[62,63,-2,-1],"filename":"test.bin"}

      BinaryData deserializedByte = byteJsonb.fromJson(byteJson, BinaryData.class);
      System.out.println(deserializedByte);
      // Output: BinaryData[filename=test.bin, content=[B@42661c1b]

      base64Jsonb.close();
      base64UrlJsonb.close();
      byteJsonb.close();

      JsonbConfig namingConfig =
          new JsonbConfig()
              .withPropertyNamingStrategy(PropertyNamingStrategy.LOWER_CASE_WITH_UNDERSCORES);

      Jsonb namingJsonb = JsonbBuilder.create(namingConfig);

      UserProfile userProfile =
          new UserProfile("john.doe", "John Doe", "john@example.com", "Software Engineer");
      String namingJson = namingJsonb.toJson(userProfile);
      System.out.println(namingJson);
      // Output:
      // {"display_name":"John Doe","email_address":"john@example.com","job_title":"Software
      // Engineer","user_name":"john.doe"}

      UserProfile deserializedUserProfile = namingJsonb.fromJson(namingJson, UserProfile.class);
      System.out.println(deserializedUserProfile);
      // Output:
      // UserProfile[userName=john.doe, displayName=John Doe, emailAddress=john@example.com,
      // jobTitle=Software Engineer]

      namingJsonb.close();

    } catch (Exception e) {
      System.err.println("Error during JSON processing: " + e.getMessage());
    } finally {
      try {
        jsonb.close();
      } catch (Exception e) {
        System.err.println("Error closing Jsonb: " + e.getMessage());
      }
      try {
        if (jsonb2 != null) jsonb2.close();
      } catch (Exception e) {
        System.err.println("Error closing jsonb2: " + e.getMessage());
      }
    }
  }
}
