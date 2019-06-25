package workflow

import (
	"context"
	"log"
	"time"

	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

var (
	// applicationName = "This is my application Name"
	signalName = "This is my signal Name"
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

func createWait(ctx workflow.Context, data string, timeout time.Duration) {
	logger := workflow.GetLogger(ctx)

	ch := workflow.GetSignalChannel(ctx, signalName)
	s := workflow.NewSelector(ctx)
	var signal string
	logger.Info("Signal received.", zap.String("signal", signal))
	timeoutFuture := workflow.NewTimer(ctx, timeout)
	var signal1 string
	s.AddFuture(timeoutFuture, func(f workflow.Future) {
	})
	s.AddReceive(ch, func(c workflow.Channel, more bool) {
		for {
			c.Receive(ctx, &signal1)
			log.Printf("received %s \n", signal1)
			if signal1 == data {
				break
			}
		}
	})

	s.Select(ctx)
}

// SignalHandlingWorkflow is a workflow that waits on signal and then sends that signal to be processed by a child workflow.
func SignalHandlingWorkflow(ctx workflow.Context) error {
	// logger := workflow.GetLogger(ctx)
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
		HeartbeatTimeout:       time.Second * 20,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	wf := createWfModel()

	nextNode := &wf.root

	// Not the best, I'm sure there's a better way ..
	for ok := true; ok; ok = nextNode != nil {
		switch nextNode.nodeType {
		case "wait":
			timeout, _ := time.ParseDuration(nextNode.args[1])
			createWait(ctx, nextNode.args[0], timeout)
		case "action":
			log.Println("Parsed action node type")
		}

		nextNode = nextNode.next.next
	}

	return nil
}

func activityForCondition0(ctx context.Context) (string, error) {

	// return fmt.Sprintf("processed %s", signal1), nil
	return "", nil
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

func createWfModel() customWorkflow {

	wf := customWorkflow{
		name: "TT",
		root: node{
			nodeType: "wait",
			// For nodeType == wait. 1st arg is value 2nd is timeout
			args: []string{"1", "30m"},
			next: &child{
				next: &node{
					nodeType: "wait",
					args:     []string{"2", "10"},
					next:     nil,
				},
			},
		},
	}

	return wf

}
