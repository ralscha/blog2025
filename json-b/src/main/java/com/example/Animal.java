package com.example;

import jakarta.json.bind.annotation.JsonbSubtype;
import jakarta.json.bind.annotation.JsonbTypeInfo;

@JsonbTypeInfo({
  @JsonbSubtype(alias = "dog", type = Dog.class),
  @JsonbSubtype(alias = "cat", type = Cat.class)
})
public abstract class Animal {

  protected String name;

  public Animal() {}

  public Animal(String name) {
    this.name = name;
  }

  public String getName() {
    return name;
  }

  public void setName(String name) {
    this.name = name;
  }

  public abstract String makeSound();
}
