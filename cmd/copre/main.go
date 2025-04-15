package main

import (
	"fmt"
	"log"

	"github.com/jsnanigans/copre/pkg/copre"
)

func main() {
	oldText := `line one-two-smile
line two-smile
line 3-smile`
	newText := `line one-two-smile
line two
line 3-smile`

	predictions, err := copre.PredictNextChanges(oldText, newText)
	if err != nil {
		log.Fatalf("Error predicting changes: %v", err)
	}

	// Visualize the predictions on the new text
	if len(predictions) > 0 {
		fmt.Println("--- Predicted Changes Preview ---")
		visualizedText := copre.VisualizePredictions(newText, predictions)
		fmt.Println(visualizedText)
		fmt.Println("---------------------------------")
	} else {
		fmt.Println("No specific next changes predicted based on anchors.")
	}

	// Keep the detailed log for debugging if needed
	// fmt.Printf("Predicted next changes (raw): %+v\n", predictions)
}
