package engine

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlexInt_UnmarshalJSON_Number(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected FlexInt
	}{
		{"positive", `3`, 3},
		{"zero", `0`, 0},
		{"negative", `-2`, -2},
		{"large positive", `8`, 8},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var fi FlexInt
			err := json.Unmarshal([]byte(tc.json), &fi)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, fi)
		})
	}
}

func TestFlexInt_UnmarshalJSON_String(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected FlexInt
	}{
		{"positive string", `"3"`, 3},
		{"zero string", `"0"`, 0},
		{"negative string", `"-2"`, -2},
		{"large positive string", `"8"`, 8},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var fi FlexInt
			err := json.Unmarshal([]byte(tc.json), &fi)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, fi)
		})
	}
}

func TestFlexInt_UnmarshalJSON_Errors(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string // substring expected in error message
	}{
		{"non-numeric string", `"abc"`, `cannot convert "abc" to int`},
		{"float string", `"3.5"`, `cannot convert "3.5" to int`},
		{"empty string", `""`, `cannot convert "" to int`},
		{"boolean", `true`, "cannot unmarshal"},
		{"array", `[1]`, "cannot unmarshal"},
		{"object", `{"a":1}`, "cannot unmarshal"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var fi FlexInt
			err := json.Unmarshal([]byte(tc.json), &fi)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestFlexInt_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    FlexInt
		expected string
	}{
		{"positive", 3, "3"},
		{"zero", 0, "0"},
		{"negative", -2, "-2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.value)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, string(data))
		})
	}
}

func TestFlexInt_Int(t *testing.T) {
	fi := FlexInt(42)
	assert.Equal(t, 42, fi.Int())
}

func TestFlexInt_UnmarshalJSON_Null(t *testing.T) {
	var fi FlexInt
	err := json.Unmarshal([]byte(`null`), &fi)
	require.NoError(t, err)
	assert.Equal(t, FlexInt(0), fi, "null should unmarshal to zero like standard Go int")
}

func TestFlexInt_IntConversion(t *testing.T) {
	fi := FlexInt(7)
	assert.Equal(t, 7, int(fi))
}

func TestFlexInt_InStruct(t *testing.T) {
	type sample struct {
		Name  string  `json:"name"`
		Value FlexInt `json:"value"`
	}

	t.Run("number in struct", func(t *testing.T) {
		input := `{"name":"test","value":5}`
		var s sample
		err := json.Unmarshal([]byte(input), &s)
		require.NoError(t, err)
		assert.Equal(t, "test", s.Name)
		assert.Equal(t, FlexInt(5), s.Value)
	})

	t.Run("string in struct", func(t *testing.T) {
		input := `{"name":"test","value":"5"}`
		var s sample
		err := json.Unmarshal([]byte(input), &s)
		require.NoError(t, err)
		assert.Equal(t, "test", s.Name)
		assert.Equal(t, FlexInt(5), s.Value)
	})

	t.Run("missing field defaults to zero", func(t *testing.T) {
		input := `{"name":"test"}`
		var s sample
		err := json.Unmarshal([]byte(input), &s)
		require.NoError(t, err)
		assert.Equal(t, FlexInt(0), s.Value)
	})
}

func TestFlexInt_RoundTrip(t *testing.T) {
	original := FlexInt(4)
	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded FlexInt
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestFlexInt_ActionParseResponseScenario(t *testing.T) {
	// Simulates the exact LLM output that causes failures:
	// difficulty returned as "3" instead of 3
	type response struct {
		ActionType string  `json:"action_type"`
		Skill      string  `json:"skill"`
		Difficulty FlexInt `json:"difficulty"`
		Confidence FlexInt `json:"confidence"`
	}

	t.Run("difficulty as string (LLM bug)", func(t *testing.T) {
		input := `{
			"action_type": "Overcome",
			"skill": "Stealth",
			"difficulty": "3",
			"confidence": 8
		}`
		var r response
		err := json.Unmarshal([]byte(input), &r)
		require.NoError(t, err)
		assert.Equal(t, FlexInt(3), r.Difficulty)
		assert.Equal(t, FlexInt(8), r.Confidence)
	})

	t.Run("difficulty as number (normal)", func(t *testing.T) {
		input := `{
			"action_type": "Overcome",
			"skill": "Stealth",
			"difficulty": 3,
			"confidence": 8
		}`
		var r response
		err := json.Unmarshal([]byte(input), &r)
		require.NoError(t, err)
		assert.Equal(t, FlexInt(3), r.Difficulty)
		assert.Equal(t, FlexInt(8), r.Confidence)
	})
}
