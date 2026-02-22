---
name: fate-core-mechanics
description: Guide for implementing and modifying Fate Core System mechanics (dice, actions, aspects, stress, consequences, conflicts). Use this when asked to work with game rules, combat, character mechanics, or the dice/action system.
---

# Fate Core Mechanics

This skill covers the Fate Core tabletop RPG rules as implemented in the `internal/core/` packages. Consult the [Fate Core SRD](https://fate-srd.com/fate-core) for the authoritative rules reference.

This work is based on Fate Core System, a product of Evil Hat Productions, LLC, licensed under the [Creative Commons Attribution 3.0 Unported license](https://creativecommons.org/licenses/by/3.0/).

## The Ladder

The Fate ladder maps adjectives to numbers. See [SRD: Dice & The Ladder](https://fate-srd.com/fate-core/taking-action-dice-ladder).

| Value | Name      |
|-------|-----------|
| +8    | Legendary |
| +7    | Epic      |
| +6    | Fantastic |
| +5    | Superb    |
| +4    | Great     |
| +3    | Good      |
| +2    | Fair      |
| +1    | Average   |
| +0    | Mediocre  |
| -1    | Poor      |
| -2    | Terrible  |

**Code:** `internal/core/dice/ladder.go` — `dice.Ladder` type with named constants (`dice.Good`, `dice.Great`, etc.).

```go
level := dice.Good       // +3
name := level.String()   // "Good (+3)"
parsed, _ := dice.ParseLadder("Great")  // dice.Great
```

## Dice Rolling (4dF)

Roll four Fate dice (each -1, 0, or +1), sum them (range -4 to +4), add skill level and modifiers. See [SRD: Rolling the Dice](https://fate-srd.com/fate-core/taking-action-dice-ladder#rolling-the-dice).

**Code:** `internal/core/dice/dice.go`

```go
roller := dice.NewRoller()                          // Random
roller := dice.NewSeededRoller(12345)                // Deterministic (tests)
result := roller.RollWithModifier(dice.Good, 2)      // skill Good(+3) + modifier 2
outcome := result.CompareAgainst(dice.Great)          // vs difficulty Great(+4)
```

Key types:
- `Roll` — raw 4dF result (`Dice [4]FateDie`, `Total int`)
- `CheckResult` — roll + skill + modifier → `FinalValue Ladder`
- `Outcome` — comparison result: `Type OutcomeType`, `Shifts int`

## Four Outcomes

The difference between your roll and opposition determines the outcome. See [SRD: Four Outcomes](https://fate-srd.com/fate-core/four-outcomes).

| Outcome            | Shifts | Code Constant             |
|--------------------|--------|---------------------------|
| Fail               | < 0    | `dice.Failure`            |
| Tie                | 0      | `dice.Tie`                |
| Succeed            | 1–2    | `dice.Success`            |
| Succeed with Style | 3+     | `dice.SuccessWithStyle`   |

**Shifts** = your result - opposition. Positive shifts on an attack become the hit value the target must absorb.

## Four Actions

Every skill roll is one of four action types. See [SRD: Four Actions](https://fate-srd.com/fate-core/four-actions).

**Code:** `internal/core/action/action.go` — `action.ActionType` constants.

| Action           | Constant                  | Purpose |
|------------------|---------------------------|---------|
| Overcome         | `action.Overcome`         | Get past obstacles, achieve goals |
| Create Advantage | `action.CreateAdvantage`  | Make/discover aspects with free invokes |
| Attack           | `action.Attack`           | Inflict stress/consequences in conflict |
| Defend           | `action.Defend`           | Resist attacks or advantage creation |

### Outcome effects per action type

**Overcome** ([SRD](https://fate-srd.com/fate-core/four-actions#overcome)):
- Fail: don't achieve goal, or succeed at serious cost
- Tie: achieve goal at minor cost
- Succeed: achieve goal
- Succeed with Style: achieve goal + gain a boost

**Create Advantage** ([SRD](https://fate-srd.com/fate-core/four-actions#create-an-advantage)):
- *New aspect:* Fail → no aspect or opponent gets free invoke; Tie → boost instead; Succeed → aspect + 1 free invoke; SWS → aspect + 2 free invokes
- *Existing aspect:* Fail → opponent gets free invoke; Tie/Succeed → 1 free invoke; SWS → 2 free invokes

**Attack** ([SRD](https://fate-srd.com/fate-core/four-actions#attack)):
- Fail: no harm
- Tie: no harm, gain a boost
- Succeed: hit equal to shifts (target absorbs with stress/consequences)
- Succeed with Style: hit equal to shifts, may reduce by 1 to gain a boost

**Defend** ([SRD](https://fate-srd.com/fate-core/four-actions#defend)):
- Fail: suffer the attack/advantage
- Tie: grant opponent a boost
- Succeed: avoid the attack/advantage
- Succeed with Style: avoid + gain a boost

### Action construction

```go
act := action.NewAction("act-1", "player-1", action.Attack, "Fight", "Slash at the orc")
act := action.NewActionWithTarget("act-2", "player-1", action.Overcome, "Athletics", "Dodge the trap", "trap-1")

// Add aspect invocations before rolling
act.AddAspectInvoke(action.AspectInvoke{
    AspectText:    "Infamous Girl with Sword",
    Source:        "character",
    IsFree:        false,
    FatePointCost: 1,
    Bonus:         2,   // +2 per invoke (standard)
})

bonus := act.CalculateBonus()  // Sum of all invoke bonuses
```

## Aspects & Invocation

Aspects are narrative truths that can be invoked for mechanical benefit. See [SRD: Invoking & Compelling](https://fate-srd.com/fate-core/invoking-compelling-aspects).

**Invoking an aspect** (spend 1 fate point or use a free invoke):
- +2 to your roll, OR
- Reroll all dice, OR
- +2 to another character's roll, OR
- +2 to passive opposition

You can invoke **multiple different** aspects on one roll but **not the same aspect twice** on one roll. Free invokes stack with paid invokes on the same aspect (+4 total).

**Code:** `action.AspectInvoke` struct, applied via `action.AddAspectInvoke()`. The `CheckResult.ApplyInvokeBonus(bonus)` method adjusts the final value.

### Character Aspects (Dynamic Model)

LlamaOfFate uses a **flexible aspect model**, not the traditional fixed 5-aspect layout.

**Code:** `internal/core/character/character.go` — `character.Aspects` struct.

```go
char.Aspects.HighConcept = "Wizard Detective"     // Required
char.Aspects.Trouble = "The Lure of Ancient Mysteries"  // Required
char.Aspects.AddAspect("Well Connected")           // Unlimited additional
char.Aspects.AddAspect("Student of the Old Ways")
allAspects := char.Aspects.GetAll()                // Returns all as []string
```

### Situation Aspects

Created during play, attached to the scene. See [SRD: Types of Aspects](https://fate-srd.com/fate-core/types-aspects).

**Code:** `internal/core/scene/scene.go` — `scene.SituationAspect`.

```go
aspect := scene.NewSituationAspect("sa-1", "On Fire", "npc-1", 1)  // 1 free invoke
scene.AddSituationAspect(aspect)
used := aspect.UseFreeInvoke()  // Returns true if free invoke was available
```

## Skills

Skills are stored as `map[string]dice.Ladder` on characters. The `internal/core/skills.go` file classifies skills for conflict resolution:

| Category | Skills |
|----------|--------|
| Physical attack | Fight, Shoot, Physique |
| Mental attack | Provoke, Deceive, Rapport, Lore |
| Physical defense | Athletics (default for Fight/Shoot/Physique attacks) |
| Mental defense | Will (default for Provoke/Deceive/Rapport attacks) |

**Key functions in `internal/core/skills.go`:**
- `DefenseSkillForAttack(attackSkill) string` — determines which skill defends
- `StressTypeForAttack(attackSkill) StressTrackType` — determines physical vs mental stress
- `ConflictTypeForSkill(skill) ConflictType` — maps skill to conflict type
- `InitiativeSkillsForConflict(type) []string` — Physical: Notice, Athletics; Mental: Empathy, Rapport
- `DefaultAttackSkillForConflict(type) string` — Physical → Fight, Mental → Provoke

```go
char.SetSkill("Fight", dice.Good)       // +3
char.SetSkill("Athletics", dice.Fair)   // +2
level := char.GetSkill("Fight")         // dice.Good; returns dice.Mediocre if unset
```

## Stress

Stress represents ephemeral harm absorbed during conflict. Clears after the scene. See [SRD: Stress](https://fate-srd.com/fate-core/resolving-attacks#stress).

**Rules:**
- Two tracks: physical and mental
- On a hit, check off a stress box **with value equal to or greater than** the shift value
- Each box can only be checked once; only **one box per hit**
- Track sizes scale with skills (per SRD):
  - **Physique** scales physical stress: Mediocre/no skill → 2 boxes, Average/Fair → 3 boxes, Good+ → 4 boxes
  - **Will** scales mental stress: same progression
- Tracks automatically recalculate when Physique or Will changes

**Code:** `internal/core/character/character.go` — `character.StressTrack`.

```go
char.SetSkill("Physique", dice.Good)  // Physical stress → 4 boxes
char.SetSkill("Will", dice.Fair)      // Mental stress → 3 boxes
char.RecalculateStressTracks()        // Explicit recalc (auto-called by SetSkill)

track := char.GetStressTrack(character.PhysicalStress)
absorbed := char.TakeStress(character.PhysicalStress, 3)  // Try to check off box 3+
char.ClearAllStress()  // After conflict ends
track.IsFull()         // All boxes checked?
track.AvailableBoxes() // Remaining unchecked count
```

`TakeStress(type, amount)` uses **1-indexed Fate Core style**: a 3-shift hit checks box 3 (or higher if 3 is taken). Returns `false` if no box available.

## Consequences

Consequences are lasting injuries/trauma that absorb hits but create negative aspects. See [SRD: Consequences](https://fate-srd.com/fate-core/resolving-attacks#consequences).

| Severity | Shifts Absorbed | Recovery Time | Code Constant |
|----------|-----------------|---------------|---------------|
| Mild     | 2               | 1 scene       | `character.MildConsequence`     |
| Moderate | 4               | 1 session     | `character.ModerateConsequence` |
| Severe   | 6               | 1 scenario    | `character.SevereConsequence`   |
| Extreme  | 8               | Permanent*    | `character.ExtremeConsequence`  |

*Extreme consequences replace a character aspect permanently.

**Rules:**
- One slot per severity level (mild, moderate, severe) — shared across physical and mental
- **Extra mild slots** at Superb+ (per SRD): Superb Physique → +1 physical mild, Superb Will → +1 mental mild
- Can combine stress + consequences to absorb a single hit
- The attacker gets a **free invoke** on your consequence aspect
- Recovery requires an overcome roll: Mild → Fair(+2), Moderate → Great(+4), Severe → Fantastic(+6)

**Code:** `internal/core/character/character.go`

```go
char.CanTakeConsequence(character.MildConsequence)  // Check slot availability
char.AddConsequence(character.Consequence{
    ID:   "con-1",
    Type: character.MildConsequence,
    Aspect: "Black Eye",
})
char.BeginConsequenceRecovery("con-1", sceneCount, scenarioCount)
recovered := char.CheckConsequenceRecovery(currentScene, currentScenario)
slots := char.AvailableConsequenceSlots()  // Returns []ConsequenceSlot
```

## Conflicts

Conflicts are structured exchanges where characters try to harm each other. See [SRD: Conflicts](https://fate-srd.com/fate-core/conflicts).

**Types:** Physical (fists, weapons) or Mental (intimidation, psychic). Code: `scene.PhysicalConflict`, `scene.MentalConflict`.

### Conflict flow

1. **Set the scene** — situation aspects, zones, participants
2. **Determine turn order** — initiative based on Notice/Athletics (physical) or Empathy/Rapport (mental)
3. **Exchanges** — each participant takes one action per round (Attack, Create Advantage, Overcome, or Defend)
4. **Resolve** — conflict ends when one side concedes or is taken out

**Code:** `internal/core/scene/scene.go` — `scene.ConflictState`, `scene.ConflictParticipant`.

```go
scene.StartConflictWithInitiator(scene.PhysicalConflict, participants, "npc-1")
actor := scene.GetCurrentActor()  // Current turn's character ID
scene.NextTurn()                  // Advance to next participant
scene.CountActiveParticipants()   // Excludes taken-out/conceded
scene.EndConflict()               // Clears conflict state
```

### Taken Out

If you can't absorb all shifts from a hit (no stress boxes or consequences left), you're **taken out**. The attacker narrates your defeat. See [SRD: Getting Taken Out](https://fate-srd.com/fate-core/getting-taken-out).

```go
scene.MarkCharacterTakenOut("npc-1")
scene.SetParticipantStatus("npc-1", scene.StatusTakenOut)
```

### Concession

A character can concede **before** a roll to preserve some narrative control. They get 1 fate point + 1 per consequence taken in the conflict. See [SRD: Conceding](https://fate-srd.com/fate-core/conflicts#conceding-a-conflict).

```go
fatePoints := ConcessionFatePoints(consequenceCount)  // 1 + consequenceCount
scene.SetParticipantStatus("player-1", scene.StatusConceded)
```

### Full Defense

Forfeit your action to gain +2 to all defense rolls for the exchange.

```go
scene.SetFullDefense("player-1")
isDefending := scene.IsFullDefense("player-1")
```

## Character Types

NPC threat levels control their mechanical weight. See `internal/core/character/character_type.go`.

| Type | Peak Skill | Stress | Consequences |
|------|-----------|--------|--------------|
| PC | Any | Full tracks | Full slots |
| Main NPC | Any | Full tracks | Full slots |
| Supporting NPC | Any | Full tracks | Mild + Moderate only |
| Nameless (Good) | Good(+3) | 2 boxes | None |
| Nameless (Fair) | Fair(+2) | 1 box | None |
| Nameless (Average) | Average(+1) | 0 boxes | None |

```go
thug := character.NewNamelessNPC("thug-1", "Street Thug", character.CharacterTypeNamelessFair, "Fight")
thug.CharacterType.IsNameless()         // true
thug.CharacterType.HasConsequences()    // false
```

## Fate Points

Characters spend fate points to invoke aspects and earn them through compels. See [SRD: Fate Points](https://fate-srd.com/fate-core/aspects-fate-points#defining-fate-points).

```go
char.FatePoints  // Current pool (default 3)
char.Refresh     // Reset value per scenario (default 3)
char.SpendFatePoint()    // Returns false if none available
char.GainFatePoint()     // +1
char.RefreshFatePoints() // Reset to Refresh value
```

## Resolving an Attack (Full Example)

Putting it all together — the complete resolution pipeline for an attack:

```go
// 1. Create the action
act := action.NewActionWithTarget("a1", "player", action.Attack, "Fight", "Slash the orc", "orc-1")

// 2. Invoke aspects for bonus
act.AddAspectInvoke(action.AspectInvoke{
    AspectText: "Infamous Girl with Sword", Source: "character",
    IsFree: false, FatePointCost: 1, Bonus: 2,
})
attacker.SpendFatePoint()

// 3. Roll with skill + bonus
roller := dice.NewRoller()
result := roller.RollWithModifier(attacker.GetSkill("Fight"), act.CalculateBonus())

// 4. Defender rolls  
defenseResult := roller.RollWithModifier(defender.GetSkill("Athletics"), 0)

// 5. Compare
outcome := result.CompareAgainst(defenseResult.FinalValue)
// outcome.Type → dice.Success, outcome.Shifts → 3

// 6. Target absorbs (stress first, then consequences)
if !defender.TakeStress(character.PhysicalStress, outcome.Shifts) {
    // Must take consequence or be taken out
    if defender.CanTakeConsequence(character.MildConsequence) && outcome.Shifts <= 2 {
        defender.AddConsequence(character.Consequence{
            Type: character.MildConsequence, Aspect: "Nasty Cut",
        })
    } else {
        scene.MarkCharacterTakenOut(defender.ID)
    }
}
```

## SRD Quick Reference

| Topic | SRD Link |
|-------|----------|
| Dice & Ladder | https://fate-srd.com/fate-core/taking-action-dice-ladder |
| Four Actions | https://fate-srd.com/fate-core/four-actions |
| Four Outcomes | https://fate-srd.com/fate-core/four-outcomes |
| Aspects & Fate Points | https://fate-srd.com/fate-core/aspects-fate-points |
| Invoking & Compelling | https://fate-srd.com/fate-core/invoking-compelling-aspects |
| Stress & Consequences | https://fate-srd.com/fate-core/stress-consequences |
| Resolving Attacks | https://fate-srd.com/fate-core/resolving-attacks |
| Conflicts | https://fate-srd.com/fate-core/conflicts |
| Getting Taken Out | https://fate-srd.com/fate-core/getting-taken-out |
| Skills List | https://fate-srd.com/fate-core/default-skill-list |
| Types of Aspects | https://fate-srd.com/fate-core/types-aspects |
