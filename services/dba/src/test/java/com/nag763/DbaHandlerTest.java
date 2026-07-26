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

    @InjectMocks
    DbaHandler handler;

    private List<DbaDoc> sampleDocs;

    @BeforeEach
    void setUp() {
        handler.searchQueriesConfig = "iphone 15 pro";
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
        when(sqsPublisher.publish(anyList(), anyString())).thenReturn(1);

        String result = handler.handleRequest(null, context);

        verify(dbaService).getQueries(any(DbaQuery.class));
        verify(sqsPublisher).publish(eq(sampleDocs), eq("iphone 15 pro"));
        assertTrue(result.contains("1 listings"));
        assertTrue(result.contains("1 events"));
    }

    @Test
    void handleRequest_emptyResults_publishesNothing() {
        handler.searchQueriesConfig = "nonexistent";
        handler.searchLimit = 10;

        when(dbaService.getQueries(any(DbaQuery.class))).thenReturn(List.of());
        when(sqsPublisher.publish(anyList(), anyString())).thenReturn(0);

        String result = handler.handleRequest(null, context);

        verify(sqsPublisher).publish(eq(List.of()), eq("nonexistent"));
        assertTrue(result.contains("0 listings"));
    }

    @Test
    void handleRequest_emptyQueriesConfig_returnsErrorMessage() {
        handler.searchQueriesConfig = " , ";

        String result = handler.handleRequest(null, context);

        assertEquals("No valid queries found in configuration", result);
        verifyNoInteractions(dbaService);
        verifyNoInteractions(sqsPublisher);
    }
}
