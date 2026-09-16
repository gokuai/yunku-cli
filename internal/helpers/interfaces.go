package helpers

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/gokuai/yunku-cli/internal/executor"
	"github.com/spf13/cobra"
)

type Handler interface {
	Name() string
	Command(runner executor.Runner) *cobra.Command
}

type Factory func() Handler

type Manifest struct {
	Vendor      string
	Name        string
	Description string
}

func (m Manifest) FullName() string {
	return strings.TrimSpace(m.Vendor) + "/" + strings.TrimSpace(m.Name)
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

const (
	vendorMinLen = 2
	vendorMaxLen = 30
	nameMinLen   = 2
	nameMaxLen   = 50
)

var (
	registryMu      sync.Mutex
	publicFactories []Factory
)

func RegisterPublic(factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	publicFactories = append(publicFactories, factory)
}

func NewPublicCommands(runner executor.Runner) []*cobra.Command {
	return buildCommands(publicFactories, runner)
}

func buildCommands(factories []Factory, runner executor.Runner) []*cobra.Command {
	registryMu.Lock()
	defer registryMu.Unlock()

	out := make([]*cobra.Command, 0, len(factories))
	for _, factory := range factories {
		handler := factory()
		out = append(out, handler.Command(runner))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Use < out[j].Use
	})
	return out
}

func ValidateNaming(vendor, name string) error {
	if err := validateSegment("vendor", vendor, vendorMinLen, vendorMaxLen); err != nil {
		return err
	}
	if err := validateSegment("name", name, nameMinLen, nameMaxLen); err != nil {
		return err
	}
	return nil
}

func validateSegment(label, value string, minLen, maxLen int) error {
	value = strings.TrimSpace(value)
	if len(value) < minLen || len(value) > maxLen {
		return fmt.Errorf("%s %q length must be %d-%d, got %d", label, value, minLen, maxLen, len(value))
	}
	if !namePattern.MatchString(value) {
		return fmt.Errorf("%s %q must be kebab-case (a-z0-9-), starting with a letter", label, value)
	}
	if strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
		return fmt.Errorf("%s %q must not start or end with a hyphen", label, value)
	}
	return nil
}
