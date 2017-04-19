// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"net/url"
	"strings"

	"code.gitea.io/gitea/models"
	"github.com/Unknwon/com"
	"github.com/go-macaron/binding"
	macaron "gopkg.in/macaron.v1"
)

// _______________________________________    _________.______________________ _______________.___.
// \______   \_   _____/\______   \_____  \  /   _____/|   \__    ___/\_____  \\______   \__  |   |
//  |       _/|    __)_  |     ___//   |   \ \_____  \ |   | |    |    /   |   \|       _//   |   |
//  |    |   \|        \ |    |   /    |    \/        \|   | |    |   /    |    \    |   \\____   |
//  |____|_  /_______  / |____|   \_______  /_______  /|___| |____|   \_______  /____|_  // ______|
//         \/        \/                   \/        \/                        \/       \/ \/

// CreateRepoForm form for creating repository
type CreateRepoForm struct {
	UID         int64  `binding:"Required"`
	RepoName    string `binding:"Required;AlphaDashDot;MaxSize(100)"`
	Private     bool
	Description string `binding:"MaxSize(255)"`
	AutoInit    bool
	Gitignores  string
	License     string
	Readme      string
}

// Validate validates the fields
func (f *CreateRepoForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// MigrateRepoForm form for migrating repository
type MigrateRepoForm struct {
	CloneAddr    string `json:"clone_addr" binding:"Required"`
	AuthUsername string `json:"auth_username"`
	AuthPassword string `json:"auth_password"`
	UID          int64  `json:"uid" binding:"Required"`
	RepoName     string `json:"repo_name" binding:"Required;AlphaDashDot;MaxSize(100)"`
	Mirror       bool   `json:"mirror"`
	Private      bool   `json:"private"`
	Description  string `json:"description" binding:"MaxSize(255)"`
}

// Validate validates the fields
func (f *MigrateRepoForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// ParseRemoteAddr checks if given remote address is valid,
// and returns composed URL with needed username and password.
// It also checks if given user has permission when remote address
// is actually a local path.
func (f MigrateRepoForm) ParseRemoteAddr(user *models.User) (string, error) {
	remoteAddr := strings.TrimSpace(f.CloneAddr)

	// Remote address can be HTTP/HTTPS/Git URL or local path.
	if strings.HasPrefix(remoteAddr, "http://") ||
		strings.HasPrefix(remoteAddr, "https://") ||
		strings.HasPrefix(remoteAddr, "git://") {
		u, err := url.Parse(remoteAddr)
		if err != nil {
			return "", models.ErrInvalidCloneAddr{IsURLError: true}
		}
		if len(f.AuthUsername)+len(f.AuthPassword) > 0 {
			u.User = url.UserPassword(f.AuthUsername, f.AuthPassword)
		}
		remoteAddr = u.String()
	} else if !user.CanImportLocal() {
		return "", models.ErrInvalidCloneAddr{IsPermissionDenied: true}
	} else if !com.IsDir(remoteAddr) {
		return "", models.ErrInvalidCloneAddr{IsInvalidPath: true}
	}

	return remoteAddr, nil
}

// RepoSettingForm form for changing repository settings
type RepoSettingForm struct {
	RepoName      string `binding:"Required;AlphaDashDot;MaxSize(100)"`
	Description   string `binding:"MaxSize(255)"`
	Website       string `binding:"ValidUrl;MaxSize(255)"`
	Interval      string
	MirrorAddress string
	Private       bool
	EnablePrune   bool

	// Advanced settings
	EnableWiki            bool
	EnableExternalWiki    bool
	ExternalWikiURL       string
	EnableIssues          bool
	EnableExternalTracker bool
	ExternalTrackerURL    string
	TrackerURLFormat      string
	TrackerIssueStyle     string
	EnablePulls           bool
}

// Validate validates the fields
func (f *RepoSettingForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

//  __      __      ___.   .__    .__            __
// /  \    /  \ ____\_ |__ |  |__ |  |__   ____ |  | __
// \   \/\/   // __ \| __ \|  |  \|  |  \ /  _ \|  |/ /
//  \        /\  ___/| \_\ \   Y  \   Y  (  <_> )    <
//   \__/\  /  \___  >___  /___|  /___|  /\____/|__|_ \
//        \/       \/    \/     \/     \/            \/

// WebhookForm form for changing web hook
type WebhookForm struct {
	Events      string
	Create      bool
	Push        bool
	PullRequest bool
	Active      bool
}

// PushOnly if the hook will be triggered when push
func (f WebhookForm) PushOnly() bool {
	return f.Events == "push_only"
}

// SendEverything if the hook will be triggered any event
func (f WebhookForm) SendEverything() bool {
	return f.Events == "send_everything"
}

// ChooseEvents if the hook will be triggered choose events
func (f WebhookForm) ChooseEvents() bool {
	return f.Events == "choose_events"
}

// NewWebhookForm form for creating web hook
type NewWebhookForm struct {
	PayloadURL  string `binding:"Required;ValidUrl"`
	ContentType int    `binding:"Required"`
	Secret      string
	WebhookForm
}

// Validate validates the fields
func (f *NewWebhookForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// NewSlackHookForm form for creating slack hook
type NewSlackHookForm struct {
	PayloadURL string `binding:"Required;ValidUrl"`
	Channel    string `binding:"Required"`
	Username   string
	IconURL    string
	Color      string
	WebhookForm
}

// Validate validates the fields
func (f *NewSlackHookForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// .___
// |   | ______ ________ __   ____
// |   |/  ___//  ___/  |  \_/ __ \
// |   |\___ \ \___ \|  |  /\  ___/
// |___/____  >____  >____/  \___  >
//          \/     \/            \/

// CreateIssueForm form for creating issue
type CreateIssueForm struct {
	Title       string `binding:"Required;MaxSize(255)"`
	LabelIDs    string `form:"label_ids"`
	MilestoneID int64
	AssigneeID  int64
	Content     string
	Files       []string
}

// Validate validates the fields
func (f *CreateIssueForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// CreateCommentForm form for creating comment
type CreateCommentForm struct {
	Content string
	Status  string `binding:"OmitEmpty;In(reopen,close)"`
	Files   []string
}

// Validate validates the fields
func (f *CreateCommentForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

//    _____  .__.__                   __
//   /     \ |__|  |   ____   _______/  |_  ____   ____   ____
//  /  \ /  \|  |  | _/ __ \ /  ___/\   __\/  _ \ /    \_/ __ \
// /    Y    \  |  |_\  ___/ \___ \  |  | (  <_> )   |  \  ___/
// \____|__  /__|____/\___  >____  > |__|  \____/|___|  /\___  >
//         \/             \/     \/                   \/     \/

// CreateMilestoneForm form for creating milestone
type CreateMilestoneForm struct {
	Title    string `binding:"Required;MaxSize(50)"`
	Content  string
	Deadline string
}

// Validate validates the fields
func (f *CreateMilestoneForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// .____          ___.          .__
// |    |   _____ \_ |__   ____ |  |
// |    |   \__  \ | __ \_/ __ \|  |
// |    |___ / __ \| \_\ \  ___/|  |__
// |_______ (____  /___  /\___  >____/
//         \/    \/    \/     \/

// CreateLabelForm form for creating label
type CreateLabelForm struct {
	ID    int64
	Title string `binding:"Required;MaxSize(50)" locale:"repo.issues.label_name"`
	Color string `binding:"Required;Size(7)" locale:"repo.issues.label_color"`
}

// Validate validates the fields
func (f *CreateLabelForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// InitializeLabelsForm form for initializing labels
type InitializeLabelsForm struct {
	TemplateName string `binding:"Required"`
}

// Validate validates the fields
func (f *InitializeLabelsForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// __________       .__
// \______   \ ____ |  |   ____ _____    ______ ____
//  |       _// __ \|  | _/ __ \\__  \  /  ___// __ \
//  |    |   \  ___/|  |_\  ___/ / __ \_\___ \\  ___/
//  |____|_  /\___  >____/\___  >____  /____  >\___  >
//         \/     \/          \/     \/     \/     \/

// NewReleaseForm form for creating release
type NewReleaseForm struct {
	TagName    string `binding:"Required"`
	Target     string `form:"tag_target" binding:"Required"`
	Title      string `binding:"Required"`
	Content    string
	Draft      string
	Prerelease bool
	Files      []string
}

// Validate validates the fields
func (f *NewReleaseForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// EditReleaseForm form for changing release
type EditReleaseForm struct {
	Title      string `form:"title" binding:"Required"`
	Content    string `form:"content"`
	Draft      string `form:"draft"`
	Prerelease bool   `form:"prerelease"`
	Files      []string
}

// Validate validates the fields
func (f *EditReleaseForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

//  __      __.__ __   .__
// /  \    /  \__|  | _|__|
// \   \/\/   /  |  |/ /  |
//  \        /|  |    <|  |
//   \__/\  / |__|__|_ \__|
//        \/          \/

// NewWikiForm form for creating wiki
type NewWikiForm struct {
	OldTitle string
	Title    string `binding:"Required"`
	Content  string `binding:"Required"`
	Message  string
}

// Validate validates the fields
// FIXME: use code generation to generate this method.
func (f *NewWikiForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// ___________    .___.__  __
// \_   _____/  __| _/|__|/  |_
//  |    __)_  / __ | |  \   __\
//  |        \/ /_/ | |  ||  |
// /_______  /\____ | |__||__|
//         \/      \/

// EditRepoFileForm form for changing repository file
type EditRepoFileForm struct {
	TreePath      string `binding:"Required;MaxSize(500)"`
	Content       string `binding:"Required"`
	CommitSummary string `binding:"MaxSize(100)"`
	CommitMessage string
	CommitChoice  string `binding:"Required;MaxSize(50)"`
	NewBranchName string `binding:"GitRefName;MaxSize(100)"`
	LastCommit    string
}

// Validate validates the fields
func (f *EditRepoFileForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// EditPreviewDiffForm form for changing preview diff
type EditPreviewDiffForm struct {
	Content string
}

// Validate validates the fields
func (f *EditPreviewDiffForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

//  ____ ___        .__                    .___
// |    |   \______ |  |   _________     __| _/
// |    |   /\____ \|  |  /  _ \__  \   / __ |
// |    |  / |  |_> >  |_(  <_> ) __ \_/ /_/ |
// |______/  |   __/|____/\____(____  /\____ |
//           |__|                   \/      \/
//

// UploadRepoFileForm form for uploading repository file
type UploadRepoFileForm struct {
	TreePath      string `binding:"MaxSize(500)"`
	CommitSummary string `binding:"MaxSize(100)"`
	CommitMessage string
	CommitChoice  string `binding:"Required;MaxSize(50)"`
	NewBranchName string `binding:"GitRefName;MaxSize(100)"`
	Files         []string
}

// Validate validates the fields
func (f *UploadRepoFileForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// RemoveUploadFileForm form for removing uploaded file
type RemoveUploadFileForm struct {
	File string `binding:"Required;MaxSize(50)"`
}

// Validate validates the fields
func (f *RemoveUploadFileForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// ________         .__          __
// \______ \   ____ |  |   _____/  |_  ____
// |    |  \_/ __ \|  | _/ __ \   __\/ __ \
// |    `   \  ___/|  |_\  ___/|  | \  ___/
// /_______  /\___  >____/\___  >__|  \___  >
//         \/     \/          \/          \/

// DeleteRepoFileForm form for deleting repository file
type DeleteRepoFileForm struct {
	CommitSummary string `binding:"MaxSize(100)"`
	CommitMessage string
	CommitChoice  string `binding:"Required;MaxSize(50)"`
	NewBranchName string `binding:"GitRefName;MaxSize(100)"`
}

// Validate validates the fields
func (f *DeleteRepoFileForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}
