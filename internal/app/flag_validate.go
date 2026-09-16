package app

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/spf13/cobra"
)

func ValidateDeadline(value string) error {
	if value == "" {
		return nil
	}
	if value == "-1" {
		return nil
	}
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return nil
	}
	if len(value) < 2 {
		return apperrors.NewValidation(fmt.Sprintf("--deadline 的值 %q 不合法，支持: 时间戳(秒)或字符串(如2d/1w/1m/1y), -1永不失效", value))
	}
	unit := strings.ToLower(string(value[len(value)-1]))
	if unit != "d" && unit != "w" && unit != "m" && unit != "y" {
		return apperrors.NewValidation(fmt.Sprintf("--deadline 的值 %q 不合法，支持: 时间戳(秒)或字符串(如2d/1w/1m/1y), -1永不失效", value))
	}
	if _, err := strconv.ParseInt(value[:len(value)-1], 10, 64); err != nil {
		return apperrors.NewValidation(fmt.Sprintf("--deadline 的值 %q 不合法，单位前必须是纯数字", value))
	}
	return nil
}

// ValidateLifecycleRule validates the --rule parameter of the file
// add-lifecycle command according to the chosen --type.
//
//   - daysago: rule is a number of days and must be a positive integer.
//   - expire: rule is a date and must be formatted as YYYY-MM-DD.
//   - time: rule must be "<周期>-<数字>": day-N(N 为 0-23 小时),
//     week-N(N 为 1-7 星期), month-N(N 为 1-31 日期).
func ValidateLifecycleRule(lifecycleType, rule string) error {
	switch lifecycleType {
	case "daysago":
		v, err := strconv.ParseInt(rule, 10, 64)
		if err != nil || v <= 0 {
			return apperrors.NewValidation(fmt.Sprintf("--rule 的值 %q 不合法: type 为 daysago 时，--rule 必须为正整数(代表天数)", rule))
		}
	case "expire":
		if _, err := time.Parse("2006-01-02", rule); err != nil {
			return apperrors.NewValidation(fmt.Sprintf("--rule 的值 %q 不合法: type 为 expire 时，--rule 必须为日期(格式 YYYY-MM-DD)", rule))
		}
	case "time":
		if err := validateLifecycleTimeRule(rule); err != nil {
			return apperrors.NewValidation(fmt.Sprintf("--rule 的值 %q 不合法: type 为 time 时，%v", rule, err))
		}
	}
	return nil
}

// validateLifecycleTimeRule validates the --rule value of a time-type
// lifecycle, whose format is "<周期>-<数字>", e.g. day-6/week-3/month-1.
func validateLifecycleTimeRule(rule string) error {
	fields := strings.Split(rule, "-")
	if len(fields) != 2 {
		return fmt.Errorf("格式必须为 day-N/week-N/month-N")
	}
	min, max, ok := lifecycleTimeRange(fields[0])
	if !ok {
		return fmt.Errorf("周期 %q 不合法, 只能是 day/week/month", fields[0])
	}
	v, err := strconv.Atoi(fields[1])
	if err != nil {
		return fmt.Errorf("%q 必须是整数", fields[1])
	}
	if v < min || v > max {
		return fmt.Errorf("%d 超出允许范围 [%d, %d]", v, min, max)
	}
	return nil
}

// lifecycleTimeRange returns the allowed [min, max] range of the number that
// follows the period name in a lifecycle time rule. ok is false for unknown
// period names.
func lifecycleTimeRange(period string) (min, max int, ok bool) {
	switch period {
	case "day": // 小时
		return 0, 23, true
	case "week": // 星期
		return 1, 7, true
	case "month": // 日期
		return 1, 31, true
	}
	return 0, 0, false
}

// ValidateRemindTimer validates the --timer parameter of the file set-remind
// command.
//
// One or more periods joined by ";", each period being "<周期>-<字段>...":
//
//	day-<小时>-<分钟>          每天, 如 day-9,11-30
//	week-<星期>-<小时>-<分钟>   每周, 如 week-3,7-14,17-00
//	month-<日期>-<小时>-<分钟>  每月, 如 month-1,11,21-10,16-00
//
// 字段为逗号分隔的整数: 星期 1-7, 日期 1-31, 小时 0-23, 分钟 0-59。
func ValidateRemindTimer(value string) error {
	if value == "" {
		return nil
	}
	for _, period := range strings.Split(value, ";") {
		if err := validateTimerPeriod(period); err != nil {
			return apperrors.NewValidation(fmt.Sprintf("--timer 的值 %q 不合法: %v", value, err))
		}
	}
	return nil
}

// timerFieldRanges returns the allowed [min, max] range of each field that
// follows the period name. A nil result means the period name is unknown.
func timerFieldRanges(period string) [][2]int {
	switch period {
	case "day": // 小时, 分钟
		return [][2]int{{0, 23}, {0, 59}}
	case "week": // 星期, 小时, 分钟
		return [][2]int{{1, 7}, {0, 23}, {0, 59}}
	case "month": // 日期, 小时, 分钟
		return [][2]int{{1, 31}, {0, 23}, {0, 59}}
	}
	return nil
}

func validateTimerPeriod(period string) error {
	fields := strings.Split(period, "-")
	ranges := timerFieldRanges(fields[0])
	if ranges == nil {
		return fmt.Errorf("周期 %q 不合法, 只能是 day/week/month", fields[0])
	}
	if len(fields) != len(ranges)+1 {
		return fmt.Errorf("%s 需要 %d 个字段, 当前为 %q", fields[0], len(ranges), period)
	}
	for i, r := range ranges {
		if err := validateTimerField(fields[i+1], r[0], r[1]); err != nil {
			return err
		}
	}
	return nil
}

// validateTimerField validates one comma-separated integer list against [min, max].
func validateTimerField(field string, min, max int) error {
	for _, item := range strings.Split(field, ",") {
		v, err := strconv.Atoi(item)
		if err != nil {
			return fmt.Errorf("%q 必须是逗号分隔的整数", field)
		}
		if v < min || v > max {
			return fmt.Errorf("%d 超出允许范围 [%d, %d]", v, min, max)
		}
	}
	return nil
}

func ValidateStartline(value string) error {
	if value == "" {
		return nil
	}
	ts, err := strconv.ParseInt(value, 10, 64)
	if err != nil || ts <= 0 {
		return apperrors.NewValidation(fmt.Sprintf("--startline 的值 %q 不合法，必须是正整数时间戳(秒)", value))
	}
	return nil
}

// Use math.MinInt64 / math.MaxInt64 as min or max to leave that side of the
// range unbounded (e.g. timestamp flags with only a lower bound).

// getStringFlagInt64InRange reads a string flag holding an integer and
// validates the parsed value lies within [min, max]. An empty string is
// treated as unset and returns 0 without validation.
func getStringFlagInt64InRange(cmd *cobra.Command, name string, min, max int64) (int64, error) {
	raw, err := cmd.Flags().GetString(name)
	if err != nil {
		return 0, err
	}
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, apperrors.NewValidation(fmt.Sprintf("--%s 的值必须是整数，当前为 %q", name, raw))
	}
	if err := checkRange(name, value, min, max); err != nil {
		return 0, err
	}
	return value, nil
}

// checkIntFlagInRange validates that an int flag lies within [min, max].
func checkIntFlagInRange(cmd *cobra.Command, name string, min, max int64) error {
	value, err := cmd.Flags().GetInt(name)
	if err != nil {
		return err
	}
	if cmd.Flags().Changed(name) {
		return checkRange(name, int64(value), min, max)
	}
	return nil
}

func checkRange(name string, value, min, max int64) error {
	if value < min || value > max {
		return apperrors.NewValidation(fmt.Sprintf("--%s 的值 %d 超出允许范围 %s",
			name, value, formatRange(min, max)))
	}
	return nil
}

func formatRange(min, max int64) string {
	switch {
	case min == math.MinInt64 && max == math.MaxInt64:
		return "(任意整数)"
	case min == math.MinInt64:
		return fmt.Sprintf("<= %d", max)
	case max == math.MaxInt64:
		return fmt.Sprintf(">= %d", min)
	default:
		return fmt.Sprintf("[%d, %d]", min, max)
	}
}

// getStringFlagInSet reads a string flag and validates that a non-empty value
// is one of allowed. An empty string is treated as unset and returned without
// validation.
func getStringFlagInSet(cmd *cobra.Command, name string, allowed ...string) (string, error) {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", nil
	}
	for _, a := range allowed {
		if value == a {
			return value, nil
		}
	}
	return "", apperrors.NewValidation(fmt.Sprintf("--%s 的值 %q 不合法，允许值: %s",
		name, value, strings.Join(allowed, ", ")))
}

// checkStringFlagInSet validates that a string flag, when explicitly set, is
// one of allowed. Unlike getStringFlagInSet it does not return the value and
// does not treat the empty string specially if it was passed explicitly.
func checkStringFlagInSet(cmd *cobra.Command, name string, allowed ...string) error {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return err
	}
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return apperrors.NewValidation(fmt.Sprintf("--%s 的值 %q 不合法，允许值: %s",
		name, value, strings.Join(allowed, ", ")))
}

// checkIntFlagInSet validates that an int flag, when set, is one of allowed.
func checkIntFlagInSet(cmd *cobra.Command, name string, allowed ...int) error {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	value, err := cmd.Flags().GetInt(name)
	if err != nil {
		return err
	}
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	strs := make([]string, len(allowed))
	for i, a := range allowed {
		strs[i] = strconv.Itoa(a)
	}
	return apperrors.NewValidation(fmt.Sprintf("--%s 的值 %d 不合法，允许值: %s",
		name, value, strings.Join(strs, ", ")))
}

// maxFileNameLength 文件(夹)名最大长度(字符数)，与前端保持一致
const maxFileNameLength = 255

// checkFileName 校验单个文件(夹)名，规则与前端 checkFilename 一致:
//   - 不能为空
//   - 不能包含 / \ : * ? " < > | 中的任何字符
//   - 长度不能超过 255 个字符
//   - 不能以 . 结尾
func checkFileName(name string) error {
	if name == "" {
		return fmt.Errorf("文件名不能为空")
	}
	if strings.ContainsAny(name, `/\:*?"<>|`) {
		return fmt.Errorf("文件名「%s」不能包含下列任何字符: / \\ : * ? \" < > |", name)
	}
	if utf8.RuneCountInString(name) > maxFileNameLength {
		return fmt.Errorf("文件名「%s」的长度不能超过 %d 个字符", name, maxFileNameLength)
	}
	if strings.HasSuffix(name, ".") {
		return fmt.Errorf("文件名「%s」无效: 不能以 . 结尾", name)
	}
	return nil
}

// ValidateFileName 校验单个文件(夹)名 (如 file rename 的 --newname)。
func ValidateFileName(name string) error {
	if err := checkFileName(name); err != nil {
		return apperrors.NewValidation(err.Error())
	}
	return nil
}

// ValidateFullpath 校验库路径 fullpath 中的每一级文件(夹)名，规则同 checkFileName。
// 空路径视为未设置，直接通过 (必填性由命令自行保证)。
// 路径开头/结尾的 "/" 不产生文件(夹)名，对应的空段会被跳过；
// 路径中间的空段 (如 "a//b") 视为空文件名，校验失败。
func ValidateFullpath(fullpath string) error {
	if fullpath == "" {
		return nil
	}
	segments := strings.Split(fullpath, "/")
	last := len(segments) - 1
	for i, seg := range segments {
		if seg == "" && (i == 0 || i == last) {
			continue
		}
		if err := checkFileName(seg); err != nil {
			return apperrors.NewValidation(fmt.Sprintf("路径「%s」中的文件(夹)名不合法: %v", fullpath, err))
		}
	}
	return nil
}

// ValidateFullpaths 校验以 "|" 分隔的一组库路径 (如 --fullpaths)。
func ValidateFullpaths(fullpaths string) error {
	if fullpaths == "" {
		return nil
	}
	for _, p := range strings.Split(fullpaths, "|") {
		if err := ValidateFullpath(p); err != nil {
			return err
		}
	}
	return nil
}
