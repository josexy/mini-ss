package ping

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func PingWithoutRoot(dst string, count int) (time.Duration, error) {
	s := fmt.Sprintf("ping %s -n %d", dst, count)
	args := strings.Split(s, " ")

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		return 0, errTimeout
	}

	data := buf.Bytes()
	index := bytes.LastIndex(data, []byte("="))
	if index == -1 {
		return 0, errTimeout
	}
	t := strings.TrimSpace(strings.Trim(string(data[index:]), "="))
	return time.ParseDuration(t)
}
