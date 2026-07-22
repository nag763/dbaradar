package com.nag763;

import org.eclipse.microprofile.rest.client.inject.RegisterRestClient;

import io.quarkus.rest.client.reactive.ClientQueryParam;
import jakarta.ws.rs.GET;
import jakarta.ws.rs.Path;
import jakarta.ws.rs.QueryParam;

@Path("/search/SEARCH_ID_BAP_COMMON")
@RegisterRestClient(configKey = "dba")
public interface DbaRestClient {

    @GET
    @ClientQueryParam(name = "dealer_segment", value = "1")
    @ClientQueryParam(name = "sort", value = "PUBLISHED_DESC")
    DbaSearchResponse getPage(@QueryParam("q") String query, @QueryParam("page") Integer page);
}
