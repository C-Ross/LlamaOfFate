---
name: session-log-parsing
description: Guide for reading and parsing session log files to extract test cases or analyze gameplay. Use this when asked to parse session logs, extract tests from gameplay, or analyze session transcripts.
---

# Session Log Parsing

Session logs capture the full back-and-forth of gameplay and evaluation tool output in YAML format for analysis and test extraction.

## Log File Types

| File Pattern | Source Tool | Description |
|---|---|---|
| `session_*.yaml` | `llm-scene-loop` | Full gameplay session logs |
| `scenario_gen_*.yaml` | `scenario-generator` | Scenario generation evaluation logs |
| `scene_gen_*.yaml` | `scene-generator` | Scene generation evaluation logs |
| `walkthrough_*.yaml` | `scenario-walkthrough` | Full scenario walkthrough logs |

## Log File Structure

All log files use YAML with `---` document separators between entries. Each entry has:
- `timestamp`: When the event occurred
- `type`: The event type
- `data`: Event-specific payload

## Event Types

### Gameplay Events (session_*.yaml)

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

### Scenario Generator Events (scenario_gen_*.yaml)

| Type | Description | Key Data Fields |
|------|-------------|-----------------|
| `scenario_generation_input` | Input data and rendered prompt | `player_name`, `high_concept`, `trouble`, `genre`, `theme_hint`, `rendered_prompt` |
| `scenario_generation_output` | Raw response and parsed scenario | `raw_response`, `generated_scenario` (title, problem, story_questions, genre, setting) |

### Scene Generator Events (scene_gen_*.yaml)

| Type | Description | Key Data Fields |
|------|-------------|-----------------|
| `scene_generation_input` | Input data and rendered prompt | `transition_hint`, `scenario`, `player_name`, `high_concept`, `trouble`, `summaries_count`, `rendered_prompt` |
| `scene_generation_output` | Raw response and parsed scene | `raw_response`, `generated_scene` (scene_name, description, situation_aspects, npcs) |

### Walkthrough Events (walkthrough_*.yaml)

| Type | Description | Key Data Fields |
|------|-------------|-----------------|
| `scenario` | Scenario used for walkthrough | Full `Scenario` struct (title, problem, story_questions, genre, setting) |
| `scenario_generation_prompt` | Rendered scenario generation prompt | `prompt` |
| `scene_generation_prompt` | Rendered scene generation prompt | `prompt` |
| `player_narration` | Player's narrated scene outcome | `scene_number`, `scene_name`, `narration` |
| `summary_generation_prompt` | Rendered summary prompt | `prompt` |
| `scene_summary` | Generated scene summary | `scene_number`, `summary` (narrative_prose, key_events, npcs_encountered, unresolved_threads) |
| `resolution_check_prompt` | Rendered resolution check prompt | `prompt` |
| `resolution_check` | Scenario resolution result | `scene_number`, `resolution` (is_resolved, answered_questions, reasoning) |
| `walkthrough_end` | Walkthrough ended | `reason` (quit/resolved/max_scenes), `scene_count` |

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
