package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyCodexOAuthTransform_ToolContinuationPreservesInput(t *testing.T) {
	reqBody := map[string]any{
		"model": "gpt-5.2",
		"input": []any{
			map[string]any{"type": "item_reference", "id": "ref1", "text": "x"},
			map[string]any{"type": "function_call_output", "call_id": "call_1", "output": "ok", "id": "o1"},
		},
		"tool_choice": "auto",
	}

	applyCodexOAuthTransform(reqBody, false)

	store, ok := reqBody["store"].(bool)
	require.True(t, ok)
	require.False(t, store)

	input, ok := reqBody["input"].([]any)
	require.True(t, ok)
	require.Len(t, input, 2)

	first, ok := input[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "item_reference", first["type"])
	require.Equal(t, "ref1", first["id"])

	second, ok := input[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "o1", second["id"])
}

func TestApplyCodexOAuthTransform_ExplicitStoreFalsePreserved(t *testing.T) {
	reqBody := map[string]any{
		"model": "gpt-5.1",
		"store": false,
		"input": []any{
			map[string]any{"type": "function_call_output", "call_id": "call_1"},
		},
		"tool_choice": "auto",
	}

	applyCodexOAuthTransform(reqBody, false)

	store, ok := reqBody["store"].(bool)
	require.True(t, ok)
	require.False(t, store)
}

func TestApplyCodexOAuthTransform_ExplicitStoreTrueForcedFalse(t *testing.T) {
	reqBody := map[string]any{
		"model": "gpt-5.1",
		"store": true,
		"input": []any{
			map[string]any{"type": "function_call_output", "call_id": "call_1"},
		},
		"tool_choice": "auto",
	}

	applyCodexOAuthTransform(reqBody, false)

	store, ok := reqBody["store"].(bool)
	require.True(t, ok)
	require.False(t, store)
}

func TestApplyCodexOAuthTransform_NonContinuationDefaultsStoreFalseAndStripsIDs(t *testing.T) {
	reqBody := map[string]any{
		"model": "gpt-5.1",
		"input": []any{
			map[string]any{"type": "text", "id": "t1", "text": "hi"},
		},
	}

	applyCodexOAuthTransform(reqBody, false)

	store, ok := reqBody["store"].(bool)
	require.True(t, ok)
	require.False(t, store)

	input, ok := reqBody["input"].([]any)
	require.True(t, ok)
	require.Len(t, input, 1)

	item, ok := input[0].(map[string]any)
	require.True(t, ok)
	_, hasID := item["id"]
	require.False(t, hasID)
}

func TestFilterCodexInput_RemovesItemReferenceWhenNotPreserved(t *testing.T) {
	input := []any{
		map[string]any{"type": "item_reference", "id": "ref1"},
		map[string]any{"type": "text", "id": "t1", "text": "hi"},
	}

	filtered := filterCodexInput(input, false)
	require.Len(t, filtered, 1)

	item, ok := filtered[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "text", item["type"])
	_, hasID := item["id"]
	require.False(t, hasID)
}

func TestApplyCodexOAuthTransform_NormalizeCodexTools_PreservesResponsesFunctionTools(t *testing.T) {
	reqBody := map[string]any{
		"model": "gpt-5.1",
		"tools": []any{
			map[string]any{
				"type":        "function",
				"name":        "bash",
				"description": "desc",
				"parameters":  map[string]any{"type": "object"},
			},
			map[string]any{
				"type":     "function",
				"function": nil,
			},
		},
	}

	applyCodexOAuthTransform(reqBody, false)

	tools, ok := reqBody["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)

	first, ok := tools[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "function", first["type"])
	require.Equal(t, "bash", first["name"])
}

func TestApplyCodexOAuthTransform_EmptyInput(t *testing.T) {
	reqBody := map[string]any{
		"model": "gpt-5.1",
		"input": []any{},
	}

	applyCodexOAuthTransform(reqBody, false)

	input, ok := reqBody["input"].([]any)
	require.True(t, ok)
	require.Len(t, input, 0)
}

func TestNormalizeCodexModel_Gpt53(t *testing.T) {
	cases := map[string]string{
		"gpt-5.4":                   "gpt-5.4",
		"gpt-5.4-high":              "gpt-5.4",
		"gpt-5.4-chat-latest":       "gpt-5.4",
		"gpt 5.4":                   "gpt-5.4",
		"gpt-5.3":                   "gpt-5.3-codex",
		"gpt-5.3-codex":             "gpt-5.3-codex",
		"gpt-5.3-codex-xhigh":       "gpt-5.3-codex",
		"gpt-5.3-codex-spark":       "gpt-5.3-codex",
		"gpt-5.3-codex-spark-high":  "gpt-5.3-codex",
		"gpt-5.3-codex-spark-xhigh": "gpt-5.3-codex",
		"gpt 5.3 codex":             "gpt-5.3-codex",
		"gpt-5.5-codex-ultra":       "gpt-5.5-codex-ultra",
		"vendor/custom-model":       "custom-model",
	}

	for input, expected := range cases {
		require.Equal(t, expected, normalizeCodexModel(input))
	}
}

func TestApplyCodexOAuthTransform_CodexCLI_PreservesExistingInstructions(t *testing.T) {
	reqBody := map[string]any{
		"model":        "gpt-5.1",
		"instructions": "existing instructions",
	}

	result := applyCodexOAuthTransform(reqBody, true)

	instructions, ok := reqBody["instructions"].(string)
	require.True(t, ok)
	require.Equal(t, "existing instructions", instructions)
	_ = result
}

func TestApplyCodexOAuthTransform_CodexCLI_MapsSystemToInstructions(t *testing.T) {
	reqBody := map[string]any{
		"model":  "gpt-5.1",
		"system": "system instructions",
	}

	result := applyCodexOAuthTransform(reqBody, true)

	instructions, ok := reqBody["instructions"].(string)
	require.True(t, ok)
	require.Equal(t, "system instructions", instructions)
	_, hasSystem := reqBody["system"]
	require.False(t, hasSystem)
	require.True(t, result.Modified)
}

func TestApplyCodexOAuthTransform_NonCodexCLI_StripsPromptFields(t *testing.T) {
	reqBody := map[string]any{
		"model":        "gpt-5.1",
		"instructions": "old instructions",
		"system":       "system prompt",
	}

	result := applyCodexOAuthTransform(reqBody, false)

	_, hasInstructions := reqBody["instructions"]
	require.False(t, hasInstructions)
	_, hasSystem := reqBody["system"]
	require.False(t, hasSystem)
	require.True(t, result.Modified)
}

func TestIsInstructionsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		reqBody  map[string]any
		expected bool
	}{
		{"missing field", map[string]any{}, true},
		{"nil value", map[string]any{"instructions": nil}, true},
		{"empty string", map[string]any{"instructions": ""}, true},
		{"whitespace only", map[string]any{"instructions": "   "}, true},
		{"non-string", map[string]any{"instructions": 123}, true},
		{"valid string", map[string]any{"instructions": "hello"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInstructionsEmpty(tt.reqBody)
			require.Equal(t, tt.expected, result)
		})
	}
}
