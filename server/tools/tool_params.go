package tools

import "github.com/anthropics/anthropic-sdk-go"

func CreateToolParams() []anthropic.ToolUnionParam {
	toolParams := []anthropic.ToolParam{
		{
			Name:        "getTicketPrices",
			Description: anthropic.String("Get the price of a return ticket to the destination city."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"destinationCity": map[string]any{
						"type":        "string",
						"description": "The city that the customer wants to travel to",
					},
				},
				Required: []string{"destinationCity"},
			},
		},
		{
			Name:        "calculateEquation",
			Description: anthropic.String("Accepts an equation as string, then returns the result"),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"Var1": map[string]any{
						"type":        "number",
						"description": "First operand",
					},
					"Var2": map[string]any{
						"type":        "number",
						"description": "Second operand",
					},
					"Op": map[string]any{
						"type":        "string",
						"description": "Operator",
					},
				},
				Required: []string{"Var1", "Var2", "Op"},
			},
		},
	}

	tools := make([]anthropic.ToolUnionParam, len(toolParams))

	for i, toolParam := range toolParams {
		tools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
	}

	return tools
}
