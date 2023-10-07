package testplan

import (
	"bytes"
	"io"
	"os"
	"path"
	"strings"
	"text/template"

	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	githubactions "github.com/sethvargo/go-githubactions"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
)

// Input parameters for this action
type Testplan struct {
	Actions    *githubactions.Action
	files      []string
	separator  string
	set_output bool
	set_env    bool
	logfile    string
	loglevel   string
	Data       map[string]interface{}
	Github     *githubactions.GitHubContext
	Env        map[string]string
}

// Creates a new testplan, loading the files and preparing the contexts
func New() (*Testplan, error) {
	plan := new(Testplan)
	plan.Actions = githubactions.New()
	err := plan.setupLogging()
	if err != nil {
		return nil, err
	}

	logger := log.With().Str("func", "NewTestplan").Str("package", "testplan").Logger()
	logger.Trace().Msg("Enter func")

	plan.parseEnv()
	g, err := githubactions.Context()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to get github context")
		return nil, err
	}
	plan.Github = g

	err = plan.getFileList()
	if err != nil {
		return nil, err
	}
	err = plan.loadFiles()
	if err != nil {
		return nil, err
	}
	plan.separator = plan.Actions.GetInput("separator")
	if plan.separator == "" {
		plan.separator = "_"
	}
	o := plan.Actions.GetInput("set_output")
	plan.set_output = strings.ToLower(o) == "true"

	e := plan.Actions.GetInput("set_env")
	plan.set_env = strings.ToLower(e) == "true"

	a := zerolog.Arr()
	for _, f := range plan.files {
		a = a.Str(f)
	}
	logger.Debug().
		Array("files", a).
		Str("separator", plan.separator).
		Bool("set_output", plan.set_output).
		Bool("set_env", plan.set_env).
		Msg("Inputs")

	return plan, nil
}

// parse the comma separated list of files into an array
func (plan *Testplan) getFileList() error {
	logger := log.With().Str("func", "getFiles").Str("package", "testplan").Logger()
	logger.Trace().Msg("Enter func")
	raw_files := plan.Actions.GetInput("files")
	if raw_files == "" {
		logger.Error().Str("error", "Missing parameter").Msg("Mandatory parameter 'files' is not defined")
		return errors.New("Missing parameter")
	}
	f := strings.Split(raw_files, ",")
	plan.files = f
	return nil
}

// parse the environment into a key/value map to be used in the template
func (plan *Testplan) parseEnv() {
	logger := log.With().Str("func", "ParseEnv").Str("package", "testplan").Logger()
	logger.Trace().Msg("Enter func")
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		env[pair[0]] = pair[1]
	}
	plan.Env = env
}

// Merge two maps
func mergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// Load all files as templates, convert them into maps and merge them together
func (plan *Testplan) loadFiles() error {
	logger := log.With().Str("func", "loadFiles").Str("package", "testplan").Logger()
	logger.Trace().Msg("Enter func")
	for _, f := range plan.files {
		log.Debug().Str("file", f).Msg("Load file")

		input, err := plan.parseFile(f)
		if err != nil {
			log.Fatal().Err(err).Str("file", f).Msg("Failed to read file")
			return err
		}
		var data map[string]interface{}
		if err := yaml.Unmarshal(input, &data); err != nil {
			log.Fatal().Err(err).Str("file", f).Msg("Could not unmarshall yaml")
			return err
		}
		plan.Data = mergeMaps(plan.Data, data)
	}
	return nil
}

// Uses text/template templating when loading the yaml files
func (plan *Testplan) parseFile(name string) ([]byte, error) {
	logger := log.With().Str("func", "readFile").Str("package", "testplan").Logger()
	logger.Trace().Msg("Enter func")
	var b bytes.Buffer

	t := template.New(path.Base(name))
	t, err := t.ParseFiles(name)
	if err != nil {
		log.Fatal().Err(err).Str("file", name).Msg("Failed to read file")
		return b.Bytes(), err
	}
	if err = t.Execute(&b, plan); err != nil {
		log.Fatal().Err(err).Str("file", name).Msg("Failed to parse template")
		return b.Bytes(), err
	}
	return b.Bytes(), nil
}

// Output the map as environment variables and/or outputs
func (plan *Testplan) Output() {
	logger := log.With().Str("func", "Output").Str("package", "testplan").Logger()
	logger.Trace().Msg("Enter func")

	for key, value := range plan.Data {
		plan.outputKey("", key, value)
	}
}

// Recursively output a key with its value. If the value is an array, a multiline output wikk be generated, if its a map, we will descend
func (plan *Testplan) outputKey(prefix string, key string, value interface{}) {
	logger := log.With().
		Str("func", "outputKey").
		Str("package", "testplan").
		Str("prefix", prefix).
		Str("key", key).
		Logger()
	logger.Trace().Msg("Enter func")

	switch value.(type) {
	case map[string]interface{}:
		for k, v := range value.(map[string]interface{}) {
			plan.outputKey(prefix+key+plan.separator, k, v)
		}
	case []interface{}:
		o := ""
		for _, v := range value.([]interface{}) {
			o = o + "\n" + fmt.Sprintf("%v", v)
		}
		o = o[1:]
		logger.Debug().Str("prefix", prefix).Str("key", key).Str("value", o).Msg("Output Multiline")

		if plan.set_output {
			plan.Actions.SetOutput(prefix+key, o)
		}
		if plan.set_env {
			plan.Actions.SetEnv(prefix+key, o)
		}
	default:
		v := fmt.Sprintf("%v", value)
		logger.Debug().Str("prefix", prefix).Str("key", key).Str("value", v).Msg("Output Single")
		if plan.set_output {
			plan.Actions.SetOutput(prefix+key, v)
		}
		if plan.set_env {
			plan.Actions.SetEnv(prefix+key, v)
		}
	}
}

// Configure the logging.
func (plan *Testplan) setupLogging() error {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	var output io.Writer
	plan.logfile = plan.Actions.GetInput("logfile")
	if plan.logfile == "-" {
		output = os.Stdout
	} else {
		output = &lumberjack.Logger{
			Filename:   plan.logfile,
			MaxBackups: 10,
			MaxAge:     1,
			Compress:   true,
		}
	}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
	plan.loglevel = strings.ToUpper(plan.Actions.GetInput("loglevel"))
	switch plan.loglevel {
	case "TRACE":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "DEBUG":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "INFO":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "WARN":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "ERROR":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "FATAL":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "PANIC":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		err := errors.New("Illegal log level " + plan.loglevel)
		log.Error().Err(err).Msg("Wrong parameter")
		return err
	}
	log.Debug().
		Str("func", "setupLogging").
		Str("package", "testplan").
		Str("logfile", plan.logfile).
		Str("loglevel", plan.loglevel).
		Msg("Logging initialized")
	return nil
}
