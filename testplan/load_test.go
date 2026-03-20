package testplan

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	if os.Getenv("GITHUB_ENV") == "" {
		// running locally
		os.Setenv("GITHUB_ENV", "../env.tmp")
	}
	t.Setenv("INPUT_LOGLEVEL", "TRACE")

	t.Run("Non-existing file", func(t *testing.T) {
		t.Setenv("INPUT_FILES", "nonexistant_yaml_file")
		_, err := New()
		if err == nil {
			t.Errorf("Error is nil when the yaml file does not exist")
		} else {
			t.Logf("Got expected error %v", err)
		}
	})

	t.Run("Broken yaml file", func(t *testing.T) {
		t.Setenv("INPUT_FILES", "../test/kaputt.yaml")
		_, err := New()
		if err == nil {
			t.Errorf("Error is nil when the yaml file is broken")
		} else {
			t.Logf("Got expected error %v", err)
		}
	})

	t.Run("Yaml without templating", func(t *testing.T) {
		t.Setenv("INPUT_FILES", "../example/defaults.yaml")
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
			fmt.Fprintf(w, "%v\n", yaml)
		}))
		defer svr.Close()

		t.Setenv("INPUT_FILES", svr.URL)
		t.Setenv("INPUT_INPUT_TYPE", "yaml")
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

	t.Run("Yaml with templating", func(t *testing.T) {
		t.Setenv("INPUT_FILES", "../example/with_template.yaml")
		t.Setenv("GITHUB_REPOSITORY", "joernott/load_testplan")
		t.Setenv("PATH", "/bin:/usr/bin")
		plan, err := New()
		if err != nil {
			t.Logf("Got error %v", err)
			t.Errorf("Error when loading a yaml file with template")
		}

		if plan.Data["template_string"].(string) != "joernott/load_testplan" {
			t.Errorf("Expected content 'joernott/load_testplan' of field 'template_string' not found")
		}
	})

	t.Run("Yaml with broken template", func(t *testing.T) {
		t.Setenv("INPUT_FILES", "../test/broken_template.yaml")
		t.Setenv("GITHUB_REPOSITORY", "joernott/load_testplan")
		t.Setenv("PATH", "/bin:/usr/bin")
		_, err := New()
		if err == nil {
			t.Errorf("Error is nil when the template code in a yaml file is broken")
		} else {
			t.Logf("Got expected error %v", err)
		}
	})

	t.Run("Broken json file", func(t *testing.T) {
		t.Setenv("INPUT_FILES", "../test/kaputt.json")
		_, err := New()
		if err == nil {
			t.Errorf("Error is nil when the json file is broken")
		} else {
			t.Logf("Got expected error %v", err)
		}
	})

	t.Run("Json without templating", func(t *testing.T) {
		t.Setenv("INPUT_FILES", "../example/defaults.json")
		plan, err := New()
		if err != nil {
			t.Logf("Got error %v", err)
			t.Errorf("Error when loading a simple json file")
		}

		s, ok := plan.Data["string"].(string)
		if !ok {
			t.Errorf("Could not find key 'string'")
		}
		if s != "String" {
			t.Errorf("Expected content 'String' of field 'string' not found")
		}

		n, ok := plan.Data["number"].(float64)
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

	t.Run("Json with broken template", func(t *testing.T) {
		t.Setenv("INPUT_FILES", "../test/broken_template.json")
		t.Setenv("GITHUB_REPOSITORY", "joernott/load_testplan")
		t.Setenv("PATH", "/bin:/usr/bin")
		_, err := New()
		if err == nil {
			t.Errorf("Error is nil when the template code in a yaml file is broken")
		} else {
			t.Logf("Got expected error %v", err)
		}
	})

	t.Run("Json from URL", func(t *testing.T) {
		json := `{
			"string":"String",
			"number":42,
			"another_string":"Another string",
			"root":{
				"branch":{
					"leaf":"A small leaf"
				}
			},
			"array":["First","Second"]
		}
		`
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "%v\n", json)
		}))
		defer svr.Close()

		t.Setenv("INPUT_FILES", svr.URL)
		t.Setenv("INPUT_INPUT_TYPE", "json")
		t.Logf("URL: %v", svr.URL)
		plan, err := New()
		if err != nil {
			t.Logf("Got error %v", err)
			t.Errorf("Error when loading a simple json file from URL")
		}

		s, ok := plan.Data["string"].(string)
		if !ok {
			t.Errorf("Could not find key 'string'")
		}
		if s != "String" {
			t.Errorf("Expected content 'String' of field 'string' not found")
		}

		n, ok := plan.Data["number"].(float64)
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

	t.Run("Json with templating", func(t *testing.T) {
		t.Setenv("INPUT_FILES", "../example/with_template.json")
		t.Setenv("GITHUB_REPOSITORY", "joernott/load_testplan")
		t.Setenv("PATH", "/bin:/usr/bin")
		plan, err := New()
		if err != nil {
			t.Logf("Got error %v", err)
			t.Errorf("Error when loading a json file with template")
		}

		if plan.Data["template_string"].(string) != "joernott/load_testplan" {
			t.Errorf("Expected content 'joernott/load_testplan' of field 'template_string' not found")
		}
	})

	t.Run("404 from URL", func(t *testing.T) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		}))
		defer svr.Close()

		t.Setenv("INPUT_FILES", svr.URL)
		t.Logf("URL: %v", svr.URL)
		_, err := New()
		if err == nil {
			t.Errorf("Didn't get an error on a 404 http error")
		} else {
			t.Logf("Got expected error %v", err)
		}
	})

	t.Run("Illegal URL", func(t *testing.T) {
		t.Setenv("INPUT_FILES", "http://localhost:99999/")
		t.Logf("URL: %v", "http://localhost:99999/")
		_, err := New()
		if err == nil {
			t.Errorf("Didn't get an error on an illegal URL")
		} else {
			t.Logf("Got expected error %v", err)
		}
	})

	t.Run("Unknown file", func(t *testing.T) {
		t.Setenv("INPUT_FILES", "../test/README.md")
		_, err := New()
		if err == nil {
			t.Errorf("Error is nil for an unknown file suffix")
		} else {
			t.Logf("Got expected error %v", err)
		}
	})
}
