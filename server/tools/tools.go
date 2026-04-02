package tools

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
)

var ticketPrices = map[string]string{
	"london": "$799",
	"paris":  "$899",
	"tokyo":  "$1400",
	"berlin": "$499",
}

var (
	failureMu         sync.RWMutex
	FailureSimulation = map[string]float64{
		"getTicketPrices":   0.0,
		"calculateEquation": 0.0,
	}
)

func SetFailureRate(tool string, failureRate float64) {
	failureMu.Lock()
	defer failureMu.Unlock()
	FailureSimulation[tool] = failureRate
}

func GetFailureRate(tool string) float64 {
	failureMu.Lock()
	defer failureMu.Unlock()
	return FailureSimulation[tool]
}

func HandleToolUse(block anthropic.ContentBlockUnion, variant anthropic.ToolUseBlock) (any, error) {

	if rate := GetFailureRate(block.Name); rate > 0 {
		if rand.Float64() < rate {
			return nil, fmt.Errorf("simulated failure for tool %s", block.Name)
		}
	}

	var response any
	switch block.Name {
	case "getTicketPrices":

		var input struct {
			DestinationCity string `json:"destinationCity"`
		}

		err := json.Unmarshal([]byte(variant.JSON.Input.Raw()), &input)
		if err != nil {
			return nil, fmt.Errorf("failed to process input for GetTicketPrices: %v", err)
		}
		response = getTicketPrices(input.DestinationCity)

	case "calculateEquation":
		var input struct {
			Var1 int    `json:"Var1"`
			Var2 int    `json:"Var2"`
			Op   string `json:"Op"`
		}

		err := json.Unmarshal([]byte(variant.JSON.Input.Raw()), &input)
		if err != nil {
			return nil, fmt.Errorf("failed to process input for calculateEquation: %v", err)
		}

		response = calculateEquation(input.Var1, input.Var2, input.Op)
	}

	return response, nil
}

func getTicketPrices(destinationCity string) string {
	fmt.Printf("Tool called for city %s", destinationCity)
	var price string
	if _, ok := ticketPrices[strings.ToLower(destinationCity)]; !ok {
		return "Unknown ticket price"
	}
	price = ticketPrices[strings.ToLower(destinationCity)]
	return fmt.Sprintf("The price of a ticket to %s is %s", destinationCity, price)
}

func calculateEquation(var1, var2 int, op string) float64 {

	operators := map[string]func(v1, v2 int) float64{
		"+": func(a, b int) float64 { return float64(a + b) },
		"-": func(a, b int) float64 { return float64(a - b) },
		"*": func(a, b int) float64 { return float64(a * b) },
		"/": func(a, b int) float64 { return float64(a) / float64(b) },
	}

	return operators[op](var1, var2)
}
