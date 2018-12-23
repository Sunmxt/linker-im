package log

import (
    "github.com/sirupsen/logrus"
    "runtime"
    "fmt"
    "path"
)

var globalLogLevel uint = 0

func init() {
    logrus.SetFormatter(&logrus.TextFormatter{
        FullTimestamp: true,
    })
    SetGlobalLogLevel(0)
}

func sourceInfo() string {
    _, file, line_no, ok := runtime.Caller(2)
    if !ok {
        return "?:?"
    }
    return fmt.Sprintf("%v:%v", path.Base(file), line_no)
}

// Global logging.
func GlobalLogLevel() uint {
    return globalLogLevel
}

func SetGlobalLogLevel(uint loglevel) {
    logrus_level = logrus.InfoLevel
    if loglevel > 3 {
        switch loglevel {
        case 4:
            logrus_level = logrus.DebugLevel
        default:
            logrus_level = logrus.TraceLevel    
        }
    }
    logrus.SetLevel(logrus_level)
    globalLogLevel = logrus_level
}

func Trace(args ...interface{}) {
    logrus.WithFields(logrus.Fields{
        "src": sourceInfo(),
    }).Trace(args)
}

func Debug(args ...interface{}) {
    logrus.WithFields(logrus.Fields{
        "src": sourceInfo(),
    }).Debug(args)
}

func Info(info_level uint ,args ...interface{}) {
    if info_level > globalLogLevel {
        return 
    }

    logrus.WithFields(logrus.Fields{
        "src": sourceInfo(),
    }).Info(args)
}

func Info0(args ...interface{}) {
    Info(0, args)
}

func Info1(args ...interface{}) {
    Info(1, args)
}

func Info2(args ...interface{}) {
    Info(2, args)
}

func Info3(args ...interface{}) {
    Info(3, args)
}

func Warn(args ...interface{}) {
    logrus.WithFields(logrus.Fields{
        "src": sourceInfo(),
    }).Warn(args)
}

func Error(args ...interface{}) {
    logrus.WithFields(logrus.Fields{
        "src": sourceInfo(),
    }).Error(args)
}

func Fatal(args ...interface{}) {
    logrus.WithFields(logrus.Fields{
        "src": sourceInfo(),
    }).Fatal(args)
}

func Panic(args ...interface{}) {
    logrus.WithFields(logrus.Fields{
        "src": sourceInfo(),
    }).Panic(args)
}

func Fatalf(format string, args ...interface{}) {
    logrus.WithFields(logrus.Fields{
        "src": sourceInfo(),
    }).Fatalf(format, args)
}

func TraceMap(kv map[string]interface{}, args ...interface{}) {
    kv["src"] = sourceInfo()
    logrus.WithFields(kv).Trace(args)
}

func DebugMap(kv map[string]interface{}, args ...interface{}) {
    kv["src"] = sourceInfo()
    logrus.WithFields(kv).Debug(args)
}

func WarnMap(kv map[string]interface{}, args ...interface{}) {
    kv["src"] = sourceInfo()
    logrus.WithFields(kv).Warn(args)
}

func ErrorMap(kv map[string]interface{}, args ...interface{}) {
    kv["src"] = sourceInfo()
    logrus.WithFields(kv).Error(args)
}

func FatalMap(kv map[string]interface{}, args ...interface{}) {
    kv["src"] = sourceInfo()
    logrus.WithFields(kv).Fatal(args)
}

func PanicMap(kv map[string]interface{}, args ...interface{}) {
    kv["src"] = sourceInfo()
    logrus.WithFields(kv).Panic(args)
}
