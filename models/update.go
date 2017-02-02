// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"container/list"
	"fmt"
	"os/exec"
	"strings"

	"code.gitea.io/git"

	"code.gitea.io/gitea/modules/log"
)

// UpdateTask defines an UpdateTask
type UpdateTask struct {
	ID          int64  `xorm:"pk autoincr"`
	UUID        string `xorm:"index"`
	RefName     string
	OldCommitID string
	NewCommitID string
}

// AddUpdateTask adds an UpdateTask
func AddUpdateTask(task *UpdateTask) error {
	_, err := x.Insert(task)
	return err
}

// GetUpdateTaskByUUID returns update task by given UUID.
func GetUpdateTaskByUUID(uuid string) (*UpdateTask, error) {
	task := &UpdateTask{
		UUID: uuid,
	}
	has, err := x.Get(task)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrUpdateTaskNotExist{uuid}
	}
	return task, nil
}

// DeleteUpdateTaskByUUID deletes an UpdateTask from the database
func DeleteUpdateTaskByUUID(uuid string) error {
	_, err := x.Delete(&UpdateTask{UUID: uuid})
	return err
}

// CommitToPushCommit transforms a git.Commit to PushCommit type.
func CommitToPushCommit(commit *git.Commit) *PushCommit {
	return &PushCommit{
		Sha1:           commit.ID.String(),
		Message:        commit.Message(),
		AuthorEmail:    commit.Author.Email,
		AuthorName:     commit.Author.Name,
		CommitterEmail: commit.Committer.Email,
		CommitterName:  commit.Committer.Name,
		Timestamp:      commit.Author.When,
	}
}

// ListToPushCommits transforms a list.List to PushCommits type.
func ListToPushCommits(l *list.List) *PushCommits {
	var commits []*PushCommit
	var actEmail string
	for e := l.Front(); e != nil; e = e.Next() {
		commit := e.Value.(*git.Commit)
		if actEmail == "" {
			actEmail = commit.Committer.Email
		}
		commits = append(commits, CommitToPushCommit(commit))
	}
	return &PushCommits{l.Len(), commits, "", nil}
}

// PushUpdateOptions defines the push update options
type PushUpdateOptions struct {
	PusherID     int64
	PusherName   string
	RepoUserName string
	RepoName     string
	RefFullName  string
	OldCommitID  string
	NewCommitID  string
}

// PushUpdate must be called for any push actions in order to
// generates necessary push action history feeds.
func PushUpdate(opts PushUpdateOptions) (err error) {
	isNewRef := opts.OldCommitID == git.EmptySHA
	isDelRef := opts.NewCommitID == git.EmptySHA
	if isNewRef && isDelRef {
		return fmt.Errorf("Old and new revisions are both %s", git.EmptySHA)
	}

	repoPath := RepoPath(opts.RepoUserName, opts.RepoName)

	gitUpdate := exec.Command("git", "update-server-info")
	gitUpdate.Dir = repoPath
	if err = gitUpdate.Run(); err != nil {
		return fmt.Errorf("Failed to call 'git update-server-info': %v", err)
	}

	if isDelRef {
		log.GitLogger.Info("Reference '%s' has been deleted from '%s/%s' by %s",
			opts.RefFullName, opts.RepoUserName, opts.RepoName, opts.PusherName)
		return nil
	}

	gitRepo, err := git.OpenRepository(repoPath)
	if err != nil {
		return fmt.Errorf("OpenRepository: %v", err)
	}

	owner, err := GetUserByName(opts.RepoUserName)
	if err != nil {
		return fmt.Errorf("GetUserByName: %v", err)
	}

	repo, err := GetRepositoryByName(owner.ID, opts.RepoName)
	if err != nil {
		return fmt.Errorf("GetRepositoryByName: %v", err)
	}

	// Push tags.
	if strings.HasPrefix(opts.RefFullName, git.TagPrefix) {
		if err := CommitRepoAction(CommitRepoActionOptions{
			PusherName:  opts.PusherName,
			RepoOwnerID: owner.ID,
			RepoName:    repo.Name,
			RefFullName: opts.RefFullName,
			OldCommitID: opts.OldCommitID,
			NewCommitID: opts.NewCommitID,
			Commits:     &PushCommits{},
		}); err != nil {
			return fmt.Errorf("CommitRepoAction (tag): %v", err)
		}
		return nil
	}

	newCommit, err := gitRepo.GetCommit(opts.NewCommitID)
	if err != nil {
		return fmt.Errorf("gitRepo.GetCommit: %v", err)
	}

	// Push new branch.
	var l *list.List
	if isNewRef {
		l, err = newCommit.CommitsBeforeLimit(10)
		if err != nil {
			return fmt.Errorf("newCommit.CommitsBeforeLimit: %v", err)
		}
	} else {
		l, err = newCommit.CommitsBeforeUntil(opts.OldCommitID)
		if err != nil {
			return fmt.Errorf("newCommit.CommitsBeforeUntil: %v", err)
		}
	}

	if err := CommitRepoAction(CommitRepoActionOptions{
		PusherName:  opts.PusherName,
		RepoOwnerID: owner.ID,
		RepoName:    repo.Name,
		RefFullName: opts.RefFullName,
		OldCommitID: opts.OldCommitID,
		NewCommitID: opts.NewCommitID,
		Commits:     ListToPushCommits(l),
	}); err != nil {
		return fmt.Errorf("CommitRepoAction (branch): %v", err)
	}
	return nil
}
