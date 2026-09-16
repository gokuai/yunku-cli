package app

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newTestCommandTree() *cobra.Command {
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().String("global", "", "全局参数")
	root.MarkPersistentFlagRequired("global")

	child := &cobra.Command{Use: "child"}
	child.Flags().String("name", "", "名称")
	child.Flags().String("tag", "", "标签 (必需)")
	child.Flags().String("secret", "", "隐藏参数")
	child.MarkFlagRequired("name")
	child.MarkFlagRequired("secret")
	child.Flags().MarkHidden("secret")
	root.AddCommand(child)

	return root
}

func TestAnnotateRequiredFlagUsagesAppendsMarker(t *testing.T) {
	root := newTestCommandTree()
	annotateRequiredFlagUsages(root)

	child, _, err := root.Find([]string{"child"})
	if err != nil {
		t.Fatalf("find child: %v", err)
	}

	cases := []struct {
		flag     string
		wantTail string
	}{
		{"global", "全局参数" + requiredFlagSuffix}, // persistent required, merged into child flags
		{"name", "名称" + requiredFlagSuffix},       // local required
		{"tag", "标签 (必需)"},                        // existing half-width marker not duplicated
	}
	for _, c := range cases {
		f := child.Flags().Lookup(c.flag)
		if f == nil {
			t.Errorf("flag %q not found", c.flag)
			continue
		}
		if f.Usage != c.wantTail {
			t.Errorf("%s usage = %q, want %q", c.flag, f.Usage, c.wantTail)
		}
	}

	// Hidden required flags are not decorated.
	if f := child.Flags().Lookup("secret"); f != nil && strings.HasSuffix(f.Usage, requiredFlagSuffix) {
		t.Errorf("hidden flag usage should not be decorated: %q", f.Usage)
	}
}

func TestAnnotateRequiredFlagUsagesIsIdempotent(t *testing.T) {
	root := newTestCommandTree()
	annotateRequiredFlagUsages(root)
	annotateRequiredFlagUsages(root)

	child, _, _ := root.Find([]string{"child"})
	f := child.Flags().Lookup("name")
	if strings.Count(f.Usage, requiredFlagSuffix) != 1 {
		t.Errorf("expected exactly one marker, got %q", f.Usage)
	}
}
