package cobracmd

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const overridePriorityAnnotation = "ykc.override-priority"

// SetOverridePriority sets the override priority annotation on cmd.
func SetOverridePriority(cmd *cobra.Command, priority int) {
	if cmd == nil {
		return
	}
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[overridePriorityAnnotation] = strconv.Itoa(priority)
}

// OverridePriority returns the override priority annotation value, or 0.
func OverridePriority(cmd *cobra.Command) int {
	if cmd == nil || cmd.Annotations == nil {
		return 0
	}
	raw := strings.TrimSpace(cmd.Annotations[overridePriorityAnnotation])
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}
