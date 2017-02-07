// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateOrUpdateIssueNotifications(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	issue := AssertExistsAndLoadBean(t, &Issue{ID: 1}).(*Issue)

	assert.NoError(t, CreateOrUpdateIssueNotifications(issue, 2))

	notf := AssertExistsAndLoadBean(t, &Notification{UserID: 1, IssueID: issue.ID}).(*Notification)
	assert.Equal(t, NotificationStatusUnread, notf.Status)
	notf = AssertExistsAndLoadBean(t, &Notification{UserID: 4, IssueID: issue.ID}).(*Notification)
	assert.Equal(t, NotificationStatusUnread, notf.Status)
	CheckConsistencyFor(t, &Issue{ID: issue.ID})
}

func TestNotificationsForUser(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	user := AssertExistsAndLoadBean(t, &User{ID: 2}).(*User)
	statuses := []NotificationStatus{NotificationStatusRead, NotificationStatusUnread}
	notfs, err := NotificationsForUser(user, statuses, 1, 10)
	assert.NoError(t, err)
	assert.Len(t, notfs, 1)
	assert.EqualValues(t, 2, notfs[0].ID)
	assert.EqualValues(t, user.ID, notfs[0].UserID)
}

func TestNotification_GetRepo(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	notf := AssertExistsAndLoadBean(t, &Notification{RepoID: 1}).(*Notification)
	repo, err := notf.GetRepo()
	assert.NoError(t, err)
	assert.Equal(t, repo, notf.Repository)
	assert.EqualValues(t, notf.RepoID, repo.ID)
}

func TestNotification_GetIssue(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	notf := AssertExistsAndLoadBean(t, &Notification{RepoID: 1}).(*Notification)
	issue, err := notf.GetIssue()
	assert.NoError(t, err)
	assert.Equal(t, issue, notf.Issue)
	assert.EqualValues(t, notf.IssueID, issue.ID)
}

func TestGetNotificationCount(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	user := AssertExistsAndLoadBean(t, &User{ID: 2}).(*User)
	cnt, err := GetNotificationCount(user, NotificationStatusUnread)
	assert.NoError(t, err)
	assert.EqualValues(t, 0, cnt)

	cnt, err = GetNotificationCount(user, NotificationStatusRead)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, cnt)
}

func TestSetNotificationStatus(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	user := AssertExistsAndLoadBean(t, &User{ID: 2}).(*User)
	notf := AssertExistsAndLoadBean(t,
		&Notification{UserID: user.ID, Status: NotificationStatusRead}).(*Notification)
	assert.NoError(t, SetNotificationStatus(notf.ID, user, NotificationStatusPinned))
	AssertExistsAndLoadBean(t,
		&Notification{ID: notf.ID, Status: NotificationStatusPinned})

	assert.Error(t, SetNotificationStatus(1, user, NotificationStatusRead))
	assert.Error(t, SetNotificationStatus(NonexistentID, user, NotificationStatusRead))
}
