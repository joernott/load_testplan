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

	"github.com/davecgh/go-spew/spew"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	githubactions "github.com/sethvargo/go-githubactions"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
	"net/http"
	"net/url"
)

// Input parameters for this action
type Testplan struct {
	Actions     *githubactions.Action
	Files       []string
	Separator   string
	SetOutput   bool
	SetEnv      bool
	SetPrint    bool
	YamlName    string
	file        *os.File
	GenerateJob bool
	LogFile     string
	LogLevel    string
	Data        map[string]interface{}
	Github      *githubactions.GitHubContext
	Env         map[string]string
	Outputs     map[string]string
	Token       string
}

var generate_template = `
jobs:
  load_testplan:
    runs-on: 'ubuntu-latest'
    steps:
      - name: 'Load Testplan'
        id: ltp
        uses: 'joernott/load_testplan@v1'
        with:
          files: '{{ range $i, $file := .Files }}{{ if gt $i 0 }},{{ end }}{{ $file }}{{ end }}'
          separator: '{{ .Separator }}'
          set_output: {{ .SetOutput }}
          set_env: {{ .SetEnv }}
          set_print: {{ .SetPrint }}
          yaml: '{{ .YamlName }}'
          loglevel: '{{ .LogLevel }}'
          logfile: '{{ .LogFile }}'
    outputs:
{{- range $key, $value := .Outputs }}
      {{ $key }}: ${{"{{"}} steps.ltp.outputs.{{ $key }} {{"}}"}}
{{- end }}
`

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
	plan.Separator = plan.Actions.GetInput("separator")
	if plan.Separator == "" {
		plan.Separator = "_"
	}
	x := plan.Actions.GetInput("set_output")
	plan.SetOutput = strings.ToLower(x) == "true"

	x = plan.Actions.GetInput("set_env")
	plan.SetEnv = strings.ToLower(x) == "true"

	x = plan.Actions.GetInput("set_print")
	plan.SetPrint = strings.ToLower(x) == "true"

	x = plan.Actions.GetInput("generate_job")
	plan.GenerateJob = strings.ToLower(x) == "true"

	plan.YamlName = plan.Actions.GetInput("yaml")

	plan.Token = plan.Actions.GetInput("token")

	a := zerolog.Arr()
	for _, f := range plan.Files {
		a = a.Str(f)
	}
	logger.Debug().
		Array("files", a).
		Str("separator", plan.Separator).
		Bool("set_output", plan.SetOutput).
		Bool("set_env", plan.SetEnv).
		Bool("set_print", plan.SetPrint).
		Bool("generate_job", plan.GenerateJob).
		Str("yaml_name", plan.YamlName).
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

// Merge two maps
func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

// Load all files as templates, convert them into maps and merge them together
func (plan *Testplan) loadFiles() error {
	logger := log.With().Str("func", "loadFiles").Str("package", "testplan").Logger()
	logger.Trace().Msg("Enter func")
	for _, f := range plan.Files {
		log.Debug().Str("file", f).Msg("Load file")

		input, err := plan.parseFile(f)
		if err != nil {
			return err
		}
		var data map[string]interface{}
		err = yaml.Unmarshal(input, &data)
		if err != nil {
			log.Error().Err(err).Str("file", f).Msg("Could not unmarshall yaml")
			return err
		}
		plan.Data = mergeMaps(plan.Data, data)
		if plan.LogLevel == "TRACE" {
			spew.Dump(plan.Data)
		}
	}
	return nil
}

// get a file content from an URL
func getFromURL(url string) (string, error) {
	logger := log.With().Str("func", "getFromURL").Str("package", "testplan").Str("url", url).Logger()
	logger.Trace().Msg("Enter func")

	r, err := http.Get(url)
	if err != nil {
		logger.Error().Err(err).Msg("Get from URL failed")
		return "", err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		e := fmt.Errorf("HTTP status %v is not OK", r.StatusCode)
		logger.Error().Err(e).Msg("Unsupported HTTP status")
		return "", e
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to read HTTP response body")
		return "", err
	}

	return string(data), nil
}

// Uses text/template templating when loading the yaml files
func (plan *Testplan) parseFile(name string) ([]byte, error) {
	logger := log.With().Str("func", "readFile").Str("package", "testplan").Logger()
	logger.Trace().Msg("Enter func")
	var b bytes.Buffer
	var t *template.Template

	u, err := url.ParseRequestURI(name)
	if err != nil || u.Scheme == "file" {
		t = template.New(path.Base(name))
		t, err = t.ParseFiles(name)
		if err != nil {
			log.Error().Err(err).Str("file", name).Msg("Failed to read file")
			return b.Bytes(), err
		}
	} else {
		n := name
		if plan.Token != "" {
			n = n + "?token=" + plan.Token
			log.Debug().Msg("Adding token to url")
		}
		s, err1 := getFromURL(n)
		if err1 != nil {
			return b.Bytes(), err1
		}
		t = template.New(path.Base(u.Path))
		t, err1 = t.Parse(s)
		if err1 != nil {
			log.Error().Err(err).Str("file", name).Msg("Failed to parse template from URL")
			return b.Bytes(), err1
		}
	}

	if err = t.Execute(&b, plan); err != nil {
		log.Error().Err(err).Str("file", name).Msg("Failed to parse template")
		return b.Bytes(), err
	}
	return b.Bytes(), nil
}

// Output the map as environment variables and/or outputs
func (plan *Testplan) Output() error {
	logger := log.With().Str("func", "Output").Str("package", "testplan").Logger()
	logger.Trace().Msg("Enter func")

	if plan.YamlName != "" {
		f, err := os.Create(plan.YamlName)
		if err != nil {
			logger.Warn().Err(err).Str("file", plan.YamlName).Msg("Can't open yaml file, skipping yaml output")
			plan.YamlName = ""
		} else {
			plan.file = f
			defer plan.file.Close()
			fmt.Fprintln(plan.file, "---")
		}
	}

	for key, value := range plan.Data {
		plan.outputKey("", key, value, "")
	}
	err := plan.OutputJob()
	if plan.LogLevel == "TRACE" {
		plan.debugOutputFile()
	}
	return err
}

// debug the github output
func (plan *Testplan) debugOutputFile() {
	logger := log.With().Str("func", "Output").Str("package", "testplan").Logger()
	logger.Trace().Msg("Enter func")
	if plan.SetOutput {
		filename, ok := os.LookupEnv("GITHUB_OUTPUT")
		if !ok {
			logger.Error().Str("error", "Could not get GITHUB_OUTPUT").Msg("Failed to dump outputs")
		} else {
			data, err := os.ReadFile(filename)
			if err != nil {
				logger.Error().Err(err).Str("file", filename).Msg("Failed read file")
			} else {
				logger.Trace().Str("output", string(data)).Str("file", filename).Msg("Content of GITHUB_OUTPUT")
			}
		}
	}
}

// Recursively output a key with its value. If the value is an array, a multiline output will be generated, if its a map, we will descend
func (plan *Testplan) outputKey(prefix string, key string, value interface{}, yaml_indentation string) {
	logger := log.With().
		Str("func", "outputKey").
		Str("package", "testplan").
		Str("prefix", prefix).
		Str("key", key).
		Logger()
	logger.Trace().Msg("Enter func")

	switch value.(type) {
	case map[string]interface{}:
		if plan.SetPrint && prefix == "" {
			fmt.Println("\033[35m" + prefix + key + "\033[0m")
		}
		if plan.YamlName != "" {
			fmt.Fprintf(plan.file, "%v%v:\n", yaml_indentation, key)
		}
		for k, v := range value.(map[string]interface{}) {
			plan.outputKey(prefix+key+plan.Separator, k, v, yaml_indentation+"  ")
		}
	case []interface{}:
		if plan.YamlName != "" {
			fmt.Fprintf(plan.file, "%v%v:\n", yaml_indentation, key)
		}
		o := ""
		for _, v := range value.([]interface{}) {
			o = o + "\n" + fmt.Sprintf("%v", v)
			if plan.YamlName != "" {
				fmt.Fprintf(plan.file, "%v  - %v\n", yaml_indentation, v)
			}
		}
		o = o[1:]
		logger.Debug().Str("prefix", prefix).Str("key", key).Str("value", o).Msg("Output Multiline")
		if plan.SetOutput {
			plan.Actions.SetOutput(prefix+key, o)
		}
		if plan.SetEnv {
			plan.Actions.SetEnv(prefix+key, o)
		}
		if plan.SetPrint {
			fmt.Printf("%v=%v\n", prefix+key, o)
		}
	default:
		v := fmt.Sprintf("%v", value)
		logger.Debug().Str("prefix", prefix).Str("key", key).Str("value", v).Msg("Output Single")
		if plan.SetOutput {
			plan.Actions.SetOutput(prefix+key, v)
		}
		if plan.SetEnv {
			plan.Actions.SetEnv(prefix+key, v)
		}
		if plan.SetPrint {
			fmt.Printf("%v=%v\n", prefix+key, v)
		}
		if plan.YamlName != "" {
			switch value.(type) {
			case string:
				fmt.Fprintf(plan.file, "%v%v: '%v'\n", yaml_indentation, key, v)
			default:
				fmt.Fprintf(plan.file, "%v%v: %v\n", yaml_indentation, key, v)
			}
		}
		if plan.GenerateJob {
			plan.Outputs[prefix+key] = v
		}
	}
}

// Write a yaml file as copy/paste template for the github workflow
func (plan *Testplan) OutputJob() error {
	logger := log.With().Str("func", "OutputJob").Str("package", "testplan").Logger()
	logger.Trace().Msg("Enter func")
	if !plan.GenerateJob {
		return nil
	}
	filename := "job_load_testplan.yml"
	genfile, err := os.Create(filename)
	if err != nil {
		logger.Error().Err(err).Str("file", filename).Msg("Can't open " + filename)
		return err
	}
	defer genfile.Close()
	t := template.New("generate_job")
	t, err = t.Parse(generate_template)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse template for GenerateJob")
		return err
	}
	if err = t.Execute(genfile, plan); err != nil {
		log.Fatal().Err(err).Str("file", filename).Msg("Failed to write template output")
		return err
	}
	return nil
}

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
