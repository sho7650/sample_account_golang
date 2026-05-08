package cli

import (
	"fmt"
	"strconv"
	"strings"

	"sample_account/internal/field"
)

// parseJobs validates a jobs argument: must be a non-negative integer.
func parseJobs(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid jobs value %q: not an integer", s)
	}
	if n < 0 {
		return 0, fmt.Errorf("invalid jobs value %d: must be >= 0", n)
	}
	return n, nil
}

// DefaultRowCount mirrors the C++ DEFAULT_ROW_COUNT constant.
const DefaultRowCount = 100

// ParseResult is the shape returned by Parse.
type ParseResult struct {
	// Selected lists the chosen fields in argv order. With no flags this
	// defaults to a single IDField (preserves historical behavior).
	Selected []field.Field
	Count    int
	Help     bool
	// Jobs controls runner concurrency: 0 = auto (NumCPU above
	// serialThreshold), 1 = forced serial, N>1 = N workers.
	Jobs int
}

// Parse walks argv (including argv[0] as the program name) and resolves
// every short/long flag against the registry. Returns a ParseResult or
// the first parse error.
//
// Conventions (getopt-compatible):
//   - "-h" / "--help" → Help=true, parsing stops
//   - "-x"            → single short flag
//   - "-xyz"          → cluster: x, y, z processed in order
//   - "--name"        → long flag (alias-aware via Registry)
//   - trailing token  → COUNT (positive integer); non-numeric tokens are
//                       silently ignored to match the C++ atoi behavior.
func Parse(argv []string, reg *field.Registry) (ParseResult, error) {
	args := ParseResult{Count: DefaultRowCount}
	if len(argv) <= 1 {
		args.Selected = defaultSelection(reg)
		return args, nil
	}

	for i := 1; i < len(argv); i++ {
		tok := argv[i]
		switch {
		case tok == "-h" || tok == "--help":
			args.Help = true
			return args, nil
		case tok == "--jobs" || tok == "-j":
			// Argument-taking flag: consume the next token.
			if i+1 >= len(argv) {
				return args, fmt.Errorf("option %q requires a non-negative integer argument", tok)
			}
			n, err := parseJobs(argv[i+1])
			if err != nil {
				return args, fmt.Errorf("option %q: %w", tok, err)
			}
			args.Jobs = n
			i++
		case strings.HasPrefix(tok, "--jobs="):
			n, err := parseJobs(strings.TrimPrefix(tok, "--jobs="))
			if err != nil {
				return args, fmt.Errorf("option \"--jobs\": %w", err)
			}
			args.Jobs = n
		case strings.HasPrefix(tok, "-j") && len(tok) > 2:
			// Short attached form: -j2, -j=2.
			val := tok[2:]
			if val[0] == '=' {
				val = val[1:]
			}
			n, err := parseJobs(val)
			if err != nil {
				return args, fmt.Errorf("option \"-j\": %w", err)
			}
			args.Jobs = n
		case len(tok) > 2 && tok[:2] == "--":
			name := tok[2:]
			f := reg.FindLong(name)
			if f == nil {
				return args, fmt.Errorf("unrecognized option '--%s'", name)
			}
			args.Selected = append(args.Selected, f)
		case len(tok) >= 2 && tok[0] == '-':
			for j := 1; j < len(tok); j++ {
				c := tok[j]
				f := reg.FindShort(c)
				if f == nil {
					return args, fmt.Errorf("unrecognized option '-%c'", c)
				}
				args.Selected = append(args.Selected, f)
			}
		default:
			if n, err := strconv.Atoi(tok); err == nil && n > 0 {
				args.Count = n
			}
			// non-numeric trailing arg silently ignored, matching atoi semantics
		}
	}

	if len(args.Selected) == 0 {
		args.Selected = defaultSelection(reg)
	}
	return args, nil
}

func defaultSelection(reg *field.Registry) []field.Field {
	if f := reg.FindShort('i'); f != nil {
		return []field.Field{f}
	}
	return nil
}
