// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTeam_IsOwnerTeam(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	team := AssertExistsAndLoadBean(t, &Team{ID: 1}).(*Team)
	assert.True(t, team.IsOwnerTeam())

	team = AssertExistsAndLoadBean(t, &Team{ID: 2}).(*Team)
	assert.False(t, team.IsOwnerTeam())
}

func TestTeam_IsMember(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	team := AssertExistsAndLoadBean(t, &Team{ID: 1}).(*Team)
	assert.True(t, team.IsMember(2))
	assert.False(t, team.IsMember(4))
	assert.False(t, team.IsMember(NonexistentID))

	team = AssertExistsAndLoadBean(t, &Team{ID: 2}).(*Team)
	assert.True(t, team.IsMember(2))
	assert.True(t, team.IsMember(4))
	assert.False(t, team.IsMember(NonexistentID))
}

func TestTeam_GetRepositories(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	test := func(teamID int64) {
		team := AssertExistsAndLoadBean(t, &Team{ID: teamID}).(*Team)
		assert.NoError(t, team.GetRepositories())
		assert.Len(t, team.Repos, team.NumRepos)
		for _, repo := range team.Repos {
			AssertExistsAndLoadBean(t, &TeamRepo{TeamID: teamID, RepoID: repo.ID})
		}
	}
	test(1)
	test(3)
}

func TestTeam_GetMembers(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	test := func(teamID int64) {
		team := AssertExistsAndLoadBean(t, &Team{ID: teamID}).(*Team)
		assert.NoError(t, team.GetMembers())
		assert.Len(t, team.Members, team.NumMembers)
		for _, member := range team.Members {
			AssertExistsAndLoadBean(t, &TeamUser{UID: member.ID, TeamID: teamID})
		}
	}
	test(1)
	test(3)
}

func TestTeam_AddMember(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	test := func(teamID, userID int64) {
		team := AssertExistsAndLoadBean(t, &Team{ID: teamID}).(*Team)
		assert.NoError(t, team.AddMember(userID))
		AssertExistsAndLoadBean(t, &TeamUser{UID: userID, TeamID: teamID})
		CheckConsistencyFor(t, &Team{ID: teamID}, &User{ID: team.OrgID})
	}
	test(1, 2)
	test(1, 4)
	test(3, 2)
}

func TestTeam_RemoveMember(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	testSuccess := func(teamID, userID int64) {
		team := AssertExistsAndLoadBean(t, &Team{ID: teamID}).(*Team)
		assert.NoError(t, team.RemoveMember(userID))
		AssertNotExistsBean(t, &TeamUser{UID: userID, TeamID: teamID})
		CheckConsistencyFor(t, &Team{ID: teamID})
	}
	testSuccess(1, 4)
	testSuccess(2, 2)
	testSuccess(3, 2)
	testSuccess(3, NonexistentID)

	team := AssertExistsAndLoadBean(t, &Team{ID: 1}).(*Team)
	err := team.RemoveMember(2)
	assert.True(t, IsErrLastOrgOwner(err))
}

func TestTeam_HasRepository(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	test := func(teamID, repoID int64, expected bool) {
		team := AssertExistsAndLoadBean(t, &Team{ID: teamID}).(*Team)
		assert.Equal(t, expected, team.HasRepository(repoID))
	}
	test(1, 1, false)
	test(1, 3, true)
	test(1, 5, true)
	test(1, NonexistentID, false)

	test(2, 3, true)
	test(2, 5, false)
}

func TestTeam_AddRepository(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	testSuccess := func(teamID, repoID int64) {
		team := AssertExistsAndLoadBean(t, &Team{ID: teamID}).(*Team)
		repo := AssertExistsAndLoadBean(t, &Repository{ID: repoID}).(*Repository)
		assert.NoError(t, team.AddRepository(repo))
		AssertExistsAndLoadBean(t, &TeamRepo{TeamID: teamID, RepoID: repoID})
		CheckConsistencyFor(t, &Team{ID: teamID}, &Repository{ID: repoID})
	}
	testSuccess(2, 3)
	testSuccess(2, 5)

	team := AssertExistsAndLoadBean(t, &Team{ID: 1}).(*Team)
	repo := AssertExistsAndLoadBean(t, &Repository{ID: 1}).(*Repository)
	assert.Error(t, team.AddRepository(repo))
	CheckConsistencyFor(t, &Team{ID: 1}, &Repository{ID: 1})
}

func TestTeam_RemoveRepository(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	testSuccess := func(teamID, repoID int64) {
		team := AssertExistsAndLoadBean(t, &Team{ID: teamID}).(*Team)
		assert.NoError(t, team.RemoveRepository(repoID))
		AssertNotExistsBean(t, &TeamRepo{TeamID: teamID, RepoID: repoID})
		CheckConsistencyFor(t, &Team{ID: teamID}, &Repository{ID: repoID})
	}
	testSuccess(2, 3)
	testSuccess(2, 5)
	testSuccess(1, NonexistentID)
}

func TestIsUsableTeamName(t *testing.T) {
	assert.NoError(t, IsUsableTeamName("usable"))
	assert.True(t, IsErrNameReserved(IsUsableTeamName("new")))
}

func TestNewTeam(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	const teamName = "newTeamName"
	team := &Team{Name: teamName, OrgID: 3}
	assert.NoError(t, NewTeam(team))
	AssertExistsAndLoadBean(t, &Team{Name: teamName})
	CheckConsistencyFor(t, &Team{}, &User{ID: team.OrgID})
}

func TestGetTeam(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	testSuccess := func(orgID int64, name string) {
		team, err := GetTeam(orgID, name)
		assert.NoError(t, err)
		assert.EqualValues(t, orgID, team.OrgID)
		assert.Equal(t, name, team.Name)
	}
	testSuccess(3, "Owners")
	testSuccess(3, "team1")

	_, err := GetTeam(3, "nonexistent")
	assert.Error(t, err)
	_, err = GetTeam(NonexistentID, "Owners")
	assert.Error(t, err)
}

func TestGetTeamByID(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	testSuccess := func(teamID int64) {
		team, err := GetTeamByID(teamID)
		assert.NoError(t, err)
		assert.EqualValues(t, teamID, team.ID)
	}
	testSuccess(1)
	testSuccess(2)
	testSuccess(3)
	testSuccess(4)

	_, err := GetTeamByID(NonexistentID)
	assert.Error(t, err)
}

func TestUpdateTeam(t *testing.T) {
	// successful update
	assert.NoError(t, PrepareTestDatabase())

	team := AssertExistsAndLoadBean(t, &Team{ID: 2}).(*Team)
	team.LowerName = "newname"
	team.Name = "newName"
	team.Description = strings.Repeat("A long description!", 100)
	team.Authorize = AccessModeAdmin
	assert.NoError(t, UpdateTeam(team, true))

	team = AssertExistsAndLoadBean(t, &Team{Name: "newName"}).(*Team)
	assert.True(t, strings.HasPrefix(team.Description, "A long description!"))

	access := AssertExistsAndLoadBean(t, &Access{UserID: 4, RepoID: 3}).(*Access)
	assert.EqualValues(t, AccessModeAdmin, access.Mode)

	CheckConsistencyFor(t, &Team{ID: team.ID})
}

func TestUpdateTeam2(t *testing.T) {
	// update to already-existing team
	assert.NoError(t, PrepareTestDatabase())

	team := AssertExistsAndLoadBean(t, &Team{ID: 2}).(*Team)
	team.LowerName = "owners"
	team.Name = "Owners"
	team.Description = strings.Repeat("A long description!", 100)
	err := UpdateTeam(team, true)
	assert.True(t, IsErrTeamAlreadyExist(err))

	CheckConsistencyFor(t, &Team{ID: team.ID})
}

func TestDeleteTeam(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	team := AssertExistsAndLoadBean(t, &Team{ID: 2}).(*Team)
	assert.NoError(t, DeleteTeam(team))
	AssertNotExistsBean(t, &Team{ID: team.ID})
	AssertNotExistsBean(t, &TeamRepo{TeamID: team.ID})
	AssertNotExistsBean(t, &TeamUser{TeamID: team.ID})

	// check that team members don't have "leftover" access to repos
	user := AssertExistsAndLoadBean(t, &User{ID: 4}).(*User)
	repo := AssertExistsAndLoadBean(t, &Repository{ID: 3}).(*Repository)
	accessMode, err := AccessLevel(user.ID, repo)
	assert.NoError(t, err)
	assert.True(t, accessMode < AccessModeWrite)
}

func TestIsTeamMember(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	assert.True(t, IsTeamMember(3, 1, 2))
	assert.False(t, IsTeamMember(3, 1, 4))
	assert.False(t, IsTeamMember(3, 1, NonexistentID))

	assert.True(t, IsTeamMember(3, 2, 2))
	assert.True(t, IsTeamMember(3, 2, 4))

	assert.False(t, IsTeamMember(3, NonexistentID, NonexistentID))
	assert.False(t, IsTeamMember(NonexistentID, NonexistentID, NonexistentID))
}

func TestGetTeamMembers(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	test := func(teamID int64) {
		team := AssertExistsAndLoadBean(t, &Team{ID: teamID}).(*Team)
		members, err := GetTeamMembers(teamID)
		assert.NoError(t, err)
		assert.Len(t, members, team.NumMembers)
		for _, member := range members {
			AssertExistsAndLoadBean(t, &TeamUser{UID: member.ID, TeamID: teamID})
		}
	}
	test(1)
	test(3)
}

func TestGetUserTeams(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())
	test := func(orgID, userID int64) {
		teams, err := GetUserTeams(orgID, userID)
		assert.NoError(t, err)
		for _, team := range teams {
			assert.EqualValues(t, orgID, team.OrgID)
			AssertExistsAndLoadBean(t, &TeamUser{TeamID: team.ID, UID: userID})
		}
	}
	test(3, 2)
	test(3, 4)
	test(3, NonexistentID)
}

func TestAddTeamMember(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	test := func(teamID, userID int64) {
		team := AssertExistsAndLoadBean(t, &Team{ID: teamID}).(*Team)
		assert.NoError(t, AddTeamMember(team, userID))
		AssertExistsAndLoadBean(t, &TeamUser{UID: userID, TeamID: teamID})
		CheckConsistencyFor(t, &Team{ID: teamID}, &User{ID: team.OrgID})
	}
	test(1, 2)
	test(1, 4)
	test(3, 2)
}

func TestRemoveTeamMember(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	testSuccess := func(teamID, userID int64) {
		team := AssertExistsAndLoadBean(t, &Team{ID: teamID}).(*Team)
		assert.NoError(t, RemoveTeamMember(team, userID))
		AssertNotExistsBean(t, &TeamUser{UID: userID, TeamID: teamID})
		CheckConsistencyFor(t, &Team{ID: teamID})
	}
	testSuccess(1, 4)
	testSuccess(2, 2)
	testSuccess(3, 2)
	testSuccess(3, NonexistentID)

	team := AssertExistsAndLoadBean(t, &Team{ID: 1}).(*Team)
	err := RemoveTeamMember(team, 2)
	assert.True(t, IsErrLastOrgOwner(err))
}

func TestHasTeamRepo(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	test := func(teamID, repoID int64, expected bool) {
		team := AssertExistsAndLoadBean(t, &Team{ID: teamID}).(*Team)
		assert.Equal(t, expected, HasTeamRepo(team.OrgID, teamID, repoID))
	}
	test(1, 1, false)
	test(1, 3, true)
	test(1, 5, true)
	test(1, NonexistentID, false)

	test(2, 3, true)
	test(2, 5, false)
}
