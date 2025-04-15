package main

import (
	"fmt"
	"log"

	"github.com/jsnanigans/copre/copre"
)

func main() {
	oldText := `line one-smile
line two-smile
line 3-smile`
	newText := `line one-smile
line two
line 3-smile`

	prediction, err := copre.PredictNextChanges(oldText, newText)
	if err != nil {
		log.Fatalf("Error predicting changes: %v", err)
	}

	fmt.Printf("Predicted next change: %s\n", prediction)
}
