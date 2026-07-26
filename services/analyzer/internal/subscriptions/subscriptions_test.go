package subscriptions

import "testing"

func TestSubscriptionHelpers(t *testing.T) {
	svc := &Service{
		chatIDToQueries: map[string]map[string]struct{}{
			"123": {"camera": {}},
			"456": {"lens": {}},
		},
		queryToChatIDs: map[string][]string{
			"camera": {"123"},
			"lens":   {"456"},
		},
	}

	if !svc.IsQuerySubscribed("camera") {
		t.Fatalf("expected camera to be subscribed")
	}

	ids := svc.GetChatIDsForQuery("camera")
	if len(ids) != 1 || ids[0] != "123" {
		t.Fatalf("expected chat id 123 for camera, got %v", ids)
	}

	if svc.IsQuerySubscribed("unknown") {
		t.Fatalf("did not expect unknown to be subscribed")
	}
}
