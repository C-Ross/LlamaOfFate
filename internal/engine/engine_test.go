package engine

import (
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestNPC(id, name string) *core.Character {
	return core.NewCharacter(id, name)
}

func TestResolveCharacter(t *testing.T) {
	t.Run("exact ID match", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("scene_4_npc_1", "Outlaw Scout"))

		result := eng.ResolveCharacter("scene_4_npc_1")
		require.NotNil(t, result)
		assert.Equal(t, "scene_4_npc_1", result.ID)
		assert.Equal(t, "Outlaw Scout", result.Name)
	})

	t.Run("exact name match", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("scene_4_npc_1", "Outlaw Scout"))

		result := eng.ResolveCharacter("Outlaw Scout")
		require.NotNil(t, result)
		assert.Equal(t, "scene_4_npc_1", result.ID)
	})

	t.Run("name match is case insensitive", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("scene_4_npc_1", "Outlaw Scout"))

		for _, input := range []string{"outlaw scout", "OUTLAW SCOUT", "Outlaw scout", "oUtLaW sCOUT"} {
			result := eng.ResolveCharacter(input)
			require.NotNil(t, result, "should resolve %q", input)
			assert.Equal(t, "scene_4_npc_1", result.ID)
		}
	})

	t.Run("Name (ID) format extracts ID", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("scene_4_npc_1", "Outlaw Scout"))

		result := eng.ResolveCharacter("Outlaw Scout (scene_4_npc_1)")
		require.NotNil(t, result)
		assert.Equal(t, "scene_4_npc_1", result.ID)
		assert.Equal(t, "Outlaw Scout", result.Name)
	})

	t.Run("Name (ID) format with wrong name still resolves via ID", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("bandit_1", "Bandit Leader"))

		result := eng.ResolveCharacter("The Bandit Boss (bandit_1)")
		require.NotNil(t, result)
		assert.Equal(t, "bandit_1", result.ID)
		assert.Equal(t, "Bandit Leader", result.Name)
	})

	t.Run("Name (ID) format with wrong ID resolves via extracted name", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("npc_42", "Sheriff Brown"))

		result := eng.ResolveCharacter("Sheriff Brown (wrong_id)")
		require.NotNil(t, result)
		assert.Equal(t, "npc_42", result.ID)
	})

	t.Run("Name (ID) format with both wrong returns nil", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("npc_42", "Sheriff Brown"))

		result := eng.ResolveCharacter("Nobody (wrong_id)")
		assert.Nil(t, result)
	})

	t.Run("nested parentheses uses last pair for ID extraction", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("npc_1", "Guard (Elite)"))

		// LLM produces "Guard (Elite) (npc_1)" — LastIndex finds the outer parens
		result := eng.ResolveCharacter("Guard (Elite) (npc_1)")
		require.NotNil(t, result)
		assert.Equal(t, "npc_1", result.ID)
		assert.Equal(t, "Guard (Elite)", result.Name)
	})

	t.Run("whitespace around target is trimmed", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("npc_1", "Scout"))

		result := eng.ResolveCharacter("  npc_1  ")
		require.NotNil(t, result)
		assert.Equal(t, "npc_1", result.ID)
	})

	t.Run("whitespace inside Name (ID) is trimmed", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("npc_1", "Scout"))

		result := eng.ResolveCharacter("Scout ( npc_1 )")
		require.NotNil(t, result)
		assert.Equal(t, "npc_1", result.ID)
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		assert.Nil(t, eng.ResolveCharacter(""))
	})

	t.Run("whitespace-only string returns nil", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		assert.Nil(t, eng.ResolveCharacter("   "))
		assert.Nil(t, eng.ResolveCharacter("\t"))
		assert.Nil(t, eng.ResolveCharacter("\n"))
	})

	t.Run("no match returns nil", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("scene_4_npc_1", "Outlaw Scout"))

		assert.Nil(t, eng.ResolveCharacter("Nobody Here"))
		assert.Nil(t, eng.ResolveCharacter("scene_999_npc_0"))
		assert.Nil(t, eng.ResolveCharacter("Outlaw"))  // partial name doesn't match
		assert.Nil(t, eng.ResolveCharacter("scene_4")) // partial ID doesn't match
	})

	t.Run("empty registry returns nil for any input", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})

		assert.Nil(t, eng.ResolveCharacter("npc_1"))
		assert.Nil(t, eng.ResolveCharacter("Some Name"))
		assert.Nil(t, eng.ResolveCharacter("Name (npc_1)"))
	})

	t.Run("multiple characters resolves correct one", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("npc_0", "Bandit Leader"))
		eng.AddCharacter(newTestNPC("npc_1", "Outlaw Scout"))
		eng.AddCharacter(newTestNPC("npc_2", "Sheriff Morgan"))

		result := eng.ResolveCharacter("npc_1")
		require.NotNil(t, result)
		assert.Equal(t, "Outlaw Scout", result.Name)

		result = eng.ResolveCharacter("Sheriff Morgan")
		require.NotNil(t, result)
		assert.Equal(t, "npc_2", result.ID)

		result = eng.ResolveCharacter("Bandit Leader (npc_0)")
		require.NotNil(t, result)
		assert.Equal(t, "npc_0", result.ID)
	})

	t.Run("ID takes priority over name when both could match", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		// Contrived: a character whose name is another character's ID
		eng.AddCharacter(newTestNPC("alpha", "Bravo"))
		eng.AddCharacter(newTestNPC("bravo", "Charlie"))

		// "bravo" matches alpha by name AND bravo by ID; ID lookup runs first
		result := eng.ResolveCharacter("bravo")
		require.NotNil(t, result)
		assert.Equal(t, "bravo", result.ID, "exact ID match should take priority over name match")
	})

	t.Run("parentheses with no content inside", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("npc_1", "Scout"))

		// "Scout ()" — empty parens; extracted ID is empty, extracted name is "Scout"
		result := eng.ResolveCharacter("Scout ()")
		require.NotNil(t, result, "should fall through to extracted name match")
		assert.Equal(t, "npc_1", result.ID)
	})

	t.Run("unmatched parentheses ignored gracefully", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("npc_1", "Scout"))

		// Only opening paren, no closing — should not crash, just fail to match
		assert.Nil(t, eng.ResolveCharacter("Scout (npc_1"))

		// Only closing paren
		assert.Nil(t, eng.ResolveCharacter("Scout npc_1)"))
	})

	t.Run("parentheses at start of string", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		eng.AddCharacter(newTestNPC("npc_1", "Scout"))

		// "(npc_1)" — idx would be 0, which is not > 0, so this branch is skipped
		// Falls through to nil since "(npc_1)" doesn't match any ID or name
		assert.Nil(t, eng.ResolveCharacter("(npc_1)"))
	})

	t.Run("extracted ID tried as name", func(t *testing.T) {
		eng, _ := New(session.NullLogger{})
		// Character whose name looks like an ID
		eng.AddCharacter(newTestNPC("real_id", "fake_id"))

		// "Whatever (fake_id)" — extracted content isn't a real ID, but matches by name
		result := eng.ResolveCharacter("Whatever (fake_id)")
		require.NotNil(t, result)
		assert.Equal(t, "real_id", result.ID)
	})
}
