package main

import (
	"encoding/json"
	"fmt"

	"github.com/Octo-sh/Octo-CLI/internal/intelligence"
	"github.com/Octo-sh/Octo-CLI/internal/ui"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:     "verify [path]",
	Aliases: []string{"v", "check"},
	Short:   "Verify expected runtime state and lockfile integrity",
	Long: `The verify command evaluates deterministic runtime health checks (listening TCP
ports, HTTP endpoints, container health) and validates the integrity of .octo.lock strategies.

In CI workflows, 'octo verify' serves as an automated smoke gate. Use '--json' for
machine-readable diagnostics or '--strict' to enforce verified lockfile presence.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) == 1 {
			path = args[0]
		}

		model, err := intelligence.Analyze(path)
		if err != nil {
			return err
		}

		plan, err := (intelligence.DeterministicPlanner{}).Plan(cmd.Context(), model)
		if err != nil {
			return err
		}

		checks := intelligence.BuildVerificationChecks(model, plan)
		results, verifyErr := intelligence.Verify(cmd.Context(), checks)

		lock, _ := intelligence.LoadOctoLock(path)
		lockPresent := len(lock.Strategies) > 0
		var strategyStatus intelligence.VerifiedStrategyApplication
		if lockPresent {
			strategyStatus, _ = intelligence.ApplyVerifiedStrategies(path, &model, lock)
		}

		strict, err := cmd.Flags().GetBool("strict")
		if err != nil {
			return err
		}

		var strictErr error
		if strict {
			if !lockPresent {
				strictErr = fmt.Errorf("strict verification failed: no cached strategies found in .octo.lock")
			} else if len(strategyStatus.Invalidated) > 0 {
				strictErr = fmt.Errorf("strict verification failed: %d strategy(s) invalidated by repository changes", len(strategyStatus.Invalidated))
			}
		}

		jsonOutput, err := cmd.Flags().GetBool("json")
		if err != nil {
			return err
		}

		outWriter := cmd.OutOrStdout()

		if jsonOutput {
			effectiveSuccess := verifyErr == nil && strictErr == nil
			report := map[string]interface{}{
				"project_name":     model.Name,
				"success":          effectiveSuccess,
				"strict":           strict,
				"checks":           results,
				"lock_present":     lockPresent,
				"lock_entries":     len(lock.Strategies),
				"lock_reused":      len(strategyStatus.Reused),
				"lock_invalidated": len(strategyStatus.Invalidated),
			}
			if verifyErr != nil {
				report["error"] = verifyErr.Error()
				diag := intelligence.DiagnoseExecutionFailure(verifyErr.Error(), verifyErr)
				report["error_code"] = diag.Code
				report["root_cause"] = diag.RootCause
				report["suggested_fix"] = diag.SuggestedFix
			} else if strictErr != nil {
				report["error"] = strictErr.Error()
				report["error_code"] = "STRICT_LOCK_INVALID"
				report["root_cause"] = strictErr.Error()
				report["suggested_fix"] = "Run 'octo run' locally to verify execution candidates and update .octo.lock."
			}

			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(outWriter, string(data))
			if verifyErr != nil {
				return verifyErr
			}
			if strictErr != nil {
				return strictErr
			}
			return nil
		}

		fmt.Fprintf(outWriter, "Verification report for %s:\n\n", model.Name)
		if len(checks) == 0 {
			fmt.Fprintln(outWriter, ui.Muted.Render("No active runtime verification checks required for this project."))
		} else {
			for _, res := range results {
				if res.Passed {
					fmt.Fprintln(outWriter, ui.SuccessLine(fmt.Sprintf("%s: passed (%s)", res.CheckID, res.Reason)))
				} else {
					fmt.Fprintln(outWriter, ui.ErrorLine(fmt.Sprintf("%s: failed (%s)", res.CheckID, res.Reason)))
					if res.RootCause != "" {
						fmt.Fprintf(outWriter, "    Root Cause: %s\n", res.RootCause)
					}
					if res.SuggestedFix != "" {
						fmt.Fprintf(outWriter, "    Suggested Fix: %s\n", ui.InfoLine(res.SuggestedFix))
					}
				}
			}
		}

		if lockPresent {
			fmt.Fprintf(outWriter, "\n.octo.lock: %d verified candidate strategy(s) cached (%d active, %d invalidated).\n",
				len(lock.Strategies), len(strategyStatus.Reused), len(strategyStatus.Invalidated))
			for _, s := range strategyStatus.Reused {
				fmt.Fprintf(outWriter, "  • %s: %s (fingerprint verified)\n", s.Component, s.Command)
			}
			for _, s := range strategyStatus.Invalidated {
				fmt.Fprintf(outWriter, "  ! %s: %s (invalidated by repository edits)\n", s.Component, s.Command)
			}
		} else {
			fmt.Fprintln(outWriter, "\n.octo.lock: no cached strategies found.")
		}

		if verifyErr != nil {
			fmt.Fprintf(outWriter, "\n%s\n", ui.ErrorLine(fmt.Sprintf("Verification gate failed: %v", verifyErr)))
			return verifyErr
		}
		if strictErr != nil {
			fmt.Fprintf(outWriter, "\n%s\n", ui.ErrorLine(strictErr.Error()))
			return strictErr
		}
		return nil
	},
}

func init() {
	verifyCmd.GroupID = "other"
	verifyCmd.Flags().BoolP("json", "j", false, "Output verification results as JSON")
	verifyCmd.Flags().Bool("strict", false, "Fail if .octo.lock is missing or contains invalidated strategies")
	rootCmd.AddCommand(verifyCmd)
}
