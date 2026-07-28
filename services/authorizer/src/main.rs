use std::env;

use aws_config::BehaviorVersion;
use aws_sdk_ssm::Client;
use ipnet::IpNet;
use lambda_runtime::{
    run, service_fn,
    tracing::{self, error, log::info},
    Error,
};

mod authorization_handler;
use authorization_handler::{function_handler, Parameters};

async fn access_parameter(
    client: &Client,
    parameter_name: &str,
    secure: bool,
) -> Result<Option<String>, Error> {
    let value = client
        .get_parameter()
        .name(parameter_name)
        .with_decryption(secure)
        .send()
        .await?
        .parameter()
        .and_then(|p| p.value().map(|v| v.to_string()));
    Ok(value)
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    tracing::init_default_subscriber();
    info!("Starting authorization logic...");

    let config = aws_config::load_defaults(BehaviorVersion::latest()).await;
    let ssm_client = Client::new(&config);

    let telegram_token = access_parameter(
        &ssm_client,
        env::var("TELEGRAM_SECRET_TOKEN_PARAM")?.as_str(),
        true,
    )
    .await?
    .ok_or("No telegram token found")?;

    if telegram_token.is_empty() {
        error!("Received telegram token is empty");
        return Err(Error::from("Empty token for telegram"));
    }

    let allowed_ips: Vec<IpNet> = access_parameter(
        &ssm_client,
        env::var("TELEGRAM_IP_RANGES_PARAM")?.as_str(),
        false,
    )
    .await?
    .ok_or("No IP ranges parameter found")?
    .split(',')
    .map(|ip_range| ip_range.trim())
    .flat_map(|ip_range| ip_range.parse::<IpNet>())
    .collect();

    if allowed_ips.is_empty() {
        return Err(Error::from("No allowed IP range found"));
    }

    let func = service_fn(move |event| {
        let params = Parameters {
            telegram_token: telegram_token.clone(),
            allowed_ips: allowed_ips.clone(),
        };
        function_handler(event, params)
    });

    run(func).await
}
