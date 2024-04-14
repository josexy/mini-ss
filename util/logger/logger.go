package logger

import (
	"time"

	"github.com/fatih/color"
	"github.com/josexy/logx"
)

// default global LogContext
var LogContext = logx.NewLogContext().
	WithColor(true).
	WithTime(true, func(t time.Time) any { return t.Format(time.DateTime) }).
	WithCaller(true, true, true, true).
	WithLevel(true, true).
	WithEncoder(logx.Json).
	WithEscapeQuote(true).
	WithWriter(color.Output)

// default global Logger
var Logger = LogContext.BuildConsoleLogger(logx.LevelTrace)
