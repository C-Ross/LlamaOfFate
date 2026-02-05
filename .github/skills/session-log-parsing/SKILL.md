---
name: session-log-parsing
description: Guide for reading and parsing session log files to extract test cases or analyze gameplay. Use this when asked to parse session logs, extract tests from gameplay, or analyze session transcripts.
---

# Session Log Parsing

Session logs (`session_*.yaml`) capture the full back-and-forth of gameplay in YAML format for analysis and test extraction.

## Log File Structure

Session logs use YAML with `---` document separators between entries. Each entry has:
- `timestamp`: When the event occurred
- `type`: The event type
- `data`: Event-specific payload

## Event Types

| Type | Description | Key Data Fields |
|------|-------------|-----------------|
| `scene_start` | Scene initialization | `scene_name`, `scene_description`, `characters`, `player_id` |
| `player_input` | Raw player text | `input` |
| `input_classification` | How input was classified | `input`, `classification` (dialog/action) |
| `action_parse` | Parsed action details | `id`, `characterid`, `type` (int), `skill`, `description`, `rawinput`, `target`, `difficulty` |
| `dice_roll` | Fate dice results | `skill`, `skill_level`, `bonus`, `difficulty`, `roll_result`, `final_value`, `outcome`, `shifts` |
| `narrative` | GM narrative after action | `action`, `outcome`, `text` |
| `dialog` | Conversational exchange | `player_input`, `gm_response` |
| `scene_transition` | Scene change triggered | `hint` |
| `taken_out` | Character defeated | `character_id`, `description` |

## Extracting Test Cases

When extracting test cases from session logs:

1. Find `input_classification` entries to get player inputs and their expected classification
2. Find `action_parse` entries following player inputs to get expected parse results
3. Use the preceding `scene_start` for scene context

Example test case extraction from a session log `action_parse` entry:
```go
testCase := ActionParseTestCase{
    Name:           "rapport_information_gathering",
    PlayerInput:    "Jesse smirks and lays down another dollar. \"Just a few questions. Where can I find the Cortez gang.\"",
    ExpectedSkill:  "Rapport",
    ExpectedTarget: "Maggie Two-Rivers",
    Description:    "Extracted from session_saloon_20260205.yaml",
}
```
