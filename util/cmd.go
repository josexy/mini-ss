package util

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func ExeCmd(cmd string) error {
	args := strings.Split(cmd, " ")
	if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
		return fmt.Errorf("%s: %v", cmd, err)
	}

	return nil
}

// ExeShell execute shell by "sh -c ..." on Linux/macOS
func ExeShell(shell string) (string, error) {
	cmd := exec.Command("sh", "-c", shell)
	out := bytes.Buffer{}
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}
