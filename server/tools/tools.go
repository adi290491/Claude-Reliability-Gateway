package tools

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

var ticketPrices = map[string]string{
	"london": "$799",
	"paris":  "$899",
	"tokyo":  "$1400",
	"berlin": "$499",
}

var FailureSimulation = map[string]float64{
	"getTicketPrices":   0.0,
	"calculateEquation": 0.0,
}

func HandleToolUse(block anthropic.ContentBlockUnion, variant anthropic.ToolUseBlock) (any, error) {

	if rate, exist := FailureSimulation[block.Name]; exist {
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
			panic(err)
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

type CoordinateResponse struct {
	Lat  float64 `json:"lat"`
	Long float64 `json:"long"`
}

func getCoordinates(location string) CoordinateResponse {
	return CoordinateResponse{
		Long: -122.4194,
		Lat:  37.7749,
	}
}

type WeatherResponse struct {
	Unit        string  `json:"unit"`
	Temperature float64 `json:"temperature"`
}

func getWeather(lat, long float64, unit string) WeatherResponse {
	return WeatherResponse{
		Unit:        "fahrenheit",
		Temperature: 122,
	}
}
