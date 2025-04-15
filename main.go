package main

import (
	"fmt"
	"log"

	"github.com/jsnanigans/copre/copre"
)

func main() {
	oldText := `line 1
line 2
line 3`
	newText := `line 1
line two
line 3`

	prediction, err := copre.PredictNextChanges(oldText, newText)
	if err != nil {
		log.Fatalf("Error predicting changes: %v", err)
	}

	fmt.Printf("Predicted next change: %s\n", prediction)
}
