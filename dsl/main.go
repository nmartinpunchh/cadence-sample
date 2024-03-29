package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/nmartinpunchh/cadence-sample/common"
	"github.com/nmartinpunchh/cadence-sample/workflow"
	"github.com/pborman/uuid"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/worker"
	"gopkg.in/yaml.v2"
)

// SignalName ..
const SignalName = "trigger-signal"

// This needs to be done as part of a bootstrap step when the process starts.
// The workers are supposed to be long running.
func startWorkers(h *common.SampleHelper) {
	// Configure worker options.
	workerOptions := worker.Options{
		MetricsScope: h.Scope,
		Logger:       h.Logger,
	}
	h.StartWorkers(h.Config.DomainName, workflow.ApplicationName, workerOptions)
}

func startWorkflow(h *common.SampleHelper, w workflow.Workflow) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                              "dsl_" + uuid.New(),
		TaskList:                        workflow.ApplicationName,
		ExecutionStartToCloseTimeout:    time.Minute,
		DecisionTaskStartToCloseTimeout: time.Minute,
	}
	h.StartWorkflow(workflowOptions, workflow.SimpleDSLWorkflow, w)
}

func main() {
	// var mode, dslConfig, workflowID, runID, queryType string
	var mode, dslConfig, workflowID string
	flag.StringVar(&mode, "m", "trigger", "Mode is worker or trigger.")
	flag.StringVar(&dslConfig, "dslConfig", "cmd/samples/dsl/workflow1.yaml", "dslConfig specify the yaml file for the dsl workflow.")
	flag.StringVar(&workflowID, "w", "", "WorkflowID")
	flag.Parse()

	var h common.SampleHelper
	h.SetupServiceConfig()

	switch mode {
	case "worker":
		startWorkers(&h)

		// The workers are supposed to be long running process that should not exit.
		// Use select{} to block indefinitely for samples, you can quit by CMD+C.
		select {}
	case "trigger":

		data, err := ioutil.ReadFile(dslConfig)
		if err != nil {
			panic(fmt.Sprintf("failed to load dsl config file %v", err))
		}
		var workflow workflow.Workflow
		if err := yaml.Unmarshal(data, &workflow); err != nil {
			panic(fmt.Sprintf("failed to unmarshal dsl config %v", err))
		}
		startWorkflow(&h, workflow)
	case "signal":
		h.SignalWorkflow(workflowID, SignalName, "WHAT IS DATA?")

	}
}
