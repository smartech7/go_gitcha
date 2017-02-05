// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import "strings"

// RepoRedirect represents that a repo name should be redirected to another
type RepoRedirect struct {
	ID             int64  `xorm:"pk autoincr"`
	OwnerID        int64  `xorm:"UNIQUE(s)"`
	LowerName      string `xorm:"UNIQUE(s) INDEX NOT NULL"`
	RedirectRepoID int64  // repoID to redirect to
}

// LookupRepoRedirect look up if a repository has a redirect name
func LookupRepoRedirect(ownerID int64, repoName string) (int64, error) {
	repoName = strings.ToLower(repoName)
	redirect := &RepoRedirect{OwnerID: ownerID, LowerName: repoName}
	if has, err := x.Get(redirect); err != nil {
		return 0, err
	} else if !has {
		return 0, ErrRepoRedirectNotExist{OwnerID: ownerID, RepoName: repoName}
	}
	return redirect.RedirectRepoID, nil
}

// NewRepoRedirect create a new repo redirect
func NewRepoRedirect(ownerID, repoID int64, oldRepoName, newRepoName string) error {
	oldRepoName = strings.ToLower(oldRepoName)
	newRepoName = strings.ToLower(newRepoName)
	sess := x.NewSession()
	defer sess.Close()

	if err := sess.Begin(); err != nil {
		return err
	}

	if err := deleteRepoRedirect(sess, ownerID, newRepoName); err != nil {
		sess.Rollback()
		return err
	}

	if _, err := sess.Insert(&RepoRedirect{
		OwnerID:        ownerID,
		LowerName:      oldRepoName,
		RedirectRepoID: repoID,
	}); err != nil {
		sess.Rollback()
		return err
	}
	return sess.Commit()
}

// deleteRepoRedirect delete any redirect from the specified repo name to
// anything else
func deleteRepoRedirect(e Engine, ownerID int64, repoName string) error {
	repoName = strings.ToLower(repoName)
	_, err := e.Delete(&RepoRedirect{OwnerID: ownerID, LowerName: repoName})
	return err
}
