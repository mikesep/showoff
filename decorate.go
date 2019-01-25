package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"mvdan.cc/sh/syntax"
)

func decorateScript(r io.Reader, w io.Writer) error {
	parser := syntax.NewParser(syntax.KeepComments)

	var stmtErr error
	parseErr := parser.Stmts(r, func(stmt *syntax.Stmt) bool {
		stmtErr = decorateStmt(stmt, w)
		return stmtErr == nil
	})

	if stmtErr == nil && parseErr != nil {
		return parseErr
	}

	return stmtErr
}

func decorateStmt(stmt *syntax.Stmt, w io.Writer) error {
	handleHashplings(w, stmt)

	writeSpacerLineIfNeeded(w, stmt)

	writeLineComment(w, stmt)

	pauseBeforePrinting, pauseBeforeRunning := detectPauses(stmt)

	if pauseBeforePrinting {
		mustFprintf(w, "read -r -s -n 1    # pause before printing\n")
	}

	printer := syntax.NewPrinter()

	{
		eof := fmt.Sprintf("EOF_%d", rand.Int())

		mustFprintf(w, "cat <<'%s' | pv -qL 20\n", eof)

		stmtBuf := bytes.Buffer{}
		if err := printer.Print(&stmtBuf, stmt); err != nil {
			return err
		}
		if err := prefixLinesCopy("> ", &stmtBuf, w); err != nil {
			return err
		}

		mustFprintf(w, "%s\n", eof)
	}

	if pauseBeforeRunning {
		mustFprintf(w, "read -r -s -n 1    # pause before running\n")
	}

	if err := printer.Print(w, stmt); err != nil {
		return err
	}

	mustFprintf(w, "\n")
	mustFprintf(w, "sleep 0.8    # wait between statements\n")

	return nil
}

// Special treatment for the #! line(s)
//
// To use this program as a wrapping interpreter for a script, there should be two #! lines:
//   #!/usr/bin/env this_program
//   #!/usr/bin/env shell_interpreter
func handleHashplings(w io.Writer, stmt *syntax.Stmt) {
	if len(stmt.Comments) == 0 || stmt.Comments[0].Hash.Line() != 1 {
		return
	}

	for len(stmt.Comments) > 0 {
		comment := stmt.Comments[0]
		if comment.Text[0] != '!' {
			return
		}

		// If the hashpling line invokes this program, we need to remove it.
		fields := strings.Fields(comment.Text[1:])
		if len(fields) == 0 {
			return
		}
		if filepath.Base(fields[len(fields)-1]) == filepath.Base(os.Args[0]) {
			stmt.Comments = stmt.Comments[1:]
			continue
		}

		// Pass any other #! line through
		mustFprintf(w, "#%s\n", comment.Text)
		stmt.Comments = stmt.Comments[1:]
	}
}

func writeSpacerLineIfNeeded(w io.Writer, stmt *syntax.Stmt) {
	if len(stmt.Comments) > 0 {
		if stmt.Comments[0].Hash.Line() != 1 {
			mustFprintf(w, "\n")
		}
	} else {
		if stmt.Pos().Line() != 1 {
			mustFprintf(w, "\n")
		}
	}
}

// Write a comment showing the line in the original script for this statement.
func writeLineComment(w io.Writer, stmt *syntax.Stmt) {
	// Comments are attached to the statement below or the statement to the left
	// So, if there's any comment, it's either before or on the same line as the command.

	if len(stmt.Comments) > 0 {
		mustFprintf(w, "# line %d\n", stmt.Comments[0].Hash.Line())
	} else {
		mustFprintf(w, "# line %d\n", stmt.Pos().Line())
	}
}

// If the pause comment is before the command's line, like
//   # pause
//   command args...
// then we pause before printing (and running) the command.
//
// If the pause comment is on the same line as the start of the command, like
//   command args... # pause
// then we print the comment and pause before running it.
func detectPauses(stmt *syntax.Stmt) (pauseBeforePrinting, pauseBeforeRunning bool) {
	pattern := regexp.MustCompile("(?i)^[[:space:]]*pause[[:space:]]*$")

	for _, comm := range stmt.Comments {
		if pattern.MatchString(comm.Text) {
			if comm.Hash.Line() < stmt.Pos().Line() {
				pauseBeforePrinting = true
			} else {
				pauseBeforeRunning = true
			}
		}
	}

	return
}

func prefixLinesCopy(prefix string, r io.Reader, w io.Writer) error {
	s := bufio.NewScanner(r)
	for s.Scan() {
		if _, err := fmt.Fprintf(w, "%s%s\n", prefix, s.Text()); err != nil {
			return err
		}
	}
	return s.Err()
}
