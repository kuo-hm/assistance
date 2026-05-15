package llm

import (
	"strings"
	"testing"

	"assistance/internal/memory"
)

func TestBuildPromptIncludesMemoryAndLanguage(t *testing.T) {
	prompt := BuildPrompt(ConversationInput{
		SessionID: "s1",
		UserText:  "salam, kif dayr?",
		Language:  "ar-MA",
		Memory: memory.Context{
			Summary: "User likes brief replies.",
			Facts:   map[string]string{"name": "Harmony"},
			Recent:  []memory.Turn{{Role: "user", Text: "hello"}},
		},
	})

	for _, want := range []string{"User likes brief replies.", "name: Harmony", "ar-MA", "salam, kif dayr?"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}
