package util

import (
	"os"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger zerolog.Logger

func Init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
	// Create a lumberjack logger for rotating log files
	logFile := &lumberjack.Logger{
		Filename:   "blogotron-application.log",
		MaxSize:    10, // Maximum size in megabytes before log rotation
		MaxBackups: 3,  // Maximum number of old log files to retain
		MaxAge:     7,  // Maximum number of days to retain log files
		Compress:   true,
	}
	multi := zerolog.MultiLevelWriter(consoleWriter, logFile)

	// Set the output to console and file writer
	Logger = zerolog.New(multi).With().Timestamp().Caller().Logger()

	// Remember to close the log file when you're done
	defer logFile.Close()
}
