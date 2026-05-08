package main

import (
	"fmt"
	"os"

	"sample_account/internal/cli"
	"sample_account/internal/field"
	"sample_account/internal/gen"
	"sample_account/internal/repo"
	"sample_account/internal/runner"
)

func main() {
	if err := run(os.Args, os.Stdout, os.Stderr); err != nil {
		os.Exit(err.Code)
	}
}

type exitErr struct {
	Code int
}

func (e *exitErr) Error() string { return "" }

func run(argv []string, stdout, stderr *os.File) *exitErr {
	prog := argv[0]
	reg := field.DefaultRegistry()

	args, err := cli.Parse(argv, reg)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %s\n", prog, err)
		fmt.Fprintf(stderr, "Try '%s --help' for usage.\n", prog)
		return &exitErr{Code: 2}
	}
	if args.Help {
		cli.PrintHelp(stdout, prog, reg)
		return nil
	}

	persons, err := repo.DefaultPersons()
	if err != nil {
		fmt.Fprintf(stderr, "%s: %s\n", prog, err)
		return &exitErr{Code: 1}
	}
	prefRepo, err := repo.DefaultPrefectures()
	if err != nil {
		fmt.Fprintf(stderr, "%s: %s\n", prog, err)
		return &exitErr{Code: 1}
	}
	ageRepo, err := repo.DefaultAges()
	if err != nil {
		fmt.Fprintf(stderr, "%s: %s\n", prog, err)
		return &exitErr{Code: 1}
	}

	deps := runner.Deps{
		Persons:     persons,
		Prefectures: prefRepo,
		Ages:        ageRepo,
		AddressGen:  gen.NewAddressGen(prefRepo),
	}

	if err := runner.RunWithJobs(stdout, args.Count, args.Selected, deps, gen.MasterSeed(), args.Jobs); err != nil {
		fmt.Fprintf(stderr, "%s: %s\n", prog, err)
		return &exitErr{Code: 1}
	}
	return nil
}
