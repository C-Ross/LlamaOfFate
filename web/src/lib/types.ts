// TypeScript types mirroring Go GameEvent structs from uicontract/events.go.
// Event names match the wire protocol snake_case names used by MarshalEvent.

// ---------------------------------------------------------------------------
// Wire protocol envelope
// ---------------------------------------------------------------------------

/** Server → client JSON message envelope. */
export interface ServerMessage {
  event: string
  data: unknown
}

/** result_meta is sent after each InputResult to communicate flow-control state. */
export interface ResultMeta {
  awaitingInvoke: boolean
  awaitingMidFlow: boolean
  gameOver: boolean
  sceneEnded: boolean
}

// ---------------------------------------------------------------------------
// Client → server messages
// ---------------------------------------------------------------------------

export type ClientMessageType = "input" | "invoke_response" | "mid_flow_response" | "setup"

export interface ClientInputMessage {
  type: "input"
  text: string
}

export interface ClientInvokeMessage {
  type: "invoke_response"
  aspectIndex: number
  isReroll: boolean
}

export interface ClientMidFlowMessage {
  type: "mid_flow_response"
  choiceIndex: number
  freeText?: string
}

export interface ClientSetupPresetMessage {
  type: "setup"
  presetId: string
}

export interface CustomSetup {
  name: string
  highConcept: string
  trouble: string
  genre: string
}

export interface ClientSetupCustomMessage {
  type: "setup"
  custom: CustomSetup
}

export type ClientSetupMessage = ClientSetupPresetMessage | ClientSetupCustomMessage

export type ClientMessage =
  | ClientInputMessage
  | ClientInvokeMessage
  | ClientMidFlowMessage
  | ClientSetupMessage

// ---------------------------------------------------------------------------
// Server → client event data types (mirrors Go uicontract structs)
// ---------------------------------------------------------------------------

export interface NarrativeEventData {
  Text: string
  SceneName?: string
  Purpose?: string
}

export interface DialogEventData {
  PlayerInput: string
  GMResponse: string
  IsRecap?: boolean
  RecapType?: string // "dialog" | "action" | "conflict"
}

export interface SystemMessageEventData {
  Message: string
}

export interface ErrorNotificationEventData {
  Message: string
}

export interface ActionAttemptEventData {
  Description: string
}

export interface ActionResultEventData {
  Skill: string
  SkillRank: string      // Fate ladder name, e.g. "Average"
  SkillBonus: number     // Numeric skill level, e.g. 1
  Bonuses: number
  Result: string         // Legacy display string (fallback)
  Outcome: string
  DiceFaces?: number[]   // Individual die values (-1, 0, +1) from server
  Total: number          // Final roll value (dice + skill + bonuses)
  TotalRank: string      // Fate ladder name, e.g. "Fair"
  Difficulty: number     // Opposition difficulty or defense value
  DiffRank: string       // Fate ladder name, e.g. "Fair"
  DefenderName?: string  // Non-empty when rolling against a character
}

export interface SceneTransitionEventData {
  Narrative: string
  NewSceneHint: string
}

export interface GameOverEventData {
  Reason: string
}

export interface ConflictParticipantInfo {
  CharacterID: string
  CharacterName: string
  Initiative: number
  IsPlayer: boolean
}

export interface ConflictStartEventData {
  ConflictType: string
  InitiatorName: string
  Participants: ConflictParticipantInfo[]
}

export interface ConflictEscalationEventData {
  FromType: string
  ToType: string
  TriggerCharName: string
}

export interface TurnAnnouncementEventData {
  CharacterName: string
  TurnNumber: number
  IsPlayer: boolean
}

export interface ConflictEndEventData {
  Reason: string
}

export interface InvokableAspect {
  Name: string
  Source: string
  SourceID?: string
  FreeInvokes: number
  AlreadyUsed?: boolean
}

export interface InvokePromptEventData {
  Available: InvokableAspect[]
  FatePoints: number
  CurrentResult: string
  ShiftsNeeded: number
}

export interface InputOption {
  Label: string
  Description?: string
}

export type InputRequestType = "numbered_choice" | "free_text"

export interface InputRequestEventData {
  Type: InputRequestType
  Prompt: string
  Options?: InputOption[]
  Context?: Record<string, unknown>
}

export interface DefenseRollEventData {
  DefenderName: string
  Skill: string
  Result: string
  DiceFaces?: number[] // Individual die values (-1, 0, +1) from server
}

export interface StressAbsorptionDetail {
  TrackType: string
  Shifts: number
  TrackState: string
}

export interface ConsequenceDetail {
  Severity: string
  Aspect: string
  Absorbed: number
}

export interface DamageResolutionEventData {
  TargetName: string
  Absorbed?: StressAbsorptionDetail
  Consequence?: ConsequenceDetail
  RemainingAbsorbed?: StressAbsorptionDetail
  TakenOut: boolean
  VictoryEnd: boolean
}

export interface PlayerAttackResultEventData {
  TargetName: string
  Shifts: number
  IsTie: boolean
  TargetMissing: boolean
  TargetHint?: string
}

export interface AspectCreatedEventData {
  AspectName: string
  FreeInvokes: number
  IsBoost: boolean
}

export interface NPCAttackEventData {
  AttackerName: string
  TargetName: string
  AttackSkill: string
  AttackResult: string
  DefenseSkill: string
  DefenseResult: string
  FullDefense: boolean
  InitialOutcome: string
  FinalOutcome: string
  Narrative: string
}

export interface PlayerStressEventData {
  Shifts: number
  StressType: string
  TrackState: string
}

export interface PlayerDefendedEventData {
  IsTie: boolean
}

export interface PlayerConsequenceEventData {
  Severity: string
  Aspect: string
  Absorbed: number
  StressAbsorbed?: StressAbsorptionDetail
}

export interface PlayerTakenOutEventData {
  AttackerName: string
  Narrative: string
  Outcome: string
  NewSceneHint?: string
}

export interface ConcessionEventData {
  FatePointsGained: number
  ConsequenceCount: number
  CurrentFatePoints: number
}

export interface OutcomeChangedEventData {
  FinalOutcome: string
}

export interface InvokeEventData {
  AspectName: string
  IsFree: boolean
  IsReroll: boolean
  FatePointsLeft: number
  NewRoll?: string
  NewTotal: string
  Failed: boolean
}

export interface NPCActionResultEventData {
  NPCName: string
  ActionType: string
  Skill: string
  RollResult: string
  Difficulty: string
  Outcome: string
  AspectCreated?: string
  FreeInvokes?: number
}

export interface RecoveryEventData {
  Action: string
  Aspect: string
  Severity: string
  Skill?: string
  RollResult?: number
  Difficulty?: string
  Success?: boolean
}

export interface StressOverflowEventData {
  Shifts: number
  NoConsequences: boolean
  RemainingOverflow: boolean
}

export interface MilestoneEventData {
  Type: string
  ScenarioTitle: string
  FatePoints: number
}

export interface GameResumedEventData {
  ScenarioTitle: string
  SceneName: string
}

// ---------------------------------------------------------------------------
// Game state snapshot (sent once after Start() for sidebar initialisation)
// ---------------------------------------------------------------------------

export interface StressTrackSnapshot {
  boxes: boolean[]
  maxBoxes: number
}

export interface ConsequenceSnapshotEntry {
  severity: string
  aspect: string
  recovering: boolean
}

export interface PlayerSnapshot {
  name: string
  highConcept: string
  trouble: string
  aspects: string[]
  fatePoints: number
  refresh: number
  stressTracks: Record<string, StressTrackSnapshot>
  consequences: ConsequenceSnapshotEntry[]
}

export interface SituationAspectSnapshot {
  name: string
  freeInvokes: number
  isBoost?: boolean
}

export interface NPCSnapshot {
  name: string
  highConcept: string
  aspects: string[]
  isTakenOut: boolean
}

export interface GameStateSnapshotEventData {
  player: PlayerSnapshot
  sceneName: string
  situationAspects: SituationAspectSnapshot[]
  npcs: NPCSnapshot[]
  inConflict: boolean
}

// ---------------------------------------------------------------------------
// Discriminated union for typed event handling
// ---------------------------------------------------------------------------

/** All known event type names from the wire protocol. */
// ---------------------------------------------------------------------------
// Setup event data types
// ---------------------------------------------------------------------------

/** A preset scenario available for selection. */
export interface ScenarioPreset {
  id: string
  title: string
  genre: string
  description: string
}

/** Data for the setup_request event listing available presets. */
export interface SetupRequestEventData {
  presets: ScenarioPreset[]
  allowCustom: boolean
}

/** Data for the setup_generating event shown during LLM scenario generation. */
export interface SetupGeneratingEventData {
  message: string
}

export type GameEventType =
  | "player_input"
  | "setup_request"
  | "setup_generating"
  | "narrative"
  | "dialog"
  | "system_message"
  | "action_attempt"
  | "action_result"
  | "scene_transition"
  | "game_over"
  | "conflict_start"
  | "conflict_escalation"
  | "turn_announcement"
  | "conflict_end"
  | "invoke_prompt"
  | "input_request"
  | "defense_roll"
  | "damage_resolution"
  | "player_attack_result"
  | "aspect_created"
  | "npc_attack"
  | "player_stress"
  | "player_defended"
  | "player_consequence"
  | "player_taken_out"
  | "concession"
  | "outcome_changed"
  | "invoke"
  | "npc_action_result"
  | "recovery"
  | "stress_overflow"
  | "milestone"
  | "game_resumed"
  | "game_state_snapshot"
  | "error_notification"
  | "session_init"
  | "result_meta"
  | "setup_request"
  | "setup_generating"

/** A parsed game event with its type tag and data. */
export interface GameEvent {
  id: string
  event: GameEventType
  data: unknown
}

/** Data for a locally-generated player input echo. */
export interface PlayerInputEventData {
  text: string
}

/** Data for the session_init event sent immediately after WebSocket connection. */
export interface SessionInitEventData {
  gameId: string
}

/** Event types that should be displayed in the chat panel. */
export const CHAT_DISPLAYABLE_EVENTS: Set<string> = new Set([
  "player_input",
  "narrative",
  "dialog",
  "system_message",
  "action_attempt",
  "action_result",
  "scene_transition",
  "game_over",
  "conflict_start",
  "conflict_end",
  "turn_announcement",
  "input_request",
  "npc_attack",
  "player_attack_result",
  "damage_resolution",
  "defense_roll",
  "player_stress",
  "player_defended",
  "player_consequence",
  "player_taken_out",
  "concession",
  "outcome_changed",
  "invoke",
  "aspect_created",
  "npc_action_result",
  "recovery",
  "stress_overflow",
  "milestone",
  "game_resumed",
  "conflict_escalation",
])
