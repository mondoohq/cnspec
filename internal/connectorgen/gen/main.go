// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Command gen regenerates the connector metadata artifact from an mql checkout.
//
// It is run by hand, not by the build:
//
//	make connectors/generate MQL=../mql
//
// cnspec has to compile with no mql source present, so the artifact is the
// contract and this command is how it is refreshed. With no checkout to read it
// refuses and changes nothing, which leaves the checked-in artifact -- the one
// every downstream test reads -- intact.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/internal/connectorgen"
)

func main() {
	mqlPath := flag.String("mql", "", "path to an mql checkout (required)")
	enterprisePath := flag.String("enterprise", "",
		"path to an mql-enterprise-providers checkout (optional; without it those connectors are absent from the artifact)")
	out := flag.String("out", "internal/connectors/connectors.json", "artifact to write")
	reportOnly := flag.Bool("report", false, "print the extraction report and write nothing")
	flag.Parse()

	if err := run(*mqlPath, *enterprisePath, *out, *reportOnly); err != nil {
		fmt.Fprintln(os.Stderr, "connectorgen:", err)
		os.Exit(1)
	}
}

func run(mqlPath, enterprisePath, out string, reportOnly bool) error {
	if mqlPath == "" {
		// Refusing, rather than writing an empty artifact: the checked-in file
		// is what every downstream test reads, and leaving it alone is the only
		// safe thing to do with no source to regenerate it from.
		fmt.Fprintln(os.Stderr,
			"connectorgen reads mql provider source and cannot run without a checkout of it.\n"+
				"  Clone one with `make prep/repos`, then run `make connectors/generate MQL=./mql`.\n"+
				"  The artifact is checked in, so not running this breaks nothing downstream.")
		return errors.New("no mql checkout given")
	}

	roots := []connectorgen.Root{{Name: "mql", Path: mqlPath}}
	if enterprisePath != "" {
		roots = append(roots, connectorgen.Root{Name: "mql-enterprise-providers", Path: enterprisePath})
	}

	art, err := connectorgen.Extract(roots)
	if err != nil {
		return err
	}

	// Anything the checkout no longer covers is kept rather than dropped, so a
	// regeneration cannot silently shrink the artifact. Done for -report too,
	// so the report describes the file that would be written rather than a
	// smaller one nobody will see.
	if err := connectorgen.CarryForward(art, out); err != nil {
		return err
	}

	// The report goes to stderr so `-report` can be read while the artifact
	// goes to a file, and so a redirect of stdout never swallows the gap list.
	connectorgen.WriteReport(os.Stderr, art)
	if reportOnly {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := connectorgen.WriteJSON(f, art); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "\nwrote %d connectors to %s\n", len(art.Connectors), out)
	return f.Close()
}
