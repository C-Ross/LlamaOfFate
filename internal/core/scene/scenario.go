package scene

// Scenario represents a Fate Core scenario with its problem and story questions.
// Per Fate Core, a scenario is "a unit of game time usually lasting from one to four sessions"
// with "some kind of big, urgent, open-ended problem" to resolve.
type Scenario struct {
	Title          string   `json:"title" yaml:"title"`
	Problem        string   `json:"problem" yaml:"problem"`                 // The big urgent issue to resolve
	StoryQuestions []string `json:"story_questions" yaml:"story_questions"` // 2-4 yes/no questions answered during play
	Setting        string   `json:"setting" yaml:"setting"`                 // World/setting description
	Genre          string   `json:"genre" yaml:"genre"`                     // e.g., "Western", "Cyberpunk", "Fantasy"
	IsResolved     bool     `json:"is_resolved" yaml:"is_resolved"`
}
