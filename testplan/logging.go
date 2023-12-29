package testplan

import (
	"errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"strings"
)

// Configure the logging.
func (plan *Testplan) setupLogging() error {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	var output io.Writer
	plan.LogFile = plan.Actions.GetInput("logfile")
	if plan.LogFile == "-" || plan.LogFile == "" {
		output = os.Stdout
	} else {
		output = &lumberjack.Logger{
			Filename:   plan.LogFile,
			MaxBackups: 10,
			MaxAge:     1,
			Compress:   true,
		}
	}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
	plan.LogLevel = strings.ToUpper(plan.Actions.GetInput("loglevel"))
	switch plan.LogLevel {
	case "TRACE":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "DEBUG":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "INFO":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "WARN":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "":
		plan.LogLevel = "WARN"
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "ERROR":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "FATAL":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "PANIC":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		err := errors.New("Illegal log level " + plan.LogLevel)
		log.Error().Err(err).Msg("Wrong parameter")
		return err
	}
	log.Debug().
		Str("func", "setupLogging").
		Str("package", "testplan").
		Str("logfile", plan.LogFile).
		Str("loglevel", plan.LogLevel).
		Msg("Logging initialized")
	return nil
}
