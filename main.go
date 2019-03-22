package main

import (
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	flags "github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
)

type options struct {
	Output flags.Filename `short:"o" long:"output" value-name:"FILE" description:"Instead of running the script, output to FILE."`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	opts := options{}
	flagParser := flags.NewParser(&opts, flags.Default)
	args, err := flagParser.Parse()
	if err != nil {
		if flags.WroteHelp(err) {
			return
		}
		os.Exit(1)
	}

	if len(args) == 0 {
		mustFprintf(os.Stderr, "ERROR: no input file given\n\n")
		flagParser.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	if err := mainApp(context.Background(), opts, args); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(ws.ExitStatus())
			}
		}

		mustFprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}

func mainApp(ctx context.Context, opts options, args []string) error {
	input, err := setInputStream(args)
	if err != nil {
		return errors.Wrap(err, "chooseInputStream")
	}

	defer func() {
		if closeErr := input.Close(); closeErr != nil {
			mustFprintf(os.Stderr, "WARN: input.Close: %v\n", closeErr)
		}
	}()

	output, err := setOutputStream(opts)
	if err != nil {
		return errors.Wrap(err, "chooseOutputStream")
	}

	err = decorateScript(input, output)

	if closeErr := output.Close(); err != nil {
		mustFprintf(os.Stderr, "WARN: output.Close: %v\n", closeErr)
	}

	if err != nil {
		return err
	}

	if opts.Output != "" {
		return nil
	}

	scriptToRun := output.(*os.File).Name()

	defer func() {
		if rmErr := os.Remove(scriptToRun); rmErr != nil {
			mustFprintf(os.Stderr, "WARN: os.Remove: %v\n", rmErr)
		}
	}()

	scriptPath, err := filepath.Abs(scriptToRun)
	if err != nil {
		return err
	}

	return runScript(ctx, scriptPath)
}

func setInputStream(args []string) (io.ReadCloser, error) {
	switch filename := args[len(args)-1]; filename {
	case "-":
		return os.Stdin, nil
	default:
		file, err := os.Open(filename)
		if err != nil {
			return nil, errors.Wrap(err, "bad input file")
		}
		return file, nil
	}
}

func setOutputStream(opts options) (io.WriteCloser, error) {
	switch filename := opts.Output; filename {

	case "-":
		return os.Stdout, nil

	case "":
		file, err := ioutil.TempFile("", "demoer.")
		if err != nil {
			return nil, errors.Wrap(err, "tempfile create")
		}
		if err := file.Chmod(0700); err != nil {
			return nil, errors.Wrap(err, "tempfile chmod")
		}
		return file, nil

	default:
		file, err := os.Create(string(filename))
		if err != nil {
			return nil, errors.Wrap(err, "output file")
		}
		return file, nil
	}
}
