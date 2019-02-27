package log

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"path"
	"runtime"
)

var globalLogLevel uint = 0

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	SetGlobalLogLevel(0)
}

func sourceInfo() string {
	_, file, line_no, ok := runtime.Caller(3)
	if !ok {
		return "?:?"
	}
	return fmt.Sprintf("%v:%v", path.Base(file), line_no)
}

// Global logging.
func GlobalLogLevel() uint {
	return globalLogLevel
}

func SetGlobalLogLevel(loglevel uint) {
	logrus_level := logrus.InfoLevel
	if loglevel > 3 {
		switch loglevel {
		case 4:
			logrus_level = logrus.DebugLevel
		default:
			logrus_level = logrus.TraceLevel
		}
	}
	logrus.SetLevel(logrus_level)
	globalLogLevel = loglevel
}

func DebugLazy(export func() string) {
	if globalLogLevel >= 4 {
		Debug(export())
	}
}

func TraceLazy(export func() string) {
	if globalLogLevel >= 5 {
		Trace(export())
	}
}

func Trace(args ...interface{}) {
	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Trace(args...)
}

func Debug(args ...interface{}) {
	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Debug(args...)
}

func Info(info_level uint, args ...interface{}) {
	if info_level > globalLogLevel {
		return
	}

	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Info(args...)
}

func Info0(args ...interface{}) {
	Info(0, args...)
}

func Info1(args ...interface{}) {
	Info(1, args...)
}

func Info2(args ...interface{}) {
	Info(2, args...)
}

func Info3(args ...interface{}) {
	Info(3, args...)
}

func Infof(info_level uint, format string, args ...interface{}) {
	if info_level > globalLogLevel {
		return
	}

	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Infof(format, args...)
}

func Infof0(format string, args ...interface{}) {
	Infof(0, format, args...)
}

func Infof1(format string, args ...interface{}) {
	Infof(1, format, args...)
}

func Infof2(format string, args ...interface{}) {
	Infof(2, format, args...)
}

func Infof3(format string, args ...interface{}) {
	Infof(3, format, args...)
}

func Warn(args ...interface{}) {
	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Warn(args...)
}

func Error(args ...interface{}) {
	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Error(args...)
}

func Fatal(args ...interface{}) {
	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Fatal(args...)
}

func Panic(args ...interface{}) {
	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Panic(args...)
}

func Debugf(format string, args ...interface{}) {
	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Debugf(format, args...)
}

func Tracef(format string, args ...interface{}) {
	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Tracef(format, args...)
}

func Warnf(format string, args ...interface{}) {
	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Fatalf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	logrus.WithFields(logrus.Fields{
		"src": sourceInfo(),
	}).Panicf(format, args...)
}

func copyKV(dst, src map[string]interface{}) {
	for k, v := range src {
		dst[k] = v
	}
}

func TraceMap(kv map[string]interface{}, args ...interface{}) {
	nkv := make(map[string]interface{}, len(kv)+1)
	copyKV(nkv, kv)
	nkv["src"] = sourceInfo()
	logrus.WithFields(nkv).Trace(args...)
}

func DebugMap(kv map[string]interface{}, args ...interface{}) {
	nkv := make(map[string]interface{}, len(kv)+1)
	copyKV(nkv, kv)
	nkv["src"] = sourceInfo()
	logrus.WithFields(kv).Debug(args...)
}

func WarnMap(kv map[string]interface{}, args ...interface{}) {
	nkv := make(map[string]interface{}, len(kv)+1)
	copyKV(nkv, kv)
	nkv["src"] = sourceInfo()
	logrus.WithFields(nkv).Warn(args...)
}

func InfoMap(kv map[string]interface{}, args ...interface{}) {
	nkv := make(map[string]interface{}, len(kv)+1)
	copyKV(nkv, kv)
	nkv["src"] = sourceInfo()
	logrus.WithFields(nkv).Info(args...)
}

func ErrorMap(kv map[string]interface{}, args ...interface{}) {
	nkv := make(map[string]interface{}, len(kv)+1)
	copyKV(nkv, kv)
	nkv["src"] = sourceInfo()
	logrus.WithFields(nkv).Error(args...)
}

func FatalMap(kv map[string]interface{}, args ...interface{}) {
	nkv := make(map[string]interface{}, len(kv)+1)
	copyKV(nkv, kv)
	nkv["src"] = sourceInfo()
	logrus.WithFields(nkv).Fatal(args...)
}

func PanicMap(kv map[string]interface{}, args ...interface{}) {
	nkv := make(map[string]interface{}, len(kv)+1)
	copyKV(nkv, kv)
	nkv["src"] = sourceInfo()
	logrus.WithFields(nkv).Panic(args...)
}

// Logger
type Logger struct {
	Fields map[string]interface{}
}

func NewLogger() *Logger {
	return &Logger{
		Fields: make(map[string]interface{}),
	}
}

func (logger *Logger) Trace(args ...interface{}) {
	TraceMap(logger.Fields, args...)
}

func (logger *Logger) Debug(args ...interface{}) {
	DebugMap(logger.Fields, args...)
}

func (logger *Logger) Info(info_level uint, args ...interface{}) {
	InfoMap(logger.Fields, args...)
}

func (logger *Logger) Info0(args ...interface{}) {
	logger.Info(0, args...)
}

func (logger *Logger) Info1(args ...interface{}) {
	logger.Info(1, args...)
}

func (logger *Logger) Info2(args ...interface{}) {
	logger.Info(2, args...)
}

func (logger *Logger) Info3(args ...interface{}) {
	logger.Info(3, args...)
}

func (logger *Logger) Infof(info_level uint, format string, args ...interface{}) {
	if info_level > globalLogLevel {
		return
	}
	InfoMap(logger.Fields, fmt.Sprintf(format, args...))
}

func (logger *Logger) Infof0(format string, args ...interface{}) {
	Infof(0, format, args...)
}

func (logger *Logger) Infof1(format string, args ...interface{}) {
	Infof(1, format, args...)
}

func (logger *Logger) Infof2(format string, args ...interface{}) {
	Infof(2, format, args...)
}

func (logger *Logger) Infof3(format string, args ...interface{}) {
	Infof(3, format, args...)
}

func (logger *Logger) Warn(args ...interface{}) {
	WarnMap(logger.Fields, args...)
}

func (logger *Logger) Error(args ...interface{}) {
	ErrorMap(logger.Fields, args...)
}

func (logger *Logger) Fatal(args ...interface{}) {
	FatalMap(logger.Fields, args...)
}

func (logger *Logger) Panic(args ...interface{}) {
	PanicMap(logger.Fields, args...)
}

func (logger *Logger) Debugf(format string, args ...interface{}) {
	ErrorMap(logger.Fields, fmt.Sprintf(format, args...))
}

func (logger *Logger) Errorf(format string, args ...interface{}) {
	ErrorMap(logger.Fields, fmt.Sprintf(format, args...))
}

func (logger *Logger) Fatalf(format string, args ...interface{}) {
	FatalMap(logger.Fields, fmt.Sprintf(format, args...))
}

func (logger *Logger) Panicf(format string, args ...interface{}) {
	PanicMap(logger.Fields, fmt.Sprintf(format, args...))
}

func (logger *Logger) DebugLazy(export func() string) {
	if globalLogLevel >= 4 {
		logger.Debug(export())
	}
}

func (logger *Logger) DebugMapLazy(export func() string, fields map[string]interface{}) {
	if globalLogLevel >= 4 {
		logger.Debug(fields, export())
	}
}

func (logger *Logger) TraceLazy(export func() string) {
	if globalLogLevel >= 5 {
		logger.Trace(export())
	}
}

func (logger *Logger) TraceMapLazy(export func() string, fields map[string]interface{}) {
	if globalLogLevel >= 5 {
		logger.Trace(fields, export())
	}
}

func updateMap(dst map[string]interface{}, src map[string]interface{}) {
	for k, v := range src {
		dst[k] = v
	}
}

func (logger *Logger) TraceMap(kv map[string]interface{}, args ...interface{}) {
	updateMap(kv, logger.Fields)
	kv["src"] = sourceInfo()
	logrus.WithFields(kv).Trace(args...)
}

func (logger *Logger) DebugMap(kv map[string]interface{}, args ...interface{}) {
	updateMap(kv, logger.Fields)
	kv["src"] = sourceInfo()
	logrus.WithFields(kv).Debug(args...)
}

func (logger *Logger) WarnMap(kv map[string]interface{}, args ...interface{}) {
	updateMap(kv, logger.Fields)
	kv["src"] = sourceInfo()
	logrus.WithFields(kv).Warn(args...)
}

func (logger *Logger) InfoMap(kv map[string]interface{}, args ...interface{}) {
	updateMap(kv, logger.Fields)
	kv["src"] = sourceInfo()
	logrus.WithFields(kv).Info(args...)
}

func (logger *Logger) ErrorMap(kv map[string]interface{}, args ...interface{}) {
	updateMap(kv, logger.Fields)
	kv["src"] = sourceInfo()
	logrus.WithFields(kv).Error(args...)
}

func (logger *Logger) FatalMap(kv map[string]interface{}, args ...interface{}) {
	updateMap(kv, logger.Fields)
	kv["src"] = sourceInfo()
	logrus.WithFields(kv).Fatal(args...)
}

func (logger *Logger) PanicMap(kv map[string]interface{}, args ...interface{}) {
	updateMap(kv, logger.Fields)
	kv["src"] = sourceInfo()
	logrus.WithFields(kv).Panic(args...)
}
