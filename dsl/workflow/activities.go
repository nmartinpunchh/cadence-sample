package workflow

import (
	"fmt"

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"
)

// SignalName ..
const SignalName = "trigger-signal"

func sampleActivity1(input []string, ch workflow.Channel) (string, error) {
	name := "sampleActivity1"
	fmt.Printf("Run %s with input %v \n", name, input)
	return "Result_" + name, nil
}

func sampleActivity2(input []string) (string, error) {
	name := "sampleActivity2"
	fmt.Printf("Run %s with input %v \n", name, input)
	return "Result_" + name, nil
}

func sampleActivity3(input []string) (string, error) {
	name := "sampleActivity3"
	fmt.Printf("Run %s with input %v \n", name, input)
	return "Result_" + name, nil
}

func sampleActivity4(input []string) (string, error) {
	name := "sampleActivity4"
	fmt.Printf("Run %s with input %v \n", name, input)
	return "Result_" + name, nil
}

func sampleActivity5(input []string) (string, error) {
	name := "sampleActivity5"
	fmt.Printf("Run %s with input %v \n", name, input)
	return "Result_" + name, nil
}

func signalActivity() (string, error) {

	var processResult string
	ch := workflow.GetSignalChannel(ctx, SignalName)
	for {
		var signal string
		if more := ch.Receive(ctx, &signal); !more {
			fmt.Println("Signal channel closed")
			return "", cadence.NewCustomError("signal_channel_closed")
		}

		fmt.Printf("Signal received %v.", signal)

		if signal == "exit" {

			processResult = fmt.Sprintf("Result of %v", signal)

			fmt.Printf("Processed signal: %v, result: %v", signal, processResult)
			break
		}
	}

	return processResult, nil
}
