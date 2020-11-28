package helpers

import (
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"
)

type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Warn
	Error
	Fatal
)

func (this LogLevel) Format(format string) string {
	switch format {
	case "letter", "l", "1":
		switch this {
		case Debug:
			return "D"
		case Info:
			return "I"
		case Warn:
			return "W"
		case Error:
			return "E"
		case Fatal:
			return "F"
		default:
			return "?"
		}
	case "short", "s", "3":
		switch this {
		case Debug:
			return "DBG"
		case Info:
			return "INF"
		case Warn:
			return "WRN"
		case Error:
			return "ERR"
		case Fatal:
			return "FTL"
		default:
			return fmt.Sprintf("LVL%d", int(this))
		}
	case "normal", "norm", "n":
		switch this {
		case Debug:
			return "DEBUG"
		case Info:
			return "INFO"
		case Warn:
			return "WARN"
		case Error:
			return "ERROR"
		case Fatal:
			return "FATAL"
		default:
			return fmt.Sprintf("LVL%d", int(this))
		}
	case "full", "f":
		switch this {
		case Debug:
			return "DEBUG"
		case Info:
			return "INFORMATION"
		case Warn:
			return "WARNING"
		case Error:
			return "ERROR"
		case Fatal:
			return "FATAL"
		default:
			return fmt.Sprintf("LVL%d", int(this))
		}
	default:
		return "UNKNOWN_FORMAT"
	}
}
func (this LogLevel) String() string { return this.Format("n") }

type LogFactory interface {
	CreateLogger(name string) Logger
}

type Logger interface {
	LogFactory

	Debug(message interface{})
	Debugf(format string, args ...interface{})
	Info(message interface{})
	Infof(format string, args ...interface{})
	Warn(message interface{})
	Warnf(format string, args ...interface{})
	Error(message interface{})
	Errorf(format string, args ...interface{})
	Fatal(message interface{})
	Fatalf(format string, args ...interface{})
}

type LogRecord struct {
	Level     LogLevel
	LogSource string
	LogTime   time.Time
	Content   interface{}
}

type FileLogFactory struct {
	dispatcher chan *LogRecord
	format     *template.Template
	output     *os.File
	stopped    chan struct{}
}

func NewFileLogFactory(format *template.Template, output *os.File) *FileLogFactory {
	result := &FileLogFactory{
		dispatcher: make(chan *LogRecord),
		format:     format,
		output:     output,
		stopped:    make(chan struct{}),
	}

	context := GetDefaultContext(output)
	go func() {
		for {
			rec := <-result.dispatcher
			if rec == nil {
				break
			}

			if _, ok := rec.Content.(ColoredContent); ok {
				builder := &strings.Builder{}
				CWrite(builder, rec.Content, context)
				rec.Content = builder.String()
			}

			result.format.Execute(result.output, rec)
		}
		close(result.stopped)
	}()

	return result
}

func (this *FileLogFactory) CreateLogger(name string) Logger {
	return FileLogger{
		factory: this,
		name:    name,
	}
}
func (this *FileLogFactory) Close() {
	this.dispatcher <- nil
	<-this.stopped
}

type FileLogger struct {
	factory *FileLogFactory
	name    string
}

func (this FileLogger) CreateLogger(name string) Logger {
	return FileLogger{
		factory: this.factory,
		name:    this.name + "." + name,
	}
}
func (this FileLogger) log(level LogLevel, message interface{}) {
	rec := &LogRecord{
		Level:     level,
		LogSource: this.name,
		LogTime:   time.Now(),
		Content:   message,
	}

	this.factory.dispatcher <- rec
}
func (this FileLogger) logf(level LogLevel, format string, args ...interface{}) {
	this.log(level, CreateFormatContent(format, args...))
}
func (this FileLogger) Debug(message interface{})                 { this.log(Debug, message) }
func (this FileLogger) Debugf(format string, args ...interface{}) { this.logf(Debug, format, args...) }
func (this FileLogger) Info(message interface{})                  { this.log(Info, message) }
func (this FileLogger) Infof(format string, args ...interface{})  { this.logf(Info, format, args...) }
func (this FileLogger) Warn(message interface{})                  { this.log(Warn, message) }
func (this FileLogger) Warnf(format string, args ...interface{})  { this.logf(Warn, format, args...) }
func (this FileLogger) Error(message interface{})                 { this.log(Error, message) }
func (this FileLogger) Errorf(format string, args ...interface{}) { this.logf(Error, format, args...) }
func (this FileLogger) Fatal(message interface{})                 { this.log(Fatal, message) }
func (this FileLogger) Fatalf(format string, args ...interface{}) { this.logf(Fatal, format, args...) }
