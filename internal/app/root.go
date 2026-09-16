package app

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gokuai/yunku-cli/internal/cache"
	"github.com/gokuai/yunku-cli/internal/cli"
	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/gokuai/yunku-cli/internal/logging"
	"github.com/gokuai/yunku-cli/internal/output"
	"github.com/gokuai/yunku-cli/internal/pipeline"
	"github.com/gokuai/yunku-cli/internal/pipeline/handlers"
	"github.com/gokuai/yunku-cli/internal/recovery"
	"github.com/gokuai/yunku-cli/pkg/edition"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type outputFileContextKey struct{}

const recoveryEventStderrPrefix = "RECOVERY_EVENT_ID="

// Execute runs the root command and returns the process exit code.
func Execute() int {
	// Load .env file from current directory and executable directory.
	// Existing env vars take precedence (godotenv will not overwrite).
	loadDotEnv()

	timing := NewTimingCollector()
	defer func() {
		timing.PrintIfEnabled()
		timing.WriteReportIfEnabled(RawVersion(), SanitizeCommand(os.Args))
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Attach timing collector to context for use by child components
	ctx = WithTimingCollector(ctx, timing)

	initStart := time.Now()
	recovery.ResetRuntimeState()
	engine := newPipelineEngine()
	root := NewRootCommandWithEngine(ctx, engine)
	timing.Record("cmd_init", time.Since(initStart))

	// Run PreParse handlers on raw argv before Cobra parses flags.
	// This corrects model-generated errors like --userId → --user-id
	// and --limit100 → --limit 100.
	pipeline.RunPreParse(root, engine)

	executed, err := root.ExecuteC()
	if err != nil {
		if executed == nil {
			executed = root
		}
		if isUnknownCommandError(err) {
			executed.SetOut(os.Stderr)
			_ = executed.Help()
			_, _ = fmt.Fprintln(os.Stderr)
		}
		err = classifyCobraUsageError(err)
		_ = printExecutionError(executed, os.Stdout, os.Stderr, err)
		if last := recovery.LatestCapture(); last != nil && last.EventID != "" {
			_, _ = fmt.Fprintf(os.Stderr, "%s%s\n", recoveryEventStderrPrefix, last.EventID)
		}
		return apperrors.ExitCode(err)
	}
	return 0
}

func isUnknownCommandError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "unknown command")
}

// cobraUsageErrorMarkers lists the stable substrings Cobra uses in its own
// argument/flag validation errors (command.go, args.go, flag_groups.go).
// These errors occur before RunE is ever invoked, so they always mean the
// invocation itself was malformed — i.e. a validation failure, not an
// internal one.
var cobraUsageErrorMarkers = []string{
	"required flag(s)",
	"unknown command",
	"unknown flag:",
	"unknown shorthand flag:",
	"invalid argument",
	"accepts at most",
	"accepts between",
	"arg(s), received",
	"flag needs an argument",
	"flags in the group",
}

// classifyCobraUsageError wraps unclassified Cobra usage errors (missing
// required flags, unknown flags, bad argument counts, etc.) as Validation
// so their exit code matches the documented contract (3=Validation) instead
// of defaulting to 5=Internal. Errors that are already categorized (or don't
// match a known Cobra usage pattern) are returned unchanged.
func classifyCobraUsageError(err error) error {
	if err == nil {
		return nil
	}
	if apperrors.ExitCode(err) != 5 {
		return err
	}
	var typed *apperrors.Error
	if stderrors.As(err, &typed) {
		return err
	}
	msg := err.Error()
	for _, marker := range cobraUsageErrorMarkers {
		if strings.Contains(msg, marker) {
			return apperrors.NewValidation(msg)
		}
	}
	return err
}

// flagErrorWithSuggestions provides helpful suggestions for common flag mistakes.
func flagErrorWithSuggestions(cmd *cobra.Command, err error) error {
	errMsg := err.Error()

	// Common flag aliases and suggestions
	suggestions := map[string]string{
		"--json": "提示: 请使用 --format json 或 -f json 来输出 JSON 格式",
	}

	for flag, suggestion := range suggestions {
		if strings.Contains(errMsg, "unknown flag: "+flag) {
			return fmt.Errorf("%w\n%s", err, suggestion)
		}
	}

	return err
}

func printExecutionError(root *cobra.Command, stdout, stderr io.Writer, err error) error {
	var raw apperrors.RawStderrError
	if stderrors.As(err, &raw) {
		_, writeErr := fmt.Fprintln(stderr, raw.RawStderr())
		return writeErr
	}
	if wantsJSONErrors(root) {
		return apperrors.PrintJSON(stdout, err)
	}
	return apperrors.PrintHumanAt(stderr, err, resolveVerbosity(root))
}

// resolveVerbosity derives the error verbosity level from the root command's flags.
func resolveVerbosity(cmd *cobra.Command) apperrors.Verbosity {
	if cmd == nil {
		return apperrors.VerbosityNormal
	}
	if debug, err := cmd.Flags().GetBool("debug"); err == nil && debug {
		return apperrors.VerbosityDebug
	}
	if verbose, err := cmd.Flags().GetBool("verbose"); err == nil && verbose {
		return apperrors.VerbosityVerbose
	}
	return apperrors.VerbosityNormal
}

func wantsJSONErrors(root *cobra.Command) bool {
	if root == nil {
		return false
	}
	if commandRequestsJSONErrors(root) {
		return true
	}
	if rootCmd := root.Root(); rootCmd != nil && rootCmd != root {
		return commandRequestsJSONErrors(rootCmd)
	}
	return false
}

func commandRequestsJSONErrors(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	for _, flags := range []interface {
		Lookup(string) *pflag.Flag
		GetString(string) (string, error)
		GetBool(string) (bool, error)
	}{
		cmd.Flags(),
		cmd.InheritedFlags(),
		cmd.PersistentFlags(),
	} {
		if flags == nil {
			continue
		}
		if flag := flags.Lookup("format"); flag != nil {
			if value, err := flags.GetString("format"); err == nil && strings.EqualFold(strings.TrimSpace(value), "json") {
				return true
			}
		}
		if flag := flags.Lookup("json"); flag != nil && flag.Changed {
			if value, err := flags.GetBool("json"); err == nil {
				if value {
					return true
				}
				continue
			}
			return true
		}
	}
	return false
}

// NewRootCommand constructs the root CLI command. The provided context
// is propagated to background goroutines and the Cobra command tree so
// that SIGINT/SIGTERM can cancel in-flight work.
func NewRootCommand(ctx ...context.Context) *cobra.Command {
	var rootCtx context.Context
	if len(ctx) > 0 && ctx[0] != nil {
		rootCtx = ctx[0]
	}
	return NewRootCommandWithEngine(rootCtx, nil)
}

// NewRootCommandWithEngine constructs the root CLI command with an
// optional pipeline engine for input correction. When engine is nil,
// no pipeline processing is applied.
func NewRootCommandWithEngine(rootCtx context.Context, engine *pipeline.Engine) *cobra.Command {
	if rootCtx == nil {
		rootCtx = context.Background()
	}
	// EnableTraverseRunHooks makes Cobra chain PersistentPreRunE/PersistentPostRunE
	// through every command on the parent chain (root → ... → target) instead of
	// stopping at the nearest one. Without this, defining PersistentPreRunE on a
	// subcommand (e.g. ent) silently overrides the root's hooks (log level,
	// bearer-token, output sink), breaking --debug, --bearer-token, etc.
	cobra.EnableTraverseRunHooks = true

	flags := &GlobalFlags{}
	runner := newCommandRunnerWithFlags(flags)

	root := &cobra.Command{
		Use:               "ykc",
		Short:             "ykc CLI",
		Args:              cobra.NoArgs,
		SilenceErrors:     true,
		SilenceUsage:      true,
		DisableAutoGenTag: true,
		Version:           Version(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Apply bearer token override for gokuai API clients.
			// Priority: --bearer-token flag > GOKUAI_BEARER_TOKEN env var.
			if flags.Token != "" {
				SetBearerToken(flags.Token)
			} else if v := os.Getenv("GOKUAI_BEARER_TOKEN"); v != "" {
				SetBearerToken(v)
			}

			// Configure global slog level based on --debug / --verbose flags.
			configureLogLevel(flags)

			if err := configureOutputSink(cmd); err != nil {
				return err
			}
			if fn := edition.Get().AfterPersistentPreRun; fn != nil {
				return fn(cmd, args)
			}
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			CloseFileLogger()
			return closeOutputSink(cmd)
		},
	}

	bindPersistentFlags(root, flags)

	utilityCommands := []*cobra.Command{
		newAuthCommand(),
		newCacheCommand(),
		newConfigCommand(),
		newDoctorCommand(),
		newFileCommand(),
		newAccountCommand(),
		newFavoriteCommand(),
		newContactCommand(),
		newLibraryCommand(),
		newEntCommand(flags),
		newCompletionCommand(root),
		newRecoveryCommand(rootCtx, flags),
		newUpgradeCommand(),
		newVersionCommand(),
	}
	root.AddCommand(utilityCommands...)
	root.AddCommand(newLegacyPublicCommands(rootCtx, runner)...)
	root.AddCommand(newLegacyHiddenCommands(runner)...)

	if fn := edition.Get().RegisterExtraCommands; fn != nil {
		caller := newToolCallerAdapter(runner, flags)
		fn(root, caller)
		deduplicateCommands(root)
	}

	hideNonDirectRuntimeCommands(root)
	configureRootHelp(root)
	// 为所有通过 MarkFlagRequired 标记的必填 flag 在帮助文字后追加 " (必需)"。
	annotateRequiredFlagUsages(root)
	// Set custom flag error handler for better UX
	root.SetFlagErrorFunc(flagErrorWithSuggestions)
	root.SetContext(rootCtx)

	return root
}

// loadDotEnv loads .env files from the current working directory and
// the executable's directory. Existing environment variables are not
// overwritten. Errors are silently ignored (missing .env is not an error).
func loadDotEnv() {
	// Try current working directory first
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Overload(".env"); err != nil {
			slog.Debug("failed to load .env from working directory", "error", err)
		} else {
			slog.Debug("loaded .env from working directory")
		}
	}

	// Try executable directory
	if exePath, err := os.Executable(); err == nil {
		if realPath, err := filepath.EvalSymlinks(exePath); err == nil {
			exePath = realPath
		}
		envFile := filepath.Join(filepath.Dir(exePath), ".env")
		if _, err := os.Stat(envFile); err == nil {
			if err := godotenv.Overload(envFile); err != nil {
				slog.Debug("failed to load .env from executable directory", "error", err)
			} else {
				slog.Debug("loaded .env from executable directory", "path", envFile)
			}
		}
	}
}

func newAuthCommand() *cobra.Command {
	return buildAuthCommand()
}

func newCacheCommand() *cobra.Command {
	cacheCmd := newPlaceholderParent("cache", "缓存管理")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "查看缓存状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOut, err := cmd.Flags().GetBool("json")
			if err != nil {
				return apperrors.NewInternal("读取 cache status 参数失败")
			}

			store := cacheStoreFromEnv()
			files, bytes, err := cacheDirectoryStats(store.Root)
			if err != nil {
				return apperrors.NewInternal(fmt.Sprintf("读取缓存状态失败: %v", err))
			}

			payload := map[string]any{
				"kind":       "cache_status",
				"cache_root": store.Root,
				"files":      files,
				"bytes":      bytes,
			}

			if jsonOut {
				return output.WriteJSON(cmd.OutOrStdout(), payload)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "缓存目录: %s\n文件数:   %d   大小: %d 字节\n", store.Root, files, bytes)
			return nil
		},
	}
	statusCmd.Flags().Bool("json", false, "Emit cache status as JSON")

	cleanCmd := &cobra.Command{
		Use:               "clean",
		Short:             "清理缓存",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			staleOnly, err := cmd.Flags().GetBool("stale")
			if err != nil {
				return apperrors.NewInternal("读取 cache clean stale 参数失败")
			}

			store := cacheStoreFromEnv()
			removed, err := cleanCacheFiles(store.Root, staleOnly)
			if err != nil {
				return apperrors.NewInternal(fmt.Sprintf("清理缓存失败: %v", err))
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"[OK] 缓存清理完成：已删除 %d 个文件\n",
				removed,
			)
			return err
		},
	}
	cleanCmd.Flags().Bool("stale", false, "Only remove stale cache entries")

	cacheCmd.AddCommand(statusCmd, cleanCmd)
	return cacheCmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "version",
		Short:             "显示版本信息",
		Example:           "  ykc version\n  ykc version --format json",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			wantJSON := cmd.Flags().Changed("format")
			if wantJSON {
				format, _ := cmd.Flags().GetString("format")
				wantJSON = (format == "json")
			}

			editionName := edition.Get().Name
			if editionName == "" {
				editionName = "open"
			}
			ver := RawVersion()
			bt := BuildTime()
			gc := GitCommit()
			goVer := "1.24+"

			if wantJSON {
				payload := map[string]any{
					"version": ver,
					"edition": editionName,
					"go":      goVer,
				}
				if bt != "unknown" {
					payload["build"] = bt
				}
				if gc != "unknown" {
					payload["commit"] = gc
				}
				return output.WriteJSON(cmd.OutOrStdout(), payload)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "%-16s%s\n", "Version:", ver)
			fmt.Fprintf(w, "%-16s%s\n", "Edition:", editionName)
			if bt != "unknown" {
				fmt.Fprintf(w, "%-16s%s\n", "Build:", bt)
			}
			if gc != "unknown" {
				fmt.Fprintf(w, "%-16s%s\n", "Commit:", gc)
			}
			fmt.Fprintf(w, "%-16s%s\n", "Go:", goVer)
			return nil
		},
	}
}

// hideNonDirectRuntimeCommands marks top-level product commands as hidden
// unless they correspond to a product discovered via dynamic server discovery
// or listed in the edition's VisibleProducts hook.
// Public utility commands (auth, cache, completion, version) are always kept
// visible; explicitly hidden commands stay hidden.
func hideNonDirectRuntimeCommands(root *cobra.Command) {
	var allowedProducts map[string]bool
	if fn := edition.Get().VisibleProducts; fn != nil {
		products := fn()
		allowedProducts = make(map[string]bool, len(products))
		for _, p := range products {
			allowedProducts[p] = true
		}
	} else {
		allowedProducts = DirectRuntimeProductIDs()
	}
	staticCommands := map[string]bool{
		"auth":       true,
		"cache":      true,
		"config":     true,
		"doctor":     true,
		"completion": true,
		"version":    true,
		"help":       true,
		"recovery":   true,
		"file":       true,
		"ent":        true,
		"account":    true,
		"contact":    true,
		"favorite":   true,
		"library":    true,
	}
	for _, cmd := range root.Commands() {
		name := cmd.Name()
		if cmd.Hidden {
			continue
		}
		if staticCommands[name] {
			continue
		}
		if allowedProducts[name] {
			continue
		}
		cmd.Hidden = true
	}
}

// deduplicateCommands removes duplicate top-level commands, keeping the last
// registered one. This ensures overlay commands take precedence over
// open-source defaults when both register the same product name.
func deduplicateCommands(root *cobra.Command) {
	seen := make(map[string]*cobra.Command)
	var dups []*cobra.Command
	for _, cmd := range root.Commands() {
		name := cmd.Name()
		if prev, ok := seen[name]; ok {
			dups = append(dups, prev)
		}
		seen[name] = cmd
	}
	for _, dup := range dups {
		root.RemoveCommand(dup)
	}
}

func cacheStoreFromEnv() *cache.Store {
	cacheDir := strings.TrimSpace(os.Getenv(cli.CacheDirEnv))
	return cache.NewStore(cacheDir)
}

func configureOutputSink(cmd *cobra.Command) error {
	if local := cmd.LocalFlags().Lookup("output"); local != nil {
		return nil
	}
	outputPath, err := cmd.Flags().GetString("output")
	if err != nil {
		return apperrors.NewInternal("读取 output 参数失败")
	}
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		return nil
	}
	if err := validateOptionalPath("--output", outputPath); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return apperrors.NewInternal(fmt.Sprintf("创建输出目录失败: %v", err))
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return apperrors.NewInternal(fmt.Sprintf("创建输出文件失败: %v", err))
	}
	cmd.SetOut(file)
	cmd.SetContext(context.WithValue(cmd.Context(), outputFileContextKey{}, file))
	return nil
}

func closeOutputSink(cmd *cobra.Command) error {
	file, ok := cmd.Context().Value(outputFileContextKey{}).(*os.File)
	if !ok || file == nil {
		return nil
	}
	if err := file.Close(); err != nil {
		return apperrors.NewInternal(fmt.Sprintf("关闭输出文件失败: %v", err))
	}
	return nil
}

func validateOptionalPath(flagName, path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := apperrors.SafePath(path); err != nil {
		return apperrors.NewValidation(fmt.Sprintf("%s 包含不安全的路径: %v", flagName, err))
	}
	return nil
}

func cacheDirectoryStats(root string) (int, int64, error) {
	if strings.TrimSpace(root) == "" {
		return 0, 0, nil
	}
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}

	files := 0
	var bytes int64
	err := filepath.WalkDir(root, func(entryPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files++
		bytes += info.Size()
		return nil
	})
	return files, bytes, err
}

func cleanCacheFiles(root string, staleOnly bool) (int, error) {
	if strings.TrimSpace(root) == "" {
		return 0, nil
	}
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	staleCutoff := time.Now().UTC().Add(-24 * time.Hour)
	removed := 0

	err := filepath.WalkDir(root, func(entryPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if staleOnly && info.ModTime().After(staleCutoff) {
			return nil
		}
		if err := os.Remove(entryPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		removed++
		return nil
	})
	if err != nil {
		return 0, err
	}
	return removed, nil
}

// fileLogger holds the package-level file logger for diagnostics.
// It is initialized by configureLogLevel and closed by CloseFileLogger.
var fileLogger *logging.FileLogger

// configureLogLevel sets the global slog level based on --debug and --verbose flags
// and initializes the file logger for diagnostics.
// --debug → slog.LevelDebug; --verbose → slog.LevelInfo; default → slog.LevelWarn.
func configureLogLevel(flags *GlobalFlags) {
	if flags == nil {
		return
	}
	var level slog.Level
	switch {
	case flags.Debug:
		level = slog.LevelDebug
	case flags.Verbose:
		level = slog.LevelInfo
	default:
		level = slog.LevelWarn
	}
	stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})

	// Initialize file logger — writes to ~/.ykc/logs/ykc.log at DEBUG level
	// regardless of stderr level. All slog calls are captured for diagnostics.
	fileLogger = logging.Setup(defaultConfigDir())
	fileHandler := slog.NewJSONHandler(fileLogger.Writer(), &slog.HandlerOptions{Level: slog.LevelDebug})

	slog.SetDefault(slog.New(logging.NewMultiHandler(stderrHandler, fileHandler)))
}

// FileLoggerInstance returns the package-level file logger, or nil if not initialized.
func FileLoggerInstance() *slog.Logger {
	if fileLogger == nil {
		return nil
	}
	return fileLogger.Logger
}

// CloseFileLogger flushes and closes the file logger.
func CloseFileLogger() {
	if fileLogger != nil {
		fileLogger.Close()
	}
}

// newPipelineEngine creates and configures the pipeline engine with
// handlers for all five pipeline phases. The phases execute in order:
// Register → PreParse → PostParse → PreRequest → PostResponse.
//
// Phases are invoked at their respective integration points:
//   - Register:     during command tree construction
//   - PreParse:     before Cobra parses raw argv (RunPreParse)
//   - PostParse:    after Cobra parsing, before validation (canonical RunE)
//   - PreRequest:   after validation, before request dispatch (canonical RunE)
//   - PostResponse: after transport returns, before stdout (canonical RunE)
func newPipelineEngine() *pipeline.Engine {
	engine := pipeline.NewEngine()
	engine.RegisterAll(
		// Register handler runs during command tree building.
		handlers.RegisterHandler{},

		// PreParse handlers run in order: alias → sticky → paramname.
		// Alias normalises case first (--userId → --user-id), then
		// sticky splits glued values (--limit100 → --limit 100), then
		// paramname fixes near-miss typos (--limt → --limit).
		handlers.AliasHandler{},
		handlers.StickyHandler{},
		handlers.ParamNameHandler{},

		// PostParse handlers normalise structured values.
		handlers.ParamValueHandler{},

		// PreRequest handler inspects the validated payload before dispatch.
		handlers.PreRequestHandler{},

		// PostResponse handler processes the response before output.
		handlers.PostResponseHandler{},
	)
	return engine
}

