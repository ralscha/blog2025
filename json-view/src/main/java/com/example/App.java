package com.example;

import tools.jackson.databind.MapperFeature;
import tools.jackson.databind.ObjectMapper;
import tools.jackson.databind.json.JsonMapper;

public class App {

  public static void main(String[] args) throws Exception {
    ObjectMapper mapper = new ObjectMapper();

    User user = new User("John Doe", "john@example.com", "johndoe", "123-45-6789", "INT001");

    String publicJson = mapper.writerWithView(View.Public.class).writeValueAsString(user);
    System.out.println(publicJson);
    // Output: {"name":"John Doe","email":"john@example.com","username":"johndoe"}

    String internalJson = mapper.writerWithView(View.Internal.class).writeValueAsString(user);
    System.out.println(internalJson);
    // Output: {"name":"John Doe","email":"john@example.com","username":"johndoe","ssn":"123-45-6789","internalId":"INT001"}

    String fullJson = mapper.writeValueAsString(user);
    System.out.println(fullJson);
    // Output: {"name":"John Doe","email":"john@example.com","username":"johndoe","ssn":"123-45-6789","internalId":"INT001"}

    String fullJsonForDeser =
        "{\"name\":\"Jane Smith\",\"email\":\"jane@example.com\",\"username\":\"janesmith\",\"ssn\":\"987-65-4321\",\"internalId\":\"INT002\"}";

    User publicUser =
        mapper.readerWithView(View.Public.class).forType(User.class).readValue(fullJsonForDeser);
    System.out.println(publicUser);
    // Output: User[name=Jane Smith, email=jane@example.com, username=janesmith, ssn=null, internalId=null]

    User internalUser =
        mapper.readerWithView(View.Internal.class).forType(User.class).readValue(fullJsonForDeser);
    System.out.println(internalUser);
    // Output: User[name=Jane Smith, email=jane@example.com, username=janesmith, ssn=987-65-4321, internalId=INT002]

    User fullUser = mapper.readValue(fullJsonForDeser, User.class);
    System.out.println(fullUser);
    // Output: User[name=Jane Smith, email=jane@example.com, username=janesmith, ssn=987-65-4321, internalId=INT002]

    Article article = new Article("Hello Views", "internal notes");

    ObjectMapper defaultNoInclusion = new ObjectMapper();
    String withViewDefault =
    	defaultNoInclusion.writerWithView(View.Public.class).writeValueAsString(article);
    System.out.println(withViewDefault);
    // Output: {"title":"Hello Views"}

    ObjectMapper withInclusion =
        JsonMapper.builder().configure(MapperFeature.DEFAULT_VIEW_INCLUSION, true).build();
    String withViewDisabled =
    	withInclusion.writerWithView(View.Public.class).writeValueAsString(article);
    System.out.println(withViewDisabled);
    // Output: {"title":"Hello Views","notes":"internal notes"}

    String noView = mapper.writeValueAsString(article);
    System.out.println(noView);
    // Output: {"title":"Hello Views","notes":"internal notes"}

    String articleJson = "{\"title\":\"Sample Article\",\"notes\":\"secret notes\"}";

    Article publicArticleDefault =
        mapper.readerWithView(View.Public.class).forType(Article.class).readValue(articleJson);
    System.out.println(publicArticleDefault);
    // Output: Article[title=Sample Article, notes=null]

    withInclusion =
        JsonMapper.builder().configure(MapperFeature.DEFAULT_VIEW_INCLUSION, true).build();
    Article publicArticleNoDefault =
    	withInclusion
            .readerWithView(View.Public.class)
            .forType(Article.class)
            .readValue(articleJson);
    System.out.println(publicArticleNoDefault);
    // Output: Article[title=Sample Article, notes=secret notes]

    Article fullArticle = mapper.readValue(articleJson, Article.class);
    System.out.println(fullArticle);
    // Output: Article[title=Sample Article, notes=secret notes]
  }
}
