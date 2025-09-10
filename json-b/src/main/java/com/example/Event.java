package com.example;

import jakarta.json.bind.adapter.JsonbAdapter;
import jakarta.json.bind.annotation.JsonbTypeAdapter;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;

public record Event(
    String title,
    @JsonbTypeAdapter(LocalDateTimeAdapter.class) LocalDateTime eventDate,
    String description) {

  public static class LocalDateTimeAdapter implements JsonbAdapter<LocalDateTime, String> {

    private static final DateTimeFormatter FORMATTER = DateTimeFormatter.ISO_LOCAL_DATE_TIME;

    @Override
    public String adaptToJson(LocalDateTime obj) throws Exception {
      return obj.format(FORMATTER);
    }

    @Override
    public LocalDateTime adaptFromJson(String obj) throws Exception {
      return LocalDateTime.parse(obj, FORMATTER);
    }
  }
}
