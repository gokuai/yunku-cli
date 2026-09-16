package app

import (
	"math"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newRangeTestCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("str", "", "string flag holding an int")
	cmd.Flags().Int("num", 0, "int flag")
	return cmd
}

func TestGetStringFlagInt64InRange(t *testing.T) {
	t.Run("empty is unset", func(t *testing.T) {
		cmd := newRangeTestCommand()
		v, err := getStringFlagInt64InRange(cmd, "str", 1, math.MaxInt64)
		if err != nil || v != 0 {
			t.Fatalf("got (%d, %v), want (0, nil)", v, err)
		}
	})

	t.Run("valid in range", func(t *testing.T) {
		cmd := newRangeTestCommand()
		cmd.Flags().Set("str", "1700000000")
		v, err := getStringFlagInt64InRange(cmd, "str", 1, math.MaxInt64)
		if err != nil || v != 1700000000 {
			t.Fatalf("got (%d, %v), want (1700000000, nil)", v, err)
		}
	})

	t.Run("not an integer", func(t *testing.T) {
		cmd := newRangeTestCommand()
		cmd.Flags().Set("str", "abc")
		_, err := getStringFlagInt64InRange(cmd, "str", 1, math.MaxInt64)
		if err == nil || !strings.Contains(err.Error(), "必须是整数") {
			t.Fatalf("expected integer validation error, got %v", err)
		}
	})

	t.Run("out of range", func(t *testing.T) {
		cmd := newRangeTestCommand()
		cmd.Flags().Set("str", "-5")
		_, err := getStringFlagInt64InRange(cmd, "str", 1, math.MaxInt64)
		if err == nil || !strings.Contains(err.Error(), "超出允许范围") {
			t.Fatalf("expected range error, got %v", err)
		}
	})

	t.Run("bounded range both sides", func(t *testing.T) {
		cmd := newRangeTestCommand()
		cmd.Flags().Set("str", "11")
		if _, err := getStringFlagInt64InRange(cmd, "str", 1, 10); err == nil {
			t.Fatal("expected range error for 11 in [1,10]")
		}
		cmd.Flags().Set("str", "0")
		if _, err := getStringFlagInt64InRange(cmd, "str", 1, 10); err == nil {
			t.Fatal("expected range error for 0 in [1,10]")
		}
		cmd.Flags().Set("str", "10")
		if _, err := getStringFlagInt64InRange(cmd, "str", 1, 10); err != nil {
			t.Fatalf("boundary value 10 should pass, got %v", err)
		}
	})
}

func TestCheckIntFlagInRange(t *testing.T) {
	t.Run("unset flag skipped", func(t *testing.T) {
		cmd := newRangeTestCommand()
		if err := checkIntFlagInRange(cmd, "num", 1, math.MaxInt64); err != nil {
			t.Fatalf("unset flag should not be validated, got %v", err)
		}
	})

	t.Run("set flag validated", func(t *testing.T) {
		cmd := newRangeTestCommand()
		cmd.Flags().Set("num", "0")
		if err := checkIntFlagInRange(cmd, "num", 1, math.MaxInt64); err == nil {
			t.Fatal("expected range error for num=0")
		}
	})
}

func TestGetStringFlagInSet(t *testing.T) {
	t.Run("empty is unset", func(t *testing.T) {
		cmd := newRangeTestCommand()
		v, err := getStringFlagInSet(cmd, "str", "0", "1")
		if err != nil || v != "" {
			t.Fatalf("got (%q, %v), want (\"\", nil)", v, err)
		}
	})

	t.Run("allowed value", func(t *testing.T) {
		cmd := newRangeTestCommand()
		cmd.Flags().Set("str", "1")
		v, err := getStringFlagInSet(cmd, "str", "0", "1")
		if err != nil || v != "1" {
			t.Fatalf("got (%q, %v), want (\"1\", nil)", v, err)
		}
	})

	t.Run("disallowed value", func(t *testing.T) {
		cmd := newRangeTestCommand()
		cmd.Flags().Set("str", "2")
		_, err := getStringFlagInSet(cmd, "str", "0", "1")
		if err == nil || !strings.Contains(err.Error(), "允许值: 0, 1") {
			t.Fatalf("expected allowed-values error, got %v", err)
		}
	})
}

func TestCheckStringFlagInSet(t *testing.T) {
	t.Run("unset flag skipped even with zero value", func(t *testing.T) {
		cmd := newRangeTestCommand()
		if err := checkStringFlagInSet(cmd, "str", "a", "b"); err != nil {
			t.Fatalf("unset flag should not be validated, got %v", err)
		}
	})

	t.Run("explicit empty is rejected when set", func(t *testing.T) {
		cmd := newRangeTestCommand()
		cmd.Flags().Set("str", "")
		if err := checkStringFlagInSet(cmd, "str", "a", "b"); err == nil {
			t.Fatal("explicit empty value not in set should error")
		}
	})

	t.Run("allowed value passes", func(t *testing.T) {
		cmd := newRangeTestCommand()
		cmd.Flags().Set("str", "b")
		if err := checkStringFlagInSet(cmd, "str", "a", "b"); err != nil {
			t.Fatalf("allowed value should pass, got %v", err)
		}
	})
}

func TestCheckIntFlagInSet(t *testing.T) {
	t.Run("unset flag skipped", func(t *testing.T) {
		cmd := newRangeTestCommand()
		if err := checkIntFlagInSet(cmd, "num", 0, 1); err != nil {
			t.Fatalf("unset flag should not be validated, got %v", err)
		}
	})

	t.Run("disallowed value", func(t *testing.T) {
		cmd := newRangeTestCommand()
		cmd.Flags().Set("num", "5")
		err := checkIntFlagInSet(cmd, "num", 0, 1)
		if err == nil || !strings.Contains(err.Error(), "允许值: 0, 1") {
			t.Fatalf("expected allowed-values error, got %v", err)
		}
	})

	t.Run("allowed value", func(t *testing.T) {
		cmd := newRangeTestCommand()
		cmd.Flags().Set("num", "1")
		if err := checkIntFlagInSet(cmd, "num", 0, 1); err != nil {
			t.Fatalf("allowed value should pass, got %v", err)
		}
	})
}

func TestValidateDeadline(t *testing.T) {
	cases := []struct {
		input   string
		wantErr bool
	}{
		// empty
		{"", false},
		// special -1 (never expire)
		{"-1", false},
		// second-level timestamps (any int)
		{"0", false},
		{"1700000000", false},
		{"-100", false},
		{"1735660800", false},
		// relative durations — case-insensitive units
		{"1d", false},
		{"1D", false},
		{"10d", false},
		{"10D", false},
		{"2W", false},
		{"3m", false},
		{"1Y", false},
		{"0d", false},
		// invalid
		{"abc", true},
		{"10dd", true}, // non-numeric + invalid unit
		{"1.5d", true}, // non-integer digits
		{"10x", true},  // unsupported unit
		{"1", false},   // plain seconds
		{"d", true},    // no digits
		{"10", false},  // plain seconds
		{"1d ", true},  // trailing space
		{" 1d", true},  // leading space
	}
	for _, c := range cases {
		err := ValidateDeadline(c.input)
		gotErr := err != nil
		if gotErr != c.wantErr {
			t.Errorf("ValidateDeadline(%q) err = %v, wantErr = %v", c.input, err, c.wantErr)
		}
	}
}

func TestValidateLifecycleRule(t *testing.T) {
	cases := []struct {
		lifecycleType string
		rule          string
		wantErr       bool
	}{
		// daysago requires a positive integer
		{"daysago", "1", false},
		{"daysago", "30", false},
		{"daysago", "365", false},
		{"daysago", "0", true},
		{"daysago", "-1", true},
		{"daysago", "abc", true},
		{"daysago", "1.5", true},
		{"daysago", "30 ", true},  // trailing space
		{"daysago", "", true},     // rule is required, non-empty
		// expire requires a YYYY-MM-DD date
		{"expire", "2019-10-10", false},
		{"expire", "2026-08-27", false},
		{"expire", "2019-13-10", true},  // invalid month
		{"expire", "2019-02-30", true},  // invalid day
		{"expire", "2019/10/10", true},  // wrong separator
		{"expire", "abc", true},
		{"expire", "", true},
		// time requires <周期>-<数字>: day(0-23), week(1-7), month(1-31)
		{"time", "day-6", false},
		{"time", "day-0", false},
		{"time", "day-23", false},
		{"time", "week-1", false},
		{"time", "week-7", false},
		{"time", "month-1", false},
		{"time", "month-31", false},
		{"time", "day-24", true},   // hour out of range
		{"time", "day--1", true},   // splits into 3 fields
		{"time", "week-0", true},   // weekday out of range
		{"time", "week-8", true},
		{"time", "month-0", true},  // day of month out of range
		{"time", "month-32", true},
		{"time", "garbage", true},  // unknown period
		{"time", "day-6-1", true},  // too many fields
		{"time", "day-x", true},    // not an integer
		{"time", "", true},
	}
	for _, c := range cases {
		err := ValidateLifecycleRule(c.lifecycleType, c.rule)
		gotErr := err != nil
		if gotErr != c.wantErr {
			t.Errorf("ValidateLifecycleRule(%q, %q) err = %v, wantErr = %v", c.lifecycleType, c.rule, err, c.wantErr)
		}
	}
}

func TestValidateRemindTimer(t *testing.T) {
	cases := []struct {
		input   string
		wantErr bool
	}{
		// empty (不设置定时提醒)
		{"", false},
		// day: 小时, 分钟
		{"day-9-30", false},
		{"day-9,11-30", false},
		{"day-0-0", false},
		{"day-23-59", false},
		// week: 星期, 小时, 分钟
		{"week-3,7-14,17-00", false},
		{"week-1-0-0", false},
		{"week-7-23-59", false},
		// month: 日期, 小时, 分钟
		{"month-1,11,21-10,16-00", false},
		{"month-31-23-59", false},
		// 多个周期
		{"day-9,11-30;week-3,7-14,17-00;month-1,11,21-10,16-00", false},
		// 未知周期
		{"year-1-9-30", true},
		{"Day-9-30", true},
		// 字段个数不符
		{"day-9", true},
		{"day-9-30-0", true},
		{"week-3-14", true},
		{"month-1-10-00-1", true},
		// 超出范围
		{"day-24-30", true},   // 小时
		{"day-9-60", true},    // 分钟
		{"week-0-14-00", true},
		{"week-8-14-00", true},
		{"month-0-10-00", true},
		{"month-32-10-00", true},
		// 非整数 / 空元素
		{"day-9-abc", true},
		{"day-9,11-", true},
		{"day-9,,11-30", true},
		{"day--30", true},
		// 空周期段
		{"day-9-30;", true},
		{";", true},
		// 空格
		{"day-9-30 ", true},
		{" day-9-30", true},
		{"day-9,11-30; week-3-14-00", true},
	}
	for _, c := range cases {
		err := ValidateRemindTimer(c.input)
		gotErr := err != nil
		if gotErr != c.wantErr {
			t.Errorf("ValidateRemindTimer(%q) err = %v, wantErr = %v", c.input, err, c.wantErr)
		}
	}
}

func TestValidateStartline(t *testing.T) {
	cases := []struct {
		input   string
		wantErr bool
	}{
		// empty (immediate effect)
		{"", false},
		// positive integer timestamps
		{"1700000000", false},
		{"2147483647", false},
		{"9999999999", false},   // 10 digits, within length limit
		{"99999999999", false},  // 11 digits, at length limit
		// length limit exceeded (12+ chars)
		{"999999999999", true},
		{"2026-08-17T12:00:00", true}, // pasted ISO string
		// not a positive integer
		{"0", true},
		{"-100", true},
		{"abc", true},
	}
	for _, c := range cases {
		err := ValidateStartline(c.input)
		gotErr := err != nil
		if gotErr != c.wantErr {
			t.Errorf("ValidateStartline(%q) err = %v, wantErr = %v", c.input, err, c.wantErr)
		}
	}
}

func TestValidateFileName(t *testing.T) {
	cases := []struct {
		input   string
		wantErr bool
	}{
		// valid names
		{"a.txt", false},
		{"新建文件夹", false},
		{".hidden", false},
		{"a.b.c", false},
		{strings.Repeat("a", 255), false},
		{strings.Repeat("文", 255), false},
		// empty
		{"", true},
		// illegal characters: / \ : * ? " < > |
		{"a/b", true},
		{"a\\b", true},
		{"a:b", true},
		{"a*b", true},
		{"a?b", true},
		{"a\"b", true},
		{"a<b", true},
		{"a>b", true},
		{"a|b", true},
		// length over 255 characters (counted by character, not byte)
		{strings.Repeat("a", 256), true},
		{strings.Repeat("文", 256), true},
		// ending with "."
		{"name.", true},
		{".", true},
		{"..", true},
	}
	for _, c := range cases {
		err := ValidateFileName(c.input)
		gotErr := err != nil
		if gotErr != c.wantErr {
			t.Errorf("ValidateFileName(%q) err = %v, wantErr = %v", c.input, err, c.wantErr)
		}
	}
}

func TestValidateFullpath(t *testing.T) {
	cases := []struct {
		input   string
		wantErr bool
	}{
		// unset
		{"", false},
		// root only
		{"/", false},
		// valid paths
		{"/a.txt", false},
		{"a.txt", false},
		{"/文件夹/文件.txt", false},
		{"documents/report.pdf", false},
		// trailing "/" (e.g. target directory)
		{"/folder/", false},
		{"folder/", false},
		// empty segment in the middle
		{"/a//b", true},
		{"//a", true},
		{"a//", true},
		// illegal characters in any segment
		{"/a?b/c.txt", true},
		{"/a/b|c", true},
		{"/a\\b/c", true},
		// segment ending with "."
		{"/folder./file.txt", true},
		{"/a/./b", true},
		{"/a/../b", true},
		// segment too long
		{"/" + strings.Repeat("a", 256), true},
	}
	for _, c := range cases {
		err := ValidateFullpath(c.input)
		gotErr := err != nil
		if gotErr != c.wantErr {
			t.Errorf("ValidateFullpath(%q) err = %v, wantErr = %v", c.input, err, c.wantErr)
		}
	}
}

func TestValidateFullpaths(t *testing.T) {
	cases := []struct {
		input   string
		wantErr bool
	}{
		// unset
		{"", false},
		// valid lists
		{"/a.txt", false},
		{"/a.txt|/b.txt", false},
		{"/folder/|/b.txt", false},
		{"/a.txt|", false}, // 空元素视为未设置
		// any invalid path fails the whole list
		{"/a.txt|/b?.txt", true},
		{"/ok.txt|/folder./x.txt", true},
	}
	for _, c := range cases {
		err := ValidateFullpaths(c.input)
		gotErr := err != nil
		if gotErr != c.wantErr {
			t.Errorf("ValidateFullpaths(%q) err = %v, wantErr = %v", c.input, err, c.wantErr)
		}
	}
}
