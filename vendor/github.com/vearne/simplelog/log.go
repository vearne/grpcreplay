package simplelog

import (
	"io"
	"log"
	"os"
)

const (
	DebugLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

var (
	Level  = WarnLevel
	LogMap map[string]int
)

func init() {
	LogMap = make(map[string]int)
	LogMap["debug"] = DebugLevel
	LogMap["info"] = InfoLevel
	LogMap["warn"] = WarnLevel
	LogMap["error"] = ErrorLevel

	// use environment variables to set log level
	logLevel := os.Getenv("SIMPLE_LOG_LEVEL")
	if _, ok := LogMap[logLevel]; ok {
		SetLevel(LogMap[logLevel])
	}
}

func SetLevel(loglevel int) {
	Level = loglevel
}

func SetOutput(w io.Writer) {
	log.SetOutput(w)
}

func Debug(format string, v ...interface{}) {
	if Level <= DebugLevel {
		log.Printf("[debug] "+format+"\n", v...)
	}
}

func Info(format string, v ...interface{}) {
	if Level <= InfoLevel {
		log.Printf("[info] "+format+"\n", v...)
	}
}

func Warn(format string, v ...interface{}) {
	if Level <= WarnLevel {
		log.Printf("[warn] "+format+"\n", v...)
	}
}

func Error(format string, v ...interface{}) {
	if Level <= ErrorLevel {
		log.Printf("[error] "+format+"\n", v...)
	}
}

func Fatal(format string, v ...interface{}) {
	if Level <= FatalLevel {
		log.Printf("[fatal] "+format+"\n", v...)
	}
	os.Exit(1)
}
