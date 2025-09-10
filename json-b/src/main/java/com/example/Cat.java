package com.example;

public class Cat extends Animal {

  private boolean isIndoor;

  public Cat() {}

  public Cat(String name, boolean isIndoor) {
    super(name);
    this.isIndoor = isIndoor;
  }

  public boolean isIndoor() {
    return isIndoor;
  }

  public void setIndoor(boolean indoor) {
    this.isIndoor = indoor;
  }

  @Override
  public String makeSound() {
    return "Meow!";
  }
}
