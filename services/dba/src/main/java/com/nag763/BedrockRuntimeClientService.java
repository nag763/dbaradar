package com.nag763;

import io.quarkus.runtime.LaunchMode;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.enterprise.inject.Produces;
import software.amazon.awssdk.auth.credentials.ProfileCredentialsProvider;
import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.services.bedrockruntime.BedrockRuntimeClient;

@ApplicationScoped
public class BedrockRuntimeClientService {

    @Produces
    @ApplicationScoped
    public BedrockRuntimeClient bedrockClient() {

        var builder = BedrockRuntimeClient.builder().region(Region.EU_CENTRAL_1);
        if (LaunchMode.current() == LaunchMode.DEVELOPMENT) {
            builder.credentialsProvider(ProfileCredentialsProvider.create("default"));
        }
        return builder.build();
    }

}
