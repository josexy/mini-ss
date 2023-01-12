//go:build !windows

package ping

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func PingWithoutRoot(dst string, count int) (time.Duration, error) {
	s := fmt.Sprintf("ping %s -c %d", dst, count)
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

	br := bufio.NewReader(&buf)
	cnt := 0
	succCnt := 0

	var rtts []time.Duration
	for {
		line, _, err := br.ReadLine()
		if err != nil {
			break
		}

		// timeout
		if bytes.Contains(line, []byte("timeout")) {
			continue
		}

		// success
		if cnt > count {
			break
		}

		cnt++
		str := strings.TrimSpace(string(line))

		var index int
		if index = strings.Index(str, "time="); index == -1 {
			continue
		}
		rtt, err := time.ParseDuration(strings.ReplaceAll(str[index+5:], " ", ""))
		if err != nil {
			continue
		}
		rtts = append(rtts, rtt)
		succCnt++
	}
	if succCnt > 0 {
		return calcAvgRtt(rtts)
	}
	return 0, errTimeout
}
