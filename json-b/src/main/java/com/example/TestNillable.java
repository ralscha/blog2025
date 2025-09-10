package com.example;

import jakarta.json.bind.annotation.JsonbNillable;

public record TestNillable(@JsonbNillable String nillableField, String nonNillableField) {}
