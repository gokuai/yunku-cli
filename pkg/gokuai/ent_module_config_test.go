package gokuai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gokuai/yunku-cli/internal/cache"
)

func newTestEntModuleConfig(t *testing.T, client *UserClient, secret string) *EntModuleConfig {
	t.Helper()
	store := cache.NewStore(t.TempDir())
	return NewEntModuleConfigWithStore(client, secret, store)
}

// entListBody 模拟 /m-api/1/account/ent 返回的全部企业列表。
func entListBody(ents ...map[string]any) map[string]any {
	return map[string]any{"list": ents}
}

func TestEntModuleConfigModuleSetting(t *testing.T) {
	var entCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/m-api/1/account/ent":
			atomic.AddInt32(&entCalls, 1)
			if got := r.URL.Query().Get("ent_id"); got != "" {
				t.Errorf("ent_id should not be sent, got %q", got)
			}
			_ = json.NewEncoder(w).Encode(entListBody(
				map[string]any{
					"ent_id":  42,
					"modules": []string{"doc", "mail"},
					"modules_setting": map[string]any{
						"doc": map[string]any{"watermark": true},
					},
				},
			))
		case "/m-api/1/account/mount":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"list": []map[string]any{{"mount_id": 7, "ent_id": 42}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewUserClient("token")
	client.BaseURL = srv.URL
	client.HTTPClient = srv.Client()

	emc := newTestEntModuleConfig(t, client, "secret")

	// 已开启模块返回配置
	setting, err := emc.ModuleSetting(7, "doc")
	if err != nil {
		t.Fatalf("ModuleSetting(doc) returned error: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(setting, &cfg); err != nil {
		t.Fatalf("unmarshal doc setting: %v", err)
	}
	if cfg["watermark"] != true {
		t.Errorf("doc setting = %v, want watermark=true", cfg)
	}

	// 未开启模块抛错
	if _, err := emc.ModuleSetting(7, "billing"); err == nil {
		t.Fatal("ModuleSetting(billing) expected error, got nil")
	}

	// 重复调用命中长期缓存，不再请求接口
	if _, err := emc.ModuleSetting(7, "doc"); err != nil {
		t.Fatalf("second ModuleSetting(doc) returned error: %v", err)
	}
	if got := atomic.LoadInt32(&entCalls); got != 1 {
		t.Errorf("ent API called %d times, want 1 (cached)", got)
	}
}

func TestEntModuleConfigMountEntPersistentCache(t *testing.T) {
	var mountCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/m-api/1/account/ent":
			_ = json.NewEncoder(w).Encode(entListBody(
				map[string]any{
					"ent_id":          42,
					"modules":         []string{"doc"},
					"modules_setting": map[string]any{"doc": map[string]any{"a": 1}},
				},
			))
		case "/m-api/1/account/mount":
			atomic.AddInt32(&mountCalls, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"list": []map[string]any{{"mount_id": 7, "ent_id": 42}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewUserClient("token")
	client.BaseURL = srv.URL
	client.HTTPClient = srv.Client()

	// 两个独立的访问器共享同一份长期缓存，mount->ent 只解析一次
	store := cache.NewStore(t.TempDir())
	emc1 := NewEntModuleConfigWithStore(client, "secret", store)
	emc2 := NewEntModuleConfigWithStore(client, "secret", store)

	if _, err := emc1.ModuleSetting(7, "doc"); err != nil {
		t.Fatalf("emc1.ModuleSetting returned error: %v", err)
	}
	if _, err := emc2.ModuleSetting(7, "doc"); err != nil {
		t.Fatalf("emc2.ModuleSetting returned error: %v", err)
	}

	if got := atomic.LoadInt32(&mountCalls); got != 1 {
		t.Errorf("mount API called %d times, want 1 (persistent cache)", got)
	}
}

func TestEntModuleConfigByEntID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(entListBody(
			map[string]any{
				"ent_id":  42,
				"modules": []string{"doc"},
				"modules_setting": map[string]any{
					"doc": map[string]any{"a": 1},
				},
			},
			map[string]any{
				"ent_id":  99,
				"modules": []string{"other"},
				"modules_setting": map[string]any{
					"other": map[string]any{"b": 2},
				},
			},
		))
	}))
	defer srv.Close()

	client := NewUserClient("token")
	client.BaseURL = srv.URL
	client.HTTPClient = srv.Client()

	emc := newTestEntModuleConfig(t, client, "secret")
	setting, err := emc.ModuleSettingByEntID("42", "doc")
	if err != nil {
		t.Fatalf("ModuleSettingByEntID returned error: %v", err)
	}
	if len(setting) == 0 {
		t.Fatal("expected non-empty setting")
	}
}

func TestEntModuleConfigRequiresIdentifier(t *testing.T) {
	client := NewUserClient("token")
	emc := newTestEntModuleConfig(t, client, "secret")
	if _, err := emc.ModuleSetting(0, "doc"); err == nil {
		t.Fatal("expected error when neither mount_id nor ent_id provided")
	}
}
