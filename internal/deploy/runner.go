// internal/deploy/runner.go
package deploy

import (
	"bufio"
	"fmt"
	"os/exec"
	"sync"
)

// Result is the outcome of a deployment run.
type Result struct {
	ExitCode int
	Err      error
}

// Run executes `terraform init` then `terraform apply -auto-approve` in workDir.
// Each line of stdout/stderr is sent to lines. The final Result is sent to done.
// Both channels are closed when Run completes.
// workDir must contain valid .tf files before Run is called.
func Run(workDir string, lines chan<- string, done chan<- Result) {
	go func() {
		defer close(lines)
		defer close(done)

		if err := runCmd(workDir, lines, "terraform", "init"); err != nil {
			done <- Result{ExitCode: 1, Err: fmt.Errorf("terraform init: %w", err)}
			return
		}
		if err := runCmd(workDir, lines, "terraform", "apply", "-auto-approve"); err != nil {
			done <- Result{ExitCode: 1, Err: fmt.Errorf("terraform apply: %w", err)}
			return
		}
		done <- Result{ExitCode: 0}
	}()
}

func runCmd(workDir string, lines chan<- string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	// wg ensures scanner goroutines finish before runCmd returns.
	// Without this, a goroutine could send to lines after close(lines) fires,
	// causing a panic.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			lines <- "ERR: " + sc.Text()
		}
	}()

	err = cmd.Wait()
	wg.Wait() // wait for scanners to finish draining pipes before returning
	return err
}
