package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"time"
)

const (
	Debug LogLevel = iota
	Info
	Warn
	Error
	Fatal
)

const InvalidLogLevel = StringError("Invalid LogLevel")

var (
	EOL = []byte{'\n'}
)

type LogLevel int

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

type LogLevelUnmarshaller struct {
	Level LogLevel
}

func (this *LogLevelUnmarshaller) fromInt(n int) error {
	if n < int(Debug) || n > int(Fatal) {
		return fmt.Errorf("`%d` is not a valid LogLevel, accepted range is [%d, %d]", n, int(Debug), int(Fatal))
	}
	this.Level = LogLevel(n)
	return nil
}
func (this *LogLevelUnmarshaller) fromString(s string) error {
	switch strings.ToLower(s) {
	case "debug", "dbg":
		this.Level = Debug
	case "information", "info":
		this.Level = Info
	case "warning", "warn":
		this.Level = Warn
	case "error", "err":
		this.Level = Error
	case "fatal", "ftl":
		this.Level = Fatal
	default:
		return fmt.Errorf("`%s` is not a valid LogLevel", s)
	}
	return nil
}
func (this *LogLevelUnmarshaller) UnmarshalJSON(data []byte) error {
	var strLevel string
	if err := json.Unmarshal(data, &strLevel); err != nil {
		return this.fromString(strLevel)
	}

	var nLevel int
	if err := json.Unmarshal(data, &nLevel); err != nil {
		return this.fromInt(nLevel)
	}

	return InvalidLogLevel
}
func (this *LogLevelUnmarshaller) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strLevel string
	if err := unmarshal(&strLevel); err == nil {
		return this.fromString(strLevel)
	}

	var nLevel int
	if err := unmarshal(&nLevel); err == nil {
		return this.fromInt(nLevel)
	}

	return InvalidLogLevel
}

type LogRecord struct {
	Level     LogLevel
	LogSource string
	LogTime   time.Time
	Content   interface{}
	context   ColorContext
	colorMap  *ColorNameMap
}

// Support for colored templating
func (this *LogRecord) GetContext() ColorContext   { return this.context }
func (this *LogRecord) GetColorMap() *ColorNameMap { return this.colorMap }
func (this *LogRecord) GetDefaultColor() Color {
	colorName := "log:" + this.Level.Format("letter")
	code := this.colorMap.GetColorCodeByName(colorName)
	return code.ToColor()
}

type LogFactory interface {
	io.Closer
	CreateLogger(name string, level *LogLevel, verbosityLevel *int) Logger
}
type Logger interface {
	GetName() string
	GetLogFactory() LogFactory
	GetMinimumLevel() LogLevel
	GetVerbosityLevel() int

	CreateLogger(name string, level *LogLevel, verbosityLevel *int) Logger

	V(verbosityLevel int) bool
	IsEnabled(level LogLevel) bool

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

	Verbose(verbosityLevel int, message interface{})
	Verbosef(verbosityLevel int, format string, args ...interface{})
}

type FileLogFactory struct {
	name           string
	dispatcher     chan *LogRecord
	format         *template.Template
	output         *os.File
	closeOutput    bool
	stopped        chan struct{}
	minimumLevel   LogLevel
	verbosityLevel int
	colorMap       *ColorNameMap
}

// NewFileLogFactory Create a a ``FileLogFactory``
func NewFileLogFactory(
	format *template.Template,
	output *os.File,
	minimumLogLevel LogLevel,
	verbosityLevel int,
	mustCloseOutput bool) *FileLogFactory {
	result := &FileLogFactory{
		dispatcher:     make(chan *LogRecord),
		format:         format,
		output:         output,
		closeOutput:    mustCloseOutput,
		stopped:        make(chan struct{}),
		minimumLevel:   minimumLogLevel,
		verbosityLevel: verbosityLevel,
		colorMap: GetGlobalColorMap().Clone().
			AddName("log:D", Grey.Code()).
			AddName("log:I", White.Code()).
			AddName("log:W", Orange.Code()).
			AddName("log:E", Red.Code()).
			AddName("log:F", DarkRed.Code()),
	}

	go result.dispatch()

	return result
}

func (this *FileLogFactory) dispatch() {
	context := GetDefaultContext(this.output)
	for {
		rec := <-this.dispatcher
		if rec == nil {
			break
		}

		rec.context = context
		if _, ok := rec.Content.(ColoredContent); ok {
			rec.Content = BindContentToContext(context, rec.Content)
		}

		err := this.format.Execute(this.output, rec)
		this.output.Write(EOL)
		if err != nil {
			fmt.Printf("LOG FAILED: %v\n", err)
		}
	}
	close(this.stopped)
}
func (this *FileLogFactory) SetColor(level LogLevel, color Color) *FileLogFactory {
	this.colorMap.AddName("log:"+level.Format("letter"), color.Code())
	return this
}
func (this *FileLogFactory) CreateLogger(name string, minimumLogLevel *LogLevel, verbosityLevel *int) Logger {
	if minimumLogLevel == nil {
		minimumLogLevel = &this.minimumLevel
	}
	if verbosityLevel == nil {
		verbosityLevel = &this.verbosityLevel
	}
	return FileLogger{
		factory:        this,
		name:           name,
		minimumLevel:   *minimumLogLevel,
		verbosityLevel: *verbosityLevel,
	}
}
func (this *FileLogFactory) Close() error {
	this.dispatcher <- nil
	<-this.stopped
	if this.closeOutput {
		return this.output.Close()
	}
	return nil
}

type FileLogger struct {
	factory        *FileLogFactory
	name           string
	minimumLevel   LogLevel
	verbosityLevel int
}

func (this FileLogger) doLog(level LogLevel, message interface{}) {
	rec := &LogRecord{
		Level:     level,
		LogSource: this.name,
		LogTime:   time.Now(),
		Content:   message,
		colorMap:  this.factory.colorMap,
	}

	this.factory.dispatcher <- rec
}
func (this FileLogger) doLogf(level LogLevel, format string, args ...interface{}) {
	this.doLog(level, CreateFormatContent(format, args...))
}

func (this FileLogger) log(level LogLevel, message interface{}) {
	if level >= this.minimumLevel {
		this.doLog(level, message)
	}
}
func (this FileLogger) logf(level LogLevel, format string, args ...interface{}) {
	if level >= this.minimumLevel {
		this.doLogf(level, format, args...)
	}
}

func (this FileLogger) GetName() string           { return this.name }
func (this FileLogger) GetLogFactory() LogFactory { return this.factory }
func (this FileLogger) GetMinimumLevel() LogLevel { return this.minimumLevel }
func (this FileLogger) GetVerbosityLevel() int    { return this.verbosityLevel }
func (this FileLogger) CreateLogger(name string, minimumLogLevel *LogLevel, verbosityLevel *int) Logger {
	if minimumLogLevel == nil {
		minimumLogLevel = &this.minimumLevel
	}
	if verbosityLevel == nil {
		verbosityLevel = &this.verbosityLevel
	}
	return FileLogger{
		factory:        this.factory,
		name:           this.name + "." + name,
		minimumLevel:   *minimumLogLevel,
		verbosityLevel: *verbosityLevel,
	}
}
func (this FileLogger) V(verbosityLevel int) bool                 { return verbosityLevel >= this.verbosityLevel }
func (this FileLogger) IsEnabled(level LogLevel) bool             { return level >= this.minimumLevel }
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
func (this FileLogger) Verbose(verbosityLevel int, message interface{}) {
	if verbosityLevel >= this.verbosityLevel {
		this.doLog(Info, message)
	}
}
func (this FileLogger) Verbosef(verbosityLevel int, format string, args ...interface{}) {
	if verbosityLevel >= this.verbosityLevel {
		this.doLogf(Info, format, args...)
	}
}
