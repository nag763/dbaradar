// Package llm provides deal analysis via Amazon Bedrock's Converse API.
package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/nag763/dbaradar/services/analyzer/internal/model"
)

const modelID = "eu.amazon.nova-2-lite-v1:0"

const systemPrompt = `You are a deal evaluation assistant for second-hand marketplace listings in Denmark (DBA.dk).
Analyze the listing and determine if it is a good deal (priced attractively for its category and condition).

Output requirements (MANDATORY):
- Respond ONLY with a single, compact, valid JSON object and nothing else.
- The JSON MUST contain exactly these fields: "verdict" and "reason".
- "verdict" must be either "GOOD_DEAL" or "NOT_A_DEAL" (use these exact strings).
- "reason" must be a brief human-readable explanation.
- Do NOT wrap the JSON in markdown code fences, backticks, or any additional commentary. Avoid any surrounding formatting. For example, do NOT return the JSON wrapped in markdown code fences (three backticks) or with extra text around it.
- If uncertain or missing important data, return {"verdict":"NOT_A_DEAL","reason":"brief explanation"}.

Policy (decision guidance):
- Be pragmatic and fair.
- Mark GOOD_DEAL when the price is clearly below market value (historical/market estimate) — e.g., >20% below market.
- Also mark GOOD_DEAL for listings that are slightly below market (discounts of ~5% or more) or that are at fair market price but have clearly favorable attributes (excellent condition, complete accessories, recent model, low supply). In short: don't only look for deep bargains — slight under-market offers or fair-priced listings with positive context can also be GOOD_DEAL.
- When in doubt or missing key info, prefer NOT_A_DEAL.

Examples (valid outputs — exact raw text, no fences):
1) Deep bargain
Input:
- Heading: iPhone 15 Pro 128GB
- Price: 3500 DKK
- Brand: Apple
- Condition: Som ny
Valid Output (exact raw text):
{"verdict":"GOOD_DEAL","reason":"iPhone 15 Pro 128GB usually sells around 5500-6000 DKK second-hand in like-new condition; 3500 DKK is >35% below market value."}

2) Slightly under market (should also pass)
Input:
- Heading: iPhone 15 Pro 128GB
- Price: 5200 DKK
- Brand: Apple
- Condition: Som ny
Valid Output (exact raw text):
{"verdict":"GOOD_DEAL","reason":"iPhone 15 Pro 128GB typically sells ~5500-6000 DKK; 5200 DKK is slightly (~5-6%) below market and acceptable as a good deal given like-new condition."}

3) Fair price without upside
Input:
- Heading: iPhone 13 128GB
- Price: 3200 DKK
- Brand: Apple
- Condition: Brugt
Valid Output (exact raw text):
{"verdict":"NOT_A_DEAL","reason":"3200 DKK is a fair market price for a used iPhone 13 128GB."}`

// BedrockAPI is the subset of the Bedrock runtime client used by this service.
type BedrockAPI interface {
	Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
}

// Service provides LLM-based deal analysis using Amazon Bedrock.
type Service struct {
	client BedrockAPI
}

// NewService creates an LLM service backed by the given Bedrock client.
func NewService(client BedrockAPI) *Service {
	return &Service{client: client}
}

// Analyze evaluates whether a DBA listing is a good deal.
// Returns a structured verdict. On malformed LLM responses, defaults to NOT_A_DEAL.
func (s *Service) Analyze(ctx context.Context, event model.DbaEvent) (model.LLMVerdict, error) {
	prompt := buildPrompt(event)

	out, err := s.client.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId: aws.String(modelID),
		System: []brtypes.SystemContentBlock{
			&brtypes.SystemContentBlockMemberText{Value: systemPrompt},
		},
		Messages: []brtypes.Message{
			{
				Role:    brtypes.ConversationRoleUser,
				Content: []brtypes.ContentBlock{&brtypes.ContentBlockMemberText{Value: prompt}},
			},
		},
	})
	if err != nil {
		return model.LLMVerdict{}, fmt.Errorf("bedrock converse: %w", err)
	}

	return parseResponse(out)
}

func buildPrompt(event model.DbaEvent) string {
	extras := ""
	for _, e := range event.Extras {
		extras += fmt.Sprintf("- %s: %v\n", e.Label, e.Values)
	}

	return fmt.Sprintf(`Listing details:
- Heading: %s
- Price: %d %s
- Location: %s
- Brand: %s
- Search query: %s
- Additional info:
%s`, event.Heading, event.Price.Amount, event.Price.CurrencyCode, event.Location, event.Brand, event.Query, extras)
}

func parseResponse(out *bedrockruntime.ConverseOutput) (model.LLMVerdict, error) {
	if out.Output == nil {
		return defaultVerdict("empty output"), nil
	}

	msg, ok := out.Output.(*brtypes.ConverseOutputMemberMessage)
	if !ok || len(msg.Value.Content) == 0 {
		return defaultVerdict("no message content"), nil
	}

	textBlock, ok := msg.Value.Content[0].(*brtypes.ContentBlockMemberText)
	if !ok {
		return defaultVerdict("non-text content block"), nil
	}

	var verdict model.LLMVerdict
	if err := json.Unmarshal([]byte(textBlock.Value), &verdict); err != nil {
		return defaultVerdict(fmt.Sprintf("malformed JSON: %s", textBlock.Value)), nil
	}

	if verdict.Verdict != model.VerdictGoodDeal && verdict.Verdict != "NOT_A_DEAL" {
		return defaultVerdict(fmt.Sprintf("unknown verdict: %s", verdict.Verdict)), nil
	}

	return verdict, nil
}

func defaultVerdict(reason string) model.LLMVerdict {
	return model.LLMVerdict{Verdict: "NOT_A_DEAL", Reason: reason}
}
