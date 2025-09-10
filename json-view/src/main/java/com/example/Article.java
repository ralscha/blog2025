package com.example;

import com.fasterxml.jackson.annotation.JsonView;

public record Article(@JsonView(View.Public.class) String title, String notes) {}
