// Copyright 2016 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"fmt"

	"github.com/Unknwon/com"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/markdown"
	"code.gitea.io/gitea/modules/setting"
)

func (issue *Issue) mailSubject() string {
	return fmt.Sprintf("[%s] %s (#%d)", issue.Repo.Name, issue.Title, issue.Index)
}

// mailIssueCommentToParticipants can be used for both new issue creation and comment.
// This function sends two list of emails:
// 1. Repository watchers and users who are participated in comments.
// 2. Users who are not in 1. but get mentioned in current issue/comment.
func mailIssueCommentToParticipants(issue *Issue, doer *User, comment *Comment, mentions []string) error {
	if !setting.Service.EnableNotifyMail {
		return nil
	}

	watchers, err := GetWatchers(issue.RepoID)
	if err != nil {
		return fmt.Errorf("GetWatchers [repo_id: %d]: %v", issue.RepoID, err)
	}
	participants, err := GetParticipantsByIssueID(issue.ID)
	if err != nil {
		return fmt.Errorf("GetParticipantsByIssueID [issue_id: %d]: %v", issue.ID, err)
	}

	// In case the issue poster is not watching the repository,
	// even if we have duplicated in watchers, can be safely filtered out.
	if issue.PosterID != doer.ID {
		participants = append(participants, issue.Poster)
	}

	tos := make([]string, 0, len(watchers)) // List of email addresses.
	names := make([]string, 0, len(watchers))
	for i := range watchers {
		if watchers[i].UserID == doer.ID {
			continue
		}

		to, err := GetUserByID(watchers[i].UserID)
		if err != nil {
			return fmt.Errorf("GetUserByID [%d]: %v", watchers[i].UserID, err)
		}
		if to.IsOrganization() {
			continue
		}

		tos = append(tos, to.Email)
		names = append(names, to.Name)
	}
	for i := range participants {
		if participants[i].ID == doer.ID {
			continue
		} else if com.IsSliceContainsStr(names, participants[i].Name) {
			continue
		}

		tos = append(tos, participants[i].Email)
		names = append(names, participants[i].Name)
	}

	SendIssueCommentMail(issue, doer, comment, tos)

	// Mail mentioned people and exclude watchers.
	names = append(names, doer.Name)
	tos = make([]string, 0, len(mentions)) // list of user names.
	for i := range mentions {
		if com.IsSliceContainsStr(names, mentions[i]) {
			continue
		}

		tos = append(tos, mentions[i])
	}
	SendIssueMentionMail(issue, doer, comment, GetUserEmailsByNames(tos))

	return nil
}

// MailParticipants sends new issue thread created emails to repository watchers
// and mentioned people.
func (issue *Issue) MailParticipants() (err error) {
	mentions := markdown.FindAllMentions(issue.Content)
	if err = UpdateIssueMentions(x, issue.ID, mentions); err != nil {
		return fmt.Errorf("UpdateIssueMentions [%d]: %v", issue.ID, err)
	}

	if err = mailIssueCommentToParticipants(issue, issue.Poster, nil, mentions); err != nil {
		log.Error(4, "mailIssueCommentToParticipants: %v", err)
	}

	return nil
}
