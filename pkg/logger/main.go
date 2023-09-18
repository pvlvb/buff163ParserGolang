package logger

import (
	"github.com/sirupsen/logrus"
	"io"
	"os"
)

var Log = logrus.New()

func init() {
	// Set the log output to write to both os.Stderr and a file.
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// If there's an error with the log file, just use the default stderr.
		Log.SetOutput(os.Stderr)
	} else {
		mw := io.MultiWriter(os.Stderr, file)
		Log.SetOutput(mw)
	}

	// Set log format to colored terminal output with a custom timestamp format.
	Log.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true, // Force colored log even if stdout is not a terminal
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Set the log level. In a real-world application, you might want this to be configurable.
	Log.SetLevel(logrus.DebugLevel)
}

// WithFields This utility function helps in logging with fields easily.
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Log.WithFields(fields)
}
