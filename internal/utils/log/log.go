package log

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	origLogger = logrus.New()
	// default logger we use
	defaultLogger = &logger{
		Logger:    origLogger,
		entry:     logrus.NewEntry(origLogger),
		fmt:       "short",
		logFilter: make(map[string]bool),
	}
)

type logger struct {
	*logrus.Logger
	entry     *logrus.Entry
	fmt       string
	logFilter map[string]bool
}

func (l *logger) Debug(args ...interface{}) {
	l.withSource().Debug(args...)
}

func (l *logger) Debugln(args ...interface{}) {
	l.withSource().Debugln(args...)
}

func (l *logger) Debugf(msg string, args ...interface{}) {
	l.withSource().Debugf(msg, args...)
}

func (l *logger) Info(args ...interface{}) {
	l.withSource().Info(args...)
}

func (l *logger) Infoln(args ...interface{}) {
	l.withSource().Infoln(args...)
}

func (l *logger) Infof(msg string, args ...interface{}) {
	l.withSource().Infof(msg, args...)
}

// InfoFilter will log info only if 'filter' was previously added via UpdateFilter of AddFilter
func (l *logger) InfoFilter(filter string, args ...interface{}) {
	if _, ok := l.logFilter[filter]; ok {
		l.withSource().Info(args...)
	}
}

// InfoFilterLn will log info only if 'filter' was previously added via UpdateFilter of AddFilter
func (l *logger) InfoFilterLn(filter string, args ...interface{}) {
	if _, ok := l.logFilter[filter]; ok {
		l.withSource().Infoln(args...)
	}

}

// InfoFilterf will log info only if 'filter' was previously added via UpdateFilter of AddFilter
func (l *logger) InfoFilterf(filter string, fmt string, args ...interface{}) {
	if _, ok := l.logFilter[filter]; ok {
		l.withSource().Infof(fmt, args...)
	}
}

// InfoFilters will log info only if one of 'filters' was previously added via UpdateFilter of AddFilter
func (l *logger) InfoFilters(filters []string, args ...interface{}) {
	for _, filter := range filters {
		if _, ok := l.logFilter[filter]; ok {
			l.withSource().Info(args...)
			break
		}
	}
}

// InfoFilterLn will log info only if one of 'filters' was previously added via UpdateFilter of AddFilter
func (l *logger) InfoFiltersLn(filters []string, args ...interface{}) {
	for _, filter := range filters {
		if _, ok := l.logFilter[filter]; ok {
			l.withSource().Infoln(args...)
			break
		}
	}
}

// InfoFilterf will log info only if one of 'filters' was previously added via UpdateFilter of AddFilter
func (l *logger) InfoFiltersf(filters []string, fmt string, args ...interface{}) {
	for _, filter := range filters {
		if _, ok := l.logFilter[filter]; ok {
			l.withSource().Infof(fmt, args...)
			break
		}
	}
}

func (l *logger) RemoveFilter(filter string) {
	delete(l.logFilter, filter)
}

func (l *logger) AddFilter(filter string) {
	l.logFilter[filter] = true
}

func (l *logger) UpdateFilter(filter map[string]bool) {
	l.logFilter = filter
}

func (l *logger) Warn(args ...interface{}) {
	l.withSource().Warn(args...)
}

func (l *logger) Warnln(args ...interface{}) {
	l.withSource().Warnln(args...)
}

func (l *logger) Warnf(fmt string, args ...interface{}) {
	l.withSource().Warnf(fmt, args...)
}

func (l *logger) Error(args ...interface{}) {
	l.withSource().Error(args...)
}

func (l *logger) Errorln(args ...interface{}) {
	l.withSource().Errorln(args...)
}

func (l *logger) Errorf(fmt string, args ...interface{}) {
	l.withSource().Errorf(fmt, args...)
}

func (l *logger) Fatal(args ...interface{}) {
	l.withSource().Fatal(args...)
}

func (l *logger) Fatalln(args ...interface{}) {
	l.withSource().Fatalln(args...)
}

func (l *logger) Fatalf(fmt string, args ...interface{}) {
	l.withSource().Fatalf(fmt, args...)
}

func (l *logger) Panic(args ...interface{}) {
	l.withSource().Panic(args...)
}

func (l *logger) Panicln(args ...interface{}) {
	l.withSource().Panicln(args...)
}

func (l *logger) Panicf(fmt string, args ...interface{}) {
	l.withSource().Panicf(fmt, args...)
}

func (l *logger) With(key string, value interface{}) Logger {
	return &logger{origLogger, l.entry.WithField(key, value), l.fmt, l.logFilter}
}

func (l *logger) WithFields(fields map[string]interface{}) Logger {
	return &logger{origLogger, l.entry.WithFields(logrus.Fields(fields)), l.fmt, l.logFilter}
}

func AddHook(hook logrus.Hook) {
	defaultLogger.AddHook(hook)
}

func Entry() *logrus.Entry {
	return defaultLogger.entry
}

func (l *logger) withSource() *logrus.Entry {
	if l.fmt == "none" {
		return l.entry
	}
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		if l.fmt == "short" {
			slash := strings.LastIndex(file, "/")
			file = file[slash+1:]
		}
	}
	return l.entry.WithField("source", fmt.Sprintf(" %s:%d ", file, line))
}

// sets the output format to 'json'|'text'|'nocolor' .. only supported for now
func SetFormat(format string) {
	switch format {
	case "json":
		defaultLogger.entry.Logger.Formatter = &logrus.JSONFormatter{}
	case "nocolor":
		defaultLogger.entry.Logger.Formatter = &logrus.TextFormatter{ForceColors: false, DisableColors: true}
	default:
		defaultLogger.entry.Logger.Formatter = &logrus.TextFormatter{}
	}
}

// Logger is interface used for logging
// currently delegates to underlying logger impl..
type Logger interface {
	Debug(...interface{})
	Debugln(...interface{})
	Debugf(string, ...interface{})

	Info(...interface{})
	Infoln(...interface{})
	Infof(string, ...interface{})

	Warn(...interface{})
	Warnln(...interface{})
	Warnf(string, ...interface{})

	Error(...interface{})
	Errorln(...interface{})
	Errorf(string, ...interface{})

	Fatal(...interface{})
	Fatalln(...interface{})
	Fatalf(string, ...interface{})

	Panic(...interface{})
	Panicln(...interface{})
	Panicf(string, ...interface{})

	RemoveFilter(filter string)
	AddFilter(filter string)
	UpdateFilter(map[string]bool)
	InfoFilter(string, ...interface{})
	InfoFilterLn(string, ...interface{})
	InfoFilterf(string, string, ...interface{})

	InfoFilters([]string, ...interface{})
	InfoFiltersLn([]string, ...interface{})
	InfoFiltersf([]string, string, ...interface{})

	WithFields(map[string]interface{}) Logger
	With(key string, value interface{}) Logger
}

// set log output
func SetOutput(out io.Writer) {
	defaultLogger.entry.Logger.Out = out
}

// set the source format output to either 'long'|'short'
func SetSourceFormat(format string) {
	switch format {
	case "short":
		defaultLogger.fmt = format
	case "long":
		defaultLogger.fmt = format
	default:
		defaultLogger.fmt = "short"
	}
}

// set logging level
func SetLevel(level string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		defaultLogger.entry.Logger.Level = logrus.InfoLevel
		return
	}
	defaultLogger.entry.Logger.Level = lvl
}

func IsDebugEnabled() bool {
	return defaultLogger.Level == logrus.DebugLevel
}

func GetLevel() (level string) {
	level = defaultLogger.entry.Logger.Level.String()
	return level
}

// get the source format output 'long'|'short'
func GetSourceFormat() (format string) {
	format = defaultLogger.fmt
	return format
}

// gets the output format to 'json'|'text'|'nocolor'
func GetFormat() (format string) {
	switch v := defaultLogger.entry.Logger.Formatter.(type) {
	case *logrus.JSONFormatter:
		{
			format = "json"
		}
	case *logrus.TextFormatter:
		{
			if !v.ForceColors && v.DisableColors {
				format = "nocolor"
			} else {
				format = "text"
			}
		}
	}
	return format
}

func Debug(args ...interface{}) {
	defaultLogger.withSource().Debug(args...)
}

func Debugln(args ...interface{}) {
	defaultLogger.withSource().Debugln(args...)
}

func Debugf(msg string, args ...interface{}) {
	defaultLogger.withSource().Debugf(msg, args...)
}

func Info(args ...interface{}) {
	defaultLogger.withSource().Info(args...)
}

func Infoln(args ...interface{}) {
	defaultLogger.withSource().Infoln(args...)
}

func Infof(msg string, args ...interface{}) {
	defaultLogger.withSource().Infof(msg, args...)
}

// remove a filter
func RemoveFilter(filter string) {
	defaultLogger.RemoveFilter(filter)
}

// add value to filter
func AddFilter(filter string) {
	defaultLogger.AddFilter(filter)
}

// updatefilter updates all filters with filters
func UpdateFilter(filter map[string]bool) {
	defaultLogger.UpdateFilter(filter)
}

// InfoFilter will log info only if 'filter' was previously added via UpdateFilter of AddFilter
func InfoFilter(filter string, args ...interface{}) {
	defaultLogger.InfoFilter(filter, args...)
}

// InfoFilterLn will log info only if 'filter' was previously added via UpdateFilter of AddFilter
func InfoFilterLn(filter string, args ...interface{}) {
	defaultLogger.InfoFilterLn(filter, args...)
}

// InfoFilterf will log info only if 'filter' was previously added via UpdateFilter of AddFilter
func InfoFilterf(filter string, fmt string, args ...interface{}) {
	defaultLogger.InfoFilterf(filter, fmt, args...)
}

// InfoFilter will log info only if one of 'filters' was previously added via UpdateFilter of AddFilter
func InfoFilters(filters []string, args ...interface{}) {
	defaultLogger.InfoFilters(filters, args...)
}

// InfoFilterLn will log info only if one of 'filters' was previously added via UpdateFilter of AddFilter
func InfoFiltersLn(filters []string, args ...interface{}) {
	defaultLogger.InfoFiltersLn(filters, args...)
}

// InfoFilterf will log info only if one of 'filters' was previously added via UpdateFilter of AddFilter
func InfoFiltersf(filters []string, fmt string, args ...interface{}) {
	defaultLogger.InfoFiltersf(filters, fmt, args...)
}

func Warn(args ...interface{}) {
	defaultLogger.withSource().Warn(args...)
}

func Warnln(args ...interface{}) {
	defaultLogger.withSource().Warnln(args...)
}

func Warnf(msg string, args ...interface{}) {
	defaultLogger.withSource().Warnf(msg, args...)
}

func Error(args ...interface{}) {
	defaultLogger.withSource().Error(args...)
}

func Errorln(args ...interface{}) {
	defaultLogger.withSource().Errorln(args...)
}

func Errorf(msg string, args ...interface{}) {
	defaultLogger.withSource().Errorf(msg, args...)
}

func Fatal(args ...interface{}) {
	defaultLogger.withSource().Fatal(args...)
}

func Fatalln(args ...interface{}) {
	defaultLogger.withSource().Fatalln(args...)
}

func Fatalf(msg string, args ...interface{}) {
	defaultLogger.withSource().Fatalf(msg, args...)
}

func Panic(args ...interface{}) {
	defaultLogger.withSource().Panic(args...)
}

func Panicln(args ...interface{}) {
	defaultLogger.withSource().Panicln(args...)
}

func Panicf(msg string, args ...interface{}) {
	defaultLogger.withSource().Panicf(msg, args...)
}

func With(key string, value interface{}) Logger {
	return defaultLogger.With(key, value)
}

type Fields map[string]interface{}

func WithFields(fields map[string]interface{}) Logger {
	return defaultLogger.WithFields(fields)
}

// Handler is an http handler for exposing log configuration.
// you can modify the logging via ?level&format&sourceFormat
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if level := r.FormValue("level"); level != "" {
			Warn("updating log level to ", level)
			SetLevel(level)
		}
		if format := r.FormValue("format"); format != "" {
			Warn("updating format to ", format)
			SetFormat(format)
		}
		if sourceFormat := r.FormValue("sourceFormat"); sourceFormat != "" {
			Warn("updating sourceFormat to ", sourceFormat)
			SetSourceFormat(sourceFormat)
		}
	})
}
