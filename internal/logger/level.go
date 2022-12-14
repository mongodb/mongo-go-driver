package logger

import (
	"strings"
)

// Level is an enumeration representing the supported log severity levels supported by the driver.
type Level int

const (
	// OffLevel disables logging and is the default logging priority.
	OffLevel Level = iota

	// InfoLevel enables logging of informational messages. These logs are High-level information about normal
	// driver behavior. Example: MongoClient creation or close.
	InfoLevel

	// DebugLevel enables logging of debug messages. These logs can be voluminous and are intended for detailed
	// information that may be helpful when debugging an application. Example: A command starting.
	DebugLevel
)

// LevelLiteral are the logging levels defined in the specification. LevelLiteral string values are meant to be used to
// read from environment variables, mapping them to a log level supported by the driver. See the "LevelLiteral.getLevel"
// method for more information.
type LevelLiteral string

const (
	OffLevelLiteral       LevelLiteral = "off"
	EmergencyLevelLiteral LevelLiteral = "emergency"
	AlertLevelLiteral     LevelLiteral = "alert"
	CriticalLevelLiteral  LevelLiteral = "critical"
	ErrorLevelLiteral     LevelLiteral = "error"
	WarnLevelLiteral      LevelLiteral = "warn"
	NoticeLevelLiteral    LevelLiteral = "notice"
	InfoLevelLiteral      LevelLiteral = "info"
	DebugLevelLiteral     LevelLiteral = "debug"
	TraceLevelLiteral     LevelLiteral = "trace"
)

// Level will return the Level associated with the level literal. If the literal is not a valid level, then the
// default level is returned.
func (levell LevelLiteral) Level() Level {
	switch levell {
	case ErrorLevelLiteral:
		return InfoLevel
	case WarnLevelLiteral:
		return InfoLevel
	case NoticeLevelLiteral:
		return InfoLevel
	case InfoLevelLiteral:
		return InfoLevel
	case DebugLevelLiteral:
		return DebugLevel
	case TraceLevelLiteral:
		return DebugLevel
	default:
		return OffLevel
	}
}

// equalFold will check if the “str” value is case-insensitive equal to the environment variable literal value.
func (llevel LevelLiteral) equalFold(str string) bool {
	return strings.EqualFold(string(llevel), str)
}

// parseLevel will check if the given string is a valid environment variable literal for a logging severity level. If it
// is, then it will return the Level. The default Level is “Off”.
func parseLevel(level string) Level {
	for _, llevel := range []LevelLiteral{
		ErrorLevelLiteral,
		WarnLevelLiteral,
		NoticeLevelLiteral,
		InfoLevelLiteral,
		DebugLevelLiteral,
		TraceLevelLiteral,
	} {
		if llevel.equalFold(level) {
			return llevel.Level()
		}
	}

	return OffLevel
}
