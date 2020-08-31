// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongologger

import (
	"errors"
	"strings"
)

// Level indicates the user's preference on logging.
type Level uint8

// Level constants
const (
	_ Level = iota
	Trace
	Debug
	Info
	Notice
	Warning
	Error
)

// Includes returns if Level l prints out logs of Level other
func (l Level) Includes(other Level) bool {
	if l == Level(0) || other == Level(0) {
		return false
	}
	return l <= other
}

// String returns the string representation of the level.
func (l Level) String() string {
	switch l {
	case Trace:
		return "trace"
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Notice:
		return "notice"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// StringToLevel returns the level corresponding to the string.
func StringToLevel(s string) (Level, error) {
	lower := strings.ToLower(s)
	switch lower {
	case "trace":
		return Trace, nil
	case "debug":
		return Debug, nil
	case "info":
		return Info, nil
	case "notice":
		return Notice, nil
	case "warning":
		return Warning, nil
	case "error":
		return Error, nil
	default:
		return Level(0), errors.New("invalid level")
	}
}
