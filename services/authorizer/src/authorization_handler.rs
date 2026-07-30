use std::{net::IpAddr, str::FromStr};

use aws_lambda_events::{
    apigw::{ApiGatewayV2CustomAuthorizerSimpleResponse, ApiGatewayV2CustomAuthorizerV2Request},
    http::Method,
};
use ipnet::IpNet;
use lambda_runtime::{
    tracing::{debug, log::info, warn},
    Error, LambdaEvent,
};
use serde::{Deserialize, Serialize};

pub(crate) type Response = ApiGatewayV2CustomAuthorizerSimpleResponse<AuthorizerContext>;

pub(crate) struct Parameters {
    pub telegram_token: String,
    pub allowed_ips: Vec<IpNet>,
}

#[derive(Debug, Serialize, Deserialize, Default)]
pub(crate) struct AuthorizerContext {
    caller_ip: Option<String>,
    reason: Option<String>,
}

pub fn create_rejection(reason: &str) -> Response {
    warn!("Incoming call rejected for reason {}", reason);
    return Response::builder()
        .is_authorized(false)
        .context(AuthorizerContext {
            reason: Some(reason.to_owned()),
            caller_ip: None,
        })
        .build();
}

pub(crate) async fn function_handler(
    event: LambdaEvent<ApiGatewayV2CustomAuthorizerV2Request>,
    params: Parameters,
) -> Result<Response, Error> {
    let payload = event.payload;

    info!("Checking incoming payload");
    if payload.request_context.http.method != Method::POST {
        return Ok(create_rejection("method isn't allowed"));
    }
    if payload
        .request_context
        .http
        .path
        .unwrap_or("null".to_string())
        != "/webhook"
    {
        return Ok(create_rejection("path not allowed"));
    }

    debug!("Checking headers...");
    debug!("Checking IP...");
    let Some(ip_str) = payload.request_context.http.source_ip else {
        return Ok(create_rejection("IP not found"));
    };
    let Ok(ip) = IpAddr::from_str(&ip_str) else {
        return Ok(create_rejection("IP not in IPv4"));
    };
    if !params.allowed_ips.iter().any(|r| r.contains(&ip)) {
        return Ok(create_rejection("IP not in allowed ranges"));
    }
    debug!("IP verified successfully");

    debug!("Checking telegram token");
    let Some(telegram_token) = payload.headers.get("X-Telegram-Bot-Api-Secret-Token") else {
        return Ok(create_rejection("Telegram token header not found"));
    };
    if telegram_token.is_empty() || telegram_token.to_str()? != &params.telegram_token {
        return Ok(create_rejection("Telegram token header value mismatch"));
    }
    debug!("Telegram token verified");

    info!("Authorization successful");

    Ok(ApiGatewayV2CustomAuthorizerSimpleResponse::builder()
        .is_authorized(true)
        .context(AuthorizerContext {
            caller_ip: Some(ip_str),
            reason: None,
        })
        .build())
}
