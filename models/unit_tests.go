// Copyright 2016 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"testing"

	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	_ "github.com/mattn/go-sqlite3" // for the test engine
	"github.com/stretchr/testify/assert"
	"gopkg.in/testfixtures.v2"
)

// NonexistentID an ID that will never exist
const NonexistentID = 9223372036854775807

// CreateTestEngine create an xorm engine for testing
func CreateTestEngine() error {
	var err error
	x, err = xorm.NewEngine("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		return err
	}
	x.SetMapper(core.GonicMapper{})
	if err = x.StoreEngine("InnoDB").Sync2(tables...); err != nil {
		return err
	}

	return InitFixtures(&testfixtures.SQLite{}, "fixtures/")
}

// PrepareTestDatabase load test fixtures into test database
func PrepareTestDatabase() error {
	return LoadFixtures()
}

func loadBeanIfExists(bean interface{}, conditions ...interface{}) (bool, error) {
	sess := x.NewSession()
	defer sess.Close()

	for _, cond := range conditions {
		sess = sess.Where(cond)
	}
	return sess.Get(bean)
}

// BeanExists for testing, check if a bean exists
func BeanExists(t *testing.T, bean interface{}, conditions ...interface{}) bool {
	exists, err := loadBeanIfExists(bean, conditions...)
	assert.NoError(t, err)
	return exists
}

// AssertExistsAndLoadBean assert that a bean exists and load it from the test
// database
func AssertExistsAndLoadBean(t *testing.T, bean interface{}, conditions ...interface{}) interface{} {
	exists, err := loadBeanIfExists(bean, conditions...)
	assert.NoError(t, err)
	assert.True(t, exists,
		"Expected to find %+v (of type %T, with conditions %+v), but did not",
		bean, bean, conditions)
	return bean
}

// AssertNotExistsBean assert that a bean does not exist in the test database
func AssertNotExistsBean(t *testing.T, bean interface{}, conditions ...interface{}) {
	exists, err := loadBeanIfExists(bean, conditions...)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// AssertSuccessfulInsert assert that beans is successfully inserted
func AssertSuccessfulInsert(t *testing.T, beans ...interface{}) {
	_, err := x.Insert(beans...)
	assert.NoError(t, err)
}

// AssertCount assert the count of a bean
func AssertCount(t *testing.T, bean interface{}, expected interface{}) {
	actual, err := x.Count(bean)
	assert.NoError(t, err)
	assert.EqualValues(t, expected, actual)
}
