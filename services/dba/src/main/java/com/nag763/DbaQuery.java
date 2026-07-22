package com.nag763;

import java.util.List;
import java.util.Optional;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotEmpty;
import jakarta.validation.constraints.PositiveOrZero;

public record DbaQuery(
        Optional<@PositiveOrZero Integer> limit,
        Optional<@PositiveOrZero Integer> page,
        @NotEmpty List<@NotBlank String> queries) {

}
