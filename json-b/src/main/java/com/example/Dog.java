package com.example;

public class Dog extends Animal {

  private String breed;

  public Dog() {}

  public Dog(String name, String breed) {
    super(name);
    this.breed = breed;
  }

  public String getBreed() {
    return breed;
  }

  public void setBreed(String breed) {
    this.breed = breed;
  }

  @Override
  public String makeSound() {
    return "Woof!";
  }
}
