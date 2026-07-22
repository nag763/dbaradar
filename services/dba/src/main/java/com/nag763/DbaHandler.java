package com.nag763;

import java.util.List;

import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.RequestHandler;
import com.amazonaws.services.lambda.runtime.events.ScheduledEvent;
import com.amazonaws.services.lambda.runtime.logging.LogLevel;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.nag763.DbaSearchResponse.DbaDoc;

import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Inject;
import jakarta.inject.Named;

@Named("dbaHandler")
@ApplicationScoped
public class DbaHandler implements RequestHandler<ScheduledEvent, List<DbaDoc>> {

    @Inject
    ObjectMapper objectMapper;

    @Inject 
    DbaService dbaService;

    @Override
    public List<DbaDoc> handleRequest(ScheduledEvent input, Context context) {
        var logger = context.getLogger();
        logger.log("Handler started...\n", LogLevel.INFO);
        logger.log(String.format("Payload received : %s\n", input), LogLevel.DEBUG);
        DbaQuery dbaQuery = objectMapper.convertValue(input.getDetail(), DbaQuery.class);
        var results = dbaService.getQueries(dbaQuery);
        logger.log(String.format("Number of results fetched from DBA : %d\n", results.size()));
        return results;
    }

}