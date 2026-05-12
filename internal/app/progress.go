package app

// ProgressFunc is called by profile operations to report step-level status.
// Pass NoopProgress to silence output (e.g. in tests).
type ProgressFunc func(msg string)

// NoopProgress discards all progress messages.
func NoopProgress(_ string) {}
