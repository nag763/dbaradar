package com.nag763;

import java.util.List;
import java.util.Optional;

public record DbaQuery(
        Optional<Integer> limit,
        Optional<Integer> page,
        List<String> queries) {

}
