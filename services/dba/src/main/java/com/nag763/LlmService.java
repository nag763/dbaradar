package com.nag763;


import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Inject;
import lombok.extern.slf4j.Slf4j;
import software.amazon.awssdk.services.bedrockruntime.BedrockRuntimeClient;
import software.amazon.awssdk.services.bedrockruntime.model.ContentBlock;
import software.amazon.awssdk.services.bedrockruntime.model.ConversationRole;
import software.amazon.awssdk.services.bedrockruntime.model.ConverseRequest;
import software.amazon.awssdk.services.bedrockruntime.model.Message;

@ApplicationScoped
@Slf4j
class LlmService {

    @Inject
    private BedrockRuntimeClient client;

    private final String MODEL_ID = "eu.amazon.nova-2-lite-v1:0";

    String callLlm(String input) {
        log.info("Initiating LLM Call...");
        var message = builMessage(input);
        ConverseRequest request = ConverseRequest.builder()
                .modelId(MODEL_ID)
                .messages(message)
                .build();
        var resp = client.converse(request);

        log.debug("Resp {}", resp);
        return resp.toString();
    }

    private Message builMessage(String input) {
        log.debug("Building bedrock message...");
        return Message.builder().content(ContentBlock.fromText(input)).role(ConversationRole.USER).build();
    }

}
