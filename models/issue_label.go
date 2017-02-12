// Copyright 2016 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"fmt"
	"html/template"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-xorm/xorm"

	api "code.gitea.io/sdk/gitea"
)

var labelColorPattern = regexp.MustCompile("#([a-fA-F0-9]{6})")

// GetLabelTemplateFile loads the label template file by given name,
// then parses and returns a list of name-color pairs.
func GetLabelTemplateFile(name string) ([][2]string, error) {
	data, err := getRepoInitFile("label", name)
	if err != nil {
		return nil, fmt.Errorf("getRepoInitFile: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	list := make([][2]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if len(line) == 0 {
			continue
		}

		fields := strings.SplitN(line, " ", 2)
		if len(fields) != 2 {
			return nil, fmt.Errorf("line is malformed: %s", line)
		}

		if !labelColorPattern.MatchString(fields[0]) {
			return nil, fmt.Errorf("bad HTML color code in line: %s", line)
		}

		fields[1] = strings.TrimSpace(fields[1])
		list = append(list, [2]string{fields[1], fields[0]})
	}

	return list, nil
}

// Label represents a label of repository for issues.
type Label struct {
	ID              int64 `xorm:"pk autoincr"`
	RepoID          int64 `xorm:"INDEX"`
	Name            string
	Color           string `xorm:"VARCHAR(7)"`
	NumIssues       int
	NumClosedIssues int
	NumOpenIssues   int  `xorm:"-"`
	IsChecked       bool `xorm:"-"`
}

// APIFormat converts a Label to the api.Label format
func (label *Label) APIFormat() *api.Label {
	return &api.Label{
		ID:    label.ID,
		Name:  label.Name,
		Color: strings.TrimLeft(label.Color, "#"),
	}
}

// CalOpenIssues calculates the open issues of label.
func (label *Label) CalOpenIssues() {
	label.NumOpenIssues = label.NumIssues - label.NumClosedIssues
}

// ForegroundColor calculates the text color for labels based
// on their background color.
func (label *Label) ForegroundColor() template.CSS {
	if strings.HasPrefix(label.Color, "#") {
		if color, err := strconv.ParseUint(label.Color[1:], 16, 64); err == nil {
			r := float32(0xFF & (color >> 16))
			g := float32(0xFF & (color >> 8))
			b := float32(0xFF & color)
			luminance := (0.2126*r + 0.7152*g + 0.0722*b) / 255

			if luminance < 0.66 {
				return template.CSS("#fff")
			}
		}
	}

	// default to black
	return template.CSS("#000")
}

// NewLabels creates new label(s) for a repository.
func NewLabels(labels ...*Label) error {
	_, err := x.Insert(labels)
	return err
}

// getLabelInRepoByName returns a label by Name in given repository.
// If pass repoID as 0, then ORM will ignore limitation of repository
// and can return arbitrary label with any valid ID.
func getLabelInRepoByName(e Engine, repoID int64, labelName string) (*Label, error) {
	if len(labelName) <= 0 {
		return nil, ErrLabelNotExist{0, repoID}
	}

	l := &Label{
		Name:   labelName,
		RepoID: repoID,
	}
	has, err := e.Get(l)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrLabelNotExist{0, l.RepoID}
	}
	return l, nil
}

// getLabelInRepoByID returns a label by ID in given repository.
// If pass repoID as 0, then ORM will ignore limitation of repository
// and can return arbitrary label with any valid ID.
func getLabelInRepoByID(e Engine, repoID, labelID int64) (*Label, error) {
	if labelID <= 0 {
		return nil, ErrLabelNotExist{labelID, repoID}
	}

	l := &Label{
		ID:     labelID,
		RepoID: repoID,
	}
	has, err := e.Get(l)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrLabelNotExist{l.ID, l.RepoID}
	}
	return l, nil
}

// GetLabelByID returns a label by given ID.
func GetLabelByID(id int64) (*Label, error) {
	return getLabelInRepoByID(x, 0, id)
}

// GetLabelInRepoByName returns a label by name in given repository.
func GetLabelInRepoByName(repoID int64, labelName string) (*Label, error) {
	return getLabelInRepoByName(x, repoID, labelName)
}

// GetLabelInRepoByID returns a label by ID in given repository.
func GetLabelInRepoByID(repoID, labelID int64) (*Label, error) {
	return getLabelInRepoByID(x, repoID, labelID)
}

// GetLabelsInRepoByIDs returns a list of labels by IDs in given repository,
// it silently ignores label IDs that are not belong to the repository.
func GetLabelsInRepoByIDs(repoID int64, labelIDs []int64) ([]*Label, error) {
	labels := make([]*Label, 0, len(labelIDs))
	return labels, x.
		Where("repo_id = ?", repoID).
		In("id", labelIDs).
		Asc("name").
		Find(&labels)
}

// GetLabelsByRepoID returns all labels that belong to given repository by ID.
func GetLabelsByRepoID(repoID int64, sortType string) ([]*Label, error) {
	labels := make([]*Label, 0, 10)
	sess := x.Where("repo_id = ?", repoID)

	switch sortType {
	case "reversealphabetically":
		sess.Desc("name")
	case "leastissues":
		sess.Asc("num_issues")
	case "mostissues":
		sess.Desc("num_issues")
	default:
		sess.Asc("name")
	}

	return labels, sess.Find(&labels)
}

func getLabelsByIssueID(e Engine, issueID int64) ([]*Label, error) {
	var labels []*Label
	return labels, e.Where("issue_label.issue_id = ?", issueID).
		Join("LEFT", "issue_label", "issue_label.label_id = label.id").
		Asc("label.name").
		Find(&labels)
}

// GetLabelsByIssueID returns all labels that belong to given issue by ID.
func GetLabelsByIssueID(issueID int64) ([]*Label, error) {
	return getLabelsByIssueID(x, issueID)
}

func updateLabel(e Engine, l *Label) error {
	_, err := e.Id(l.ID).AllCols().Update(l)
	return err
}

// UpdateLabel updates label information.
func UpdateLabel(l *Label) error {
	return updateLabel(x, l)
}

// DeleteLabel delete a label of given repository.
func DeleteLabel(repoID, labelID int64) error {
	_, err := GetLabelInRepoByID(repoID, labelID)
	if err != nil {
		if IsErrLabelNotExist(err) {
			return nil
		}
		return err
	}

	sess := x.NewSession()
	defer sessionRelease(sess)
	if err = sess.Begin(); err != nil {
		return err
	}

	if _, err = sess.Id(labelID).Delete(new(Label)); err != nil {
		return err
	} else if _, err = sess.
		Where("label_id = ?", labelID).
		Delete(new(IssueLabel)); err != nil {
		return err
	}

	// Clear label id in comment table
	if _, err = sess.Where("label_id = ?", labelID).Cols("label_id").Update(&Comment{}); err != nil {
		return err
	}

	return sess.Commit()
}

// .___                            .____          ___.          .__
// |   | ______ ________ __   ____ |    |   _____ \_ |__   ____ |  |
// |   |/  ___//  ___/  |  \_/ __ \|    |   \__  \ | __ \_/ __ \|  |
// |   |\___ \ \___ \|  |  /\  ___/|    |___ / __ \| \_\ \  ___/|  |__
// |___/____  >____  >____/  \___  >_______ (____  /___  /\___  >____/
//          \/     \/            \/        \/    \/    \/     \/

// IssueLabel represents an issue-label relation.
type IssueLabel struct {
	ID      int64 `xorm:"pk autoincr"`
	IssueID int64 `xorm:"UNIQUE(s)"`
	LabelID int64 `xorm:"UNIQUE(s)"`
}

func hasIssueLabel(e Engine, issueID, labelID int64) bool {
	has, _ := e.Where("issue_id = ? AND label_id = ?", issueID, labelID).Get(new(IssueLabel))
	return has
}

// HasIssueLabel returns true if issue has been labeled.
func HasIssueLabel(issueID, labelID int64) bool {
	return hasIssueLabel(x, issueID, labelID)
}

func newIssueLabel(e *xorm.Session, issue *Issue, label *Label, doer *User) (err error) {
	if _, err = e.Insert(&IssueLabel{
		IssueID: issue.ID,
		LabelID: label.ID,
	}); err != nil {
		return err
	}

	if err = issue.loadRepo(e); err != nil {
		return
	}

	if _, err = createLabelComment(e, doer, issue.Repo, issue, label, true); err != nil {
		return err
	}

	label.NumIssues++
	if issue.IsClosed {
		label.NumClosedIssues++
	}
	return updateLabel(e, label)
}

// NewIssueLabel creates a new issue-label relation.
func NewIssueLabel(issue *Issue, label *Label, doer *User) (err error) {
	if HasIssueLabel(issue.ID, label.ID) {
		return nil
	}

	sess := x.NewSession()
	defer sessionRelease(sess)
	if err = sess.Begin(); err != nil {
		return err
	}

	if err = newIssueLabel(sess, issue, label, doer); err != nil {
		return err
	}

	return sess.Commit()
}

func newIssueLabels(e *xorm.Session, issue *Issue, labels []*Label, doer *User) (err error) {
	for i := range labels {
		if hasIssueLabel(e, issue.ID, labels[i].ID) {
			continue
		}

		if err = newIssueLabel(e, issue, labels[i], doer); err != nil {
			return fmt.Errorf("newIssueLabel: %v", err)
		}
	}

	return nil
}

// NewIssueLabels creates a list of issue-label relations.
func NewIssueLabels(issue *Issue, labels []*Label, doer *User) (err error) {
	sess := x.NewSession()
	defer sessionRelease(sess)
	if err = sess.Begin(); err != nil {
		return err
	}

	if err = newIssueLabels(sess, issue, labels, doer); err != nil {
		return err
	}

	return sess.Commit()
}

func getIssueLabels(e Engine, issueID int64) ([]*IssueLabel, error) {
	issueLabels := make([]*IssueLabel, 0, 10)
	return issueLabels, e.
		Where("issue_id=?", issueID).
		Asc("label_id").
		Find(&issueLabels)
}

func deleteIssueLabel(e *xorm.Session, issue *Issue, label *Label, doer *User) (err error) {
	if count, err := e.Delete(&IssueLabel{
		IssueID: issue.ID,
		LabelID: label.ID,
	}); err != nil {
		return err
	} else if count == 0 {
		return nil
	}

	if err = issue.loadRepo(e); err != nil {
		return
	}

	if _, err = createLabelComment(e, doer, issue.Repo, issue, label, false); err != nil {
		return err
	}

	label.NumIssues--
	if issue.IsClosed {
		label.NumClosedIssues--
	}
	return updateLabel(e, label)
}

// DeleteIssueLabel deletes issue-label relation.
func DeleteIssueLabel(issue *Issue, label *Label, doer *User) (err error) {
	sess := x.NewSession()
	defer sessionRelease(sess)
	if err = sess.Begin(); err != nil {
		return err
	}

	if err = deleteIssueLabel(sess, issue, label, doer); err != nil {
		return err
	}

	return sess.Commit()
}
