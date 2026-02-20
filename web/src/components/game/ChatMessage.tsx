import { cn } from "@/lib/utils"
import { RollResult } from "@/components/game/RollResult"
import { ActionAttempt } from "@/components/game/ActionAttempt"
import { DefenseRoll } from "@/components/game/DefenseRoll"
import { OutcomeBadge } from "@/components/game/OutcomeBadge"
import { TurnAnnouncement } from "@/components/game/TurnAnnouncement"
import { DamageResolution } from "@/components/game/DamageResolution"
import { NPCAction } from "@/components/game/NPCAction"
import { ConflictEnd } from "@/components/game/ConflictEnd"
import type {
  GameEvent,
  PlayerInputEventData,
  NarrativeEventData,
  DialogEventData,
  SystemMessageEventData,
  ActionAttemptEventData,
  ActionResultEventData,
  SceneTransitionEventData,
  GameOverEventData,
  ConflictStartEventData,
  ConflictEndEventData,
  TurnAnnouncementEventData,
  InputRequestEventData,
  NPCAttackEventData,
  PlayerAttackResultEventData,
  DamageResolutionEventData,
  DefenseRollEventData,
  PlayerStressEventData,
  PlayerDefendedEventData,
  PlayerConsequenceEventData,
  PlayerTakenOutEventData,
  ConcessionEventData,
  OutcomeChangedEventData,
  InvokeEventData,
  AspectCreatedEventData,
  NPCActionResultEventData,
  RecoveryEventData,
  StressOverflowEventData,
  MilestoneEventData,
  GameResumedEventData,
  ConflictEscalationEventData,
} from "@/lib/types"

interface ChatMessageProps {
  event: GameEvent
  className?: string
}

export function ChatMessage({ event, className }: ChatMessageProps) {
  return (
    <div className={cn("animate-in fade-in-0 slide-in-from-bottom-2 duration-300", className)}>
      {renderEvent(event)}
    </div>
  )
}

function renderEvent(event: GameEvent) {
  switch (event.event) {
    case "player_input":
      return <PlayerInputMessage data={event.data as PlayerInputEventData} />
    case "narrative":
      return <NarrativeMessage data={event.data as NarrativeEventData} />
    case "dialog":
      return <DialogMessage data={event.data as DialogEventData} />
    case "system_message":
      return <SystemMessage data={event.data as SystemMessageEventData} />
    case "action_attempt":
      return <ActionAttemptMessage data={event.data as ActionAttemptEventData} />
    case "action_result":
      return <ActionResultMessage data={event.data as ActionResultEventData} />
    case "scene_transition":
      return <SceneTransitionMessage data={event.data as SceneTransitionEventData} />
    case "game_over":
      return <GameOverMessage data={event.data as GameOverEventData} />
    case "conflict_start":
      return <ConflictStartMessage data={event.data as ConflictStartEventData} />
    case "conflict_end":
      return <ConflictEndMessage data={event.data as ConflictEndEventData} />
    case "conflict_escalation":
      return <ConflictEscalationMessage data={event.data as ConflictEscalationEventData} />
    case "turn_announcement":
      return <TurnAnnouncementMessage data={event.data as TurnAnnouncementEventData} />
    case "input_request":
      return <InputRequestMessage data={event.data as InputRequestEventData} />
    case "npc_attack":
      return <NPCAttackMessage data={event.data as NPCAttackEventData} />
    case "player_attack_result":
      return <PlayerAttackResultMessage data={event.data as PlayerAttackResultEventData} />
    case "damage_resolution":
      return <DamageResolutionMessage data={event.data as DamageResolutionEventData} />
    case "defense_roll":
      return <DefenseRollMessage data={event.data as DefenseRollEventData} />
    case "player_stress":
      return <PlayerStressMessage data={event.data as PlayerStressEventData} />
    case "player_defended":
      return <PlayerDefendedMessage data={event.data as PlayerDefendedEventData} />
    case "player_consequence":
      return <PlayerConsequenceMessage data={event.data as PlayerConsequenceEventData} />
    case "player_taken_out":
      return <PlayerTakenOutMessage data={event.data as PlayerTakenOutEventData} />
    case "concession":
      return <ConcessionMessage data={event.data as ConcessionEventData} />
    case "outcome_changed":
      return <OutcomeChangedMessage data={event.data as OutcomeChangedEventData} />
    case "invoke":
      return <InvokeMessage data={event.data as InvokeEventData} />
    case "aspect_created":
      return <AspectCreatedMessage data={event.data as AspectCreatedEventData} />
    case "npc_action_result":
      return <NPCActionResultMessage data={event.data as NPCActionResultEventData} />
    case "recovery":
      return <RecoveryMessage data={event.data as RecoveryEventData} />
    case "stress_overflow":
      return <StressOverflowMessage data={event.data as StressOverflowEventData} />
    case "milestone":
      return <MilestoneMessage data={event.data as MilestoneEventData} />
    case "game_resumed":
      return <GameResumedMessage data={event.data as GameResumedEventData} />
    default:
      return null
  }
}

// ---------------------------------------------------------------------------
// Player input echo
// ---------------------------------------------------------------------------

function PlayerInputMessage({ data }: { data: PlayerInputEventData }) {
  return (
    <div className="flex justify-end">
      <div className="rounded-lg bg-primary/15 border border-primary/30 px-4 py-3 max-w-[80%]">
        <div className="text-xs font-heading uppercase tracking-wide text-primary/70 mb-1">
          You
        </div>
        <div className="text-sm font-body text-foreground whitespace-pre-wrap">
          {data.text}
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// GM prose / narrative
// ---------------------------------------------------------------------------

function NarrativeMessage({ data }: { data: NarrativeEventData }) {
  return (
    <div className="space-y-2">
      {data.SceneName && (
        <div className="flex items-center gap-2 text-xs font-heading uppercase tracking-widest text-primary">
          <span className="h-px flex-1 bg-primary/30" />
          {data.SceneName}
          <span className="h-px flex-1 bg-primary/30" />
        </div>
      )}
      <div className="rounded-lg bg-secondary/50 px-4 py-3 text-sm font-body text-foreground leading-relaxed whitespace-pre-wrap">
        {data.Text}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Dialog (player + GM)
// ---------------------------------------------------------------------------

function DialogMessage({ data }: { data: DialogEventData }) {
  // Conflict recap entries (e.g. "conflict initiated", "conceded") get a
  // distinctive system-style banner instead of a dialog bubble.
  if (data.IsRecap && data.RecapType === "conflict") {
    return (
      <div className="flex items-center gap-2 text-xs font-heading uppercase tracking-widest text-primary/60 opacity-70">
        <span className="h-px flex-1 bg-primary/20" />
        {data.GMResponse}
        <span className="h-px flex-1 bg-primary/20" />
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {data.IsRecap && data.PlayerInput && (
        <div className="flex justify-end">
          <div className="rounded-lg bg-primary/15 border border-primary/30 px-4 py-3 max-w-[80%] opacity-70">
            <div className="text-xs font-heading uppercase tracking-wide text-primary/70 mb-1">
              You
            </div>
            <div className="text-sm font-body text-foreground whitespace-pre-wrap">
              {data.PlayerInput}
            </div>
          </div>
        </div>
      )}
      {data.GMResponse && (
        <div className={cn(
          "rounded-lg bg-secondary/50 px-4 py-3 text-sm font-body text-foreground leading-relaxed whitespace-pre-wrap",
          data.IsRecap && "opacity-70",
        )}>
          {data.GMResponse}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// System / info messages
// ---------------------------------------------------------------------------

function SystemMessage({ data }: { data: SystemMessageEventData }) {
  return (
    <div className="px-4 py-2 text-xs text-muted-foreground italic font-body">
      {data.Message}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Mechanical events
// ---------------------------------------------------------------------------

function ActionAttemptMessage({ data }: { data: ActionAttemptEventData }) {
  return <ActionAttempt data={data} />
}

function ActionResultMessage({ data }: { data: ActionResultEventData }) {
  return <RollResult data={data} />
}

function SceneTransitionMessage({ data }: { data: SceneTransitionEventData }) {
  return (
    <div className="my-4 space-y-2">
      <div className="flex items-center gap-2 text-xs font-heading uppercase tracking-widest text-muted-foreground">
        <span className="h-px flex-1 bg-border" />
        Scene Transition
        <span className="h-px flex-1 bg-border" />
      </div>
      {data.Narrative && (
        <div className="px-4 text-sm font-body text-muted-foreground italic">
          {data.Narrative}
        </div>
      )}
    </div>
  )
}

function GameOverMessage({ data }: { data: GameOverEventData }) {
  return (
    <div className="my-4 rounded-lg border-2 border-destructive bg-destructive/10 px-4 py-3 text-center">
      <div className="font-heading text-lg font-bold text-destructive">Game Over</div>
      {data.Reason && <div className="mt-1 text-sm font-body text-muted-foreground">{data.Reason}</div>}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Conflict events
// ---------------------------------------------------------------------------

function ConflictStartMessage({ data }: { data: ConflictStartEventData }) {
  return (
    <div className="my-3 rounded-lg border border-destructive/50 bg-destructive/5 px-4 py-3 space-y-2">
      <div className="font-heading text-sm font-bold uppercase tracking-wide text-destructive">
        Conflict! ({data.ConflictType})
      </div>
      <div className="text-xs font-body text-muted-foreground">
        Initiated by {data.InitiatorName} — {data.Participants?.length ?? 0} participants
      </div>
    </div>
  )
}

function ConflictEndMessage({ data }: { data: ConflictEndEventData }) {
  return <ConflictEnd data={data} />
}

function ConflictEscalationMessage({ data }: { data: ConflictEscalationEventData }) {
  return (
    <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-2 text-sm font-body">
      <span className="font-heading text-xs uppercase tracking-wide text-destructive">Escalation: </span>
      {data.TriggerCharName} escalates from {data.FromType} to {data.ToType}!
    </div>
  )
}

function TurnAnnouncementMessage({ data }: { data: TurnAnnouncementEventData }) {
  return <TurnAnnouncement data={data} />
}

// ---------------------------------------------------------------------------
// Invoke / Input prompts
// ---------------------------------------------------------------------------

function InputRequestMessage({ data }: { data: InputRequestEventData }) {
  return (
    <div className="rounded-lg border border-primary/30 bg-card px-4 py-3 space-y-2">
      <div className="text-sm font-body text-foreground">{data.Prompt}</div>
      {data.Options?.map((opt, i) => (
        <div key={i} className="rounded bg-secondary/50 px-3 py-1 text-sm font-body">
          <span className="font-bold text-primary mr-1">{i + 1}.</span> {opt.Label}
          {opt.Description && (
            <span className="text-xs text-muted-foreground ml-1">— {opt.Description}</span>
          )}
        </div>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Attack / defense events
// ---------------------------------------------------------------------------

function NPCAttackMessage({ data }: { data: NPCAttackEventData }) {
  return <NPCAction data={data} />
}

function PlayerAttackResultMessage({ data }: { data: PlayerAttackResultEventData }) {
  if (data.TargetMissing) {
    return (
      <div className="px-4 py-2 text-sm font-body text-muted-foreground italic">
        Target not found{data.TargetHint ? ` (${data.TargetHint})` : ""}
      </div>
    )
  }
  return (
    <div className="rounded-lg border border-primary/30 bg-card px-4 py-2 text-sm font-body">
      <span className="font-heading text-xs uppercase tracking-wide text-muted-foreground">Hit: </span>
      {data.Shifts} shift{data.Shifts !== 1 ? "s" : ""} on {data.TargetName}
      {data.IsTie && <span className="text-boost ml-1">(Tie — boost gained!)</span>}
    </div>
  )
}

function DamageResolutionMessage({ data }: { data: DamageResolutionEventData }) {
  return <DamageResolution data={data} />
}

function DefenseRollMessage({ data }: { data: DefenseRollEventData }) {
  return <DefenseRoll data={data} />
}

// ---------------------------------------------------------------------------
// Player damage events
// ---------------------------------------------------------------------------

function PlayerStressMessage({ data }: { data: PlayerStressEventData }) {
  return (
    <div className="rounded-lg border border-destructive/30 bg-card px-4 py-2 text-sm font-body">
      <span className="font-heading text-xs uppercase tracking-wide text-destructive">Stress: </span>
      {data.Shifts} {data.StressType} stress absorbed {data.TrackState}
    </div>
  )
}

function PlayerDefendedMessage({ data }: { data: PlayerDefendedEventData }) {
  return (
    <div className="rounded-lg border border-boost/30 bg-boost/5 px-4 py-2 text-sm font-body text-boost">
      Defended!{data.IsTie ? " (Tie — attacker gets a boost)" : ""}
    </div>
  )
}

function PlayerConsequenceMessage({ data }: { data: PlayerConsequenceEventData }) {
  return (
    <div className={cn("rounded-lg border bg-card px-4 py-2 text-sm font-body space-y-1",
      consequenceBorderColor(data.Severity))}>
      <div className={cn("font-heading text-xs uppercase tracking-wide", consequenceColor(data.Severity))}>
        {data.Severity} consequence
      </div>
      <div className="font-bold">{data.Aspect}</div>
      <div className="text-xs text-muted-foreground">Absorbs {data.Absorbed} shift{data.Absorbed !== 1 ? "s" : ""}</div>
    </div>
  )
}

function PlayerTakenOutMessage({ data }: { data: PlayerTakenOutEventData }) {
  return (
    <div className="rounded-lg border-2 border-destructive bg-destructive/10 px-4 py-3 space-y-2">
      <div className="font-heading text-sm font-bold text-destructive">
        Taken Out by {data.AttackerName}!
      </div>
      {data.Narrative && (
        <div className="text-sm font-body text-foreground whitespace-pre-wrap">{data.Narrative}</div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Miscellaneous mechanical events
// ---------------------------------------------------------------------------

function ConcessionMessage({ data }: { data: ConcessionEventData }) {
  return (
    <div className="rounded-lg border border-primary/30 bg-card px-4 py-2 text-sm font-body">
      <span className="font-heading text-xs uppercase tracking-wide text-primary">Concession: </span>
      Gained {data.FatePointsGained} fate point{data.FatePointsGained !== 1 ? "s" : ""} (now {data.CurrentFatePoints})
    </div>
  )
}

function OutcomeChangedMessage({ data }: { data: OutcomeChangedEventData }) {
  return (
    <div className="px-4 py-1 text-xs font-heading uppercase text-primary">
      Outcome changed → {data.FinalOutcome}
    </div>
  )
}

function InvokeMessage({ data }: { data: InvokeEventData }) {
  if (data.Failed) {
    return (
      <div className="px-4 py-1 text-xs font-body text-destructive italic">
        Failed to invoke {data.AspectName} (not enough fate points)
      </div>
    )
  }
  return (
    <div className="rounded-lg border border-fate-point/30 bg-fate-point/5 px-4 py-2 text-sm font-body">
      <span className="font-heading text-xs uppercase tracking-wide text-fate-point">Invoke: </span>
      {data.AspectName}
      {data.IsFree ? " (free)" : ""}
      {data.IsReroll ? ` — reroll: ${data.NewRoll}` : " — +2"}
      {" → "}{data.NewTotal}
      {!data.IsFree && (
        <span className="text-xs text-muted-foreground ml-1">({data.FatePointsLeft} FP left)</span>
      )}
    </div>
  )
}

function AspectCreatedMessage({ data }: { data: AspectCreatedEventData }) {
  return (
    <div className="rounded-lg border border-boost/30 bg-boost/5 px-4 py-2 text-sm font-body">
      <span className="font-heading text-xs uppercase tracking-wide text-boost">New Aspect: </span>
      <span className="font-bold">{data.AspectName}</span>
      {data.FreeInvokes > 0 && (
        <span className="text-xs text-muted-foreground ml-1">
          ({data.FreeInvokes} free invoke{data.FreeInvokes !== 1 ? "s" : ""})
        </span>
      )}
    </div>
  )
}

function NPCActionResultMessage({ data }: { data: NPCActionResultEventData }) {
  return (
    <div className="rounded-lg border border-border bg-card px-4 py-2 text-sm font-body">
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <span className="font-heading text-xs uppercase tracking-wide text-muted-foreground">
          {data.NPCName} — {data.ActionType}:
        </span>
        {data.Outcome && <OutcomeBadge outcome={data.Outcome} size="sm" />}
      </div>
      <div className="mt-1">
        {data.Skill && `${data.Skill} `}
        {data.RollResult && <span className="text-muted-foreground">{data.RollResult}</span>}
        {data.AspectCreated && (
          <span className="ml-2 text-boost">→ {data.AspectCreated}</span>
        )}
      </div>
    </div>
  )
}

function RecoveryMessage({ data }: { data: RecoveryEventData }) {
  return (
    <div className="rounded-lg border border-boost/30 bg-boost/5 px-4 py-2 text-sm font-body">
      <span className="font-heading text-xs uppercase tracking-wide text-boost">Recovery: </span>
      {data.Aspect} ({data.Severity}) — {data.Action === "healed" ? "Healed!" : (
        <span>{data.Skill} roll: {data.Success ? "Success" : "Failed"}</span>
      )}
    </div>
  )
}

function StressOverflowMessage({ data }: { data: StressOverflowEventData }) {
  return (
    <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-2 text-sm font-body text-destructive">
      <span className="font-heading text-xs uppercase tracking-wide">Stress Overflow: </span>
      {data.Shifts} shift{data.Shifts !== 1 ? "s" : ""} could not be absorbed
      {data.NoConsequences && " — no consequences available!"}
    </div>
  )
}

function MilestoneMessage({ data }: { data: MilestoneEventData }) {
  return (
    <div className="my-3 rounded-lg border-2 border-primary bg-primary/10 px-4 py-3 text-center space-y-1">
      <div className="font-heading text-sm font-bold uppercase text-primary">
        Milestone: {data.ScenarioTitle}
      </div>
      <div className="text-xs font-body text-muted-foreground">
        Fate points: {data.FatePoints}
      </div>
    </div>
  )
}

function GameResumedMessage({ data }: { data: GameResumedEventData }) {
  return (
    <div className="px-4 py-2 text-xs text-muted-foreground italic font-body">
      Resumed: {data.ScenarioTitle} — {data.SceneName}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function consequenceColor(severity: string): string {
  switch (severity.toLowerCase()) {
    case "mild": return "text-consequence-mild"
    case "moderate": return "text-consequence-moderate"
    case "severe": return "text-consequence-severe"
    default: return "text-foreground"
  }
}

function consequenceBorderColor(severity: string): string {
  switch (severity.toLowerCase()) {
    case "mild": return "border-consequence-mild/30"
    case "moderate": return "border-consequence-moderate/30"
    case "severe": return "border-consequence-severe/30"
    default: return "border-border"
  }
}
