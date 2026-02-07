
// getAvailableConsequences returns consequences that can absorb the given shifts for any character
func (sm *SceneManager) getAvailableConsequences(char *character.Character, shifts int) []ConsequenceOption {
available := []ConsequenceOption{}

// Use the character's CanTakeConsequence method which respects NPC type
if char.CanTakeConsequence(character.MildConsequence) {
available = append(available, ConsequenceOption{
Type:  character.MildConsequence,
Value: character.MildConsequence.Value(),
})
}
if char.CanTakeConsequence(character.ModerateConsequence) {
available = append(available, ConsequenceOption{
Type:  character.ModerateConsequence,
Value: character.ModerateConsequence.Value(),
})
}
if char.CanTakeConsequence(character.SevereConsequence) {
available = append(available, ConsequenceOption{
Type:  character.SevereConsequence,
Value: character.SevereConsequence.Value(),
})
}

return available
}
