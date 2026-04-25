package ch.rasc.timefold.demo.web;

final class ScheduleJsonWriter {

  private ScheduleJsonWriter() {
  }

  static String toJson(ScheduleResponse response) {
    StringBuilder json = new StringBuilder(4096);
    json.append('{');
    appendStringField(json, "score", response.score());
    json.append(',');
    appendNumberField(json, "hardScore", response.hardScore());
    json.append(',');
    appendNumberField(json, "softScore", response.softScore());
    json.append(',');
    appendTimeslots(json, response.timeslots());
    json.append(',');
    appendScreens(json, response.screens());
    json.append(',');
    appendScreenings(json, response.screenings());
    json.append('}');
    return json.toString();
  }

  private static void appendTimeslots(StringBuilder json, java.util.List<ScheduleResponse.TimeslotView> timeslots) {
    json.append("\"timeslots\":[");
    for (int i = 0; i < timeslots.size(); i++) {
      ScheduleResponse.TimeslotView timeslot = timeslots.get(i);
      json.append('{');
      appendStringField(json, "id", timeslot.id());
      json.append(',');
      appendStringField(json, "label", timeslot.label());
      json.append(',');
      appendStringField(json, "start", timeslot.start());
      json.append(',');
      appendStringField(json, "end", timeslot.end());
      json.append('}');
      if (i + 1 < timeslots.size()) {
        json.append(',');
      }
    }
    json.append(']');
  }

  private static void appendScreens(StringBuilder json, java.util.List<ScheduleResponse.ScreenView> screens) {
    json.append("\"screens\":[");
    for (int i = 0; i < screens.size(); i++) {
      ScheduleResponse.ScreenView screen = screens.get(i);
      json.append('{');
      appendStringField(json, "id", screen.id());
      json.append(',');
      appendStringField(json, "name", screen.name());
      json.append(',');
      appendNumberField(json, "capacity", screen.capacity());
      json.append('}');
      if (i + 1 < screens.size()) {
        json.append(',');
      }
    }
    json.append(']');
  }

  private static void appendScreenings(StringBuilder json, java.util.List<ScheduleResponse.ScreeningView> screenings) {
    json.append("\"screenings\":[");
    for (int i = 0; i < screenings.size(); i++) {
      ScheduleResponse.ScreeningView screening = screenings.get(i);
      json.append('{');
      appendStringField(json, "id", screening.id());
      json.append(',');
      appendStringField(json, "title", screening.title());
      json.append(',');
      appendStringField(json, "director", screening.director());
      json.append(',');
      appendNullableStringField(json, "screenId", screening.screenId());
      json.append(',');
      appendNullableStringField(json, "screenName", screening.screenName());
      json.append(',');
      appendNullableStringField(json, "startSlotId", screening.startSlotId());
      json.append(',');
      appendStringField(json, "scheduleLabel", screening.scheduleLabel());
      json.append(',');
      appendNullableStringField(json, "start", screening.start());
      json.append(',');
      appendNullableStringField(json, "end", screening.end());
      json.append(',');
      appendNumberField(json, "durationMinutes", screening.durationMinutes());
      json.append(',');
      appendNumberField(json, "expectedAudience", screening.expectedAudience());
      json.append(',');
      appendStringField(json, "audienceSegment", screening.audienceSegment());
      json.append(',');
      appendBooleanField(json, "premiere", screening.premiere());
      json.append(',');
      appendBooleanField(json, "assigned", screening.assigned());
      json.append(',');
      appendBooleanField(json, "unassigned", screening.unassigned());
      json.append('}');
      if (i + 1 < screenings.size()) {
        json.append(',');
      }
    }
    json.append(']');
  }

  private static void appendStringField(StringBuilder json, String name, String value) {
    json.append('"').append(name).append("\":\"").append(escape(value)).append('"');
  }

  private static void appendNullableStringField(StringBuilder json, String name, String value) {
    json.append('"').append(name).append("\":");
    if (value == null) {
      json.append("null");
    } else {
      json.append('"').append(escape(value)).append('"');
    }
  }

  private static void appendNumberField(StringBuilder json, String name, int value) {
    json.append('"').append(name).append("\":").append(value);
  }

  private static void appendNumberField(StringBuilder json, String name, long value) {
    json.append('"').append(name).append("\":").append(value);
  }

  private static void appendBooleanField(StringBuilder json, String name, boolean value) {
    json.append('"').append(name).append("\":").append(value);
  }

  private static String escape(String value) {
    return value.replace("\\", "\\\\").replace("\"", "\\\"").replace("\r", "\\r").replace("\n", "\\n").replace("\t",
        "\\t");
  }
}