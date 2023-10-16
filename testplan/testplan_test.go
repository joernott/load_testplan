package testplan

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
)

func TestNew(t *testing.T) {

	t.Run("Empty mandatory parameters", func(t *testing.T) {
		os.Unsetenv("INPUT_files")
		_, err := New()
		if err == nil {
			t.Errorf("Error is nil when the mandatory parameters are not set")
		}
		t.Logf("Got expected error %v", err)
	})

	t.Run("Non-existing file", func(t *testing.T) {
		os.Setenv("INPUT_files", "nonexistant_yaml_file")
		_, err := New()
		if err == nil {
			t.Errorf("Error is nil when the yaml file does not exist")
		}
		t.Logf("Got expected error %v", err)
	})

	t.Run("Broken yaml file", func(t *testing.T) {
		os.Setenv("INPUT_files", "../example/kaputt.yaml")
		_, err := New()
		if err == nil {
			t.Errorf("Error is nil when the yaml file is broken")
		}
		t.Logf("Got expected error %v", err)
	})

	t.Run("Yaml without templating", func(t *testing.T) {
		os.Setenv("INPUT_files", "../example/defaults.yaml")
		plan, err := New()
		if err != nil {
			t.Logf("Got error %v", err)
			t.Errorf("Error when loading a simple yaml file")
		}

		s, ok := plan.Data["string"].(string)
		if !ok {
			t.Errorf("Could not find key 'string'")
		}
		if s != "String" {
			t.Errorf("Expected content 'String' of field 'string' not found")
		}

		n, ok := plan.Data["number"].(int)
		if !ok {
			t.Errorf("Could not find key 'number'")
		}
		if n != 42 {
			t.Errorf("Expected content '42' of field 'number' not found")
		}

		if _, ok := plan.Data["root"].(map[string]interface{}); !ok {
			t.Errorf("Wrong type for Field 'root'")
		}

		if _, ok := plan.Data["array"].([]interface{}); !ok {
			t.Errorf("Wrong type for Field 'array'")
		}
		a := plan.Data["array"].([]interface{})
		if len(a) != 2 {
			t.Errorf("Expected len=2 for 'array', got %v", len(a))
		}

	})

	t.Run("Yaml from URL", func(t *testing.T) {
		yaml := `---
string: 'String'
number: 42
another_string: 'Another string'
root:
  branch:
    leaf: 'A small leaf'
`
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, yaml)
		}))
		defer svr.Close()

		os.Setenv("INPUT_files", svr.URL)
		t.Logf("URL: %v", svr.URL)
		plan, err := New()
		if err != nil {
			t.Logf("Got error %v", err)
			t.Errorf("Error when loading a simple yaml file from URL")
		}

		s, ok := plan.Data["string"].(string)
		if !ok {
			t.Errorf("Could not find key 'string'")
		}
		if s != "String" {
			t.Errorf("Expected content 'String' of field 'string' not found")
		}

		n, ok := plan.Data["number"].(int)
		if !ok {
			t.Errorf("Could not find key 'number'")
		}
		if n != 42 {
			t.Errorf("Expected content '42' of field 'number' not found")
		}

		if _, ok := plan.Data["root"].(map[string]interface{}); !ok {
			t.Errorf("Wrong type for Field 'root'")
		}
	})

	t.Run("404 from URL", func(t *testing.T) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		}))
		defer svr.Close()

		os.Setenv("INPUT_files", svr.URL)
		t.Logf("URL: %v", svr.URL)
		_, err := New()
		if err == nil {
			t.Errorf("Didn't get an error on a 404 http error")
		}
	})

	t.Run("Yaml with templating", func(t *testing.T) {
		os.Setenv("INPUT_files", "../example/with_template.yaml")
		os.Setenv("GITHUB_REPOSITORY", "joernott/load_testplan")
		os.Setenv("PATH", "/bin:/usr/bin")
		plan, err := New()
		if err != nil {
			t.Logf("Got error %v", err)
			t.Errorf("Error when loading a simple yaml file")
		}

		if plan.Data["template_string"].(string) != "joernott/load_testplan" {
			t.Errorf("Expected content 'joernott/load_testplan' of field 'template_string' not found")
		}
	})

	t.Run("Merging yaml", func(t *testing.T) {
		os.Setenv("INPUT_files", "../example/defaults.yaml,../example/overwrite_string_with_structure.yaml")
		plan, err := New()
		if err != nil {
			t.Logf("Got error %v", err)
			t.Errorf("Error when loading a simple yaml file")
		}

		if _, ok := plan.Data["another_string"].(map[string]interface{}); !ok {
			t.Errorf("Wrong type for Field 'another_string', it should be a map now")
		}

		d := plan.Data["another_string"].(map[string]interface{})
		if d["string"].(string) != "Another string" {
			t.Errorf("Expected content 'Another string' of field 'another_string|string' not found")
		}

		if d["number"].(int) != 42 {
			t.Errorf("Expected content '42' of field 'another_string|number' not found")
		}

		if _, ok := d["array"].([]interface{}); !ok {
			t.Errorf("Wrong type for Field 'array', it should be a []interface{}")
		}
	})

	t.Run("Test inputs", func(t *testing.T) {
		os.Setenv("INPUT_files", "../example/defaults.yaml,../example/overwrite_string_with_structure.yaml")
		os.Setenv("INPUT_separator", "__")
		os.Setenv("GITHUB_OUTPUT", "../output.tmp")
		os.Setenv("INPUT_set_output", "true")
		os.Setenv("GITHUB_ENV", "../env.tmp")
		os.Setenv("INPUT_set_env", "true")
		os.Setenv("INPUT_set_print", "true")
		os.Setenv("INPUT_yaml", "../yaml.tmp")
		os.Setenv("INPUT_generate_job", "true")
		os.Setenv("INPUT_logfile", "../logfile.tmp")

		plan, err := New()
		if err != nil {
			t.Logf("Got error %v", err)
			t.Errorf("Error when loading yaml files")
		}

		l := len(plan.Files)
		if l != 2 {
			t.Errorf("Wrong number of files to parse, expected 2, got %v", l)
		}

		if plan.Separator != "__" {
			t.Errorf("Expected content '__' as separator, got %v", plan.Separator)
		}

		if !plan.SetOutput {
			t.Errorf("Expected SetOutput to be 'true', got 'false'")
		}

		if plan.Env["GITHUB_OUTPUT"] != "../output.tmp" {
			t.Errorf("Expected Env[GITHUB_OUTPUT] to be '../output.tmp', got '%v'", plan.Env["GITHUB_OUTPUT"])
		}

		if !plan.SetEnv {
			t.Errorf("Expected SetEnv to be 'true', got 'false'")
		}

		if plan.Env["GITHUB_ENV"] != "../env.tmp" {
			t.Errorf("Expected Env[GITHUB_ENV] to be '../env.tmp', got '%v'", plan.Env["GITHUB_ENV"])
		}

		if !plan.SetPrint {
			t.Errorf("Expected SetPrint to be 'true', got 'false'")
		}

		if plan.YamlName != "../yaml.tmp" {
			t.Errorf("Expected YamlName to be '../yaml.tmp', got '%v'", plan.YamlName)
		}

		if !plan.GenerateJob {
			t.Errorf("Expected GenerateJob to be 'true', got 'false'")
		}

		if plan.LogFile != "../logfile.tmp" {
			t.Errorf("Expected LogFile to be '../logfile.tmp', got '%v'", plan.LogFile)
		}
	})

	levels := [7]string{"PANIC", "FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
	for _, l := range levels {
		t.Run("Test loglevel", func(t *testing.T) {
			os.Setenv("INPUT_files", "../example/defaults.yaml")
			os.Setenv("INPUT_loglevel", l)
			plan, err := New()
			if err != nil {
				t.Logf("Got error %v", err)
				t.Errorf("Error when loading yaml files")
			}
			if plan.LogLevel != l {
				t.Errorf("Expected '%v' as LogLevel, got %v", l, plan.LogLevel)
			}
		})
	}

	t.Run("Test wrong loglevel", func(t *testing.T) {
		os.Setenv("INPUT_files", "../example/defaults.yaml")
		os.Setenv("INPUT_loglevel", "FOOBAR")
		_, err := New()
		if err == nil {
			t.Errorf("Expected an error for an invalid LogLevel")
		}
	})

	t.Run("Test empty loglevel", func(t *testing.T) {
		os.Setenv("INPUT_files", "../example/defaults.yaml")
		os.Unsetenv("INPUT_loglevel")
		plan, err := New()
		if err != nil {
			t.Logf("Got error %v", err)
			t.Errorf("Error when loading yaml files")
		}
		if plan.LogLevel != "WARN" {
			t.Errorf("Expected 'WARN' as default LogLevel, got %v", plan.LogLevel)
		}
	})
}

func TestOutput(t *testing.T) {
	os.Setenv("INPUT_files", "../example/defaults.yaml")
	os.Setenv("GITHUB_OUTPUT", "../output.tmp")
	os.Setenv("INPUT_set_output", "true")
	os.Setenv("GITHUB_ENV", "../env.tmp")
	os.Setenv("INPUT_set_env", "true")
	os.Setenv("INPUT_set_print", "true")
	os.Setenv("INPUT_yaml", "../yaml.tmp")
	os.Setenv("INPUT_generate_job", "true")
	os.Setenv("INPUT_logfile", "../logfile.tmp")

	plan, err := New()
	if err != nil {
		t.Logf("Got error %v", err)
		t.Errorf("Error when loading yaml file")
	}
	err = plan.Output()
	if err != nil {
		t.Errorf("Error generating the output")
	}

	t.Run("Test output", func(t *testing.T) {
		pattern := "(?m:^string<<_GitHubActionsFileCommandDelimeter_\r*\nString\r*\n_GitHubActionsFileCommandDelimeter_\r*\n)"
		r := regexp.MustCompile(pattern)
		b, err := os.ReadFile("../output.tmp")
		if err != nil {
			t.Errorf("Could not read file ../output.tmp")
		}
		s := string(b)
		found := r.FindAllStringSubmatch(s, -1)
		if found == nil {
			t.Errorf("Did not find output for variable 'string' with regexp search: %v", found)
		}
	})

	t.Run("Test env", func(t *testing.T) {
		pattern := "(?m:^string<<_GitHubActionsFileCommandDelimeter_\r*\nString\r*\n_GitHubActionsFileCommandDelimeter_\r*\n)"
		r := regexp.MustCompile(pattern)
		b, err := os.ReadFile("../env.tmp")
		if err != nil {
			t.Errorf("Could not read file ../env.tmp")
		}
		s := string(b)
		found := r.FindAllStringSubmatch(s, -1)
		if found == nil {
			t.Errorf("Did not find output for variable 'string' with regexp search: %v", found)
		}
	})

	t.Run("Test yaml", func(t *testing.T) {
		b, err := os.ReadFile("../yaml.tmp")
		if err != nil {
			t.Errorf("Could not read '../yaml.tmp', reason: %v", err)
		}
		var data map[string]interface{}
		if err := yaml.Unmarshal(b, &data); err != nil {
			t.Errorf("Could not unmarshall yaml, reason: %v", err)
		}
		if data["string"].(string) != "String" {
			t.Errorf("Field 'string' does not contain 'String'")
		}
	})

	t.Run("Test job", func(t *testing.T) {
		b, err := os.ReadFile("job_load_testplan.yml")
		if err != nil {
			t.Errorf("Could not read 'job_load_testplan.yml', reason: %v", err)
		}
		var data map[string]interface{}
		if err := yaml.Unmarshal(b, &data); err != nil {
			t.Errorf("Could not unmarshall yaml, reason: %v", err)
		}
		j, ok := data["jobs"].(map[string]interface{})
		if !ok {
			t.Errorf("Could not find key 'jobs'")
		}
		l, ok := j["load_testplan"].(map[string]interface{})
		if !ok {
			t.Errorf("Could not find key 'jobs.load_testplan'")
		}
		o, ok := l["outputs"].(map[string]interface{})
		if !ok {
			t.Errorf("Could not find key 'jobs.load_testplan.outputs'")
		}
		s, ok := o["string"].(string)
		if !ok {
			t.Errorf("Could not find key 'jobs.load_testplan.outputs.string'")
		}
		if s != "${{ steps.ltp.string }}" {
			t.Errorf("'jobs.load_testplan.outputs.string' does not contain '${{ steps.ltp.string }}' but '%v'", s)
		}

	})

}
