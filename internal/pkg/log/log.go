package log

import (
	"bufio"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var defaultLogger *Logger

func Info(msg string) {
	defaultLogger.Info(msg)
}

func Infof(msg string, args ...interface{}) {
	defaultLogger.Infof(msg, args...)
}

func Warn(msg string) {
	defaultLogger.Warn(msg)
}

func Warnf(msg string, args ...interface{}) {
	defaultLogger.Warnf(msg, args...)
}

func Error(msg string) {
	defaultLogger.Error(msg)
}

func Errorf(msg string, args ...interface{}) {
	defaultLogger.Errorf(msg, args...)
}

func Panic(msg string) {
	defaultLogger.Panic(msg)
}

func Panicf(msg string, args ...interface{}) {
	defaultLogger.Panicf(msg, args)
}

func Fatal(msg string) {
	defaultLogger.Fatal(msg)
}

func Fatalf(msg string, args ...interface{}) {
	defaultLogger.Fatalf(msg, args)
}

func Output() io.Writer {
	return defaultLogger.output
}

const (
	ConsoleOutput = "console"
	FileOutput    = "file"
	NetworkOutput = "network"
)

const (
	TCP = "tcp"
	UDP = "udp"
)

type Config struct {
	Level  string   `json:"level"`
	Target []Target `json:"target"`
}

type Target struct {
	Type       string `json:"type"`
	Filename   string `json:"filename" yaml:"filename"`
	MaxSize    int    `json:"max_size" yaml:"max_size"`
	MaxAge     int    `json:"max_age" yaml:"max_age"`
	Compress   bool   `json:"compress" yaml:"compress"`
	MaxBackups int    `json:"max_backups" yaml:"max_backups"`

	Addr     string `json:"addr"`
	Protocol string `json:"protocol"`
}

type Logger struct {
	cfg *Config
	*zap.SugaredLogger
	output zapcore.WriteSyncer
	conn   net.Conn
}

func InitLogger(cfg Config) {
	defaultLogger = &Logger{
		cfg: &cfg,
	}
	if len(cfg.Target) == 0 {
		cfg.Target = append(cfg.Target, Target{Type: ConsoleOutput})
	}
	if strings.TrimSpace(cfg.Level) == "" {
		cfg.Level = "info"
	}

	var output []io.Writer
	for _, item := range cfg.Target {
		switch item.Type {
		case ConsoleOutput:
			output = append(output, os.Stdout)
		case FileOutput:
			lumberjack := &lumberjack.Logger{
				Filename:   item.Filename,
				MaxSize:    item.MaxSize,
				MaxAge:     item.MaxAge,
				MaxBackups: item.MaxBackups,
				LocalTime:  true,
				Compress:   true,
			}
			output = append(output, lumberjack)
		case NetworkOutput:
			switch item.Protocol {
			case TCP:
				conn, err := net.Dial("tcp", item.Addr)
				if err != nil {
					panic(err)
				}
				defaultLogger.conn = conn
				output = append(output, bufio.NewWriter(conn))
			case UDP:
				conn, err := net.Dial("udp", item.Addr)
				if err != nil {
					panic(err)
				}
				defaultLogger.conn = conn
				output = append(output, bufio.NewWriter(conn))
			}
		}
	}

	level := zapcore.InfoLevel
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	case "panic":
		level = zapcore.PanicLevel
	case "fatal":
		level = zapcore.FatalLevel
	}

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.TimeEncoderOfLayout(time.DateTime)
	enc := zapcore.NewJSONEncoder(config)
	syncers := make([]zapcore.WriteSyncer, len(output))
	for i, out := range output {
		syncers[i] = zapcore.AddSync(out)
	}
	msyncer := zapcore.NewMultiWriteSyncer(syncers...)
	core := zapcore.NewCore(enc, msyncer, level)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.ErrorOutput(msyncer), zap.AddStacktrace(zapcore.ErrorLevel))
	zap.ReplaceGlobals(zapLogger)

	defaultLogger.output = msyncer
	defaultLogger.SugaredLogger = zapLogger.Sugar()
}

func (l *Logger) GetOutput() io.Writer {
	return l.output
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.SugaredLogger.Infof(format, args...)
}
