package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	RegistryTTL     = 24 * time.Hour
	ToolsTTL        = 7 * 24 * time.Hour
	DetailTTL       = 7 * 24 * time.Hour
	RevalidateAfter = 1 * time.Hour

	// EntModuleConfigTTL 是企业模块配置长期缓存的有效期。
	EntModuleConfigTTL = 1 * time.Hour
)

type Store struct {
	Root string
	Now  func() time.Time
}

type RegistrySnapshot struct {
	SavedAt time.Time `json:"saved_at"`
}

type ToolsSnapshot struct {
	SavedAt time.Time `json:"saved_at"`
}

type DetailSnapshot struct {
	SavedAt time.Time       `json:"saved_at"`
	MCPID   int             `json:"mcp_id"`
	Payload json.RawMessage `json:"payload"`
}

// EntModuleConfigSnapshot 是企业模块配置的缓存快照，Payload 为
// AccountEntListResponse 的原始 JSON（缓存全部企业，而非单个）。
type EntModuleConfigSnapshot struct {
	SavedAt time.Time       `json:"saved_at"`
	Payload json.RawMessage `json:"payload"`
}

// MountEntSnapshot 是 mount_id -> ent_id 映射的缓存快照。
// 该映射为长期缓存，不校验新鲜度，找不到时才重新获取。
type MountEntSnapshot struct {
	SavedAt time.Time `json:"saved_at"`
	EntID   int       `json:"ent_id"`
}

func NewStore(root string) *Store {
	if strings.TrimSpace(root) == "" {
		root = defaultCacheRoot()
	}
	return &Store{
		Root: root,
		Now:  time.Now,
	}
}

func defaultCacheRoot() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".ykc", "cache")
	}
	return filepath.Join(os.TempDir(), "ykc-cache")
}

func (s *Store) SaveRegistry(partition string, snapshot RegistrySnapshot) error {
	if snapshot.SavedAt.IsZero() {
		snapshot.SavedAt = s.Now().UTC()
	}
	return s.saveJSON(s.registryPath(partition), snapshot)
}

func (s *Store) LoadRegistry(partition string) (RegistrySnapshot, Freshness, error) {
	var snapshot RegistrySnapshot
	if err := s.loadJSON(s.registryPath(partition), &snapshot); err != nil {
		return RegistrySnapshot{}, "", err
	}
	return snapshot, freshness(s.Now().UTC(), snapshot.SavedAt, RegistryTTL), nil
}

func (s *Store) DeleteTools(partition, serverKey string) error {
	path := s.toolsPath(partition, serverKey)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Store) SaveDetail(partition, serverKey string, snapshot DetailSnapshot) error {
	if snapshot.SavedAt.IsZero() {
		snapshot.SavedAt = s.Now().UTC()
	}
	return s.saveJSON(s.detailPath(partition, serverKey), snapshot)
}

func (s *Store) LoadDetail(partition, serverKey string) (DetailSnapshot, Freshness, error) {
	var snapshot DetailSnapshot
	if err := s.loadJSON(s.detailPath(partition, serverKey), &snapshot); err != nil {
		return DetailSnapshot{}, "", err
	}
	return snapshot, freshness(s.Now().UTC(), snapshot.SavedAt, DetailTTL), nil
}

func (s *Store) DeleteDetail(partition, serverKey string) error {
	path := s.detailPath(partition, serverKey)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// SaveEntModuleConfig 保存企业模块配置缓存（缓存全部企业）。
func (s *Store) SaveEntModuleConfig(partition string, snapshot EntModuleConfigSnapshot) error {
	if snapshot.SavedAt.IsZero() {
		snapshot.SavedAt = s.Now().UTC()
	}
	return s.saveJSON(s.entModuleConfigPath(partition), snapshot)
}

// LoadEntModuleConfig 读取企业模块配置缓存，并返回新鲜度（TTL 见 EntModuleConfigTTL）。
func (s *Store) LoadEntModuleConfig(partition string) (EntModuleConfigSnapshot, Freshness, error) {
	var snapshot EntModuleConfigSnapshot
	if err := s.loadJSON(s.entModuleConfigPath(partition), &snapshot); err != nil {
		return EntModuleConfigSnapshot{}, "", err
	}
	return snapshot, freshness(s.Now().UTC(), snapshot.SavedAt, EntModuleConfigTTL), nil
}

// SaveMountEnt 保存 mount_id -> ent_id 映射缓存。
func (s *Store) SaveMountEnt(partition string, mountID int, snapshot MountEntSnapshot) error {
	if snapshot.SavedAt.IsZero() {
		snapshot.SavedAt = s.Now().UTC()
	}
	return s.saveJSON(s.mountEntPath(partition, mountID), snapshot)
}

// LoadMountEnt 读取 mount_id -> ent_id 映射。长期缓存，不校验新鲜度，
// 找不到时返回错误由调用方重新获取。
func (s *Store) LoadMountEnt(partition string, mountID int) (MountEntSnapshot, error) {
	var snapshot MountEntSnapshot
	if err := s.loadJSON(s.mountEntPath(partition, mountID), &snapshot); err != nil {
		return MountEntSnapshot{}, err
	}
	return snapshot, nil
}

func (s *Store) registryPath(partition string) string {
	return filepath.Join(s.Root, sanitize(partition), "registry.json")
}

func (s *Store) toolsPath(partition, serverKey string) string {
	return filepath.Join(s.Root, sanitize(partition), "tools", sanitize(serverKey)+".json")
}

func (s *Store) detailPath(partition, serverKey string) string {
	return filepath.Join(s.Root, sanitize(partition), "detail", sanitize(serverKey)+".json")
}

func (s *Store) entModuleConfigPath(partition string) string {
	return filepath.Join(s.Root, sanitize(partition), "ent_module.json")
}

func (s *Store) mountEntPath(partition string, mountID int) string {
	return filepath.Join(s.Root, sanitize(partition), "mount_ent", strconv.Itoa(mountID)+".json")
}

func (s *Store) saveJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	writeSuccess := false
	defer func() {
		if !writeSuccess {
			tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	writeSuccess = true
	return nil
}

func (s *Store) loadJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func freshness(now, savedAt time.Time, ttl time.Duration) Freshness {
	if savedAt.IsZero() || now.Sub(savedAt) > ttl {
		return FreshnessStale
	}
	return FreshnessFresh
}

func ShouldRevalidate(now, savedAt time.Time) bool {
	if savedAt.IsZero() {
		return true
	}
	return now.Sub(savedAt) >= RevalidateAfter
}

// Freshness indicates whether cached data is fresh or stale.
type Freshness string

const (
	FreshnessFresh Freshness = "fresh"
	FreshnessStale Freshness = "stale"
)

func sanitize(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	return replacer.Replace(value)
}

func IsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

// ToolsCacheEntry represents a cached tools entry with freshness information.
type ToolsCacheEntry struct {
	ServerKey string
	Freshness Freshness
	SavedAt   time.Time
}

// ListToolsCacheEntries returns all cached tools entries for a partition.
func (s *Store) ListToolsCacheEntries(partition string) ([]ToolsCacheEntry, error) {
	dir := filepath.Join(s.Root, sanitize(partition), "tools")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	result := make([]ToolsCacheEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		serverKey := strings.TrimSuffix(entry.Name(), ".json")
		path := filepath.Join(dir, entry.Name())

		var snapshot ToolsSnapshot
		if err := s.loadJSON(path, &snapshot); err != nil {
			continue
		}

		result = append(result, ToolsCacheEntry{
			ServerKey: serverKey,
			Freshness: freshness(s.Now().UTC(), snapshot.SavedAt, ToolsTTL),
			SavedAt:   snapshot.SavedAt,
		})
	}
	return result, nil
}
