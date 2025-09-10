package com.example;

import java.util.List;
import java.util.Map;
import java.util.Set;

public record DataContainer(List<Person> people, Set<String> tags, Map<String, Integer> scores) {}
