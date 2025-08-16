package dice

import "fmt"

// Ladder represents the Fate Core adjective ladder
type Ladder int

// Fate Core ladder values
const (
	Terrible  Ladder = -2
	Poor      Ladder = -1
	Mediocre  Ladder = 0
	Average   Ladder = 1
	Fair      Ladder = 2
	Good      Ladder = 3
	Great     Ladder = 4
	Superb    Ladder = 5
	Fantastic Ladder = 6
	Epic      Ladder = 7
	Legendary Ladder = 8
)

// ladderNames maps ladder values to their adjective names
var ladderNames = map[Ladder]string{
	Terrible:  "Terrible",
	Poor:      "Poor",
	Mediocre:  "Mediocre",
	Average:   "Average",
	Fair:      "Fair",
	Good:      "Good",
	Great:     "Great",
	Superb:    "Superb",
	Fantastic: "Fantastic",
	Epic:      "Epic",
	Legendary: "Legendary",
}

// String returns the adjective name for the ladder value
func (l Ladder) String() string {
	if name, exists := ladderNames[l]; exists {
		return fmt.Sprintf("%s (%+d)", name, int(l))
	}
	if l > Legendary {
		return fmt.Sprintf("Legendary+ (%+d)", int(l))
	}
	if l < Terrible {
		return fmt.Sprintf("Terrible- (%+d)", int(l))
	}
	return fmt.Sprintf("Unknown (%+d)", int(l))
}

// IsValid checks if the ladder value is within reasonable bounds
func (l Ladder) IsValid() bool {
	return l >= -3 && l <= 10 // Allow some flexibility beyond core range
}

// Add safely adds a value to the ladder
func (l Ladder) Add(value int) Ladder {
	return Ladder(int(l) + value)
}

// Compare compares two ladder values and returns the difference
func (l Ladder) Compare(other Ladder) int {
	return int(l) - int(other)
}

// ParseLadder attempts to parse a string into a Ladder value
func ParseLadder(s string) (Ladder, error) {
	for ladder, name := range ladderNames {
		if s == name {
			return ladder, nil
		}
	}
	return Mediocre, fmt.Errorf("invalid ladder name: %s", s)
}
