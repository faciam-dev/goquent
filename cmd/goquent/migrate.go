package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/faciam-dev/goquent/orm"
	"github.com/faciam-dev/goquent/orm/migration"
	"github.com/faciam-dev/goquent/orm/review"
)

func runMigrate(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printMigrateUsage(stderr)
		return 2
	}

	switch args[0] {
	case "plan":
		return runMigrateCommand("plan", args[1:], stdout, stderr)
	case "dry-run":
		return runMigrateCommand("dry-run", args[1:], stdout, stderr)
	case "apply":
		return runMigrateCommand("apply", args[1:], stdout, stderr)
	case "-h", "--help", "help":
		printMigrateUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown migrate command %q\n", args[0])
		printMigrateUsage(stderr)
		return 2
	}
}

func runMigrateCommand(mode string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("goquent migrate "+mode, flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "pretty", "output format: pretty, json")
	failOn := fs.String("fail-on", "", "optional risk threshold that returns exit code 1")
	approve := fs.String("approve", "", "approval reason for applying risky migrations")
	driverName := fs.String("driver", "", "database driver for apply: mysql or postgres")
	dsn := fs.String("dsn", "", "database DSN for apply")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: goquent migrate %s [flags] <migration.sql ...>\n", mode)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	outputFormat := strings.ToLower(strings.TrimSpace(*format))
	switch outputFormat {
	case "", "pretty", "text", "json":
	default:
		fmt.Fprintf(stderr, "unknown migrate format %q\n", *format)
		return 2
	}

	var thresholdSet bool
	var threshold orm.RiskLevel
	if strings.TrimSpace(*failOn) != "" {
		parsed, err := review.ParseRiskLevel(*failOn)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		threshold = parsed
		thresholdSet = true
	}

	sqlText, err := readMigrationSQL(fs.Args())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	migrator := migration.New(sqlText)
	if strings.TrimSpace(*approve) != "" {
		migrator.RequireApproval(*approve)
	}

	ctx := context.Background()
	plan, err := migrator.Plan(ctx)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	if mode == "dry-run" || mode == "apply" {
		if err := migration.EnsureExecutable(plan); err != nil {
			_ = writeMigrationOutput(stdout, outputFormat, plan)
			fmt.Fprintln(stderr, err)
			return 1
		}
	}

	if mode == "apply" {
		if strings.TrimSpace(*driverName) == "" || strings.TrimSpace(*dsn) == "" {
			_ = writeMigrationOutput(stdout, outputFormat, plan)
			fmt.Fprintln(stderr, "goquent migrate apply requires --driver and --dsn")
			return 2
		}
		db, err := orm.OpenWithDriver(*driverName, *dsn)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		defer db.Close()
		plan, err = migrator.Apply(ctx, db)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	}

	if err := writeMigrationOutput(stdout, outputFormat, plan); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if thresholdSet && compareMigrationRisk(plan.RiskLevel, threshold) >= 0 {
		return 1
	}
	return 0
}

func readMigrationSQL(paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("goquent migrate requires at least one SQL file")
	}
	var b strings.Builder
	for _, path := range paths {
		var data []byte
		var err error
		if path == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			return "", err
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.Write(data)
	}
	return b.String(), nil
}

func writeMigrationOutput(w io.Writer, format string, plan *migration.MigrationPlan) error {
	switch format {
	case "", "pretty", "text":
		return migration.WritePretty(w, plan)
	case "json":
		return migration.WriteJSON(w, plan)
	default:
		return fmt.Errorf("unknown migrate format %q", format)
	}
}

func compareMigrationRisk(a, b orm.RiskLevel) int {
	return migrationRiskRank(a) - migrationRiskRank(b)
}

func migrationRiskRank(level orm.RiskLevel) int {
	switch level {
	case orm.RiskLow, "":
		return 0
	case orm.RiskMedium:
		return 1
	case orm.RiskHigh:
		return 2
	case orm.RiskDestructive:
		return 3
	case orm.RiskBlocked:
		return 4
	default:
		return 0
	}
}

func printMigrateUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: goquent migrate <command>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  plan      print a migration plan")
	fmt.Fprintln(w, "  dry-run   validate whether a migration can be applied")
	fmt.Fprintln(w, "  apply     apply migration SQL after approval checks")
}
