package com.nag763;

import java.util.List;
import java.util.Optional;

import org.eclipse.microprofile.config.inject.ConfigProperty;

import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.RequestHandler;

import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Inject;
import jakarta.inject.Named;
import lombok.extern.slf4j.Slf4j;

/**
 * Lambda handler for the DBA Ingestor.
 * Fetches listings from DBA.dk and publishes each as an event to SQS
 * for downstream analysis based on configured environment variables.
 */
@Named("dbaHandler")
@ApplicationScoped
@Slf4j
public class DbaHandler implements RequestHandler<Object, String> {

    @Inject
    @ConfigProperty(name = "dba.search.limit", defaultValue = "20")
    int searchLimit;

    @Inject
    DbaService dbaService;

    @Inject
    SqsPublisher sqsPublisher;

    @Inject
    SubscriptionsLoader subscriptionsLoader;

    @Override
    public String handleRequest(Object input, Context context) {
        log.info("DBA Ingestor started...");

        var subscriptions = subscriptionsLoader.loadAll();
        int totalPublished = 0;

        if (subscriptions.isEmpty()) {
            log.error("No subscriptions found");
            return "No subscriptions configured";
        }

        for (var entry : subscriptions.entrySet()) {
            String chatId = entry.getKey();
            for (String q : entry.getValue()) {
                DbaQuery dq = new DbaQuery(Optional.of(searchLimit), Optional.empty(), List.of(q));
                var results = dbaService.getQueries(dq);
                log.info("Fetched {} listings for query '{}'", results.size(), q);
                totalPublished += sqsPublisher.publish(results, q, chatId);
            }
        }

        String summary = String.format("Published %d events for %d subscriptions", totalPublished, subscriptions.size());
        log.info(summary);
        return summary;
    }
}

