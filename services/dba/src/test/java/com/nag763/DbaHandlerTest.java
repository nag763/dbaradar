package com.nag763;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

import java.util.List;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import com.amazonaws.services.lambda.runtime.Context;
import com.nag763.DbaSearchResponse.DbaDoc;
import com.nag763.DbaSearchResponse.Price;

@ExtendWith(MockitoExtension.class)
class DbaHandlerTest {

    @Mock
    DbaService dbaService;

    @Mock
    SqsPublisher sqsPublisher;

    @Mock
    Context context;

    @Mock
    SubscriptionsLoader subscriptionsLoader;

    @InjectMocks
    DbaHandler handler;

    private List<DbaDoc> sampleDocs;

    @BeforeEach
    void setUp() {
        handler.searchLimit = 20;

        sampleDocs = List.of(
                DbaDoc.builder()
                        .type("type")
                        .id("1")
                        .mainSearchKey("key")
                        .heading("iPhone 15 Pro")
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
                        .imageUrls(List.of("https://img.example.com/1.jpg"))
                        .brand("Apple")
                        .adId(12345678L)
                        .build()
        );
    }

    @Test
    void handleRequest_fetchesAndPublishes() {
        when(dbaService.getQueries(any(DbaQuery.class))).thenReturn(sampleDocs);
        when(subscriptionsLoader.loadAll()).thenReturn(java.util.Map.of("5253905011", List.of("iphone 15 pro")));
        when(sqsPublisher.publish(anyList(), anyString(), anyString())).thenReturn(1);

        String result = handler.handleRequest(null, context);

        verify(dbaService).getQueries(any(DbaQuery.class));
        verify(sqsPublisher).publish(eq(sampleDocs), eq("iphone 15 pro"), eq("5253905011"));
        assertTrue(result.contains("1 events"));
        assertTrue(result.contains("1 subscriptions"));
    }

    @Test
    void handleRequest_emptyResults_publishesNothing() {
        handler.searchLimit = 10;

        when(dbaService.getQueries(any(DbaQuery.class))).thenReturn(List.of());
        when(subscriptionsLoader.loadAll()).thenReturn(java.util.Map.of("5253905011", List.of("nonexistent")));
        when(sqsPublisher.publish(anyList(), anyString(), anyString())).thenReturn(0);

        String result = handler.handleRequest(null, context);

        verify(sqsPublisher).publish(eq(List.of()), eq("nonexistent"), eq("5253905011"));
        assertTrue(result.contains("0 events"));
    }

    @Test
    void handleRequest_noSubscriptions_returnsErrorMessage() {
        when(subscriptionsLoader.loadAll()).thenReturn(java.util.Collections.emptyMap());

        String result = handler.handleRequest(null, context);

        assertEquals("No subscriptions configured", result);
        verifyNoInteractions(dbaService);
        verifyNoInteractions(sqsPublisher);
    }
}
