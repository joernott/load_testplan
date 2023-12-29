package testplan

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"regexp"
	"strings"
	"text/template"
)

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
			plan.yamlfile = f
			defer plan.yamlfile.Close()
			fmt.Fprintln(plan.yamlfile, "---")
		}
	}

	for key, value := range plan.Data {
		plan.outputKey("", key, value, "")
	}

	err := plan.OutputJson()
	if err != nil {
		return err
	}

	err = plan.OutputJob()
	if plan.LogLevel == "TRACE" {
		plan.debugOutputFile()
	}
	return err
}

func (plan *Testplan) OutputJson() error {
	logger := log.With().Str("func", "OutputJson").Str("package", "testplan").Logger()
	logger.Trace().Msg("Enter func")
	if plan.JsonName != "" {
		f, err := os.Create(plan.JsonName)
		if err != nil {
			logger.Warn().Err(err).Str("file", plan.JsonName).Msg("Can't open json file, skipping json output")
			return err
		}
		plan.jsonfile = f
		defer plan.jsonfile.Close()
		j, err := json.MarshalIndent(plan.Data, "", "    ")
		if err != nil {
			logger.Warn().Err(err).Str("file", plan.JsonName).Msg("Can't marshal json, skipping json output")
			return err
		}
		fmt.Fprint(plan.jsonfile, string(j))
	}
	return nil
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
			s, err := sanitizeName(prefix + key)
			if err != nil {
				s = prefix + key
			}
			fmt.Println("\033[35m" + s + "\033[0m")
		}
		if plan.YamlName != "" {
			fmt.Fprintf(plan.yamlfile, "%v%v:\n", yaml_indentation, key)
		}
		for k, v := range value.(map[string]interface{}) {
			plan.outputKey(prefix+key+plan.Separator, k, v, yaml_indentation+"  ")
		}
	case []interface{}:
		if plan.YamlName != "" {
			fmt.Fprintf(plan.yamlfile, "%v%v:\n", yaml_indentation, key)
		}
		o := ""
		for _, v := range value.([]interface{}) {
			o = o + "\n" + fmt.Sprintf("%v", v)
			if plan.YamlName != "" {
				fmt.Fprintf(plan.yamlfile, "%v  - %v\n", yaml_indentation, v)
			}
		}
		o = o[1:]
		logger.Debug().Str("prefix", prefix).Str("key", key).Str("value", o).Msg("Output Multiline")
		s, err := sanitizeName(prefix + key)
		if err != nil {
			s = prefix + key
		}
		if plan.SetOutput {
			plan.Actions.SetOutput(s, o)
		}
		if plan.SetEnv {
			plan.Actions.SetEnv(s, o)
		}
		if plan.SetPrint {
			fmt.Printf("%v=%v\n", s, o)
		}
	default:
		v := fmt.Sprintf("%v", value)
		logger.Debug().Str("prefix", prefix).Str("key", key).Str("value", v).Msg("Output Single")
		s, err := sanitizeName(prefix + key)
		if err != nil {
			s = prefix + key
		}
		if plan.SetOutput {
			plan.Actions.SetOutput(s, v)
		}
		if plan.SetEnv {
			plan.Actions.SetEnv(s, v)
		}
		if plan.SetPrint {
			fmt.Printf("%v=%v\n", s, v)
		}
		if plan.YamlName != "" {
			switch value.(type) {
			case string:
				if strings.Contains(v, "\n") {
					v = yaml_indentation + yaml_indentation + strings.ReplaceAll(v, "\n", "\n"+yaml_indentation+yaml_indentation)
					fmt.Fprintf(plan.yamlfile, "%v%v: |\n%v\n", yaml_indentation, key, v)
				} else {
					fmt.Fprintf(plan.yamlfile, "%v%v: '%v'\n", yaml_indentation, key, v)
				}
			default:
				fmt.Fprintf(plan.yamlfile, "%v%v: %v\n", yaml_indentation, key, v)
			}
		}
		if plan.GenerateJob {
			plan.Outputs[s] = v
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

func sanitizeName(name string) (string, error) {
	logger := log.With().Str("func", "sanitizeName").Str("package", "testplan").Str("key", name).Logger()
	logger.Trace().Msg("Enter func")
	r := `[^0-9A-Za-z_]`
	m, err := regexp.Compile(r)
	if err != nil {
		log.Error().Err(err).Str("regexp", r).Msg("Failed to compile regexp.")
		return "", err
	}
	s := m.ReplaceAllString(name, "_")
	r = `^[0-9]|GITHUB_.*`
	illegal_start, err := regexp.Match(r, []byte(s))
	if err != nil {
		log.Error().Err(err).Str("regexp", r).Msg("Failed match start with regexp.")
		return "", err
	}
	if illegal_start {
		s = "_" + s
	}
	log.Debug().Str("sanitized", s).Msg("Sanitized name.")
	return s, nil
}
