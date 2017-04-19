// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"gopkg.in/macaron.v1"

	"github.com/go-macaron/binding"
)

// AdminCreateUserForm form for admin to create user
type AdminCreateUserForm struct {
	LoginType  string `binding:"Required"`
	LoginName  string
	UserName   string `binding:"Required;AlphaDashDot;MaxSize(35)"`
	Email      string `binding:"Required;Email;MaxSize(254)"`
	Password   string `binding:"MaxSize(255)"`
	SendNotify bool
}

// Validate validates form fields
func (f *AdminCreateUserForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}

// AdminEditUserForm form for admin to create user
type AdminEditUserForm struct {
	LoginType               string `binding:"Required"`
	LoginName               string
	FullName                string `binding:"MaxSize(100)"`
	Email                   string `binding:"Required;Email;MaxSize(254)"`
	Password                string `binding:"MaxSize(255)"`
	Website                 string `binding:"ValidUrl;MaxSize(255)"`
	Location                string `binding:"MaxSize(50)"`
	MaxRepoCreation         int
	Active                  bool
	Admin                   bool
	AllowGitHook            bool
	AllowImportLocal        bool
	AllowCreateOrganization bool
	ProhibitLogin           bool
}

// Validate validates form fields
func (f *AdminEditUserForm) Validate(ctx *macaron.Context, errs binding.Errors) binding.Errors {
	return validate(errs, ctx.Data, f, ctx.Locale)
}
