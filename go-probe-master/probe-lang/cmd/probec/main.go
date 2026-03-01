// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.

// Command probec is the PROBE language compiler.
//
// Usage:
//
//	probec [flags] <source.probe>
//
// Flags:
//
//	-o <output>    Output file (default: stdout)
//	-emit <stage>  Emit intermediate output: tokens, ast, ir, bytecode (default: bytecode)
//	-optimize      Enable optimization passes (default: true)
//	-verify        Run bytecode verifier (default: true)
//	-version       Print version and exit
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/probechain/go-probe/probe-lang/lang/lexer"
)

const version = "0.1.0"

func main() {
	var (
		output   = flag.String("o", "", "Output file (default: stdout)")
		emit     = flag.String("emit", "bytecode", "Emit stage: tokens, ast, ir, bytecode")
		optimize = flag.Bool("optimize", true, "Enable optimization passes")
		verify   = flag.Bool("verify", true, "Run bytecode verifier")
		ver      = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *ver {
		fmt.Printf("probec %s\n", version)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: probec [flags] <source.probe>")
		os.Exit(1)
	}

	filename := flag.Arg(0)
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Suppress unused variable warnings for flags we'll use in later phases.
	_ = output
	_ = optimize
	_ = verify

	switch *emit {
	case "tokens":
		emitTokens(filename, string(source))
	case "ast", "ir", "bytecode":
		fmt.Fprintf(os.Stderr, "emit stage %q not yet implemented\n", *emit)
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "unknown emit stage: %s\n", *emit)
		os.Exit(1)
	}
}

func emitTokens(filename, source string) {
	l := lexer.New(filename, source)
	tokens := l.Tokenize()
	for _, tok := range tokens {
		fmt.Printf("%s\t%s\t%q\n", tok.Pos, tok.Type, tok.Literal)
	}
}
