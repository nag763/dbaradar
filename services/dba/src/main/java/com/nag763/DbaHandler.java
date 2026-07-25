package com.nag763;

import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.RequestHandler;
import com.amazonaws.services.lambda.runtime.events.ScheduledEvent;
import com.fasterxml.jackson.databind.ObjectMapper;

import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Inject;
import jakarta.inject.Named;
import lombok.extern.slf4j.Slf4j;

@Named("dbaHandler")
@ApplicationScoped
@Slf4j
public class DbaHandler implements RequestHandler<ScheduledEvent, String> {

    @Inject
    ObjectMapper objectMapper;

    @Inject 
    DbaService dbaService;

    @Inject
    LlmService llmService;

    @Inject
    TelegramBotService telegramBotService;

    @Override
    public String handleRequest(ScheduledEvent input, Context context) {
        log.info("Handler started...");
        log.debug("Payload received : {}", input);
        DbaQuery dbaQuery = objectMapper.convertValue(input.getDetail(), DbaQuery.class);
        var results = dbaService.getQueries(dbaQuery);
        log.info("Number of results fetched from DBA : {}", results.size());
        var llmAnalysis = llmService.callLlm(String.format("Is there any item here that is worth buying as it is under the price you would expect ?\n%s", results.toString()));
        log.info("Updates ? {}", telegramBotService.getUpdates());
        return llmAnalysis;
    }

}