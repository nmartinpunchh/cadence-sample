// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package internal

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pborman/uuid"
	"github.com/uber-go/tally"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/.gen/go/cadence/workflowservicetest"
	"go.uber.org/cadence/.gen/go/shared"
	"go.uber.org/cadence/encoded"
	"go.uber.org/cadence/internal/common"
	"go.uber.org/zap"
)

type (
	// Worker represents objects that can be started and stopped.
	Worker interface {
		// Start starts the worker in a non-blocking fashion
		Start() error
		// Run is a blocking start and cleans up resources when killed
		// returns error only if it fails to start the worker
		Run() error
		// Stop cleans up any resources opened by worker
		Stop()
	}

	// WorkerOptions is used to configure a worker instance.
	// The current timeout resolution implementation is in seconds and uses math.Ceil(d.Seconds()) as the duration. But is
	// subjected to change in the future.
	WorkerOptions struct {
		// Optional: To set the maximum concurrent activity executions this worker can have.
		// The zero value of this uses the default value.
		// default: defaultMaxConcurrentActivityExecutionSize(1k)
		MaxConcurrentActivityExecutionSize int

		// Optional: Sets the rate limiting on number of activities that can be executed per second per
		// worker. This can be used to limit resources used by the worker.
		// Notice that the number is represented in float, so that you can set it to less than
		// 1 if needed. For example, set the number to 0.1 means you want your activity to be executed
		// once for every 10 seconds. This can be used to protect down stream services from flooding.
		// The zero value of this uses the default value. Default: 100k
		WorkerActivitiesPerSecond float64

		// Optional: To set the maximum concurrent local activity executions this worker can have.
		// The zero value of this uses the default value.
		// default: 1k
		MaxConcurrentLocalActivityExecutionSize int

		// Optional: Sets the rate limiting on number of local activities that can be executed per second per
		// worker. This can be used to limit resources used by the worker.
		// Notice that the number is represented in float, so that you can set it to less than
		// 1 if needed. For example, set the number to 0.1 means you want your local activity to be executed
		// once for every 10 seconds. This can be used to protect down stream services from flooding.
		// The zero value of this uses the default value. Default: 100k
		WorkerLocalActivitiesPerSecond float64

		// Optional: Sets the rate limiting on number of activities that can be executed per second.
		// This is managed by the server and controls activities per second for your entire tasklist
		// whereas WorkerActivityTasksPerSecond controls activities only per worker.
		// Notice that the number is represented in float, so that you can set it to less than
		// 1 if needed. For example, set the number to 0.1 means you want your activity to be executed
		// once for every 10 seconds. This can be used to protect down stream services from flooding.
		// The zero value of this uses the default value. Default: 100k
		TaskListActivitiesPerSecond float64

		// Optional: To set the maximum concurrent decision task executions this worker can have.
		// The zero value of this uses the default value.
		// default: defaultMaxConcurrentTaskExecutionSize(1k)
		MaxConcurrentDecisionTaskExecutionSize int

		// Optional: Sets the rate limiting on number of decision tasks that can be executed per second per
		// worker. This can be used to limit resources used by the worker.
		// The zero value of this uses the default value. Default: 100k
		WorkerDecisionTasksPerSecond float64

		// Optional: if the activities need auto heart beating for those activities
		// by the framework
		// default: false not to heartbeat.
		AutoHeartBeat bool

		// Optional: Sets an identify that can be used to track this host for debugging.
		// default: default identity that include hostname, groupName and process ID.
		Identity string

		// Optional: Metrics to be reported. Metrics emitted by the cadence client are not prometheus compatible by
		// default. To ensure metrics are compatible with prometheus make sure to create tally scope with sanitizer
		// options set.
		// var (
		// _safeCharacters = []rune{'_'}
		// _sanitizeOptions = tally.SanitizeOptions{
		// 	NameCharacters: tally.ValidCharacters{
		// 		Ranges:     tally.AlphanumericRange,
		// 		Characters: _safeCharacters,
		// 	},
		// 		KeyCharacters: tally.ValidCharacters{
		// 			Ranges:     tally.AlphanumericRange,
		// 			Characters: _safeCharacters,
		// 		},
		// 		ValueCharacters: tally.ValidCharacters{
		// 			Ranges:     tally.AlphanumericRange,
		// 			Characters: _safeCharacters,
		// 		},
		// 		ReplacementCharacter: tally.DefaultReplacementCharacter,
		// 	}
		// )
		// opts := tally.ScopeOptions{
		// 	Reporter:        reporter,
		// 	SanitizeOptions: &_sanitizeOptions,
		// }
		// scope, _ := tally.NewRootScope(opts, time.Second)
		// default: no metrics.
		MetricsScope tally.Scope

		// Optional: Logger framework can use to log.
		// default: default logger provided.
		Logger *zap.Logger

		// Optional: Enable logging in replay.
		// In the workflow code you can use workflow.GetLogger(ctx) to write logs. By default, the logger will skip log
		// entry during replay mode so you won't see duplicate logs. This option will enable the logging in replay mode.
		// This is only useful for debugging purpose.
		// default: false
		EnableLoggingInReplay bool

		// Optional: Disable running workflow workers.
		// default: false
		DisableWorkflowWorker bool

		// Optional: Disable running activity workers.
		// default: false
		DisableActivityWorker bool

		// Optional: Disable sticky execution.
		// default: false
		// Sticky Execution is to run the decision tasks for one workflow execution on same worker host. This is an
		// optimization for workflow execution. When sticky execution is enabled, worker keeps the workflow state in
		// memory. New decision task contains the new history events will be dispatched to the same worker. If this
		// worker crashes, the sticky decision task will timeout after StickyScheduleToStartTimeout, and cadence server
		// will clear the stickiness for that workflow execution and automatically reschedule a new decision task that
		// is available for any worker to pick up and resume the progress.
		DisableStickyExecution bool

		// Optional: Sticky schedule to start timeout.
		// default: 5s
		// The resolution is seconds. See details about StickyExecution on the comments for DisableStickyExecution.
		StickyScheduleToStartTimeout time.Duration

		// Optional: sets context for activity. The context can be used to pass any configuration to activity
		// like common logger for all activities.
		BackgroundActivityContext context.Context

		// Optional: Sets how decision worker deals with non-deterministic history events
		// (presumably arising from non-deterministic workflow definitions or non-backward compatible workflow definition changes).
		// default: NonDeterministicWorkflowPolicyBlockWorkflow, which just logs error but reply nothing back to server
		NonDeterministicWorkflowPolicy NonDeterministicWorkflowPolicy

		// Optional: Sets DataConverter to customize serialization/deserialization of arguments in Cadence
		// default: defaultDataConverter, an combination of thriftEncoder and jsonEncoder
		DataConverter encoded.DataConverter

		// Optional: worker graceful shutdown timeout
		// default: 0s
		WorkerStopTimeout time.Duration
	}
)

// NonDeterministicWorkflowPolicy is an enum for configuring how client's decision task handler deals with
// mismatched history events (presumably arising from non-deterministic workflow definitions).
type NonDeterministicWorkflowPolicy int

const (
	// NonDeterministicWorkflowPolicyBlockWorkflow is the default policy for handling detected non-determinism.
	// This option simply logs to console with an error message that non-determinism is detected, but
	// does *NOT* reply anything back to the server.
	// It is chosen as default for backward compatibility reasons because it preserves the old behavior
	// for handling non-determinism that we had before NonDeterministicWorkflowPolicy type was added to
	// allow more configurability.
	NonDeterministicWorkflowPolicyBlockWorkflow NonDeterministicWorkflowPolicy = iota
	// NonDeterministicWorkflowPolicyFailWorkflow behaves exactly the same as Ignore, up until the very
	// end of processing a decision task.
	// Whereas default does *NOT* reply anything back to the server, fail workflow replies back with a request
	// to fail the workflow execution.
	NonDeterministicWorkflowPolicyFailWorkflow
)

// NewWorker creates an instance of worker for managing workflow and activity executions.
// service 	- thrift connection to the cadence server.
// domain - the name of the cadence domain.
// taskList 	- is the task list name you use to identify your client worker, also
// 		  identifies group of workflow and activity implementations that are hosted by a single worker process.
// options 	-  configure any worker specific options like logger, metrics, identity.
func NewWorker(
	service workflowserviceclient.Interface,
	domain string,
	taskList string,
	options WorkerOptions,
) Worker {
	return newAggregatedWorker(service, domain, taskList, options)
}

// ReplayWorkflowExecution loads a workflow execution history from the Cadence service and executes a single decision task for it.
// Use for testing the backwards compatibility of code changes and troubleshooting workflows in a debugger.
// The logger is the only optional parameter. Defaults to the noop logger.
func ReplayWorkflowExecution(ctx context.Context, service workflowserviceclient.Interface, logger *zap.Logger, domain string, execution WorkflowExecution) error {
	sharedExecution := &shared.WorkflowExecution{
		RunId:      common.StringPtr(execution.RunID),
		WorkflowId: common.StringPtr(execution.ID),
	}
	request := &shared.GetWorkflowExecutionHistoryRequest{
		Domain:    common.StringPtr(domain),
		Execution: sharedExecution,
	}
	hResponse, err := service.GetWorkflowExecutionHistory(ctx, request)
	if err != nil {
		return err
	}

	return replayWorkflowHistory(logger, service, domain, hResponse.History)
}

// ReplayWorkflowHistory executes a single decision task for the given history.
// Use for testing the backwards compatibility of code changes and troubleshooting workflows in a debugger.
// The logger is an optional parameter. Defaults to the noop logger.
func ReplayWorkflowHistory(logger *zap.Logger, history *shared.History) error {

	if logger == nil {
		logger = zap.NewNop()
	}

	testReporter := logger.Sugar()
	controller := gomock.NewController(testReporter)
	service := workflowservicetest.NewMockClient(controller)

	domain := "ReplayDomain"

	return replayWorkflowHistory(logger, service, domain, history)
}

// ReplayWorkflowHistoryFromJSONFile executes a single decision task for the given json history file.
// Use for testing the backwards compatibility of code changes and troubleshooting workflows in a debugger.
// The logger is an optional parameter. Defaults to the noop logger.
func ReplayWorkflowHistoryFromJSONFile(logger *zap.Logger, jsonfileName string) error {

	history, err := extractHistoryFromFile(jsonfileName)

	if err != nil {
		return err
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	testReporter := logger.Sugar()
	controller := gomock.NewController(testReporter)
	service := workflowservicetest.NewMockClient(controller)

	domain := "ReplayDomain"

	return replayWorkflowHistory(logger, service, domain, history)
}

func replayWorkflowHistory(logger *zap.Logger, service workflowserviceclient.Interface, domain string, history *shared.History) error {
	taskList := "ReplayTaskList"
	events := history.Events
	if events == nil {
		return errors.New("empty events")
	}
	if len(events) < 3 {
		return errors.New("at least 3 events expected in the history")
	}
	first := events[0]
	if first.GetEventType() != shared.EventTypeWorkflowExecutionStarted {
		return errors.New("first event is not WorkflowExecutionStarted")
	}
	attr := first.WorkflowExecutionStartedEventAttributes
	if attr == nil {
		return errors.New("corrupted WorkflowExecutionStarted")
	}
	workflowType := attr.WorkflowType
	execution := &shared.WorkflowExecution{
		RunId:      common.StringPtr(uuid.NewUUID().String()),
		WorkflowId: common.StringPtr("ReplayId"),
	}
	task := &shared.PollForDecisionTaskResponse{
		Attempt:                common.Int64Ptr(0),
		TaskToken:              []byte("ReplayTaskToken"),
		WorkflowType:           workflowType,
		WorkflowExecution:      execution,
		History:                history,
		PreviousStartedEventId: common.Int64Ptr(math.MaxInt64),
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	metricScope := tally.NoopScope
	iterator := &historyIteratorImpl{
		nextPageToken: task.NextPageToken,
		execution:     task.WorkflowExecution,
		domain:        "ReplayDomain",
		service:       service,
		metricsScope:  metricScope,
		maxEventID:    task.GetStartedEventId(),
	}
	params := workerExecutionParameters{
		TaskList: taskList,
		Identity: "replayID",
		Logger:   logger,
	}
	taskHandler := newWorkflowTaskHandler(domain, params, nil, getHostEnvironment())
	_, _, err := taskHandler.ProcessWorkflowTask(&workflowTask{task: task, historyIterator: iterator})
	return err
}

func extractHistoryFromFile(jsonfileName string) (*shared.History, error) {
	raw, err := ioutil.ReadFile(jsonfileName)
	if err != nil {
		return nil, err
	}

	var deserializedEvents []*shared.HistoryEvent
	err = json.Unmarshal(raw, &deserializedEvents)

	if err != nil {
		return nil, err
	}
	history := &shared.History{Events: deserializedEvents}

	return history, nil
}
