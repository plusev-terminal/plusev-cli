package api

import (
	"encoding/json"
	"time"
)

// PublishRelease is the payload sent to releases/publish. PluginId and
// Version are no longer included — the server reads them from the WASM meta
// export and uses the plugin binary as the source of truth.
type PublishRelease struct {
	Sha256    string `json:"sha256"`
	Channel   string `json:"channel,omitempty"`
	Changelog string `json:"changelog,omitempty"`
}

// UploadResult is returned from files/upload.
type UploadResult struct {
	Sha256   string `json:"sha256"`
	FileSize int64  `json:"fileSize"`
	Exists   bool   `json:"exists"`
}

type SubmissionListReq struct {
	Status   string `json:"status,omitempty"`
	Kind     string `json:"kind,omitempty"`
	PluginID string `json:"pluginId,omitempty"`
}

type SubmissionCancelReq struct {
	SubmissionID uint64 `json:"submissionId"`
}

type SetActiveReq struct {
	PluginID string `json:"pluginId"`
	Active   bool   `json:"active"`
}

type DeletePluginReq struct {
	PluginID string `json:"pluginId"`
}

type DeleteReleaseReq struct {
	PluginID string `json:"pluginId"`
	Version  string `json:"version"`
}

type PruneReleasesReq struct {
	PluginID  string `json:"pluginId"`
	OlderThan string `json:"olderThan"`
}

type UpdateEmailReq struct {
	Email string `json:"email"`
}

// WhoamiResult is returned from /whoami — the repo metadata for the
// authenticated dev key. Used by `registry add` to derive the config label
// and validate credentials before saving.
type WhoamiResult struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	IsPublic    bool   `json:"isPublic"`
	Description string `json:"description"`
}

// PluginListEntry is a plugin entry as returned by plugins/list.
type PluginListEntry struct {
	ID          uint64   `json:"id"`
	PluginID    string   `json:"pluginId"`
	Name        string   `json:"name"`
	AppID       string   `json:"appId"`
	Category    string   `json:"category"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Active      bool     `json:"active"`
	Description string   `json:"description"`
	Repository  string   `json:"repository"`
	Tags        []string `json:"tags"`
}

// Submission is a row from submissions/list.
type Submission struct {
	ID              uint64          `json:"id"`
	Kind            string          `json:"kind"`
	Status          string          `json:"status"`
	PluginID        string          `json:"pluginId,omitempty"`
	RejectionReason string          `json:"rejectionReason,omitempty"`
	SubmittedAt     time.Time       `json:"submittedAt"`
	DecidedAt       *time.Time      `json:"decidedAt,omitempty"`
	Payload         json.RawMessage `json:"payload,omitempty"`
}
