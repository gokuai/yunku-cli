package gokuai

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/gokuai/yunku-cli/internal/cache"
	apperrors "github.com/gokuai/yunku-cli/internal/errors"
)

// AccountEntItem 是 /m-api/1/account/ent 返回的单个企业条目。
//
// Modules 为已开启模块的名称列表；ModulesSetting 为已开启模块对应的配置，
// key 为模块名，value 为该模块的配置（原始 JSON，调用方可自行反序列化）。
type AccountEntItem struct {
	EntID          FlexInt                    `json:"ent_id"`
	Modules        []string                   `json:"modules"`
	ModulesSetting map[string]json.RawMessage `json:"modules_setting"`
}

// AccountEntListResponse 是 /m-api/1/account/ent 的响应。
// 该接口忽略 ent_id 参数，始终返回当前账号所属的全部企业。
type AccountEntListResponse struct {
	List []AccountEntItem `json:"list"`
}

// GetAccountEnt 获取企业模块配置列表 (用户 API, GET /m-api/1/account/ent)。
// 注意：ent_id 参数无效，接口返回全部企业，需在结果中按 ent_id 自行筛选。
func (c *UserClient) GetAccountEnt(secret string) (*AccountEntListResponse, error) {
	params := map[string]string{
		"dateline": fmt.Sprintf("%d", time.Now().Unix()),
	}

	body, err := c.doUserRequest("/m-api/1/account/ent", params, secret)
	if err != nil {
		return nil, err
	}

	if err := checkAPIError(body); err != nil {
		return nil, err
	}

	var resp AccountEntListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	return &resp, nil
}

// entModuleConfigPartition 是企业模块配置长期缓存的分区。
const entModuleConfigPartition = "ent_module_config"

// EntModuleConfig 通过 mount_id 或 ent_id 获取企业模块配置，并对结果做长期缓存。
//
// - 企业模块配置缓存全部企业，有效期 1 小时（cache.EntModuleConfigTTL）。
// - mount_id -> ent_id 映射为长期缓存，不设有效期，仅在找不到时才重新获取。
//
// 提供「给定 mount_id + 模块名返回该模块配置，模块未开启则报错」的能力。
// 由于 /m-api/1/account/ent 返回全部企业，传入 mount_id 时会先解析出 ent_id
// 再在结果列表中定位对应企业。
type EntModuleConfig struct {
	client    *UserClient
	secret    string
	store     *cache.Store
	partition string
}

// NewEntModuleConfig 创建一个企业模块配置访问器，使用默认长期缓存（~/.ykc/cache）。
func NewEntModuleConfig(client *UserClient, secret string) *EntModuleConfig {
	return &EntModuleConfig{
		client:    client,
		secret:    secret,
		store:     cache.NewStore(""),
		partition: entModuleConfigPartition,
	}
}

// NewEntModuleConfigWithStore 创建一个使用指定缓存存储的企业模块配置访问器。
func NewEntModuleConfigWithStore(client *UserClient, secret string, store *cache.Store) *EntModuleConfig {
	emc := NewEntModuleConfig(client, secret)
	if store != nil {
		emc.store = store
	}
	return emc
}

// Get 获取指定企业的模块配置。mountID 与 entID 二选一，同时传入时优先使用 entID。
// 结果按全部企业列表做长期缓存，1 小时内直接返回缓存。
func (e *EntModuleConfig) Get(mountID int, entID string) (*AccountEntItem, error) {
	key, err := e.resolveEntKey(mountID, entID)
	if err != nil {
		return nil, err
	}

	list, err := e.list()
	if err != nil {
		return nil, err
	}

	for i := range list {
		if fmt.Sprintf("%d", list[i].EntID) == key {
			return &list[i], nil
		}
	}

	return nil, apperrors.NewValidation(fmt.Sprintf("未找到 ent_id=%s 对应的企业", key))
}

// ModuleSetting 返回指定模块的配置；若模块未开启则报错。
// 模块已开启但无对应配置时返回 nil（不报错）。
func (e *EntModuleConfig) ModuleSetting(mountID int, moduleName string) (json.RawMessage, error) {
	return e.moduleSetting(mountID, "", moduleName)
}

// ModuleSettingByEntID 通过 ent_id 返回指定模块的配置；若模块未开启则报错。
func (e *EntModuleConfig) ModuleSettingByEntID(entID, moduleName string) (json.RawMessage, error) {
	return e.moduleSetting(0, entID, moduleName)
}

func (e *EntModuleConfig) moduleSetting(mountID int, entID, moduleName string) (json.RawMessage, error) {
	if moduleName == "" {
		return nil, apperrors.NewValidation("模块名不能为空")
	}

	item, err := e.Get(mountID, entID)
	if err != nil {
		return nil, err
	}

	if !slices.Contains(item.Modules, moduleName) {
		return nil, apperrors.NewValidation(
			fmt.Sprintf("模块 %q 未开启", moduleName),
			apperrors.WithReason("module_not_enabled"),
		)
	}

	return item.ModulesSetting[moduleName], nil
}

// list 获取全部企业模块配置，优先读长期缓存，过期或缺失时重新请求。
func (e *EntModuleConfig) list() ([]AccountEntItem, error) {
	if snapshot, freshness, err := e.store.LoadEntModuleConfig(e.partition); err == nil && freshness == cache.FreshnessFresh {
		var resp AccountEntListResponse
		if err := json.Unmarshal(snapshot.Payload, &resp); err == nil {
			return resp.List, nil
		}
	}

	resp, err := e.client.GetAccountEnt(e.secret)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	_ = e.store.SaveEntModuleConfig(e.partition, cache.EntModuleConfigSnapshot{
		SavedAt: time.Now().UTC(),
		Payload: payload,
	})

	return resp.List, nil
}

// resolveEntKey 归一化出 ent_id：entID 非空则直接使用；否则由 mountID 解析。
func (e *EntModuleConfig) resolveEntKey(mountID int, entID string) (string, error) {
	if entID != "" {
		return entID, nil
	}
	if mountID <= 0 {
		return "", apperrors.NewValidation("需提供 mount_id 或 ent_id")
	}
	return e.resolveEntIDByMount(mountID)
}

// resolveEntIDByMount 把 mount_id 解析为 ent_id。
// 优先读取长期缓存，仅在找不到时才调用库列表接口获取。
func (e *EntModuleConfig) resolveEntIDByMount(mountID int) (string, error) {
	if snapshot, err := e.store.LoadMountEnt(e.partition, mountID); err == nil && snapshot.EntID != 0 {
		return fmt.Sprintf("%d", snapshot.EntID), nil
	}

	resp, err := e.client.GetMountList(e.secret)
	if err != nil {
		return "", err
	}
	for _, m := range resp.List {
		if m.MountID == mountID {
			_ = e.store.SaveMountEnt(e.partition, mountID, cache.MountEntSnapshot{
				SavedAt: time.Now().UTC(),
				EntID:   m.EntID,
			})
			return fmt.Sprintf("%d", m.EntID), nil
		}
	}

	return "", apperrors.NewValidation(fmt.Sprintf("未找到 mount_id=%d 对应的企业", mountID))
}
