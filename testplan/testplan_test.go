package testplan

import (
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	if os.Getenv("GITHUB_ENV") == "" {
		// running locally
		os.Setenv("GITHUB_ENV", "../env.tmp")
	}
	t.Setenv("INPUT_LOGLEVEL", "TRACE")

	t.Run("Empty mandatory parameters", func(t *testing.T) {
		t.Setenv("INPUT_FILES", "")
		_, err := New()
		if err == nil {
			t.Errorf("Error is nil when the mandatory parameters are not set")
		}
		t.Logf("Got expected error %v", err)
	})

	t.Run("Merging yaml", func(t *testing.T) {
		t.Setenv("INPUT_FILES", "../example/defaults.yaml,../example/overwrite_string_with_structure.yaml")
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
		t.Setenv("INPUT_FILES", "../example/defaults.yaml,../example/overwrite_string_with_structure.yaml")
		t.Setenv("INPUT_SEPARATOR", "__")
		t.Setenv("GITHUB_OUTPUT", "../output.tmp")
		t.Setenv("INPUT_SET_OUTPUT", "true")
		t.Setenv("GITHUB_ENV", "../env.tmp")
		t.Setenv("INPUT_SET_ENV", "true")
		t.Setenv("INPUT_SET_PRINT", "true")
		t.Setenv("INPUT_YAML", "../yaml.tmp")
		t.Setenv("INPUT_GENERATE_JOB", "true")
		t.Setenv("INPUT_LOGFILE", "../logfile.tmp")

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
}
