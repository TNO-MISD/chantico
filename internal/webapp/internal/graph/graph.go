package graph

import "strings"

type Node struct {
	Name    string
	Parents []*Node
}

func GenerateMermaidString(nodes []*Node) string {
	if len(nodes) == 0 {
		return ""
	}

	var output []string
	output = append(output, "flowchart TB")

	for _, node := range nodes {
		if len(node.Parents) == 0 {
			output = append(output, node.Name)
		} else {
			for _, parent := range node.Parents {
				output = append(output, node.Name+"-->"+parent.Name)
			}
		}
	}

	return strings.Join(output, "\n")
}
