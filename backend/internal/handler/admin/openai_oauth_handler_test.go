package admin

import "testing"

func TestMergeMapAnyOverwritesDefaults(t *testing.T) {
	t.Parallel()

	base := map[string]any{
		"access_token": "ak",
		"plan_type":    "free",
		"model_mapping": map[string]any{
			"gpt-4o": "gpt-4o",
		},
	}
	override := map[string]any{
		"plan_type": "plus",
		"model_mapping": map[string]any{
			"gpt-5.4": "gpt-5.4",
		},
	}

	mergeMapAny(base, override)

	if got := base["plan_type"]; got != "plus" {
		t.Fatalf("plan_type = %#v, want plus", got)
	}
	modelMapping, _ := base["model_mapping"].(map[string]any)
	if modelMapping["gpt-5.4"] != "gpt-5.4" {
		t.Fatalf("model_mapping = %#v", base["model_mapping"])
	}
}

func TestCloneMapAnyHandlesEmptyInput(t *testing.T) {
	t.Parallel()

	if cloneMapAny(nil) != nil {
		t.Fatal("cloneMapAny(nil) should be nil")
	}
	if cloneMapAny(map[string]any{}) != nil {
		t.Fatal("cloneMapAny(empty) should be nil")
	}
}
