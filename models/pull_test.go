// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPullRequest_LoadAttributes(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	pr := AssertExistsAndLoadBean(t, &PullRequest{ID: 1}).(*PullRequest)
	assert.NoError(t, pr.LoadAttributes())
	assert.NotNil(t, pr.Merger)
	assert.Equal(t, pr.MergerID, pr.Merger.ID)
}

func TestPullRequest_LoadIssue(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	pr := AssertExistsAndLoadBean(t, &PullRequest{ID: 1}).(*PullRequest)
	assert.NoError(t, pr.LoadIssue())
	assert.NotNil(t, pr.Issue)
	assert.Equal(t, int64(2), pr.Issue.ID)
	assert.NoError(t, pr.LoadIssue())
	assert.NotNil(t, pr.Issue)
	assert.Equal(t, int64(2), pr.Issue.ID)
}

// TODO TestPullRequest_APIFormat

func TestPullRequest_GetBaseRepo(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	pr := AssertExistsAndLoadBean(t, &PullRequest{ID: 1}).(*PullRequest)
	assert.NoError(t, pr.GetBaseRepo())
	assert.NotNil(t, pr.BaseRepo)
	assert.Equal(t, pr.BaseRepoID, pr.BaseRepo.ID)
	assert.NoError(t, pr.GetBaseRepo())
	assert.NotNil(t, pr.BaseRepo)
	assert.Equal(t, pr.BaseRepoID, pr.BaseRepo.ID)
}

func TestPullRequest_GetHeadRepo(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	pr := AssertExistsAndLoadBean(t, &PullRequest{ID: 1}).(*PullRequest)
	assert.NoError(t, pr.GetHeadRepo())
	assert.NotNil(t, pr.HeadRepo)
	assert.Equal(t, pr.HeadRepoID, pr.HeadRepo.ID)
}

// TODO TestMerge

// TODO TestNewPullRequest

func TestPullRequestsNewest(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	prs, count, err := PullRequests(1, &PullRequestsOptions{
		Page:     1,
		State:    "open",
		SortType: "newest",
		Labels:   []string{},
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assert.Len(t, prs, 2)
	assert.Equal(t, int64(2), prs[0].ID)
	assert.Equal(t, int64(1), prs[1].ID)
}

func TestPullRequestsOldest(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	prs, count, err := PullRequests(1, &PullRequestsOptions{
		Page:     1,
		State:    "open",
		SortType: "oldest",
		Labels:   []string{},
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assert.Len(t, prs, 2)
	assert.Equal(t, int64(1), prs[0].ID)
	assert.Equal(t, int64(2), prs[1].ID)
}

func TestGetUnmergedPullRequest(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	pr, err := GetUnmergedPullRequest(1, 1, "branch2", "master")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), pr.ID)

	pr, err = GetUnmergedPullRequest(1, 9223372036854775807, "branch1", "master")
	assert.Error(t, err)
	assert.True(t, IsErrPullRequestNotExist(err))
}

func TestGetUnmergedPullRequestsByHeadInfo(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	prs, err := GetUnmergedPullRequestsByHeadInfo(1, "branch2")
	assert.NoError(t, err)
	assert.Len(t, prs, 1)
	for _, pr := range prs {
		assert.Equal(t, int64(1), pr.HeadRepoID)
		assert.Equal(t, "branch2", pr.HeadBranch)
	}
}

func TestGetUnmergedPullRequestsByBaseInfo(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	prs, err := GetUnmergedPullRequestsByBaseInfo(1, "master")
	assert.NoError(t, err)
	assert.Len(t, prs, 1)
	pr := prs[0]
	assert.Equal(t, int64(2), pr.ID)
	assert.Equal(t, int64(1), pr.BaseRepoID)
	assert.Equal(t, "master", pr.BaseBranch)
}

func TestGetPullRequestByIndex(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	pr, err := GetPullRequestByIndex(1, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), pr.BaseRepoID)
	assert.Equal(t, int64(2), pr.Index)

	pr, err = GetPullRequestByIndex(9223372036854775807, 9223372036854775807)
	assert.Error(t, err)
	assert.True(t, IsErrPullRequestNotExist(err))
}

func TestGetPullRequestByID(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	pr, err := GetPullRequestByID(1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), pr.ID)
	assert.Equal(t, int64(2), pr.IssueID)

	_, err = GetPullRequestByID(9223372036854775807)
	assert.Error(t, err)
	assert.True(t, IsErrPullRequestNotExist(err))
}

func TestGetPullRequestByIssueID(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	pr, err := GetPullRequestByIssueID(2)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), pr.IssueID)

	pr, err = GetPullRequestByIssueID(9223372036854775807)
	assert.Error(t, err)
	assert.True(t, IsErrPullRequestNotExist(err))
}

func TestPullRequest_Update(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	pr := &PullRequest{
		ID:         1,
		IssueID:    100,
		BaseBranch: "baseBranch",
		HeadBranch: "headBranch",
	}
	pr.Update()

	pr = AssertExistsAndLoadBean(t, &PullRequest{ID: 1}).(*PullRequest)
	assert.Equal(t, int64(100), pr.IssueID)
	assert.Equal(t, "baseBranch", pr.BaseBranch)
	assert.Equal(t, "headBranch", pr.HeadBranch)
}

func TestPullRequest_UpdateCols(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	pr := &PullRequest{
		ID:         1,
		IssueID:    int64(100),
		BaseBranch: "baseBranch",
		HeadBranch: "headBranch",
	}
	pr.UpdateCols("issue_id", "head_branch")

	pr = AssertExistsAndLoadBean(t, &PullRequest{ID: 1}).(*PullRequest)
	assert.Equal(t, int64(100), pr.IssueID)
	assert.Equal(t, "master", pr.BaseBranch)
	assert.Equal(t, "headBranch", pr.HeadBranch)
}

// TODO TestPullRequest_UpdatePatch

// TODO TestPullRequest_PushToBaseRepo

func TestPullRequest_AddToTaskQueue(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	pr := AssertExistsAndLoadBean(t, &PullRequest{ID: 1}).(*PullRequest)
	pr.AddToTaskQueue()

	// briefly sleep so that background threads have time to run
	time.Sleep(time.Millisecond)

	assert.True(t, pullRequestQueue.Exist(pr.ID))
	pr = AssertExistsAndLoadBean(t, &PullRequest{ID: 1}).(*PullRequest)
	assert.Equal(t, PullRequestStatusChecking, pr.Status)
}

func TestPullRequestList_LoadAttributes(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	prs := []*PullRequest{
		AssertExistsAndLoadBean(t, &PullRequest{ID: 1}).(*PullRequest),
		AssertExistsAndLoadBean(t, &PullRequest{ID: 2}).(*PullRequest),
	}
	assert.NoError(t, PullRequestList(prs).LoadAttributes())
	for _, pr := range prs {
		assert.NotNil(t, pr.Issue)
		assert.Equal(t, pr.IssueID, pr.Issue.ID)
	}

	assert.NoError(t, PullRequestList([]*PullRequest{}).LoadAttributes())
}

// TODO TestAddTestPullRequestTask

func TestChangeUsernameInPullRequests(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	const newUsername = "newusername"
	assert.NoError(t, ChangeUsernameInPullRequests("user1", newUsername))

	prs := make([]*PullRequest, 0, 10)
	assert.NoError(t, x.Where("head_user_name = ?", newUsername).Find(&prs))
	assert.Len(t, prs, 2)
	for _, pr := range prs {
		assert.Equal(t, newUsername, pr.HeadUserName)
	}
}
