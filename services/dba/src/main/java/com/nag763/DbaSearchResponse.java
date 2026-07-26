package com.nag763;

import java.util.List;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonAlias;
import com.fasterxml.jackson.annotation.JsonProperty;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.experimental.Accessors;
import lombok.extern.jackson.Jacksonized;

@JsonIgnoreProperties(ignoreUnknown = true)
public record DbaSearchResponse(List<DbaDoc> docs) {

        @JsonIgnoreProperties(ignoreUnknown = true)
        @Getter
        @Builder
        @Jacksonized
        @AllArgsConstructor
        @Accessors(fluent = true)
        public static class DbaDoc {
                private final String type;
                private final String id;
                private final String mainSearchKey;
                private final String heading;
                private final String location;
                private final Image image;
                private final List<String> flags;
                private final Long timestamp;
                private final Integer adType;
                private final List<Label> labels;
                @JsonProperty("canonicalUrl")
                @JsonAlias("canonical_url")
                private final String canonicalUrl;
                private final List<Extra> extras;
                private final Price price;
                private final Integer distance;
                private final List<String> imageUrls;
                private final String brand;
                private final Long adId;
        }

        @JsonIgnoreProperties(ignoreUnknown = true)
        public static record Image(
                        String url,
                        String path) {
        }

        @JsonIgnoreProperties(ignoreUnknown = true)
        public static record Label(
                        String id,
                        String text,
                        String type) {
        }

        @JsonIgnoreProperties(ignoreUnknown = true)
        public static record Extra(
                        String id,
                        String label,
                        List<String> values) {
        }

        @JsonIgnoreProperties(ignoreUnknown = true)
        public static record Price(
                        Integer amount,
                        String currencyCode,
                        String priceUnit) {

                @JsonProperty
                public String getDefaultCurrency() {
                        return "DKK";
                }
        }

        @JsonIgnoreProperties(ignoreUnknown = true)
        public static record Metadata(
                        String source) {
        }
}
