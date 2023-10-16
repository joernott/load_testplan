// check_log_elasticsearch project main.go
package main

import (
	"github.com/joernott/load_testplan/testplan"
	"os"
)

func main() {
	plan, err := testplan.New()
	if err != nil {
		os.Exit(1)
	}
	err = plan.Output()
	if err != nil {
		os.Exit(1)
	}
}
