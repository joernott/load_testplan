package testplan

import (
	"errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	githubactions "github.com/sethvargo/go-githubactions"
	"os"
	"strings"
)

// Input parameters for this action
type Testplan struct {
	Actions     *githubactions.Action
	Files       []string
	InputType   string
	Separator   string
	SetOutput   bool
	SetEnv      bool
	SetPrint    bool
	YamlName    string
	JsonName    string
	yamlfile    *os.File
	jsonfile    *os.File
	GenerateJob bool
	LogFile     string
	LogLevel    string
	Data        map[string]interface{}
	Github      *githubactions.GitHubContext
	Env         map[string]string
	Outputs     map[string]string
	Token       string
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

	plan.Token = plan.Actions.GetInput("token")

	x := plan.Actions.GetInput("input_type")
	plan.InputType = strings.ToLower(x)
	if plan.InputType == "" {
		plan.InputType = "auto"
	}

	err = plan.getFileList()
	if err != nil {
		return nil, err
	}
	err = plan.loadFiles()
	if err != nil {
		return nil, err
	}
	plan.Separator = plan.Actions.GetInput("separator")
	if plan.Separator == "" {
		plan.Separator = "_"
	}
	x = plan.Actions.GetInput("set_output")
	plan.SetOutput = strings.ToLower(x) == "true"

	x = plan.Actions.GetInput("set_env")
	plan.SetEnv = strings.ToLower(x) == "true"

	x = plan.Actions.GetInput("set_print")
	plan.SetPrint = strings.ToLower(x) == "true"

	x = plan.Actions.GetInput("generate_job")
	plan.GenerateJob = strings.ToLower(x) == "true"

	plan.YamlName = plan.Actions.GetInput("yaml")

	plan.JsonName = plan.Actions.GetInput("json")

	a := zerolog.Arr()
	for _, f := range plan.Files {
		a = a.Str(f)
	}
	logger.Debug().
		Array("files", a).
		Str("type", plan.InputType).
		Str("separator", plan.Separator).
		Bool("set_output", plan.SetOutput).
		Bool("set_env", plan.SetEnv).
		Bool("set_print", plan.SetPrint).
		Bool("generate_job", plan.GenerateJob).
		Str("yaml_name", plan.YamlName).
		Str("json_name", plan.JsonName).
		Str("token", plan.Token).
		Msg("Inputs")

	o := make(map[string]string)
	plan.Outputs = o

	return plan, nil
}

// parse the comma separated list of files into an array
func (plan *Testplan) getFileList() error {
	logger := log.With().Str("func", "getFiles").Str("package", "testplan").Logger()
	logger.Trace().Msg("Enter func")
	raw_files := plan.Actions.GetInput("files")
	if raw_files == "" {
		logger.Error().Str("error", "Missing parameter").Msg("Mandatory parameter 'files' is not defined")
		return errors.New("missing parameter")
	}
	f := strings.Split(raw_files, ",")
	plan.Files = f
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
