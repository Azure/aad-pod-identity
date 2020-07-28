//created by @phillipgibson

package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	// Only get the slice of args passed to the go file
	cmdArgs := os.Args[1:]

	fmt.Println("Validating script arguments...")

	if len(cmdArgs) != 2 {
		fmt.Println("The wrong number of arguments were passed to the program.")
		fmt.Println("Please check your arguments and re-run the program again.")
		fmt.Println("Exiting program.")
		os.Exit(1)
	} else if strings.ToLower(cmdArgs[0]) == "deploy" {
		fmt.Println("Script is to deploy AAD Pod Identiy demo.")
	} else if strings.ToLower(cmdArgs[0]) == "clean" {
		fmt.Println("Script is to clean up AAD Pod Idenity demo.")
		fmt.Println("Starting AAD Pod Identity demo cleanup...")
	} else {
		fmt.Println("No expected script action was detected for the first parameter needed.")
		fmt.Println("The script action parameter is expecting either \"deploy\" or a \"clean\" action.")
		fmt.Println("Please ensure you read the documentation on how to use the AAD Pod Identity demo script.")
		fmt.Println("Exiting script")
		os.Exit(1)
	}
}
