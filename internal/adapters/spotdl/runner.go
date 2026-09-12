package spotdl

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Runner struct {
	binary      string
	downloadDir string
	format      string
	extraArgs   []string
}

const (
	OutputFormat string = "/{artists}/{album}/{track-number} - {title}.{output-ext}"
)

func NewRunner(binary, downloadDir, format, extraArgsRaw string) *Runner {
	var extra []string
	if strings.TrimSpace(extraArgsRaw) != "" {
		extra = strings.Fields(extraArgsRaw)
	}
	return &Runner{
		binary:      binary,
		downloadDir: downloadDir,
		format:      format,
		extraArgs:   extra,
	}
}

func (r *Runner) Download(ctx context.Context, query string, onProgress func(line string)) error {
	outputTemplate := r.downloadDir + OutputFormat
	args := []string{"download", query, "--output", outputTemplate, "--format", r.format}
	args = append(args, r.extraArgs...)

	cmd := exec.CommandContext(ctx, r.binary, args...)
	cmd.Dir = r.downloadDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not attach stdout: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start spotdl: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		onProgress(line)
	}

	return cmd.Wait()
}
