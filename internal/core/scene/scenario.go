package scene

import "strings"

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

// PredefinedScenario returns a predefined scenario by name.
// Supported names: "saloon", "heist", "tower" (case-insensitive).
// Returns nil if the name is not recognized.
func PredefinedScenario(name string) *Scenario {
	switch strings.ToLower(name) {
	case "tower":
		return &Scenario{
			Title:   "The Wizard's Tower",
			Problem: "A mysterious magical disturbance threatens the tower and its inhabitants",
			StoryQuestions: []string{
				"Can the source of the disturbance be discovered?",
				"Will the tower's secrets be revealed?",
			},
			Genre:   "Fantasy",
			Setting: "A medieval fantasy world of magic and mystery. Wizards study arcane arts in towers, adventurers seek treasure in ancient ruins, and supernatural forces are very real.",
		}
	case "heist":
		return &Scenario{
			Title:   "The Prometheus Job",
			Problem: "A high-value data core must be extracted from a heavily guarded corporate facility",
			StoryQuestions: []string{
				"Can the team breach the facility's security?",
				"Will the extraction succeed without casualties?",
				"What secrets does the data core contain?",
			},
			Genre:   "Cyberpunk",
			Setting: "A dark near-future where megacorporations rule, hackers breach digital fortresses, and chrome-enhanced mercenaries sell their skills to the highest bidder. Neon lights flicker over rain-slicked streets.",
		}
	case "saloon":
		return &Scenario{
			Title:   "Trouble in Redemption Gulch",
			Problem: "The town is under threat from outlaws and someone needs to stand up for the innocent",
			StoryQuestions: []string{
				"Will the outlaws be brought to justice?",
				"Can the town be saved?",
			},
			Genre:   "Western",
			Setting: "The American Old West in the late 1800s. Dusty frontier towns, lawless territories, and the struggle between civilization and the wild. Gunslingers, outlaws, and honest folk all seeking their fortune.",
		}
	default:
		return nil
	}
}
