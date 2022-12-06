package logger

import (
	"strings"
)

// LogLevel is an enumeration representing the supported log severity levels.
type LogLevel int

const (
	// OffLogLevel disables logging and is the default logging priority.
	OffLogLevel LogLevel = iota

	// InfoLogLevel enables logging of informational messages. These logs are High-level information about normal
	// driver behavior. Example: MongoClient creation or close.
	InfoLogLevel

	// DebugLogLevel enables logging of debug messages. These logs can be voluminous and are intended for detailed
	// information that may be helpful when debugging an application. Example: A command starting.
	DebugLogLevel
)

type levelEnv string

const (
	offLevelEnv    levelEnv = "off"
	errorLevelEnv  levelEnv = "error"
	warnLevelEnv   levelEnv = "warn"
	noticeLevelEnv levelEnv = "notice"
	infoLevelEnv   levelEnv = "info"
	debugLevelEnv  levelEnv = "debug"
	traceLevelEnv  levelEnv = "trace"
)

// level will return the level associated with the environment variable literal.
func (llevel levelEnv) level() LogLevel {
	switch llevel {
	case errorLevelEnv:
		return InfoLogLevel
	case warnLevelEnv:
		return InfoLogLevel
	case noticeLevelEnv:
		return InfoLogLevel
	case infoLevelEnv:
		return InfoLogLevel
	case debugLevelEnv:
		return DebugLogLevel
	case traceLevelEnv:
		return DebugLogLevel
	default:
		return OffLogLevel
	}
}

// equalFold will check if the “str” value is case-insensitive equal to the environment variable literal value.
func (llevel levelEnv) equalFold(str string) bool {
	return strings.EqualFold(string(llevel), str)
}

// parseLevel will check if the given string is a valid environment variable literal for a logging severity level. If it
// is, then it will return the Level. The default Level is “Off”.
func parseLevel(level string) LogLevel {
	for _, llevel := range []levelEnv{
		errorLevelEnv,
		warnLevelEnv,
		noticeLevelEnv,
		infoLevelEnv,
		debugLevelEnv,
		traceLevelEnv,
	} {
		if llevel.equalFold(level) {
			return llevel.level()
		}
	}

	return OffLogLevel
}
