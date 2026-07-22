package com.nag763;

import java.util.List;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
public record DbaSearchResponse(List<DbaDoc> docs) {

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static record DbaDoc(
            String type,
            String id,
            String mainSearchKey,
            String heading,
            String location,
            Image image,
            List<String> flags,
            Long timestamp,
            Integer adType,
            List<Label> labels,
            String canonicalUrl,
            List<Extra> extras,
            Price price,
            Integer distance,
            List<String> imageUrls,
            String brand,
            Long adId) {
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
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static record Metadata(
            String source) {
    }
}
