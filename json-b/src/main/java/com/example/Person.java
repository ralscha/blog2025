package com.example;

import jakarta.json.bind.annotation.JsonbProperty;
import jakarta.json.bind.annotation.JsonbPropertyOrder;
import jakarta.json.bind.annotation.JsonbTransient;
import java.time.LocalDate;
import java.util.List;
import java.util.Map;

@JsonbPropertyOrder({"name", "email", "age", "createdDate"})
public record Person(
    @JsonbProperty("full_name") String name,
    int age,
    String email,
    @JsonbTransient String password,
    LocalDate createdDate,
    List<String> hobbies,
    Map<String, String> metadata) {}
