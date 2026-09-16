package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/gokuai/yunku-cli/internal/output"
	"github.com/gokuai/yunku-cli/internal/recovery"
	"github.com/spf13/cobra"
)

func newRecoveryCommand(_ context.Context, flags *GlobalFlags) *cobra.Command {
	var (
		planUseLast    bool
		planEventID    string
		executeUseLast bool
		executeEventID string
		finalEventID   string
		finalOutcome   string
		executionFile  string
	)

	cmd := &cobra.Command{
		Use:               "recovery",
		Short:             "错误恢复辅助命令",
		Long:              "读取失败快照，生成恢复分析，并回写恢复结果。",
		Args:              cobra.NoArgs,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	planCmd := &cobra.Command{
		Use:               "plan",
		Short:             "基于失败快照生成恢复计划",
		Args:              cobra.NoArgs,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			store := recovery.NewStore(defaultConfigDir())
			last, err := loadRecoverySnapshot(store, planUseLast, planEventID)
			if err != nil {
				return err
			}

			planner := recovery.NewPlanner()
			plan := planner.PlanWithOptions(cmd.Context(), last.Context, recovery.PlanOptions{
				EventID:         last.EventID,
				EnableDocSearch: true,
			})
			recovery.HydratePlanForEvent(last.EventID, last.Context, last.Replay, &plan)
			if err := store.SavePlan(last.EventID, plan); err != nil {
				return fmt.Errorf("保存恢复计划失败: %w", err)
			}

			payload := map[string]any{
				"event_id": last.EventID,
				"context":  last.Context,
				"plan":     plan,
			}
			return output.WriteCommandPayload(cmd, payload, output.FormatJSON)
		},
	}
	planCmd.Flags().BoolVar(&planUseLast, "last", false, "读取最近一次失败快照")
	planCmd.Flags().StringVar(&planEventID, "event-id", "", "按 event_id 读取失败快照")

	executeCmd := &cobra.Command{
		Use:               "execute",
		Short:             "生成面向 Agent 的恢复分析包",
		Args:              cobra.NoArgs,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			store := recovery.NewStore(defaultConfigDir())
			last, err := loadRecoverySnapshot(store, executeUseLast, executeEventID)
			if err != nil {
				return err
			}

			planner := recovery.NewPlanner()
			executor := recovery.NewExecutor(planner)
			bundle := executor.Execute(cmd.Context(), *last)
			if err := store.SaveAnalysis(last.EventID, bundle.Plan, bundle); err != nil {
				return fmt.Errorf("保存恢复分析失败: %w", err)
			}

			return output.WriteCommandPayload(cmd, bundle, output.FormatJSON)
		},
	}
	executeCmd.Flags().BoolVar(&executeUseLast, "last", false, "读取最近一次失败快照")
	executeCmd.Flags().StringVar(&executeEventID, "event-id", "", "按 event_id 读取失败快照")

	finalizeCmd := &cobra.Command{
		Use:               "finalize",
		Short:             "回写恢复闭环结果",
		Args:              cobra.NoArgs,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(finalEventID) == "" {
				return apperrors.NewValidation("必须提供 --event-id")
			}
			if strings.TrimSpace(finalOutcome) == "" {
				return apperrors.NewValidation("必须提供 --outcome")
			}
			switch finalOutcome {
			case "recovered", "failed", "handoff":
			default:
				return apperrors.NewValidation("--outcome 仅支持 recovered|failed|handoff")
			}

			store := recovery.NewStore(defaultConfigDir())
			var execution *recovery.RecoveryExecution
			if strings.TrimSpace(executionFile) != "" {
				loaded, err := loadRecoveryExecution(executionFile)
				if err != nil {
					return err
				}
				execution = &loaded
			}
			if err := store.Finalize(finalEventID, finalOutcome, execution); err != nil {
				return fmt.Errorf("回写恢复结果失败: %w", err)
			}

			payload := map[string]any{
				"event_id": finalEventID,
				"outcome":  finalOutcome,
				"success":  true,
			}
			if execution != nil {
				payload["execution_recorded"] = true
			}
			return output.WriteCommandPayload(cmd, payload, output.FormatJSON)
		},
	}
	finalizeCmd.Flags().StringVar(&finalEventID, "event-id", "", "恢复事件 ID")
	finalizeCmd.Flags().StringVar(&finalOutcome, "outcome", "", "恢复结果: recovered|failed|handoff")
	finalizeCmd.Flags().StringVar(&executionFile, "execution-file", "", "Agent 执行详情 JSON 文件")

	cmd.AddCommand(planCmd, executeCmd, finalizeCmd)
	return cmd
}

func loadRecoverySnapshot(store *recovery.Store, useLast bool, eventID string) (*recovery.LastError, error) {
	if useLast && strings.TrimSpace(eventID) != "" {
		return nil, apperrors.NewValidation("--last 和 --event-id 不能同时使用")
	}
	switch {
	case useLast:
		last, err := store.LoadLastError()
		if err != nil {
			return nil, fmt.Errorf("读取失败快照失败: %w", err)
		}
		return last, nil
	case strings.TrimSpace(eventID) != "":
		last, err := store.LoadErrorByEvent(strings.TrimSpace(eventID))
		if err != nil {
			return nil, fmt.Errorf("读取失败快照失败: %w", err)
		}
		return last, nil
	default:
		return nil, apperrors.NewValidation("必须通过 --last 或 --event-id 指定失败快照")
	}
}

func loadRecoveryExecution(path string) (recovery.RecoveryExecution, error) {
	var execution recovery.RecoveryExecution
	data, err := os.ReadFile(path)
	if err != nil {
		return execution, fmt.Errorf("读取恢复执行详情失败: %w", err)
	}
	var payload recoveryExecutionPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return execution, fmt.Errorf("解析恢复执行详情失败: %w", err)
	}
	execution.Actions = append([]string(nil), payload.Actions...)
	if len(execution.Actions) == 0 && strings.TrimSpace(payload.Action) != "" {
		execution.Actions = []string{strings.TrimSpace(payload.Action)}
	}
	execution.Result = strings.TrimSpace(payload.Result)
	execution.ErrorSummary = strings.TrimSpace(payload.ErrorSummary)
	if execution.ErrorSummary == "" {
		execution.ErrorSummary = strings.TrimSpace(payload.Error)
	}

	attempts, err := decodeRecoveryAttempts(payload.Attempts, execution.Actions, execution.Result, execution.ErrorSummary)
	if err != nil {
		return execution, fmt.Errorf("解析恢复执行详情失败: %w", err)
	}
	if len(attempts) == 0 && payload.Attempt > 0 {
		attempts = legacyRecoveryAttempts(payload.Attempt, execution.Actions, execution.Result, execution.ErrorSummary)
	}
	execution.Attempts = attempts
	return execution, nil
}

type recoveryExecutionPayload struct {
	Action       string          `json:"action,omitempty"`
	Actions      []string        `json:"actions,omitempty"`
	Attempt      int             `json:"attempt,omitempty"`
	Attempts     json.RawMessage `json:"attempts,omitempty"`
	Result       string          `json:"result,omitempty"`
	Error        string          `json:"error,omitempty"`
	ErrorSummary string          `json:"error_summary,omitempty"`
}

func decodeRecoveryAttempts(raw json.RawMessage, actions []string, result, errorSummary string) ([]recovery.RecoveryAttempt, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var attempts []recovery.RecoveryAttempt
		if err := json.Unmarshal(raw, &attempts); err != nil {
			return nil, err
		}
		return attempts, nil
	}

	var count int
	if err := json.Unmarshal(raw, &count); err != nil {
		return nil, err
	}
	return legacyRecoveryAttempts(count, actions, result, errorSummary), nil
}

func legacyRecoveryAttempts(count int, actions []string, result, errorSummary string) []recovery.RecoveryAttempt {
	if count <= 0 {
		return nil
	}
	summary := strings.TrimSpace(strings.Join(actions, ", "))
	if summary == "" {
		summary = "legacy execution attempt"
	}
	attempts := make([]recovery.RecoveryAttempt, 0, count)
	for i := 0; i < count; i++ {
		attempts = append(attempts, recovery.RecoveryAttempt{
			CommandSummary: summary,
			Result:         result,
			ErrorSummary:   errorSummary,
			Source:         "legacy_execution_file",
		})
	}
	return attempts
}
