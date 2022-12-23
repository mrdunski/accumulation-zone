package logger

import (
	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"
)

const (
	componentKey = "component"
)

type LogConfig struct {
	LogLevel  string `help:"Level of logging." default:"info" enum:"trace,debug,info,warn,error"`
	LogFormat string `help:"Format for logs." default:"text" enum:"text,json"`
}

func (c LogConfig) InitLogger(kongCtx *kong.Context) {
	logrus.SetOutput(kongCtx.Stderr)
	logrus.SetLevel(c.level())
	logrus.SetFormatter(c.formatter())
}

func (c LogConfig) level() logrus.Level {
	level, err := logrus.ParseLevel(c.LogLevel)
	if err != nil {
		panic(err)
	}

	return level
}

func (c LogConfig) formatter() logrus.Formatter {
	switch c.LogFormat {
	case "json":
		return &logrus.JSONFormatter{}
	case "text":
		return &logrus.TextFormatter{}
	default:
		return &logrus.TextFormatter{}
	}
}

func Get() *logrus.Logger {
	return logrus.StandardLogger()
}

func WithComponent(component string) *logrus.Entry {
	return Get().WithField(componentKey, component)
}
