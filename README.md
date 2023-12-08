# Github action load_testplan
This github action loads one or more yaml or json files, applying golang templating and
merging them together to generate either environment variables or outputs to be
used in github actions. It can also output yaml or json files containing the merged data.

The code is maintained in the (development branch)[https://github.com/joernott/load_testplan/tree/development],
the (main branch)[https://github.com/joernott/load_testplan/tree/main] only
contains the binaries for the action itself to reduce the amount of data to
download.

## Usage
To use this action in your workflow and convert the content of some yaml files into outputs, you can use
```yaml
- uses: joernott/load_testplan
  with:
    files: 'example/defaults.yaml.example/overwrite_string_with_structure.yaml'
    set_output: true
    set_env: true
    set_print: true
    separator: '_'
```

### Inputs
The action has the following inputs:

**files**: _Required_, no default  
A comma separated list of yaml files to be parsed. You can use go template
syntax in those files (see below). Files are parsed from left to right and
content from later files overrides previous values.

**input_type**: _Not required_, 'auto'  
Set this to "yaml" and all the files specified in the _files_ parameter will be
loaded as yaml files. Set it to "json" to always assume them to be json files.
If this is set to "auto", it will try to determine the file type based on its
suffix. Files ending in ".yml" or ".yaml" will be loaded as yaml, files with
".json", ".jsn", ",jso" or ".js" will be loaded as json.

**separator**: _Optional_, default: '_'  
When flattening hierarchical yaml into key/value pairs, this separator will be
used to concatenate the keys. Using the default underscore,
```yaml
foo:
  bar: baz
```
becomes "foo_bar=baz".

**set_output**: _Optional_, default: false  
When set to true, the action will write the variables to the file defined in
the environment variable $GITHUB_OUTPUT.

**set_env**: _Optional_, default: false  
When set to true, the action will write the variables to the file defined in
the environment variable $GITHUB_ENV.

**set_print**: _Optional_, default: false  
When set to true, the action will write the variables with their values to
stdout to provide some debugging information. Keys on the first level with a
structure below will be interpreted as headings and printed in purple.

```yaml
compile:
  flags: '--foo'
```

will render as

```
compile
compile_flags='--foo'
```

**yaml**: _Optional_, default: ''  
If you provide a file name here, the merged and processed data will be written
into the file as yaml. This helps debugging merge issues or can be used as later input.

**json**: _Optional_, default: ''  
If you provide a file name here, the merged and processed data will be written
into the file as json. This helps debugging merge issues or can be used as later input.

**logfile**: _Optional_, default: '-'  
Tis defines where the actions log messages/output should go. The default "-"
means logging to stdout.

**loglevel**: _Optional_, default: 'WARN'
How verbose should the action log what it is doing. The levels (in order of
increasing verbosity) are PANIC, FATAL, ERROR, WARN, INFO, DEBUG, TRACE.

**generate_job**: _Optional_, default: false
If you set generate_job to true, this will create a file "job_load_testplan.yml" which
contains a job running the testplan with exactly the input parameters specified and defining
job outputs for every key found in the loaded yaml file(s). Copying from this
file can save you a lot of time when you have a lot of outputs you want to use in the workflow.

**token**: _Optional_, default: ''
If you want to check out a plan from a private github repository, you can set token to
${{ github.token }} to add "?token=<dynamic token>" to the URL.

## Templating

You can use (go templating)[https://pkg.go.dev/text/template] in the yaml files.
To access the environment variables available to the action, use
```
{{ .Env.<VARRIABLE_NAME> }}
```

Instead of using the environment variables, you can also use 
```
{{ .Github.<FIELD> }}
```
to access the Github context. We use (sethvargo/go-githubactions)[https://github.com/sethvargo/go-githubactions/],
so the capitalization follows their (data structure)[https://github.com/sethvargo/go-githubactions/blob/v1.1.0/actions.go#L461]
e.g. {{ .Github.RunID }} is the same as {{ .Env.GITHUB_RUN_ID }}, which
you would write as ${{ github.run_id }} in a workflow file.

As we parse multiple files, the output from the previous merge step (or first file) is available
as 
```
{{ .Data.<field> }}
```
