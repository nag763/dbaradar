package com.nag763;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

import org.eclipse.microprofile.config.inject.ConfigProperty;

import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Inject;
import lombok.extern.slf4j.Slf4j;
import software.amazon.awssdk.services.dynamodb.DynamoDbClient;
import software.amazon.awssdk.services.dynamodb.model.AttributeValue;
import software.amazon.awssdk.services.dynamodb.model.QueryRequest;
import software.amazon.awssdk.services.dynamodb.model.QueryResponse;
import software.amazon.awssdk.services.ssm.SsmClient;
import software.amazon.awssdk.services.ssm.model.GetParameterRequest;
import software.amazon.awssdk.services.ssm.model.GetParameterResponse;

@ApplicationScoped
@Slf4j
public class SubscriptionsLoader {

    @Inject
    @ConfigProperty(name = "trusted.chat.ids.param", defaultValue = "")
    String trustedChatIdsParam;

    @Inject
    @ConfigProperty(name = "trusted.chat_queries_table", defaultValue = "trusted-chat-queries")
    String tableName;

    public Map<String, List<String>> loadAll() {
        String trusted = "";
        if (trustedChatIdsParam != null && !trustedChatIdsParam.isBlank()) {
            try (SsmClient ssm = SsmClient.create()) {
                GetParameterRequest req = GetParameterRequest.builder().name(trustedChatIdsParam).withDecryption(true).build();
                GetParameterResponse resp = ssm.getParameter(req);
                trusted = resp.parameter().value();
            } catch (Exception e) {
                log.error("failed to read SSM param {}: {}", trustedChatIdsParam, e.getMessage());
            }
        }

        Map<String, List<String>> out = new HashMap<>();
        if (trusted.isBlank()) {
            log.info("No trusted chat ids configured");
            return out;
        }

        List<String> chatIds = parseCSV(trusted);
        try (DynamoDbClient dynamo = DynamoDbClient.create()) {
            for (String cid : chatIds) {
                QueryRequest qr = QueryRequest.builder()
                        .tableName(tableName)
                        .keyConditionExpression("chatId = :cid")
                        .expressionAttributeValues(Map.of(":cid", AttributeValue.builder().s(cid).build()))
                        .build();
                QueryResponse resp = dynamo.query(qr);
                List<String> queries = resp.items().stream()
                        .map(m -> m.get("query"))
                        .filter(a -> a != null && a.s() != null)
                        .map(AttributeValue::s)
                        .collect(Collectors.toList());
                out.put(cid, queries);
            }
        } catch (Exception e) {
            log.error("failed to query subscriptions table {}: {}", tableName, e.getMessage());
        }

        return out;
    }

    private List<String> parseCSV(String s) {
        if (s == null || s.isBlank()) return List.of();
        String[] parts = s.split(",");
        List<String> out = new ArrayList<>();
        for (String p : parts) {
            String t = p.trim();
            if (!t.isEmpty()) out.add(t);
        }
        return out;
    }
}
