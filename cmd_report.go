package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func runReport(args []string) error {
	if len(args) < 1 {
		printReportUsage()
		return fmt.Errorf("no report subcommand provided")
	}
	if isHelpFlag(args[0]) {
		printReportUsage()
		return flag.ErrHelp
	}

	commands := map[string]func([]string) error{
		"timesheet": runReportTimesheet,
	}

	handler, ok := commands[args[0]]
	if !ok {
		return fmt.Errorf("unknown report subcommand '%s'", args[0])
	}
	return handler(args[1:])
}

func runReportTimesheet(args []string) error {
	timesheetCommand := flag.NewFlagSet("report timesheet", flag.ContinueOnError)
	timesheetFile := timesheetCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	timesheetFrom := timesheetCommand.String("from", "", "Start date (YYYY-MM-DD, defaults to Monday of current week)")
	timesheetTo := timesheetCommand.String("to", "", "End date (YYYY-MM-DD, defaults to Sunday of current week)")
	timesheetFormat := timesheetCommand.String("format", "table", "Output format: table or csv")
	timesheetDecimal := timesheetCommand.Bool("decimal", false, "Display hours in decimal format (e.g., 1.50)")
	timesheetAll := timesheetCommand.Bool("all", false, "Use the full date range of all events")
	timesheetHideEmpty := timesheetCommand.Bool("hide-empty", false, "Hide days with no time entries")
	configureFlagSet(timesheetCommand, "Generate a timesheet showing time logged per task per day. Defaults to the current week (Monday–Sunday).", "  worklog report timesheet\n  worklog report timesheet -from 2023-08-01 -to 2023-08-15\n  worklog report timesheet -format csv -decimal\n  worklog report timesheet -all -hide-empty")
	if err := timesheetCommand.Parse(args); err != nil {
		return err
	}

	worklog, err := loadWorklog(*timesheetFile)
	if err != nil {
		return err
	}

	isDefaultRange := true
	from, to := currentWeekRange(time.Now())

	if *timesheetAll {
		isDefaultRange = false
		min, max := eventDateRange(worklog)
		if !min.IsZero() {
			from = min
		}
		if !max.IsZero() {
			to = max
		}
	}
	if *timesheetFrom != "" {
		isDefaultRange = false
		parsed, err := parseDateFlag(*timesheetFrom)
		if err != nil {
			return err
		}
		from = parsed
	}
	if *timesheetTo != "" {
		isDefaultRange = false
		parsed, err := parseDateFlag(*timesheetTo)
		if err != nil {
			return err
		}
		to = parsed
	}
	if from.After(to) {
		return fmt.Errorf("from date must not be after to date")
	}

	ts := generateTimesheet(worklog, from, to)
	if *timesheetHideEmpty {
		ts = hideEmptyColumns(ts)
	}

	if len(ts.rows) == 0 {
		fmt.Println("No time entries in the selected date range.")
		if isDefaultRange {
			fmt.Println("Use --all to see all data, or specify --from and --to.")
		}
		return nil
	}

	switch *timesheetFormat {
	case "table":
		printTimesheetTable(os.Stdout, ts, *timesheetDecimal)
	case "csv":
		printTimesheetCSV(os.Stdout, ts, *timesheetDecimal)
	default:
		return fmt.Errorf("unknown format '%s', use 'table' or 'csv'", *timesheetFormat)
	}
	return nil
}
