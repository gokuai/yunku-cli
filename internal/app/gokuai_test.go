package app

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func newOrgTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "t"}
	cmd.Flags().String("org-id", "", "")
	cmd.Flags().Int("mount-id", 0, "")
	cmd.Flags().String("title", "", "")
	cmd.Flags().String("org-client-id", "", "")
	cmd.Flags().String("org-client-secret", "", "")
	return cmd
}

func TestGetOrgClientReusesLastBindCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YKC_CONFIG_DIR", dir)
	os.Unsetenv("GOKUAI_ORG_CLIENT_ID")
	os.Unsetenv("GOKUAI_ORG_CLIENT_SECRET")

	// 无缓存、无参数 → 报错
	if _, err := getOrgClient(newOrgTestCmd()); err == nil {
		t.Fatal("expected error when no credentials and no cache")
	}

	// 写入未过期的 last_bind 缓存 → 无参数应复用
	cache := orgBindCache{
		lastBindCacheKey: {
			OrgClientID:     "cached-org-id",
			OrgClientSecret: "cached-org-secret",
			ExpiresAt:       time.Now().Add(30 * time.Minute),
		},
	}
	if err := saveOrgBindCache(cache); err != nil {
		t.Fatalf("saveOrgBindCache: %v", err)
	}
	client, err := getOrgClient(newOrgTestCmd())
	if err != nil {
		t.Fatalf("expected cache reuse, got error: %v", err)
	}
	if client.OrgClientID != "cached-org-id" {
		t.Fatalf("got OrgClientID %q, want cached-org-id", client.OrgClientID)
	}

	// 过期的 last_bind → 报错
	cache[lastBindCacheKey] = orgBindCacheEntry{
		OrgClientID: "stale", OrgClientSecret: "stale",
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	_ = saveOrgBindCache(cache)
	if _, err := getOrgClient(newOrgTestCmd()); err == nil {
		t.Fatal("expected error for expired last_bind cache")
	}

	// 显式 org-client 配置优先于缓存
	cmd := newOrgTestCmd()
	_ = cmd.Flags().Set("org-client-id", "explicit-id")
	_ = cmd.Flags().Set("org-client-secret", "explicit-secret")
	client, err = getOrgClient(cmd)
	if err != nil {
		t.Fatalf("explicit config path error: %v", err)
	}
	if client.OrgClientID != "explicit-id" {
		t.Fatalf("got %q, want explicit-id", client.OrgClientID)
	}
}
