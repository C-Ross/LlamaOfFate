package engine

import (
	"fmt"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
)

// NPCRegistry manages named NPCs that persist across scenes within a scenario.
// It tracks characters by normalized name and their last-known attitudes.
type NPCRegistry interface {
	// Register adds a named NPC with the given attitude. The name is normalized internally.
	Register(npc *character.Character, attitude string)

	// Lookup returns the NPC for the given name and true, or nil and false if not found.
	Lookup(name string) (*character.Character, bool)

	// UpdateAttitude sets the attitude for the given NPC name.
	UpdateAttitude(name, attitude string)

	// UpdateAttitudesFromSummary batch-updates attitudes from a scene summary.
	UpdateAttitudesFromSummary(summary *prompt.SceneSummary)

	// KnownSummaries returns prompt-ready summaries of all known NPCs,
	// excluding permanently removed ones and annotating temporarily defeated ones.
	KnownSummaries() []prompt.NPCSummary

	// Snapshot returns deep copies of the registry and attitude maps for persistence.
	Snapshot() (registry map[string]*character.Character, attitudes map[string]string)

	// Restore replaces the registry contents from previously saved maps.
	// Nil maps are treated as empty.
	Restore(registry map[string]*character.Character, attitudes map[string]string)

	// All returns all registered NPCs (including taken-out ones).
	All() map[string]*character.Character
}

// npcRegistryImpl is the concrete implementation of NPCRegistry.
type npcRegistryImpl struct {
	registry  map[string]*character.Character
	attitudes map[string]string
}

// NewNPCRegistry creates a new, empty NPCRegistry.
func NewNPCRegistry() NPCRegistry {
	return &npcRegistryImpl{
		registry:  make(map[string]*character.Character),
		attitudes: make(map[string]string),
	}
}

func (r *npcRegistryImpl) Register(npc *character.Character, attitude string) {
	key := normalizeNPCName(npc.Name)
	r.registry[key] = npc
	r.attitudes[key] = attitude
}

func (r *npcRegistryImpl) Lookup(name string) (*character.Character, bool) {
	npc, found := r.registry[normalizeNPCName(name)]
	return npc, found
}

func (r *npcRegistryImpl) UpdateAttitude(name, attitude string) {
	r.attitudes[normalizeNPCName(name)] = attitude
}

func (r *npcRegistryImpl) UpdateAttitudesFromSummary(summary *prompt.SceneSummary) {
	if summary == nil {
		return
	}
	for _, npc := range summary.NPCsEncountered {
		r.attitudes[normalizeNPCName(npc.Name)] = npc.Attitude
	}
}

func (r *npcRegistryImpl) KnownSummaries() []prompt.NPCSummary {
	var summaries []prompt.NPCSummary
	for normalizedName, npc := range r.registry {
		if npc.IsPermanentlyRemoved() {
			continue
		}

		attitude := r.attitudes[normalizedName]
		if attitude == "" {
			attitude = "neutral"
		}

		if npc.IsTakenOut() && !npc.Fate.Permanent {
			attitude = fmt.Sprintf("defeated (%s)", npc.Fate.Description)
		}

		summaries = append(summaries, prompt.NPCSummary{
			Name:     npc.Name,
			Attitude: attitude,
		})
	}
	return summaries
}

func (r *npcRegistryImpl) Snapshot() (map[string]*character.Character, map[string]string) {
	registry := make(map[string]*character.Character, len(r.registry))
	for k, v := range r.registry {
		registry[k] = v
	}

	attitudes := make(map[string]string, len(r.attitudes))
	for k, v := range r.attitudes {
		attitudes[k] = v
	}

	return registry, attitudes
}

func (r *npcRegistryImpl) Restore(registry map[string]*character.Character, attitudes map[string]string) {
	if registry == nil {
		registry = make(map[string]*character.Character)
	}
	if attitudes == nil {
		attitudes = make(map[string]string)
	}
	r.registry = registry
	r.attitudes = attitudes
}

func (r *npcRegistryImpl) All() map[string]*character.Character {
	return r.registry
}

// normalizeNPCName normalizes an NPC name for matching (lowercase, trimmed)
func normalizeNPCName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
