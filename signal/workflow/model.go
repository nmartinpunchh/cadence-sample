package workflow

type (
	customWorkflow struct {
		name string
		root node
	}

	node struct {
		// type is a restricted keyword
		nodeType string
		args     []string
		next     *child
	}

	child struct {
		next *node
	}
)
