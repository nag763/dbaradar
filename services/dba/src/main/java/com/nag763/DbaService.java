package com.nag763;

import java.util.Comparator;
import java.util.List;
import java.util.concurrent.ExecutorService;

import org.eclipse.microprofile.rest.client.inject.RestClient;

import com.nag763.DbaSearchResponse.DbaDoc;

import io.quarkus.virtual.threads.VirtualThreads;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Inject;

@ApplicationScoped
public class DbaService {

    @RestClient
    @Inject
    private DbaRestClient dbaRestClient;

    @Inject
    @VirtualThreads
    private ExecutorService vThreads;

    List<DbaDoc> getPage(String query, Integer page) {
        return dbaRestClient.getPage(query, page).docs();
    }

    List<DbaDoc> getFirstPage(String query) {
        return this.getPage(query, 1);
    }

    List<DbaDoc> getParallel(List<String> queries) {
        return queries.stream()
                .map(q -> vThreads.submit(() -> this.getFirstPage(q)))
                .flatMap(future -> {
                    try {
                        return future.get().stream();
                    } catch (InterruptedException e) {
                        Thread.currentThread().interrupt();
                        throw new RuntimeException("Interrupted while fetching query results", e);
                    } catch (Exception e) {
                        throw new RuntimeException("Failed to fetch query results", e);
                    }
                })
                .sorted(Comparator.comparing(DbaDoc::timestamp).reversed()).toList();
    }

    List<DbaDoc> getQueries(DbaQuery query) {
        var results = this.getParallel(query.queries());
        if(results.isEmpty() || results.size() < 20) {
            return results;
        }
        return results.subList(0, query.limit().orElse(20));
    }

}