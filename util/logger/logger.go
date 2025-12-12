package logger

import (
	"time"

	"github.com/fatih/color"
	"github.com/josexy/logx"
)

// default global LogContext
var LogContext = logx.NewLogContext().
	WithLevel(logx.LevelTrace).
	WithColorfulset(true, logx.TextColorAttri{}).
	WithCallerKey(true, logx.CallerOption{Formatter: logx.ShortFileFunc}).
	WithTimeKey(true, logx.TimeOption{Formatter: func(t time.Time) any { return t.Format("2006/01/02 15:04:05.000") }}).
	WithLevelKey(true, logx.LevelOption{LowerKey: true}).
	WithEscapeQuote(true).
	WithWriter(logx.AddSync(color.Output)).
	WithEncoder(logx.Console).
	WithReflectValue(true)

// default global Logger
var Logger = LogContext.Build()
