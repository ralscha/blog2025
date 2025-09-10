package com.example;

import com.fasterxml.jackson.annotation.JsonView;

public record User(
    @JsonView(View.Public.class) String name,
    @JsonView(View.Public.class) String email,
    @JsonView({View.Public.class}) String username,
    @JsonView(View.Internal.class) String ssn,
    @JsonView(View.Internal.class) String internalId) {}
