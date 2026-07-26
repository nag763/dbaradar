package com.nag763;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

import java.util.List;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.Spy;
import org.mockito.junit.jupiter.MockitoExtension;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.nag763.DbaSearchResponse.DbaDoc;
import com.nag763.DbaSearchResponse.Price;
import java.util.function.Consumer;

import software.amazon.awssdk.services.sqs.SqsClient;
import software.amazon.awssdk.services.sqs.model.SendMessageRequest;
import software.amazon.awssdk.services.sqs.model.SendMessageResponse;

@ExtendWith(MockitoExtension.class)
class SqsPublisherTest {

    @Mock
    SqsClient sqsClient;

    @Spy
    ObjectMapper objectMapper = new ObjectMapper();

    @InjectMocks
    SqsPublisher publisher;

    @BeforeEach
    void setUp() throws Exception {
        var field = SqsPublisher.class.getDeclaredField("queueUrl");
        field.setAccessible(true);
        field.set(publisher, "https://sqs.eu-central-1.amazonaws.com/123456789/dba-events");
    }

    @Test
    void publish_singleDoc_sendsSingletonObject() {
        List<DbaDoc> docs = List.of(
                DbaDoc.builder()
                        .type("type")
                        .id("1")
                        .mainSearchKey("key")
                        .heading("iPhone 15")
                        .location("Copenhagen")
                        .image(null)
                        .flags(List.of())
                        .timestamp(1719000000L)
                        .adType(1)
                        .labels(List.of())
                        .canonicalUrl("https://dba.dk/test")
                        .extras(List.of())
                        .price(new Price(4500, "DKK", null))
                        .distance(0)
                        .imageUrls(List.of())
                        .brand("Apple")
                        .adId(12345678L)
                        .build()
        );

        when(sqsClient.sendMessage(any(Consumer.class)))
                .thenReturn(SendMessageResponse.builder().messageId("msg-1").build());

        int result = publisher.publish(docs, "iphone 15");

        assertEquals(1, result);

        ArgumentCaptor<Consumer<SendMessageRequest.Builder>> captor = ArgumentCaptor.forClass(Consumer.class);
        verify(sqsClient).sendMessage(captor.capture());

        SendMessageRequest.Builder builder = SendMessageRequest.builder();
        captor.getValue().accept(builder);
        SendMessageRequest request = builder.build();

        assertTrue(request.messageBody().startsWith("{"));
        assertTrue(request.messageBody().contains("\"id\":\"1\""));
        assertTrue(request.messageBody().contains("\"query\":\"iphone 15\""));
    }

    @Test
    void publish_emptyList_doesNotCallSqs() {
        int result = publisher.publish(List.of(), "test");

        assertEquals(0, result);
        verifyNoInteractions(sqsClient);
    }

    @Test
    void publish_multipleDocs_sendsJsonArrayBatch() {
        List<DbaDoc> docs = new java.util.ArrayList<>();
        for (int i = 0; i < 12; i++) {
            docs.add(DbaDoc.builder()
                    .type("type")
                    .id(String.valueOf(i))
                    .mainSearchKey("key")
                    .heading("Item " + i)
                    .location("Loc")
                    .image(null)
                    .flags(List.of())
                    .timestamp(1719000000L)
                    .adType(1)
                    .labels(List.of())
                    .canonicalUrl("https://dba.dk/" + i)
                    .extras(List.of())
                    .price(new Price(100, "DKK", null))
                    .distance(0)
                    .imageUrls(List.of())
                    .brand("Brand")
                    .adId((long) i)
                    .build());
        }

        when(sqsClient.sendMessage(any(Consumer.class)))
                .thenReturn(SendMessageResponse.builder().messageId("msg-batch").build());

        int result = publisher.publish(docs, "test");

        assertEquals(12, result);

        ArgumentCaptor<Consumer<SendMessageRequest.Builder>> captor = ArgumentCaptor.forClass(Consumer.class);
        verify(sqsClient).sendMessage(captor.capture());

        SendMessageRequest.Builder builder = SendMessageRequest.builder();
        captor.getValue().accept(builder);
        SendMessageRequest request = builder.build();

        assertTrue(request.messageBody().startsWith("["));
        assertTrue(request.messageBody().contains("\"id\":\"0\""));
        assertTrue(request.messageBody().contains("\"id\":\"11\""));
    }
}
