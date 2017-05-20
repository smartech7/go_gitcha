// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package integrations

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"

	"code.gitea.io/gitea/models"

	"github.com/stretchr/testify/assert"
)

func TestDeleteUser(t *testing.T) {
	prepareTestEnv(t)

	session := loginUser(t, "user1", "password")

	req, err := http.NewRequest("GET", "/admin/users/8", nil)
	assert.NoError(t, err)
	resp := session.MakeRequest(t, req)
	assert.EqualValues(t, http.StatusOK, resp.HeaderCode)

	doc, err := NewHtmlParser(resp.Body)
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", "/admin/users/8/delete",
		bytes.NewBufferString(url.Values{
			"_csrf": []string{doc.GetInputValueByName("_csrf")},
		}.Encode()))
	assert.NoError(t, err)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp = session.MakeRequest(t, req)
	assert.EqualValues(t, http.StatusOK, resp.HeaderCode)

	models.AssertNotExistsBean(t, &models.User{ID: 8})
	models.CheckConsistencyFor(t, &models.User{})
}
