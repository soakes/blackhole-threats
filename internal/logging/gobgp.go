package logging

import (
	"os"
	"strings"

	gobgplog "github.com/osrg/gobgp/v3/pkg/log"
	log "github.com/sirupsen/logrus"
)

type GoBGPLogger struct {
	logger *log.Logger
}

func NewGoBGPLogger(level log.Level, formatter log.Formatter) *GoBGPLogger {
	logger := log.New()
	if formatter == nil {
		formatter = OperatorFormatter{}
	}
	logger.SetFormatter(formatter)
	logger.SetOutput(os.Stdout)
	logger.SetLevel(level)

	return &GoBGPLogger{logger: logger}
}

func normalizeFields(fields gobgplog.Fields) log.Fields {
	normalized := make(log.Fields, len(fields))
	for key, value := range fields {
		normalized[strings.ToLower(key)] = value
	}

	return normalized
}

func (l *GoBGPLogger) Panic(msg string, fields gobgplog.Fields) {
	l.logger.WithField("component", "gobgp").WithFields(normalizeFields(fields)).Panic(msg)
}

func (l *GoBGPLogger) Fatal(msg string, fields gobgplog.Fields) {
	l.logger.WithField("component", "gobgp").WithFields(normalizeFields(fields)).Fatal(msg)
}

func (l *GoBGPLogger) Error(msg string, fields gobgplog.Fields) {
	l.logger.WithField("component", "gobgp").WithFields(normalizeFields(fields)).Error(msg)
}

func (l *GoBGPLogger) Warn(msg string, fields gobgplog.Fields) {
	l.logger.WithField("component", "gobgp").WithFields(normalizeFields(fields)).Warn(msg)
}

func (l *GoBGPLogger) Info(msg string, fields gobgplog.Fields) {
	l.logger.WithField("component", "gobgp").WithFields(normalizeFields(fields)).Info(msg)
}

func (l *GoBGPLogger) Debug(msg string, fields gobgplog.Fields) {
	l.logger.WithField("component", "gobgp").WithFields(normalizeFields(fields)).Debug(msg)
}

func (l *GoBGPLogger) SetLevel(level gobgplog.LogLevel) {
	l.logger.SetLevel(log.Level(level))
}

func (l *GoBGPLogger) GetLevel() gobgplog.LogLevel {
	return gobgplog.LogLevel(l.logger.GetLevel())
}
