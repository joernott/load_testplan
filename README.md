# Github action load_testplan
This github action loads one or more yaml files, applying golang templating and
merging them together to generate either environment variables or outputs to be
used in github actions.

## Usage
To use this action in your workflow and convert the content of some yaml files into outputs, you can use
```yaml
- uses: joernott/load_testplan
  with:
    files: 'example/defaults.yaml.example/overwrite_string_with_structure.yaml'
    set_output: true
    set_env: true
    separator: '_'
```

### Inputs
The action has the following inputs:

**files**: _Required_, no default  
A comma separated list of yaml files to be parsed. You can use go template
syntax in those files (see below). Files are parsed from left to right and
content from later files overrides previous values.

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

**logfile**: _Optional_, default: '-'  
Tis defines where the actions log messages/output should go. The default "-"
means logging to stdout.

**loglevel**: _Optional_, default: 'WARN'
How verbose should the action log what it is doing. The levels (in order of
increasing verbosity) are PANIC, FATAL, ERROR, WARN, INFO, DEBUG, TRACE.

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
