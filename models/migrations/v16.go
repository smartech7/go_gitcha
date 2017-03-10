// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"fmt"
	"time"

	"code.gitea.io/gitea/modules/markdown"

	"github.com/go-xorm/xorm"
)

// RepoUnit describes all units of a repository
type RepoUnit struct {
	ID          int64
	RepoID      int64 `xorm:"INDEX(s)"`
	Type        int   `xorm:"INDEX(s)"`
	Index       int
	Config      map[string]string `xorm:"JSON"`
	CreatedUnix int64             `xorm:"INDEX CREATED"`
	Created     time.Time         `xorm:"-"`
}

// Enumerate all the unit types
const (
	UnitTypeCode            = iota + 1 // 1 code
	UnitTypeIssues                     // 2 issues
	UnitTypePRs                        // 3 PRs
	UnitTypeCommits                    // 4 Commits
	UnitTypeReleases                   // 5 Releases
	UnitTypeWiki                       // 6 Wiki
	UnitTypeSettings                   // 7 Settings
	UnitTypeExternalWiki               // 8 ExternalWiki
	UnitTypeExternalTracker            // 9 ExternalTracker
)

// Repo describes a repository
type Repo struct {
	ID                                                                               int64
	EnableWiki, EnableExternalWiki, EnableIssues, EnableExternalTracker, EnablePulls bool
	ExternalWikiURL, ExternalTrackerURL, ExternalTrackerFormat, ExternalTrackerStyle string
}

func addUnitsToTables(x *xorm.Engine) error {
	var repos []Repo
	err := x.Table("repository").Select("*").Find(&repos)
	if err != nil {
		return fmt.Errorf("Query repositories: %v", err)
	}

	sess := x.NewSession()
	defer sess.Close()

	if err := sess.Begin(); err != nil {
		return err
	}

	var repoUnit RepoUnit
	if exist, err := sess.IsTableExist(&repoUnit); err != nil {
		return fmt.Errorf("IsExist RepoUnit: %v", err)
	} else if exist {
		return nil
	}

	if err := sess.CreateTable(&repoUnit); err != nil {
		return fmt.Errorf("CreateTable RepoUnit: %v", err)
	}

	if err := sess.CreateUniques(&repoUnit); err != nil {
		return fmt.Errorf("CreateUniques RepoUnit: %v", err)
	}

	if err := sess.CreateIndexes(&repoUnit); err != nil {
		return fmt.Errorf("CreateIndexes RepoUnit: %v", err)
	}

	for _, repo := range repos {
		for i := 1; i <= 9; i++ {
			if (i == UnitTypeWiki || i == UnitTypeExternalWiki) && !repo.EnableWiki {
				continue
			}
			if i == UnitTypeExternalWiki && !repo.EnableExternalWiki {
				continue
			}
			if i == UnitTypePRs && !repo.EnablePulls {
				continue
			}
			if (i == UnitTypeIssues || i == UnitTypeExternalTracker) && !repo.EnableIssues {
				continue
			}
			if i == UnitTypeExternalTracker && !repo.EnableExternalTracker {
				continue
			}

			var config = make(map[string]string)
			switch i {
			case UnitTypeExternalTracker:
				config["ExternalTrackerURL"] = repo.ExternalTrackerURL
				config["ExternalTrackerFormat"] = repo.ExternalTrackerFormat
				if len(repo.ExternalTrackerStyle) == 0 {
					repo.ExternalTrackerStyle = markdown.IssueNameStyleNumeric
				}
				config["ExternalTrackerStyle"] = repo.ExternalTrackerStyle
			case UnitTypeExternalWiki:
				config["ExternalWikiURL"] = repo.ExternalWikiURL
			}

			if _, err = sess.Insert(&RepoUnit{
				RepoID: repo.ID,
				Type:   i,
				Index:  i,
				Config: config,
			}); err != nil {
				return fmt.Errorf("Insert repo unit: %v", err)
			}
		}
	}

	if err := sess.Commit(); err != nil {
		return err
	}

	return nil
}
