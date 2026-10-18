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
}
