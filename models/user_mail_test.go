// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEmailAddresses(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	emails, _ := GetEmailAddresses(int64(1))
	assert.Len(t, emails, 3)
	assert.False(t, emails[0].IsPrimary)
	assert.True(t, emails[2].IsActivated)
	assert.True(t, emails[2].IsPrimary)

	emails, _ = GetEmailAddresses(int64(2))
	assert.Len(t, emails, 2)
	assert.True(t, emails[0].IsPrimary)
	assert.True(t, emails[0].IsActivated)
}

func TestIsEmailUsed(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	isExist, _ := IsEmailUsed("")
	assert.True(t, isExist)
	isExist, _ = IsEmailUsed("user11@example.com")
	assert.True(t, isExist)
	isExist, _ = IsEmailUsed("user1234567890@example.com")
	assert.False(t, isExist)
}

func TestAddEmailAddress(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	assert.NoError(t, AddEmailAddress(&EmailAddress{
		Email:       "user1234567890@example.com",
		IsPrimary:   true,
		IsActivated: true,
	}))

	// ErrEmailAlreadyUsed
	err := AddEmailAddress(&EmailAddress{
		Email: "user1234567890@example.com",
	})
	assert.Error(t, err)
	assert.True(t, IsErrEmailAlreadyUsed(err))
}

func TestAddEmailAddresses(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	// insert multiple email address
	emails := make([]*EmailAddress, 2)
	emails[0] = &EmailAddress{
		Email:       "user1234@example.com",
		IsActivated: true,
	}
	emails[1] = &EmailAddress{
		Email:       "user5678@example.com",
		IsActivated: true,
	}
	assert.NoError(t, AddEmailAddresses(emails))

	// ErrEmailAlreadyUsed
	err := AddEmailAddresses(emails)
	assert.Error(t, err)
	assert.True(t, IsErrEmailAlreadyUsed(err))
}

func TestDeleteEmailAddress(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	assert.NoError(t, DeleteEmailAddress(&EmailAddress{
		UID:   int64(1),
		ID:    int64(1),
		Email: "user11@example.com",
	}))

	assert.NoError(t, DeleteEmailAddress(&EmailAddress{
		UID:   int64(1),
		Email: "user12@example.com",
	}))

	// Email address does not exist
	err := DeleteEmailAddress(&EmailAddress{
		UID:   int64(1),
		Email: "user1234567890@example.com",
	})
	assert.Error(t, err)
}

func TestDeleteEmailAddresses(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	// delete multiple email address
	emails := make([]*EmailAddress, 2)
	emails[0] = &EmailAddress{
		UID:   int64(2),
		ID:    int64(3),
		Email: "user2@example.com",
	}
	emails[1] = &EmailAddress{
		UID:   int64(2),
		Email: "user21@example.com",
	}
	assert.NoError(t, DeleteEmailAddresses(emails))

	// ErrEmailAlreadyUsed
	err := DeleteEmailAddresses(emails)
	assert.Error(t, err)
}

func TestMakeEmailPrimary(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	email := &EmailAddress{
		Email: "user567890@example.com",
	}
	err := MakeEmailPrimary(email)
	assert.Error(t, err)
	assert.Equal(t, ErrEmailNotExist.Error(), err.Error())

	email = &EmailAddress{
		Email: "user11@example.com",
	}
	err = MakeEmailPrimary(email)
	assert.Error(t, err)
	assert.Equal(t, ErrEmailNotActivated.Error(), err.Error())

	email = &EmailAddress{
		Email: "user9999999@example.com",
	}
	err = MakeEmailPrimary(email)
	assert.Error(t, err)
	assert.True(t, IsErrUserNotExist(err))

	email = &EmailAddress{
		Email: "user101@example.com",
	}
	err = MakeEmailPrimary(email)
	assert.NoError(t, err)

	user, _ := GetUserByID(int64(10))
	assert.Equal(t, "user101@example.com", user.Email)
}

func TestActivate(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	email := &EmailAddress{
		ID:    int64(1),
		UID:   int64(1),
		Email: "user11@example.com",
	}
	assert.NoError(t, email.Activate())

	emails, _ := GetEmailAddresses(int64(1))
	assert.Len(t, emails, 3)
	assert.True(t, emails[0].IsActivated)
	assert.True(t, emails[2].IsActivated)
	assert.True(t, emails[2].IsPrimary)
}
