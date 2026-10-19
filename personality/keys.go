package personality

// Key names one message in every catalog: dotted, stable and readable.
type Key string

// The keys, by where the message is shown (0022-PLAN P1). The boring
// catalog has each one, and nothing else.
const (
	// gobble's CLI: status lines on stderr, the spinner, the turn's last
	// line, cut tool output.
	RetryOffline            Key = "retry.offline"
	RetryEmptyReply         Key = "retry.empty_reply"
	RetryAttempt            Key = "retry.attempt"
	CompactionStart         Key = "compaction.start"
	CompactionFailed        Key = "compaction.failed"
	CompactionFailedTurn    Key = "compaction.failed.paused_turn"
	CompactionFailedSession Key = "compaction.failed.paused_session"
	TurnWorking             Key = "turn.working"
	TurnStopMaxTokens       Key = "turn.stop.max_tokens"
	TurnStopMaxRequests     Key = "turn.stop.max_turn_requests"
	TurnStopRefusal         Key = "turn.stop.refusal"
	TurnStopCancelled       Key = "turn.stop.cancelled"
	LinesHidden             Key = "lines.hidden"

	// The agent's slash commands, through any ACP client.
	CommandUnknown    Key = "command.unknown"
	HelpHeading       Key = "help.heading"
	UsageHeading      Key = "usage.heading"
	UsageCostUnknown  Key = "usage.cost_unknown"
	UsageEstimated    Key = "usage.estimated"
	UsageNoAccount    Key = "usage.no_account"
	SessionHeading    Key = "session.heading"
	ModelNone         Key = "model.none"
	ModelCurrent      Key = "model.current"
	ModelList         Key = "model.list"
	ModelUnknown      Key = "model.unknown"
	ModelSet          Key = "model.set"
	ThinkingCurrent   Key = "thinking.current"
	ThinkingList      Key = "thinking.list"
	ThinkingUnknown   Key = "thinking.unknown"
	ThinkingSet       Key = "thinking.set"
	ModeCurrent       Key = "mode.current"
	ModeList          Key = "mode.list"
	ModeUnknown       Key = "mode.unknown"
	ModeAlreadyPlan   Key = "mode.already_plan"
	ModePlanOn        Key = "mode.plan_on"
	ModeSet           Key = "mode.set"
	NameCurrent       Key = "name.current"
	NameUsage         Key = "name.usage"
	NameNormalized    Key = "name.normalized"
	NameSet           Key = "name.set"
	ForkNone          Key = "fork.none"
	ForkList          Key = "fork.list"
	ForkNothingBefore Key = "fork.nothing_before"
	ForkNoMessage     Key = "fork.no_message"
	ForkDone          Key = "fork.done"
	CloneNone         Key = "clone.none"
	CloneDone         Key = "clone.done"
	TodosNone         Key = "todos.none"
	CompactStart      Key = "compact.start"
	CompactFailed     Key = "compact.failed"
	CompactDone       Key = "compact.done"

	// /personality and /humor (0022-PLAN P3).
	PersonalityCurrent  Key = "personality.current"
	PersonalityList     Key = "personality.list"
	PersonalitySet      Key = "personality.set"
	PersonalityUnknown  Key = "personality.unknown"
	PersonalityDeclined Key = "personality.declined"
	HumorTwo            Key = "humor.two"
	HumorUnknown        Key = "humor.unknown"

	// /trust, and an untrusted project's notice (0005-PLAN F6-1).
	TrustUntrusted   Key = "trust.untrusted"
	TrustCurrent     Key = "trust.current"
	TrustSaved       Key = "trust.saved"
	TrustNotSaved    Key = "trust.not_saved"
	TrustFailed      Key = "trust.failed"
	TrustUnavailable Key = "trust.unavailable"

	// /permissions, and an "always" answer that was not saved (0005-PLAN
	// F6-2b decisions E and H).
	PermissionsDefaults Key = "permissions.defaults"
	PermissionsRule     Key = "permissions.rule"
	PermissionsNone     Key = "permissions.none"
	PermissionsLimits   Key = "permissions.limits"
	PermissionsRemoved  Key = "permissions.removed"
	PermissionsFixed    Key = "permissions.fixed"
	PermissionsNoRule   Key = "permissions.no_rule"
	PermissionsAdded    Key = "permissions.added"
	PermissionsBad      Key = "permissions.bad"
	PermissionsUsage    Key = "permissions.usage"
	PermissionsFailed   Key = "permissions.failed"
)

// Keys are every key, in the order above.
var Keys = []Key{
	RetryOffline, RetryEmptyReply, RetryAttempt, CompactionStart, CompactionFailed, CompactionFailedTurn,
	CompactionFailedSession, TurnWorking, TurnStopMaxTokens, TurnStopMaxRequests, TurnStopRefusal, TurnStopCancelled,
	LinesHidden,
	CommandUnknown, HelpHeading, UsageHeading, UsageCostUnknown, UsageEstimated, UsageNoAccount, SessionHeading,
	ModelNone, ModelCurrent, ModelList, ModelUnknown, ModelSet, ThinkingCurrent, ThinkingList, ThinkingUnknown,
	ThinkingSet, ModeCurrent, ModeList, ModeUnknown, ModeAlreadyPlan, ModePlanOn, ModeSet, NameCurrent, NameUsage,
	NameNormalized, NameSet, ForkNone, ForkList, ForkNothingBefore, ForkNoMessage, ForkDone, CloneNone, CloneDone,
	TodosNone, CompactStart, CompactFailed, CompactDone,
	PersonalityCurrent, PersonalityList, PersonalitySet, PersonalityUnknown, PersonalityDeclined, HumorTwo, HumorUnknown,
	TrustUntrusted, TrustCurrent, TrustSaved, TrustNotSaved, TrustFailed, TrustUnavailable,
	PermissionsDefaults, PermissionsRule, PermissionsNone, PermissionsLimits, PermissionsRemoved, PermissionsFixed,
	PermissionsNoRule, PermissionsAdded, PermissionsBad, PermissionsUsage, PermissionsFailed,
}

// The lines a personality may put before an error or a warning in gobble's
// line view (0022-MADR D13). boring has none: its errors stand alone.
const (
	LeadNoCredential  Key = "error.no_credential.lead" //nolint:gosec // a message key, not a credential
	LeadSessionFailed Key = "error.session_failed.lead"
	LeadImages        Key = "error.images.lead"
	LeadAuth          Key = "error.auth.lead"
	LeadTurnFailed    Key = "error.turn_failed.lead"
)

// LeadKeys are every lead key.
var LeadKeys = []Key{LeadNoCredential, LeadSessionFailed, LeadImages, LeadAuth, LeadTurnFailed}

// The ambient lines of gobble's line view, and /gobble's (0022-MADR D12).
// boring has none, so under boring the view is as it was.
const (
	SessionWelcome             Key = "session.welcome"
	SessionWelcomeLateNight    Key = "session.welcome.late_night"
	SessionWelcomeMonday       Key = "session.welcome.monday"
	SessionWelcomeWeekend      Key = "session.welcome.weekend"
	SessionWelcomeThanksgiving Key = "session.welcome.thanksgiving"
	SessionGoodbye             Key = "session.goodbye"
	TurnSlow                   Key = "turn.slow"
	TurnVerySlow               Key = "turn.very_slow"
	MilestoneFirstCompaction   Key = "milestone.first_compaction"
	MilestoneLongTurn          Key = "milestone.long_turn"
	MilestoneCost              Key = "milestone.cost"
	MilestoneHundredthCall     Key = "milestone.hundredth_call"
	EasterGobble               Key = "easter.gobble"
)

// AmbientKeys are every ambient key.
var AmbientKeys = []Key{
	SessionWelcome, SessionWelcomeLateNight, SessionWelcomeMonday, SessionWelcomeWeekend, SessionWelcomeThanksgiving,
	SessionGoodbye, TurnSlow, TurnVerySlow, MilestoneFirstCompaction, MilestoneLongTurn, MilestoneCost,
	MilestoneHundredthCall, EasterGobble,
}

// ambientFacts are the placeholders each ambient key's variants keep, as
// boring's text would hold them had it any (the facts rule, 0022-MADR D5).
var ambientFacts = map[Key]string{MilestoneCost: "{cost}"}
