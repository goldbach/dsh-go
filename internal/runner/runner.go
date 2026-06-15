package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// Options controls execution behaviour.
type Options struct {
	SSHArgs       []string // extra arguments passed to ssh before the hostname
	ShowHostname  bool     // prefix each output line with "hostname: "
	HostnameWidth int      // pad hostname prefix to this width (0 = no padding)
	Fanout        int      // max concurrent ssh connections; 0 = unlimited
	Sequential    bool     // run one host at a time (default: concurrent)
}

// Result holds the outcome for a single host.
type Result struct {
	Host     string
	ExitCode int
	Err      error // non-nil when ssh itself could not be started
}

// Run executes cmd on every host and returns one Result per host.
func Run(ctx context.Context, hosts []string, cmd []string, opts Options) []Result {
	if opts.ShowHostname && opts.HostnameWidth == 0 {
		for _, h := range hosts {
			if len(h) > opts.HostnameWidth {
				opts.HostnameWidth = len(h)
			}
		}
	}
	if opts.Sequential {
		return runSequential(ctx, hosts, cmd, opts)
	}
	return runConcurrent(ctx, hosts, cmd, opts)
}

// runSequential executes hosts one at a time, passing stdin through.
func runSequential(ctx context.Context, hosts []string, cmd []string, opts Options) []Result {
	results := make([]Result, len(hosts))
	for i, host := range hosts {
		if opts.ShowHostname {
			var mu sync.Mutex
			results[i] = execPiped(ctx, host, cmd, opts, &mu)
		} else {
			results[i] = execDirect(ctx, host, cmd, opts)
		}
	}
	return results
}

// runConcurrent fans out to all hosts up to opts.Fanout simultaneous connections.
func runConcurrent(ctx context.Context, hosts []string, cmd []string, opts Options) []Result {
	fanout := opts.Fanout
	if fanout <= 0 {
		fanout = len(hosts)
	}

	sem := make(chan struct{}, fanout)
	results := make([]Result, len(hosts))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, host := range hosts {
		wg.Add(1)
		go func(i int, host string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = execPiped(ctx, host, cmd, opts, &mu)
		}(i, host)
	}
	wg.Wait()
	return results
}

// execDirect runs ssh with stdin/stdout/stderr wired to the process directly.
// Used in sequential mode without hostname prefixing.
func execDirect(ctx context.Context, host string, cmd []string, opts Options) Result {
	c := exec.CommandContext(ctx, "ssh", sshArgs(opts.SSHArgs, host, cmd)...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return resultOf(host, c.Run())
}

// execPiped runs ssh with piped output, streaming each line with optional hostname prefix.
// mu serialises writes to stdout/stderr across goroutines.
func execPiped(ctx context.Context, host string, cmd []string, opts Options, mu *sync.Mutex) Result {
	c := exec.CommandContext(ctx, "ssh", sshArgs(opts.SSHArgs, host, cmd)...)

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	c.Stdout = stdoutW
	c.Stderr = stderrW

	prefix := ""
	if opts.ShowHostname {
		if opts.HostnameWidth > 0 {
			prefix = fmt.Sprintf("%-*s: ", opts.HostnameWidth, host)
		} else {
			prefix = host + ": "
		}
	}

	if err := c.Start(); err != nil {
		stdoutW.Close()
		stderrW.Close()
		stdoutR.Close()
		stderrR.Close()
		return Result{Host: host, ExitCode: -1, Err: fmt.Errorf("start ssh: %w", err)}
	}

	var scanners sync.WaitGroup
	scanners.Add(2)

	go func() {
		defer scanners.Done()
		streamLines(stdoutR, prefix, os.Stdout, mu)
	}()
	go func() {
		defer scanners.Done()
		streamLines(stderrR, prefix, os.Stderr, mu)
	}()

	runErr := c.Wait()
	stdoutW.CloseWithError(runErr)
	stderrW.CloseWithError(runErr)
	scanners.Wait()

	return resultOf(host, runErr)
}

func streamLines(r io.Reader, prefix string, w io.Writer, mu *sync.Mutex) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		mu.Lock()
		fmt.Fprintf(w, "%s%s\n", prefix, sc.Text())
		mu.Unlock()
	}
}

func sshArgs(extra []string, host string, cmd []string) []string {
	args := make([]string, 0, len(extra)+1+len(cmd))
	args = append(args, extra...)
	args = append(args, host)
	args = append(args, cmd...)
	return args
}

func resultOf(host string, err error) Result {
	if err == nil {
		return Result{Host: host}
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return Result{Host: host, ExitCode: exitErr.ExitCode()}
	}
	return Result{Host: host, ExitCode: -1, Err: err}
}
