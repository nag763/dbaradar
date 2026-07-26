package com.nag763;

import java.util.Arrays;
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
    @ConfigProperty(name = "dba.search.queries", defaultValue = "canon,nikon,lumix,olympus")
    String searchQueriesConfig;

    @Inject
    @ConfigProperty(name = "dba.search.limit", defaultValue = "20")
    int searchLimit;

    @Inject
    DbaService dbaService;

    @Inject
    SqsPublisher sqsPublisher;

    @Override
    public String handleRequest(Object input, Context context) {
        log.info("DBA Ingestor started...");

        List<String> queries = Arrays.stream(searchQueriesConfig.split(","))
                .map(String::trim)
                .filter(s -> !s.isEmpty())
                .toList();

        if (queries.isEmpty()) {
            log.error("No search queries configured");
            return "No valid queries found in configuration";
        }

        DbaQuery dbaQuery = new DbaQuery(Optional.of(searchLimit), Optional.empty(), queries);
        log.info("Loaded DbaQuery from environment: {}", dbaQuery);

        var results = dbaService.getQueries(dbaQuery);
        log.info("Fetched {} listings from DBA", results.size());

        int totalPublished = 0;
        for (String query : dbaQuery.queries()) {
            totalPublished += sqsPublisher.publish(results, query);
        }

        String summary = String.format("Ingested %d listings, published %d events", results.size(), totalPublished);
        log.info(summary);
        return summary;
    }
}

