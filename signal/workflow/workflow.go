package workflow

import (
	"context"
	"log"
	"time"

	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
)

func init() {
	workflow.Register(SignalHandlingWorkflow)

	activity.Register(activityForCondition0)

}

func createWait(ctx workflow.Context, signalName string, timeout time.Duration) {
	// logger := workflow.GetLogger(ctx)

	ch := workflow.GetSignalChannel(ctx, signalName)
	s := workflow.NewSelector(ctx)

	timeoutFuture := workflow.NewTimer(ctx, timeout)
	var signal1 string

	s.AddFuture(timeoutFuture, func(f workflow.Future) {
		//handle timeout here
	})
	s.AddReceive(ch, func(c workflow.Channel, more bool) {
		c.Receive(ctx, &signal1)
		log.Printf("received data %s from %s \n", signal1, signalName)
	})

	s.Select(ctx)
}

// SignalHandlingWorkflow is a workflow that waits on signal and then sends that signal to be processed by a node workflow.
func SignalHandlingWorkflow(ctx workflow.Context) error {
	// logger := workflow.GetLogger(ctx)
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
		HeartbeatTimeout:       time.Second * 20,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	wf := createWfModel()

	var nextNode *node = &wf.root

	for {
		switch nextNode.nodeType {
		case "wait":
			log.Println("I'm in wait case")
			timeout, _ := time.ParseDuration(nextNode.args[1])
			// I should be waiting here...
			createWait(ctx, nextNode.args[0], timeout)
		case "action":
			log.Println("Parsed action node type")
			//workflow.ExecuteActivity("NAMESPACE")
		}

		if nextNode.next == nil {
			break
		}

		nextNode = nextNode.next.next
	}

	return nil
}

func activityForCondition0(ctx context.Context) (string, error) {

	// return fmt.Sprintf("processed %s", signal1), nil
	return "", nil
}

func createWfModel() customWorkflow {

	wf := customWorkflow{
		name: "TT",
		root: node{
			nodeType: "wait",
			// For nodeType == wait. 1st arg is value 2nd is timeout
			args: []string{"1", "15m"},
			next: &child{
				next: &node{
					nodeType: "action",
					args:     []string{},
					next: &child{
						next: &node{
							nodeType: "wait",
							args:     []string{"2", "10m"},
							next:     nil,
						},
					},
				},
			},
		},
	}

	return wf

}
