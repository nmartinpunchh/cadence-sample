package workflow

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

var (
	applicationName = "This is my application Name"
	signalName      = "This is my signal Name"
)

/**
 * Sample workflow that uses local activities.
 */

// This is registration process where you register all your workflows
// and activity function handlers.
func init() {
	workflow.Register(SignalHandlingWorkflow)

	activity.Register(activityForCondition0)
	activity.Register(activityForCondition1)
	activity.Register(activityForCondition2)

	// no need to register local activities
}

type conditionAndAction struct {
	// condition is a function pointer to a local activity
	condition interface{}
	// action is a function pointer to a regular activity
	action interface{}
}

// SignalHandlingWorkflow is a workflow that waits on signal and then sends that signal to be processed by a child workflow.
func SignalHandlingWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	ch := workflow.GetSignalChannel(ctx, signalName)
	s := workflow.NewSelector(ctx)
	var signal string

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
		HeartbeatTimeout:       time.Second * 20,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	logger.Info("Signal received.", zap.String("signal", signal))

	var nameResult string
	err := workflow.ExecuteActivity(ctx, activityForCondition0, ch, s, ctx).Get(ctx, &nameResult)
	if err != nil {
		logger.Error("execute activity failed", zap.Error(err))
		return err
	}

	return nil
}

func activityForCondition0(ctx context.Context, ch workflow.Channel, s workflow.Selector, wctx workflow.Context) (string, error) {
	activity.GetLogger(ctx).Info("process 0")
	timeout := workflow.NewTimer(wctx, time.Minute*30)
	var signal1 string
	s.AddFuture(timeout, func(f workflow.Future) {
	})
	s.AddReceive(ch, func(c workflow.Channel, more bool) {
		for {
			c.Receive(wctx, signal1)
			log.Printf("received %s \n", signal1)
			if signal1 == "1" {
				break
			}
		}

	})

	// s.Select(ctx)

	return fmt.Sprintf("processed %s", signal1), nil
}

func activityForCondition1(ctx context.Context, ch workflow.Channel, s workflow.Selector) (string, error) {
	activity.GetLogger(ctx).Info("process 1")
	// some real processing logic goes here
	time.Sleep(time.Second * 2)
	return "processed_1", nil
}

func activityForCondition2(ctx context.Context, ch workflow.Channel, s workflow.Selector) (string, error) {
	activity.GetLogger(ctx).Info("process 2")
	// some real processing logic goes here
	time.Sleep(time.Second * 3)
	return "processed_2", nil
}
