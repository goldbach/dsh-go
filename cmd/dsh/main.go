package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/goldbach/dsh-go/internal/groups"
	"github.com/goldbach/dsh-go/internal/runner"
)

// multiFlag collects repeated flag values (e.g. -g web -g db).
type multiFlag []string

func (f *multiFlag) String() string  { return fmt.Sprint([]string(*f)) }
func (f *multiFlag) Set(v string) error { *f = append(*f, v); return nil }

func main() {
	var (
		groupNames multiFlag
		machines   multiFlag
		files      multiFlag
		sshArgs    multiFlag

		all        = flag.Bool("a", false, "use all hosts from ~/.dsh/machines.list")
		concurrent = flag.Bool("c", false, "run concurrently (concurrent-shell)")
		sequential = flag.Bool("w", false, "run sequentially, wait for each host (wait-shell)")
		hideNames  = flag.Bool("H", false, "hide machine names in output")
		_          = flag.Bool("M", false, "show machine names in output (default: on)")
		fanout     = flag.Int("F", 64, "max concurrent ssh connections")
	)

	flag.Var(&groupNames, "g", "group name; may be repeated")
	flag.Var(&machines, "m", "machine; may be repeated")
	flag.Var(&files, "f", "file of machine names; may be repeated")
	flag.Var(&sshArgs, "o", "extra ssh argument (passed before hostname); may be repeated")

	flag.Usage = usage
	flag.Parse()

	cmd := flag.Args()
	if len(cmd) == 0 {
		fmt.Fprintln(os.Stderr, "dsh: no command specified")
		usage()
		os.Exit(1)
	}

	hosts, err := resolveHosts(groupNames, machines, files, *all)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsh: %v\n", err)
		os.Exit(1)
	}
	if len(hosts) == 0 {
		fmt.Fprintln(os.Stderr, "dsh: no hosts specified (use -g, -m, -f, or -a)")
		os.Exit(1)
	}

	if *concurrent && *sequential {
		fmt.Fprintln(os.Stderr, "dsh: -c and -w are mutually exclusive")
		os.Exit(1)
	}

	show := !*hideNames

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	results := runner.Run(ctx, hosts, cmd, runner.Options{
		SSHArgs:      []string(sshArgs),
		ShowHostname: show,
		Fanout:       *fanout,
		Sequential:   *sequential,
	})

	exitCode := 0
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "dsh: %s: %v\n", r.Host, r.Err)
			if exitCode == 0 {
				exitCode = 1
			}
		} else if r.ExitCode != 0 && exitCode == 0 {
			exitCode = r.ExitCode
		}
	}
	os.Exit(exitCode)
}

func resolveHosts(groupNames, machines, files multiFlag, all bool) ([]string, error) {
	seen := make(map[string]bool)
	var hosts []string

	add := func(h string) {
		if !seen[h] {
			seen[h] = true
			hosts = append(hosts, h)
		}
	}

	if len(groupNames) > 0 {
		loaded, err := groups.Load([]string(groupNames))
		if err != nil {
			return nil, err
		}
		for _, h := range loaded {
			add(h)
		}
	}

	if len(files) > 0 {
		loaded, err := groups.LoadFiles([]string(files))
		if err != nil {
			return nil, err
		}
		for _, h := range loaded {
			add(h)
		}
	}

	if all {
		path, err := groups.MachinesListPath()
		if err != nil {
			return nil, err
		}
		loaded, err := groups.LoadFiles([]string{path})
		if err != nil {
			return nil, fmt.Errorf("-a: %w", err)
		}
		for _, h := range loaded {
			add(h)
		}
	}

	for _, m := range machines {
		add(m)
	}

	return hosts, nil
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: dsh [options] [--] <command> [args...]

options:
  -g <group>    group name from ~/.dsh/group/ (repeatable)
  -m <host>     individual host (repeatable)
  -f <file>     file containing host names (repeatable)
  -a            use all hosts from ~/.dsh/machines.list
  -c            concurrent execution
  -w            sequential execution (wait-shell)
  -F <n>        max concurrent connections (default 64, implies -c)
  -H            hide machine names in output (machine names shown by default)
  -M            no-op; machine names are shown by default
  -o <arg>      extra ssh argument, e.g. -o "-p 2222" (repeatable)`)
}
