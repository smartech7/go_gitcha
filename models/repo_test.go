// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"testing"

	"code.gitea.io/gitea/modules/markdown"

	"github.com/stretchr/testify/assert"
)

func TestRepo(t *testing.T) {
	repo := &Repository{Name: "testRepo"}
	repo.Owner = &User{Name: "testOwner"}

	repo.Units = nil
	assert.Nil(t, repo.ComposeMetas())

	externalTracker := RepoUnit{
		Type: UnitTypeExternalTracker,
		Config: &ExternalTrackerConfig{
			ExternalTrackerFormat: "https://someurl.com/{user}/{repo}/{issue}",
		},
	}

	testSuccess := func(expectedStyle string) {
		repo.Units = []*RepoUnit{&externalTracker}
		repo.ExternalMetas = nil
		metas := repo.ComposeMetas()
		assert.Equal(t, expectedStyle, metas["style"])
		assert.Equal(t, "testRepo", metas["repo"])
		assert.Equal(t, "testOwner", metas["user"])
		assert.Equal(t, "https://someurl.com/{user}/{repo}/{issue}", metas["format"])
	}

	testSuccess(markdown.IssueNameStyleNumeric)

	externalTracker.ExternalTrackerConfig().ExternalTrackerStyle = markdown.IssueNameStyleAlphanumeric
	testSuccess(markdown.IssueNameStyleAlphanumeric)

	externalTracker.ExternalTrackerConfig().ExternalTrackerStyle = markdown.IssueNameStyleNumeric
	testSuccess(markdown.IssueNameStyleNumeric)
}

func TestGetRepositoryCount(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	count, err1 := GetRepositoryCount(&User{ID: int64(10)})
	privateCount, err2 := GetPrivateRepositoryCount(&User{ID: int64(10)})
	publicCount, err3 := GetPublicRepositoryCount(&User{ID: int64(10)})
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.Equal(t, int64(3), count)
	assert.Equal(t, (privateCount + publicCount), count)
}

func TestGetPublicRepositoryCount(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	count, err := GetPublicRepositoryCount(&User{ID: int64(10)})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestGetPrivateRepositoryCount(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	count, err := GetPrivateRepositoryCount(&User{ID: int64(10)})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestUpdateRepositoryVisibilityChanged(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	// Get sample repo and change visibility
	repo, err := GetRepositoryByID(9)
	repo.IsPrivate = true

	// Update it
	err = UpdateRepository(repo, true)
	assert.NoError(t, err)

	// Check visibility of action has become private
	act := Action{}
	_, err = x.ID(3).Get(&act)

	assert.NoError(t, err)
	assert.Equal(t, true, act.IsPrivate)
}
