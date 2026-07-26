package com.nag763;

import java.util.List;

import com.nag763.DbaSearchResponse.Extra;
import com.nag763.DbaSearchResponse.Price;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.experimental.Accessors;
import lombok.extern.jackson.Jacksonized;

/**
 * Event published to the dba-events SQS queue for downstream consumption
 * by the Analyzer service.
 */
@Getter
@Builder
@Jacksonized
@AllArgsConstructor
@Accessors(fluent = true)
public class DbaEvent {
    private final String eventId;
    private final String id;
    private final String heading;
    private final Price price;
    private final String location;
    private final String canonicalUrl;
    private final List<String> imageUrls;
    private final String brand;
    private final Long timestamp;
    private final List<Extra> extras;
    private final String query;
    private final String chatId;
}
