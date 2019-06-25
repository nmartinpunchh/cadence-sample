package main

import (
	"flag"
	"time"

	"github.com/nmartinpunchh/cadence-sample/common"
	s "github.com/nmartinpunchh/cadence-sample/signal/workflow"
	"github.com/pborman/uuid"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/worker"
)

var (
	applicationName string = "This is my application Name"
	signalName      string = "This is my signal Name"
)

// This needs to be done as part of a bootstrap step when the process starts.
// The workers are supposed to be long running.
func startWorkers(h *common.SampleHelper) {
	// Configure worker options.
	workerOptions := worker.Options{
		MetricsScope: h.Scope,
		Logger:       h.Logger,
	}
	h.StartWorkers(h.Config.DomainName, applicationName, workerOptions)
}

func startWorkflow(h *common.SampleHelper) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                              "localactivity_" + uuid.New(),
		TaskList:                        applicationName,
		ExecutionStartToCloseTimeout:    time.Hours * 3,
		DecisionTaskStartToCloseTimeout: time.Minute,
		WorkflowIDReusePolicy:           client.WorkflowIDReusePolicyAllowDuplicate,
	}
	h.StartWorkflow(workflowOptions, s.SignalHandlingWorkflow)
}

func main() {
	var mode, workflowID, signal, runID string
	flag.StringVar(&mode, "m", "trigger", "Mode is worker, trigger or query.")
	flag.StringVar(&workflowID, "w", "", "WorkflowID")
	flag.StringVar(&signal, "s", "signal_data", "SignalData")
	flag.StringVar(&runID, "r", "", "RunID")
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
		startWorkflow(&h)
	case "signal":
		h.SignalWorkflow(workflowID, signalName, runID, signal)
	}
}
