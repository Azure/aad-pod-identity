package util

import (
	"fmt"
	"os/exec"
	"strings"
)

// PrintCommand prints a command string
func PrintCommand(cmd *exec.Cmd) {
	fmt.Printf("$ %s\n", strings.Join(cmd.Args, " "))
}
