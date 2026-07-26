package llm

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/nag763/dbaradar/services/analyzer/internal/model"
)

type mockBedrock struct {
	converseFunc func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
}

func (m *mockBedrock) Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	return m.converseFunc(ctx, params, optFns...)
}

func TestAnalyze_GoodDeal(t *testing.T) {
	mock := &mockBedrock{
		converseFunc: func(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return &bedrockruntime.ConverseOutput{
				Output: &brtypes.ConverseOutputMemberMessage{
					Value: brtypes.Message{
						Content: []brtypes.ContentBlock{
							&brtypes.ContentBlockMemberText{Value: `{"verdict":"GOOD_DEAL","reason":"40% below market"}`},
						},
					},
				},
			}, nil
		},
	}

	svc := NewService(mock)
	event := model.DbaEvent{
		Heading: "iPhone 15 Pro",
		Price:   model.Price{Amount: 4500, CurrencyCode: "DKK"},
		Query:   "iphone 15 pro",
	}

	verdict, err := svc.Analyze(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Verdict != model.VerdictGoodDeal {
		t.Errorf("expected GOOD_DEAL, got %s", verdict.Verdict)
	}
	if verdict.Reason != "40% below market" {
		t.Errorf("unexpected reason: %s", verdict.Reason)
	}
}

func TestAnalyze_NotADeal(t *testing.T) {
	mock := &mockBedrock{
		converseFunc: func(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return &bedrockruntime.ConverseOutput{
				Output: &brtypes.ConverseOutputMemberMessage{
					Value: brtypes.Message{
						Content: []brtypes.ContentBlock{
							&brtypes.ContentBlockMemberText{Value: `{"verdict":"NOT_A_DEAL","reason":"fair market price"}`},
						},
					},
				},
			}, nil
		},
	}

	svc := NewService(mock)
	verdict, err := svc.Analyze(context.Background(), model.DbaEvent{Heading: "Test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Verdict != "NOT_A_DEAL" {
		t.Errorf("expected NOT_A_DEAL, got %s", verdict.Verdict)
	}
}

func TestAnalyze_MalformedJSON(t *testing.T) {
	mock := &mockBedrock{
		converseFunc: func(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return &bedrockruntime.ConverseOutput{
				Output: &brtypes.ConverseOutputMemberMessage{
					Value: brtypes.Message{
						Content: []brtypes.ContentBlock{
							&brtypes.ContentBlockMemberText{Value: "this is not json"},
						},
					},
				},
			}, nil
		},
	}

	svc := NewService(mock)
	verdict, err := svc.Analyze(context.Background(), model.DbaEvent{Heading: "Test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Verdict != "NOT_A_DEAL" {
		t.Errorf("expected fallback NOT_A_DEAL, got %s", verdict.Verdict)
	}
}

func TestAnalyze_EmptyOutput(t *testing.T) {
	mock := &mockBedrock{
		converseFunc: func(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return &bedrockruntime.ConverseOutput{Output: nil}, nil
		},
	}

	svc := NewService(mock)
	verdict, err := svc.Analyze(context.Background(), model.DbaEvent{Heading: "Test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Verdict != "NOT_A_DEAL" {
		t.Errorf("expected fallback NOT_A_DEAL, got %s", verdict.Verdict)
	}
}

func TestBuildPrompt(t *testing.T) {
	event := model.DbaEvent{
		Heading:  "MacBook Pro M3",
		Price:    model.Price{Amount: 12000, CurrencyCode: "DKK"},
		Location: "Aarhus",
		Brand:    "Apple",
		Query:    "macbook pro",
		Extras:   []model.Extra{{Label: "Condition", Values: []string{"Used"}}},
	}

	prompt := buildPrompt(event)

	if prompt == "" {
		t.Fatal("prompt should not be empty")
	}
	// Verify key fields are present in the prompt
	for _, want := range []string{"MacBook Pro M3", "12000", "DKK", "Aarhus", "Apple", "macbook pro", "Condition"} {
		if !contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
