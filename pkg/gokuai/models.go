package gokuai

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FlexInt is an int type that can be unmarshaled from either a JSON number or a JSON string.
// The GoKuai API sometimes returns numeric fields as strings (e.g. "123" instead of 123).
type FlexInt int

func (fi *FlexInt) UnmarshalJSON(data []byte) error {
	// Try int first
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	s := n.String()
	parsed, err := strconv.Atoi(s)
	if err != nil {
		*fi = 0
		return nil
	}
	*fi = FlexInt(parsed)
	return nil
}

// FlexInt64 is an int64 type that can be unmarshaled from either a JSON number or a JSON string.
// The GoKuai API sometimes returns numeric fields as strings (e.g. "45979150" instead of 45979150).
type FlexInt64 int64

func (fi *FlexInt64) UnmarshalJSON(data []byte) error {
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	s := n.String()
	parsed, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		*fi = 0
		return nil
	}
	*fi = FlexInt64(parsed)
	return nil
}

// FlexStringMap is a map[string]string that tolerates the API returning an
// empty JSON array ([]) for an empty map (common with PHP backends).
// Empty arrays and null unmarshal to an empty non-nil map so output renders
// as {} rather than failing or rendering as null.
type FlexStringMap map[string]string

func (m *FlexStringMap) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == "" || s == "null" || s == "[]" {
		*m = FlexStringMap{}
		return nil
	}
	var mm map[string]string
	if err := json.Unmarshal(data, &mm); err != nil {
		return err
	}
	if mm == nil {
		mm = map[string]string{}
	}
	*m = mm
	return nil
}

// EmbeddedJSON holds a JSON value that the API returns as a JSON-encoded
// string (e.g. "{\"k\":\"v\"}"). On unmarshaling the embedded string is
// decoded into raw JSON; on marshaling it is emitted as a nested JSON value
// rather than a quoted string. Values that are already raw JSON, null, or
// empty are also handled.
type EmbeddedJSON json.RawMessage

// UnmarshalJSON accepts either a JSON string containing encoded JSON or a
// raw JSON value.
func (e *EmbeddedJSON) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*e = nil
		return nil
	}

	// Most API responses wrap the object as a string.
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			*e = nil
			return nil
		}
		if json.Valid([]byte(s)) {
			*e = EmbeddedJSON([]byte(s))
			return nil
		}
		// Not valid JSON: fall back to emitting the plain string.
		enc, err := json.Marshal(s)
		if err != nil {
			return err
		}
		*e = EmbeddedJSON(enc)
		return nil
	}

	// Already a raw JSON value (object, array, number, etc.).
	if !json.Valid(data) {
		return fmt.Errorf("invalid JSON for EmbeddedJSON: %s", string(data))
	}
	*e = EmbeddedJSON(append([]byte(nil), data...))
	return nil
}

// MarshalJSON emits the embedded value as JSON instead of a quoted string.
func (e EmbeddedJSON) MarshalJSON() ([]byte, error) {
	if len(e) == 0 {
		return []byte(`""`), nil
	}
	return []byte(e), nil
}

// FileInfo represents file information
type FileInfo struct {
	Hash             string          `json:"hash"`
	Dir              int             `json:"dir"`
	Fullpath         string          `json:"fullpath"`
	Filename         string          `json:"filename"`
	Filehash         string          `json:"filehash"`
	Filesize         FlexInt64       `json:"filesize"`
	CreateMemberName string          `json:"create_member_name"`
	CreateDateline   FlexInt64       `json:"create_dateline"`
	LastMemberName   string          `json:"last_member_name"`
	LastDateline     FlexInt64       `json:"last_dateline"`
	Property         json.RawMessage `json:"property,omitempty"`
}

// FileListResponse is the response for file list API
type FileListResponse struct {
	Count int        `json:"count"`
	List  []FileInfo `json:"list"`
}

// FileUpdateResponse is the response for file updates API
type FileUpdateResponse struct {
	FetchDateline int64      `json:"fetch_dateline"`
	List          []FileInfo `json:"list"`
}

// DownloadURLResponse is the response for download URL API
type DownloadURLResponse struct {
	URLs []string `json:"urls"`
}

// PreviewURLResponse is the response for preview URL API
type PreviewURLResponse struct {
	URL       string `json:"url"`
	Thumbnail string `json:"thumbnail,omitempty"`
}

// CeditURLResponse is the response for collaborative edit URL API
type CeditURLResponse struct {
	URL string `json:"url"`
}

// FileInfoResponse is the response for file info API
type FileInfoResponse struct {
	Hash             string `json:"hash"`
	Dir              int    `json:"dir"`
	Fullpath         string `json:"fullpath"`
	Filename         string `json:"filename"`
	Filesize         int64  `json:"filesize"`
	Filehash         string `json:"filehash,omitempty"`
	CreateMemberName string `json:"create_member_name"`
	CreateDateline   int64  `json:"create_dateline"`
	LastMemberName   string `json:"last_member_name"`
	LastDateline     int64  `json:"last_dateline"`
	URI              string `json:"uri,omitempty"`
	Preview          string `json:"preview,omitempty"`
	Thumbnail        string `json:"thumbnail,omitempty"`
	Tag              string `json:"tag,omitempty"`
	FileCount        int    `json:"file_count,omitempty"`
	FolderCount      int    `json:"folder_count,omitempty"`
	FilesSize        int64  `json:"files_size,omitempty"`
	Property         *struct {
		Tag        string `json:"tag"`
		OpName     string `json:"op_name"`
		Inherit    int    `json:"inherit"`
		Permission string `json:"permission"`
	} `json:"property,omitempty"`
}

// CreateFolderResponse is the response for create folder API
type CreateFolderResponse struct {
	Hash     string `json:"hash"`
	Fullpath string `json:"fullpath"`
}

// CreateFileResponse is the response for create file API
type CreateFileResponse struct {
	Hash     string `json:"hash"`
	Fullpath string `json:"fullpath"`
	State    int    `json:"state"`
	Server   string `json:"server"`
}

// CopyResponse is the response for copy API
type CopyResponse struct {
	Hash     string `json:"hash"`
	Fullpath string `json:"fullpath"`
	QueueID  string `json:"queue_id,omitempty"`
}

// MCopyResponse is the response for multi-copy API
type MCopyResponse struct {
	Hash     string `json:"hash"`
	Fullpath string `json:"fullpath"`
	QueueID  string `json:"queue_id,omitempty"`
}

// LinkResponse is the response for share link API
type LinkResponse struct {
	Link string `json:"link"`
	Code string `json:"code"`
}

// FileLinkItem represents a single file link entry
type FileLinkItem struct {
	URL           string        `json:"url"`
	QRURL         string        `json:"qr_url"`
	MountID       string        `json:"mount_id"`
	EntID         string        `json:"ent_id"`
	Hash          string        `json:"hash"`
	Type          string        `json:"type"`
	SettingURL    string        `json:"setting_url"`
	Code          string        `json:"code"`
	MemberID      string        `json:"member_id"`
	MemberName    string        `json:"member_name"`
	VisitCount    FlexInt       `json:"visit_count"`
	DownloadCount FlexInt       `json:"download_count"`
	Deadline      string        `json:"deadline"`
	Dateline      string        `json:"dateline"`
	Property      *LinkProperty `json:"property,omitempty"`
}

// LinkProperty represents the settings/property of a file link
type LinkProperty struct {
	Scope       string  `json:"scope"`
	Preview     string  `json:"preview"`
	Download    string  `json:"download"`
	Upload      string  `json:"upload"`
	Password    string  `json:"password"`
	Day         FlexInt `json:"day"`
	Emails      string  `json:"emails,omitempty"`
	Content     string  `json:"content,omitempty"`
	ClientID    string  `json:"client_id,omitempty"`
	Anonymity   string  `json:"anonymity,omitempty"`
	Startline   FlexInt `json:"startline,omitempty"`
	V           string  `json:"v,omitempty"`
	AccessLimit FlexInt `json:"access_limit,omitempty"`
	Edit        string  `json:"edit,omitempty"`
}

// FileLinkListResponse is the response for file_link_list API
type FileLinkListResponse struct {
	Owner []FileLinkItem `json:"owner"`
	Other interface{}    `json:"other"` // array or false
}

// LinksListResponse is the response for links list API
type LinksListResponse struct {
	Filename string `json:"filename"`
	Filesize int64  `json:"filesize"`
	Link     string `json:"link"`
	Deadline int64  `json:"deadline"`
	Password int    `json:"password"`
}

// HistoryItem represents a file history entry
type HistoryItem struct {
	HID        string          `json:"hid"`
	Act        int             `json:"act"`
	ActName    string          `json:"act_name"`
	Dir        int             `json:"dir"`
	Hash       string          `json:"hash"`
	Fullpath   string          `json:"fullpath"`
	Filename   string          `json:"filename"`
	Filehash   string          `json:"filehash"`
	Filesize   int64           `json:"filesize"`
	MemberID   int             `json:"member_id"`
	MemberName string          `json:"member_name"`
	Dateline   int64           `json:"dateline"`
	Property   json.RawMessage `json:"property,omitempty"`
}

// HistoryResponse is the response for history API
type HistoryResponse struct {
	Count int           `json:"count"`
	List  []HistoryItem `json:"list"`
}

// QueueStatusResponse is the response for queue status API
type QueueStatusResponse struct {
	Status         int    `json:"status"`
	Fullpath       string `json:"fullpath"`
	AddDateline    int64  `json:"add_dateline"`
	StartDateline  int64  `json:"start_dateline,omitempty"`
	FinishDateline int64  `json:"finish_dateline,omitempty"`
}

// StatResponse is the response for stat API
type StatResponse struct {
	OrgID        int    `json:"org_id"`
	OrgName      string `json:"org_name"`
	MountID      int    `json:"mount_id"`
	Capacity     int64  `json:"capacity"`
	Size         int64  `json:"size"`
	SizeRecycle  int64  `json:"size_recycle"`
	CountFile    int    `json:"count_file"`
	StoragePoint string `json:"storage_point"`
}

// RecycleResponse is the response for recycle list API
type RecycleResponse struct {
	Count int        `json:"count"`
	List  []FileInfo `json:"list"`
}

// Permission represents file permissions
type Permission struct {
	Members map[string][]string `json:"members"`
	Groups  map[string][]string `json:"groups"`
}

// TagResponse is the response for tag operations
type TagResponse struct {
	Result int    `json:"result"`
	Tag    string `json:"tag,omitempty"`
}

// MetadataResponse is the response for metadata operations
type MetadataResponse struct {
	Result int `json:"result"`
}

// UploadServersResponse is the response for upload servers API
type UploadServersResponse struct {
	UploadServers []string `json:"m-upload"`
	Key           string   `json:"key"`
}

// === Enterprise APIs ===

// MemberInfo represents a member
type MemberInfo struct {
	MemberID    FlexInt `json:"member_id"`
	OutID       string  `json:"out_id,omitempty"`
	Account     string  `json:"account,omitempty"`
	MemberName  string  `json:"member_name"`
	MemberEmail string  `json:"member_email,omitempty"`
	MemberPhone string  `json:"member_phone,omitempty"`
	MemberTitle string  `json:"member_title,omitempty"`
	State       FlexInt `json:"state"`
	Expire      FlexInt `json:"expire,omitempty"`
	RoleID      FlexInt `json:"role_id,omitempty"`
}

// MemberListResponse is the response for member list API
type MemberListResponse struct {
	List  []MemberInfo `json:"list"`
	Count FlexInt      `json:"count"`
}

// MemberDetailResponse is the response for member detail API
type MemberDetailResponse struct {
	MemberID    FlexInt       `json:"member_id"`
	OutID       string        `json:"out_id"`
	Account     string        `json:"account"`
	MemberName  string        `json:"member_name"`
	MemberEmail string        `json:"member_email"`
	MemberPhone string        `json:"member_phone"`
	MemberTitle string        `json:"member_title"`
	State       FlexInt       `json:"state"`
	Expire      FlexInt       `json:"expire"`
	Groups      []MemberGroup `json:"groups,omitempty"`
	Orgs        []MemberOrg   `json:"orgs,omitempty"`
}

// MemberGroup represents a member's group
type MemberGroup struct {
	ID        FlexInt `json:"id"`
	Name      string  `json:"name"`
	GroupCode string  `json:"group_code"`
	OutID     string  `json:"out_id"`
}

// MemberOrg represents a member's organization
type MemberOrg struct {
	ID          FlexInt           `json:"id"`
	Name        string            `json:"name"`
	Type        FlexInt           `json:"type"`
	Owner       string            `json:"owner"`
	OwnerID     FlexInt           `json:"owner_id"`
	MemberCount FlexInt           `json:"member_count"`
	SizeTotal   FlexInt           `json:"size_total"`
	SizeUsed    FlexInt           `json:"size_used"`
	MountID     FlexInt           `json:"mount_id"`
	Roles       map[string]string `json:"roles"`
}

// GroupInfo represents a group/department
type GroupInfo struct {
	ID       FlexInt `json:"id,omitempty"`
	GroupID  FlexInt `json:"group_id,omitempty"`
	Name     string  `json:"name"`
	OutID    string  `json:"out_id"`
	ParentID FlexInt `json:"parent_id"`
}

// GroupListResponse is the response for group list API
type GroupListResponse []GroupInfo

// GroupMemberListResponse is the response for group member list API
type GroupMemberListResponse struct {
	List  []MemberInfo `json:"list"`
	Count FlexInt      `json:"count"`
}

// RoleInfo represents a role
type RoleInfo struct {
	ID   FlexInt `json:"id"`
	Name string  `json:"name"`
}

// RolesResponse is the response for roles API
type RolesResponse []RoleInfo

// SyncMemberResponse is the response for sync member operations
type SyncMemberResponse struct {
	MemberID FlexInt `json:"member_id"`
	State    FlexInt `json:"state"`
}

// AddGroupResponse is the response for add group API
type AddGroupResponse struct {
	GroupID FlexInt `json:"group_id"`
}

// ClientOAuthResponse is the response for get client oauth API
type ClientOAuthResponse struct {
	Result       FlexInt `json:"result"`
	ClientID     string  `json:"client_id"`
	ClientSecret string  `json:"client_secret"`
	Msg          string  `json:"msg,omitempty"`
}

// === User API Models ===

// MountInfo represents a mount/library for user API
type MountInfo struct {
	EntID           int    `json:"ent_id"`
	OrgID           int    `json:"org_id"`
	OrgName         string `json:"org_name"`
	MountID         int    `json:"mount_id"`
	OrgType         int    `json:"org_type"`
	MemberCount     int    `json:"member_count"`
	OwnerMemberID   int    `json:"owner_member_id"`
	OwnerMemberName string `json:"owner_member_name"`
	OrgLogoURL      string `json:"org_logo_url"`
	SizeTotal       int64  `json:"size_total"`
	SizeOrgTotal    int64  `json:"size_org_total"`
	SizeOrgUse      int64  `json:"size_org_use"`
	SizeUse         int64  `json:"size_use"`
	StoragePoint    string `json:"storage_point"`
	StorageEthernet int    `json:"storage_ethernet"`
	MemberName      string `json:"member_name"`
	MemberID        int    `json:"member_id"`
}

// MountListResponse is the response for user mount list API
type MountListResponse struct {
	List []MountInfo `json:"list"`
}

// === Library APIs ===

// OrgInfo represents an organization/library
type OrgInfo struct {
	OrgID      int    `json:"org_id"`
	OrgName    string `json:"org_name"`
	OrgLogoURL string `json:"org_logo_url"`
	SizeTotal  int64  `json:"size_org_total"`
	SizeUsed   int64  `json:"size_org_use"`
	FileCount  int    `json:"file_count"`
	DirCount   int    `json:"dir_count"`
	MountID    int    `json:"mount_id"`
	OwnerID    int    `json:"owner_id"`
}

// OrgInfoResponse is the response for org info API
type OrgInfoResponse struct {
	Info OrgInfo `json:"info"`
}

// OrgListResponse is the response for org list API
type OrgListResponse struct {
	List []OrgInfo `json:"list"`
}

// OrgBindResponse is the response for org bind API
type OrgBindResponse struct {
	OrgClientID     string `json:"org_client_id"`
	OrgClientSecret string `json:"org_client_secret"`
}

// OrgMembersResponse is the response for org members API
type OrgMembersResponse struct {
	List  []MemberInfo `json:"list"`
	Count int          `json:"count"`
}

// OrgGroupsResponse is the response for org groups API
type OrgGroupsResponse struct {
	List []struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		RoleID int    `json:"role_id"`
	} `json:"list"`
}

// CreateOrgResponse is the response for create org API
type CreateOrgResponse struct {
	OrgID   int `json:"org_id"`
	MountID int `json:"mount_id"`
}

// OrgLogResponse is the response for org log API
type OrgLogResponse struct {
	Total int          `json:"total"`
	List  []OrgLogItem `json:"list"`
}

// OrgLogItem represents an org log entry
type OrgLogItem struct {
	Hash          string `json:"hash"`
	Dir           int    `json:"dir"`
	Act           int    `json:"act"`
	Filehash      string `json:"filehash"`
	Filesize      int64  `json:"filesize"`
	Fullpath      string `json:"fullpath"`
	MemberID      int    `json:"member_id"`
	Dateline      int64  `json:"dateline"`
	ActName       string `json:"act_name"`
	MemberName    string `json:"member_name"`
	DisplayName   string `json:"display_name"`
	MemberAccount string `json:"member_account"`
}

// EntLogResponse is the response for ent log API
type EntLogResponse struct {
	Total FlexInt      `json:"total"`
	List  []EntLogItem `json:"list"`
}

// FileHistoryResponseV2 is the response for file history API v2
type FileHistoryResponseV2 struct {
	Count int               `json:"count"`
	List  []FileHistoryItem `json:"list"`
}

// FileHistoryItem represents a file history entry for v2 API
type FileHistoryItem struct {
	HID          string          `json:"hid"`
	EntID        FlexInt         `json:"ent_id"`
	MountID      FlexInt         `json:"mount_id"`
	Dir          FlexInt         `json:"dir"`
	Fullpath     string          `json:"fullpath"`
	Filename     string          `json:"filename"`
	Type         FlexInt         `json:"type"`
	Act          FlexInt         `json:"act"`
	Hash         string          `json:"hash"`
	Filehash     string          `json:"filehash"`
	Filesize     FlexInt64       `json:"filesize"`
	Version      FlexInt         `json:"version"`
	MemberID     FlexInt         `json:"member_id"`
	Dateline     FlexInt64       `json:"dateline"`
	DatelineMS   FlexInt64       `json:"datelinems"`
	Del          bool            `json:"del"`
	ActName      string          `json:"act_name"`
	MemberName   string          `json:"member_name"`
	DisplayName  string          `json:"display_name"`
	OrgName      string          `json:"org_name"`
	StoragePoint string          `json:"storage_point"`
	Property     json.RawMessage `json:"property,omitempty"`
}

// QueueIDResponse is the response for queue operations
type QueueIDResponse struct {
	QueueID string `json:"queue_id,omitempty"`
}

// EntLogItem represents an ent log entry
type EntLogItem struct {
	Type     FlexInt `json:"type"`
	Admin    string  `json:"admin"`
	Content  string  `json:"content"`
	IP       string  `json:"ip"`
	Dateline FlexInt `json:"dateline"`
}

// ExportURLResponse is the response for export URL API
type ExportURLResponse struct {
	URL       string                 `json:"url"`
	Path      string                 `json:"path"`
	Query     map[string]interface{} `json:"query"`
	ExportMsg *struct {
		Filename string `json:"filename"`
		WM       string `json:"wm"`
	} `json:"export_msg"`
}

// === Enterprise Library (Org) API Models ===

// EntOrgInfo represents enterprise library info
type EntOrgInfo struct {
	OrgID        FlexInt `json:"org_id"`
	OrgName      string  `json:"org_name"`
	OrgLogoURL   string  `json:"org_logo_url"`
	SizeOrgTotal int64   `json:"size_org_total"`
	SizeOrgUse   int64   `json:"size_org_use"`
	FileCount    FlexInt `json:"file_count"`
	DirCount     FlexInt `json:"dir_count"`
	MountID      FlexInt `json:"mount_id"`
	OwnerID      FlexInt `json:"owner_id"`
}

// EntOrgInfoResponse is the response for enterprise org info API
type EntOrgInfoResponse struct {
	Info EntOrgInfo `json:"info"`
}

// EntOrgListResponse is the response for enterprise org list API
type EntOrgListResponse struct {
	List []EntOrgInfo `json:"list"`
}

// EntOrgCreateResponse is the response for enterprise org create API
type EntOrgCreateResponse struct {
	OrgID   FlexInt `json:"org_id"`
	MountID FlexInt `json:"mount_id"`
}

// EntOrgBindResponse is the response for enterprise org bind API
type EntOrgBindResponse struct {
	OrgClientID     string `json:"org_client_id"`
	OrgClientSecret string `json:"org_client_secret"`
}

// EntOrgMemberInfo represents a member in an enterprise library
type EntOrgMemberInfo struct {
	MemberID    FlexInt   `json:"member_id"`
	OutID       string    `json:"out_id"`
	Account     string    `json:"account"`
	MemberName  string    `json:"member_name"`
	MemberEmail string    `json:"member_email"`
	State       FlexInt   `json:"state"`
	RoleID      FlexInt   `json:"role_id"`
	RoleIDs     []FlexInt `json:"role_ids"`
}

// EntOrgMembersResponse is the response for enterprise org members API
type EntOrgMembersResponse struct {
	List  []EntOrgMemberInfo `json:"list"`
	Count FlexInt            `json:"count"`
}

// EntOrgGroupInfo represents a department in an enterprise library
type EntOrgGroupInfo struct {
	ID     FlexInt `json:"id"`
	Name   string  `json:"name"`
	RoleID FlexInt `json:"role_id"`
}

// EntOrgGroupsResponse is the response for enterprise org groups API
// The API returns a JSON array directly, not wrapped in an object.
type EntOrgGroupsResponse []EntOrgGroupInfo

// EntOrgSearchResponse is the response for enterprise org search API
type EntOrgSearchResponse struct {
	List []EntOrgInfo `json:"list"`
}

// EntOrgLogResponse is the response for enterprise org log API
type EntOrgLogResponse struct {
	Total FlexInt      `json:"total"`
	List  []OrgLogItem `json:"list"`
}

// === Enterprise Library File (File) API Models ===

// EntFileInfo represents file info in enterprise library
type EntFileInfo struct {
	Hash             string    `json:"hash"`
	Dir              FlexInt   `json:"dir"`
	Fullpath         string    `json:"fullpath"`
	Filename         string    `json:"filename"`
	Filehash         string    `json:"filehash"`
	Filesize         FlexInt64 `json:"filesize"`
	CreateMemberName string    `json:"create_member_name"`
	CreateDateline   FlexInt   `json:"create_dateline"`
	LastMemberName   string    `json:"last_member_name"`
	LastDateline     FlexInt   `json:"last_dateline"`
	Property         *struct {
		Tag        string `json:"tag"`
		OpName     string `json:"op_name"`
		Inherit    int    `json:"inherit"`
		Permission string `json:"permission"`
	} `json:"property,omitempty"`
}

// EntFileListResponse is the response for enterprise file list API
type EntFileListResponse struct {
	Count FlexInt       `json:"count"`
	List  []EntFileInfo `json:"list"`
}

// EntFileUpdateResponse is the response for enterprise file updates API
type EntFileUpdateResponse struct {
	FetchDateline int64         `json:"fetch_dateline"`
	List          []EntFileInfo `json:"list"`
}

// EntFileCreateFolderResponse is the response for enterprise create folder API
type EntFileCreateFolderResponse struct {
	Hash     string `json:"hash"`
	Fullpath string `json:"fullpath"`
}

// EntFileCreateFileResponse is the response for enterprise create file API
type EntFileCreateFileResponse struct {
	Hash     string  `json:"hash"`
	Fullpath string  `json:"fullpath"`
	State    FlexInt `json:"state"`
	Server   string  `json:"server"`
}

// EntFileCopyResponse is the response for enterprise copy file API
type EntFileCopyResponse struct {
	Hash     string `json:"hash"`
	Fullpath string `json:"fullpath"`
	QueueID  string `json:"queue_id,omitempty"`
}

// EntFileLinkResponse is the response for enterprise file link API
type EntFileLinkResponse struct {
	Link string `json:"link"`
	Code string `json:"code"`
}

// EntFileLinksResponse is the response for enterprise file links list API
type EntFileLinksResponse struct {
	Filename string    `json:"filename"`
	Filesize FlexInt64 `json:"filesize"`
	Link     string    `json:"link"`
	Deadline FlexInt64 `json:"deadline"`
	Password bool      `json:"password"`
}

// EntFileHistoryResponse is the response for enterprise file history API
type EntFileHistoryResponse struct {
	Count FlexInt              `json:"count"`
	List  []EntFileHistoryItem `json:"list"`
}

// EntFileHistoryItem represents a file history entry
type EntFileHistoryItem struct {
	HID        string          `json:"hid"`
	Act        FlexInt         `json:"act"`
	ActName    string          `json:"act_name"`
	Dir        FlexInt         `json:"dir"`
	Hash       string          `json:"hash"`
	Fullpath   string          `json:"fullpath"`
	Filename   string          `json:"filename"`
	Filehash   string          `json:"filehash"`
	Filesize   FlexInt64       `json:"filesize"`
	MemberID   FlexInt         `json:"member_id"`
	MemberName string          `json:"member_name"`
	Dateline   FlexInt         `json:"dateline"`
	Property   json.RawMessage `json:"property,omitempty"`
}

// EntFileQueueStatusResponse is the response for enterprise file queue status API
type EntFileQueueStatusResponse struct {
	Status         FlexInt `json:"status"`
	Fullpath       string  `json:"fullpath"`
	AddDateline    FlexInt `json:"add_dateline"`
	StartDateline  FlexInt `json:"start_dateline"`
	FinishDateline FlexInt `json:"finish_dateline"`
}

// EntFileStatResponse is the response for enterprise file stat API
type EntFileStatResponse struct {
	OrgID        FlexInt `json:"org_id"`
	OrgName      string  `json:"org_name"`
	MountID      FlexInt `json:"mount_id"`
	Capacity     int64   `json:"capacity"`
	Size         int64   `json:"size"`
	SizeRecycle  int64   `json:"size_recycle"`
	CountFile    FlexInt `json:"count_file"`
	StoragePoint string  `json:"storage_point"`
}

// EntFilePermissionResponse is the response for enterprise file permission API
type EntFilePermissionResponse struct {
	Members map[string][]string `json:"members"`
	Groups  map[string][]string `json:"groups"`
}

// EntFileUpdatesCountResponse is the response for file updates count API
type EntFileUpdatesCountResponse struct {
	Count FlexInt `json:"count"`
}
