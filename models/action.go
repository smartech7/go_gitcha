// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/Unknwon/com"
	"github.com/go-xorm/xorm"

	"code.gitea.io/git"
	api "code.gitea.io/sdk/gitea"

	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

// ActionType represents the type of an action.
type ActionType int

// Possible action types.
const (
	ActionCreateRepo        ActionType = iota + 1 // 1
	ActionRenameRepo                              // 2
	ActionStarRepo                                // 3
	ActionWatchRepo                               // 4
	ActionCommitRepo                              // 5
	ActionCreateIssue                             // 6
	ActionCreatePullRequest                       // 7
	ActionTransferRepo                            // 8
	ActionPushTag                                 // 9
	ActionCommentIssue                            // 10
	ActionMergePullRequest                        // 11
	ActionCloseIssue                              // 12
	ActionReopenIssue                             // 13
	ActionClosePullRequest                        // 14
	ActionReopenPullRequest                       // 15
)

var (
	// Same as Github. See
	// https://help.github.com/articles/closing-issues-via-commit-messages
	issueCloseKeywords  = []string{"close", "closes", "closed", "fix", "fixes", "fixed", "resolve", "resolves", "resolved"}
	issueReopenKeywords = []string{"reopen", "reopens", "reopened"}

	issueCloseKeywordsPat, issueReopenKeywordsPat *regexp.Regexp
	issueReferenceKeywordsPat                     *regexp.Regexp
)

func assembleKeywordsPattern(words []string) string {
	return fmt.Sprintf(`(?i)(?:%s) \S+`, strings.Join(words, "|"))
}

func init() {
	issueCloseKeywordsPat = regexp.MustCompile(assembleKeywordsPattern(issueCloseKeywords))
	issueReopenKeywordsPat = regexp.MustCompile(assembleKeywordsPattern(issueReopenKeywords))
	issueReferenceKeywordsPat = regexp.MustCompile(`(?i)(?:)(^| )\S+`)
}

// Action represents user operation type and other information to
// repository. It implemented interface base.Actioner so that can be
// used in template render.
type Action struct {
	ID           int64 `xorm:"pk autoincr"`
	UserID       int64 `xorm:"INDEX"` // Receiver user id.
	OpType       ActionType
	ActUserID    int64  `xorm:"INDEX"` // Action user id.
	ActUserName  string // Action user name.
	ActAvatar    string `xorm:"-"`
	RepoID       int64  `xorm:"INDEX"`
	RepoUserName string
	RepoName     string
	RefName      string
	IsPrivate    bool      `xorm:"INDEX NOT NULL DEFAULT false"`
	Content      string    `xorm:"TEXT"`
	Created      time.Time `xorm:"-"`
	CreatedUnix  int64     `xorm:"INDEX"`
}

// BeforeInsert will be invoked by XORM before inserting a record
// representing this object.
func (a *Action) BeforeInsert() {
	a.CreatedUnix = time.Now().Unix()
}

// AfterSet updates the webhook object upon setting a column.
func (a *Action) AfterSet(colName string, _ xorm.Cell) {
	switch colName {
	case "created_unix":
		a.Created = time.Unix(a.CreatedUnix, 0).Local()
	}
}

// GetOpType gets the ActionType of this action.
// TODO: change return type to ActionType ?
func (a *Action) GetOpType() int {
	return int(a.OpType)
}

// GetActUserName gets the action's user name.
func (a *Action) GetActUserName() string {
	return a.ActUserName
}

// ShortActUserName gets the action's user name trimmed to max 20
// chars.
func (a *Action) ShortActUserName() string {
	return base.EllipsisString(a.ActUserName, 20)
}

// GetRepoUserName returns the name of the action repository owner.
func (a *Action) GetRepoUserName() string {
	return a.RepoUserName
}

// ShortRepoUserName returns the name of the action repository owner
// trimmed to max 20 chars.
func (a *Action) ShortRepoUserName() string {
	return base.EllipsisString(a.RepoUserName, 20)
}

// GetRepoName returns the name of the action repository.
func (a *Action) GetRepoName() string {
	return a.RepoName
}

// ShortRepoName returns the name of the action repository
// trimmed to max 33 chars.
func (a *Action) ShortRepoName() string {
	return base.EllipsisString(a.RepoName, 33)
}

// GetRepoPath returns the virtual path to the action repository.
func (a *Action) GetRepoPath() string {
	return path.Join(a.RepoUserName, a.RepoName)
}

// ShortRepoPath returns the virtual path to the action repository
// trimmed to max 20 + 1 + 33 chars.
func (a *Action) ShortRepoPath() string {
	return path.Join(a.ShortRepoUserName(), a.ShortRepoName())
}

// GetRepoLink returns relative link to action repository.
func (a *Action) GetRepoLink() string {
	if len(setting.AppSubURL) > 0 {
		return path.Join(setting.AppSubURL, a.GetRepoPath())
	}
	return "/" + a.GetRepoPath()
}

// GetBranch returns the action's repository branch.
func (a *Action) GetBranch() string {
	return a.RefName
}

// GetContent returns the action's content.
func (a *Action) GetContent() string {
	return a.Content
}

// GetCreate returns the action creation time.
func (a *Action) GetCreate() time.Time {
	return a.Created
}

// GetIssueInfos returns a list of issues associated with
// the action.
func (a *Action) GetIssueInfos() []string {
	return strings.SplitN(a.Content, "|", 2)
}

// GetIssueTitle returns the title of first issue associated
// with the action.
func (a *Action) GetIssueTitle() string {
	index := com.StrTo(a.GetIssueInfos()[0]).MustInt64()
	issue, err := GetIssueByIndex(a.RepoID, index)
	if err != nil {
		log.Error(4, "GetIssueByIndex: %v", err)
		return "500 when get issue"
	}
	return issue.Title
}

// GetIssueContent returns the content of first issue associated with
// this action.
func (a *Action) GetIssueContent() string {
	index := com.StrTo(a.GetIssueInfos()[0]).MustInt64()
	issue, err := GetIssueByIndex(a.RepoID, index)
	if err != nil {
		log.Error(4, "GetIssueByIndex: %v", err)
		return "500 when get issue"
	}
	return issue.Content
}

func newRepoAction(e Engine, u *User, repo *Repository) (err error) {
	if err = notifyWatchers(e, &Action{
		ActUserID:    u.ID,
		ActUserName:  u.Name,
		OpType:       ActionCreateRepo,
		RepoID:       repo.ID,
		RepoUserName: repo.Owner.Name,
		RepoName:     repo.Name,
		IsPrivate:    repo.IsPrivate,
	}); err != nil {
		return fmt.Errorf("notify watchers '%d/%d': %v", u.ID, repo.ID, err)
	}

	log.Trace("action.newRepoAction: %s/%s", u.Name, repo.Name)
	return err
}

// NewRepoAction adds new action for creating repository.
func NewRepoAction(u *User, repo *Repository) (err error) {
	return newRepoAction(x, u, repo)
}

func renameRepoAction(e Engine, actUser *User, oldRepoName string, repo *Repository) (err error) {
	if err = notifyWatchers(e, &Action{
		ActUserID:    actUser.ID,
		ActUserName:  actUser.Name,
		OpType:       ActionRenameRepo,
		RepoID:       repo.ID,
		RepoUserName: repo.Owner.Name,
		RepoName:     repo.Name,
		IsPrivate:    repo.IsPrivate,
		Content:      oldRepoName,
	}); err != nil {
		return fmt.Errorf("notify watchers: %v", err)
	}

	log.Trace("action.renameRepoAction: %s/%s", actUser.Name, repo.Name)
	return nil
}

// RenameRepoAction adds new action for renaming a repository.
func RenameRepoAction(actUser *User, oldRepoName string, repo *Repository) error {
	return renameRepoAction(x, actUser, oldRepoName, repo)
}

func issueIndexTrimRight(c rune) bool {
	return !unicode.IsDigit(c)
}

// PushCommit represents a commit in a push operation.
type PushCommit struct {
	Sha1           string
	Message        string
	AuthorEmail    string
	AuthorName     string
	CommitterEmail string
	CommitterName  string
	Timestamp      time.Time
}

// PushCommits represents list of commits in a push operation.
type PushCommits struct {
	Len        int
	Commits    []*PushCommit
	CompareURL string

	avatars map[string]string
}

// NewPushCommits creates a new PushCommits object.
func NewPushCommits() *PushCommits {
	return &PushCommits{
		avatars: make(map[string]string),
	}
}

// ToAPIPayloadCommits converts a PushCommits object to
// api.PayloadCommit format.
func (pc *PushCommits) ToAPIPayloadCommits(repoLink string) []*api.PayloadCommit {
	commits := make([]*api.PayloadCommit, len(pc.Commits))
	for i, commit := range pc.Commits {
		authorUsername := ""
		author, err := GetUserByEmail(commit.AuthorEmail)
		if err == nil {
			authorUsername = author.Name
		}
		committerUsername := ""
		committer, err := GetUserByEmail(commit.CommitterEmail)
		if err == nil {
			// TODO: check errors other than email not found.
			committerUsername = committer.Name
		}
		commits[i] = &api.PayloadCommit{
			ID:      commit.Sha1,
			Message: commit.Message,
			URL:     fmt.Sprintf("%s/commit/%s", repoLink, commit.Sha1),
			Author: &api.PayloadUser{
				Name:     commit.AuthorName,
				Email:    commit.AuthorEmail,
				UserName: authorUsername,
			},
			Committer: &api.PayloadUser{
				Name:     commit.CommitterName,
				Email:    commit.CommitterEmail,
				UserName: committerUsername,
			},
			Timestamp: commit.Timestamp,
		}
	}
	return commits
}

// AvatarLink tries to match user in database with e-mail
// in order to show custom avatar, and falls back to general avatar link.
func (pc *PushCommits) AvatarLink(email string) string {
	_, ok := pc.avatars[email]
	if !ok {
		u, err := GetUserByEmail(email)
		if err != nil {
			pc.avatars[email] = base.AvatarLink(email)
			if !IsErrUserNotExist(err) {
				log.Error(4, "GetUserByEmail: %v", err)
			}
		} else {
			pc.avatars[email] = u.RelAvatarLink()
		}
	}

	return pc.avatars[email]
}

// UpdateIssuesCommit checks if issues are manipulated by commit message.
func UpdateIssuesCommit(doer *User, repo *Repository, commits []*PushCommit) error {
	// Commits are appended in the reverse order.
	for i := len(commits) - 1; i >= 0; i-- {
		c := commits[i]

		refMarked := make(map[int64]bool)
		for _, ref := range issueReferenceKeywordsPat.FindAllString(c.Message, -1) {
			ref = ref[strings.IndexByte(ref, byte(' '))+1:]
			ref = strings.TrimRightFunc(ref, issueIndexTrimRight)

			if len(ref) == 0 {
				continue
			}

			// Add repo name if missing
			if ref[0] == '#' {
				ref = fmt.Sprintf("%s%s", repo.FullName(), ref)
			} else if !strings.Contains(ref, "/") {
				// FIXME: We don't support User#ID syntax yet
				// return ErrNotImplemented
				continue
			}

			issue, err := GetIssueByRef(ref)
			if err != nil {
				if IsErrIssueNotExist(err) || err == errMissingIssueNumber {
					continue
				}
				return err
			}

			if refMarked[issue.ID] {
				continue
			}
			refMarked[issue.ID] = true

			message := fmt.Sprintf(`<a href="%s/commit/%s">%s</a>`, repo.Link(), c.Sha1, c.Message)
			if err = CreateRefComment(doer, repo, issue, message, c.Sha1); err != nil {
				return err
			}
		}

		refMarked = make(map[int64]bool)
		// FIXME: can merge this one and next one to a common function.
		for _, ref := range issueCloseKeywordsPat.FindAllString(c.Message, -1) {
			ref = ref[strings.IndexByte(ref, byte(' '))+1:]
			ref = strings.TrimRightFunc(ref, issueIndexTrimRight)

			if len(ref) == 0 {
				continue
			}

			// Add repo name if missing
			if ref[0] == '#' {
				ref = fmt.Sprintf("%s%s", repo.FullName(), ref)
			} else if !strings.Contains(ref, "/") {
				// We don't support User#ID syntax yet
				// return ErrNotImplemented
				continue
			}

			issue, err := GetIssueByRef(ref)
			if err != nil {
				if IsErrIssueNotExist(err) || err == errMissingIssueNumber {
					continue
				}
				return err
			}

			if refMarked[issue.ID] {
				continue
			}
			refMarked[issue.ID] = true

			if issue.RepoID != repo.ID || issue.IsClosed {
				continue
			}

			if err = issue.ChangeStatus(doer, repo, true); err != nil {
				return err
			}
		}

		// It is conflict to have close and reopen at same time, so refsMarked doesn't need to reinit here.
		for _, ref := range issueReopenKeywordsPat.FindAllString(c.Message, -1) {
			ref = ref[strings.IndexByte(ref, byte(' '))+1:]
			ref = strings.TrimRightFunc(ref, issueIndexTrimRight)

			if len(ref) == 0 {
				continue
			}

			// Add repo name if missing
			if ref[0] == '#' {
				ref = fmt.Sprintf("%s%s", repo.FullName(), ref)
			} else if !strings.Contains(ref, "/") {
				// We don't support User#ID syntax yet
				// return ErrNotImplemented
				continue
			}

			issue, err := GetIssueByRef(ref)
			if err != nil {
				if IsErrIssueNotExist(err) || err == errMissingIssueNumber {
					continue
				}
				return err
			}

			if refMarked[issue.ID] {
				continue
			}
			refMarked[issue.ID] = true

			if issue.RepoID != repo.ID || !issue.IsClosed {
				continue
			}

			if err = issue.ChangeStatus(doer, repo, false); err != nil {
				return err
			}
		}
	}
	return nil
}

// CommitRepoActionOptions represent options of a new commit action.
type CommitRepoActionOptions struct {
	PusherName  string
	RepoOwnerID int64
	RepoName    string
	RefFullName string
	OldCommitID string
	NewCommitID string
	Commits     *PushCommits
}

// CommitRepoAction adds new commit action to the repository, and prepare
// corresponding webhooks.
func CommitRepoAction(opts CommitRepoActionOptions) error {
	pusher, err := GetUserByName(opts.PusherName)
	if err != nil {
		return fmt.Errorf("GetUserByName [%s]: %v", opts.PusherName, err)
	}

	repo, err := GetRepositoryByName(opts.RepoOwnerID, opts.RepoName)
	if err != nil {
		return fmt.Errorf("GetRepositoryByName [owner_id: %d, name: %s]: %v", opts.RepoOwnerID, opts.RepoName, err)
	}

	// Change repository bare status and update last updated time.
	repo.IsBare = false
	if err = UpdateRepository(repo, false); err != nil {
		return fmt.Errorf("UpdateRepository: %v", err)
	}

	isNewBranch := false
	opType := ActionCommitRepo
	// Check it's tag push or branch.
	if strings.HasPrefix(opts.RefFullName, git.TagPrefix) {
		opType = ActionPushTag
		opts.Commits = &PushCommits{}
	} else {
		// if not the first commit, set the compare URL.
		if opts.OldCommitID == git.EmptySHA {
			isNewBranch = true
		} else {
			opts.Commits.CompareURL = repo.ComposeCompareURL(opts.OldCommitID, opts.NewCommitID)
		}

		if err = UpdateIssuesCommit(pusher, repo, opts.Commits.Commits); err != nil {
			log.Error(4, "updateIssuesCommit: %v", err)
		}
	}

	if len(opts.Commits.Commits) > setting.UI.FeedMaxCommitNum {
		opts.Commits.Commits = opts.Commits.Commits[:setting.UI.FeedMaxCommitNum]
	}

	data, err := json.Marshal(opts.Commits)
	if err != nil {
		return fmt.Errorf("Marshal: %v", err)
	}

	refName := git.RefEndName(opts.RefFullName)
	if err = NotifyWatchers(&Action{
		ActUserID:    pusher.ID,
		ActUserName:  pusher.Name,
		OpType:       opType,
		Content:      string(data),
		RepoID:       repo.ID,
		RepoUserName: repo.MustOwner().Name,
		RepoName:     repo.Name,
		RefName:      refName,
		IsPrivate:    repo.IsPrivate,
	}); err != nil {
		return fmt.Errorf("NotifyWatchers: %v", err)
	}

	defer func() {
		go HookQueue.Add(repo.ID)
	}()

	apiPusher := pusher.APIFormat()
	apiRepo := repo.APIFormat(AccessModeNone)

	var shaSum string
	switch opType {
	case ActionCommitRepo: // Push
		if err = PrepareWebhooks(repo, HookEventPush, &api.PushPayload{
			Ref:        opts.RefFullName,
			Before:     opts.OldCommitID,
			After:      opts.NewCommitID,
			CompareURL: setting.AppURL + opts.Commits.CompareURL,
			Commits:    opts.Commits.ToAPIPayloadCommits(repo.HTMLURL()),
			Repo:       apiRepo,
			Pusher:     apiPusher,
			Sender:     apiPusher,
		}); err != nil {
			return fmt.Errorf("PrepareWebhooks: %v", err)
		}

		if isNewBranch {
			gitRepo, err := git.OpenRepository(repo.RepoPath())
			if err != nil {
				log.Error(4, "OpenRepository[%s]: %v", repo.RepoPath(), err)
			}
			shaSum, err = gitRepo.GetBranchCommitID(refName)
			if err != nil {
				log.Error(4, "GetBranchCommitID[%s]: %v", opts.RefFullName, err)
			}
			return PrepareWebhooks(repo, HookEventCreate, &api.CreatePayload{
				Ref:     refName,
				Sha:     shaSum,
				RefType: "branch",
				Repo:    apiRepo,
				Sender:  apiPusher,
			})
		}

	case ActionPushTag: // Create
		gitRepo, err := git.OpenRepository(repo.RepoPath())
		if err != nil {
			log.Error(4, "OpenRepository[%s]: %v", repo.RepoPath(), err)
		}
		shaSum, err = gitRepo.GetTagCommitID(refName)
		if err != nil {
			log.Error(4, "GetTagCommitID[%s]: %v", opts.RefFullName, err)
		}
		return PrepareWebhooks(repo, HookEventCreate, &api.CreatePayload{
			Ref:     refName,
			Sha:     shaSum,
			RefType: "tag",
			Repo:    apiRepo,
			Sender:  apiPusher,
		})
	}

	return nil
}

func transferRepoAction(e Engine, doer, oldOwner *User, repo *Repository) (err error) {
	if err = notifyWatchers(e, &Action{
		ActUserID:    doer.ID,
		ActUserName:  doer.Name,
		OpType:       ActionTransferRepo,
		RepoID:       repo.ID,
		RepoUserName: repo.Owner.Name,
		RepoName:     repo.Name,
		IsPrivate:    repo.IsPrivate,
		Content:      path.Join(oldOwner.Name, repo.Name),
	}); err != nil {
		return fmt.Errorf("notifyWatchers: %v", err)
	}

	// Remove watch for organization.
	if oldOwner.IsOrganization() {
		if err = watchRepo(e, oldOwner.ID, repo.ID, false); err != nil {
			return fmt.Errorf("watchRepo [false]: %v", err)
		}
	}

	return nil
}

// TransferRepoAction adds new action for transferring repository,
// the Owner field of repository is assumed to be new owner.
func TransferRepoAction(doer, oldOwner *User, repo *Repository) error {
	return transferRepoAction(x, doer, oldOwner, repo)
}

func mergePullRequestAction(e Engine, doer *User, repo *Repository, issue *Issue) error {
	return notifyWatchers(e, &Action{
		ActUserID:    doer.ID,
		ActUserName:  doer.Name,
		OpType:       ActionMergePullRequest,
		Content:      fmt.Sprintf("%d|%s", issue.Index, issue.Title),
		RepoID:       repo.ID,
		RepoUserName: repo.Owner.Name,
		RepoName:     repo.Name,
		IsPrivate:    repo.IsPrivate,
	})
}

// MergePullRequestAction adds new action for merging pull request.
func MergePullRequestAction(actUser *User, repo *Repository, pull *Issue) error {
	return mergePullRequestAction(x, actUser, repo, pull)
}

// GetFeeds returns action list of given user in given context.
// actorID is the user who's requesting, ctxUserID is the user/org that is requested.
// actorID can be -1 when isProfile is true or to skip the permission check.
func GetFeeds(ctxUser *User, actorID, offset int64, isProfile bool) ([]*Action, error) {
	actions := make([]*Action, 0, 20)
	sess := x.
		Limit(20, int(offset)).
		Desc("id").
		Where("user_id = ?", ctxUser.ID)
	if isProfile {
		sess.
			And("is_private = ?", false).
			And("act_user_id = ?", ctxUser.ID)
	} else if actorID != -1 && ctxUser.IsOrganization() {
		env, err := ctxUser.AccessibleReposEnv(actorID)
		if err != nil {
			return nil, fmt.Errorf("AccessibleReposEnv: %v", err)
		}
		repoIDs, err := env.RepoIDs(1, ctxUser.NumRepos)
		if err != nil {
			return nil, fmt.Errorf("GetUserRepositories: %v", err)
		}
		if len(repoIDs) > 0 {
			sess.In("repo_id", repoIDs)
		}
	}

	err := sess.Find(&actions)
	return actions, err
}
