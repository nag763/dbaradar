package com.nag763;

import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

import org.eclipse.microprofile.config.inject.ConfigProperty;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.nag763.DbaSearchResponse.DbaDoc;

import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Inject;
import lombok.extern.slf4j.Slf4j;
import software.amazon.awssdk.services.sqs.SqsClient;

/**
 * Publishes DBA listing events to the downstream SQS queue in batches.
 */
@ApplicationScoped
@Slf4j
public class SqsPublisher {

    @Inject
    SqsClient sqsClient;

    @Inject
    ObjectMapper objectMapper;

    @ConfigProperty(name = "dba.sqs.queue-url")
    String queueUrl;

    /**
     * Publishes a list of DBA listings to the SQS queue.
     * If there is 1 listing, it is sent as a singleton JSON object.
     * If there are multiple listings, they are packaged into a JSON array batch
     * payload.
     *
     * @param docs  the listings to publish
     * @param query the search query that produced these results
     * @return the number of successfully published items
     */
    public int publish(List<DbaDoc> docs, String query) {
        if (docs == null || docs.isEmpty()) {
            return 0;
        }

        if (docs.size() == 1) {
            return sendSingle(docs.get(0), query);
        }

        return sendArrayBatch(docs, query);
    }

    private int sendSingle(DbaDoc doc, String query) {
        try {
            DbaEvent event = toEvent(doc, query);
            String body = objectMapper.writeValueAsString(event);
            sqsClient.sendMessage(builder -> builder.queueUrl(queueUrl).messageBody(body));
            return 1;
        } catch (JsonProcessingException e) {
            log.error("Failed to serialize single event for adId={}", doc.adId(), e);
            return 0;
        }
    }

    private int sendArrayBatch(List<DbaDoc> docs, String query) {
        List<DbaEvent> events = new ArrayList<>();
        for (DbaDoc doc : docs) {
            events.add(toEvent(doc, query));
        }

        try {
            String body = objectMapper.writeValueAsString(events);
            sqsClient.sendMessage(builder -> builder.queueUrl(queueUrl).messageBody(body));
            return events.size();
        } catch (JsonProcessingException e) {
            log.error("Failed to serialize batch events array", e);
            return 0;
        }
    }

    private DbaEvent toEvent(DbaDoc doc, String query) {
        return DbaEvent.builder()
                .eventId(UUID.randomUUID().toString())
                .id(doc.id())
                .heading(doc.heading())
                .price(doc.price())
                .location(doc.location())
                .canonicalUrl(doc.canonicalUrl())
                .imageUrls(doc.imageUrls())
                .brand(doc.brand())
                .timestamp(doc.timestamp())
                .extras(doc.extras())
                .query(query)
                .build();
    }
}
