// Copyright 2016 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package org

import (
	api "code.gitea.io/sdk/gitea"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/routers/api/v1/convert"
	"code.gitea.io/gitea/routers/api/v1/utils"
)

// ListHooks list an organziation's webhooks
func ListHooks(ctx *context.APIContext) {
	org := ctx.Org.Organization
	orgHooks, err := models.GetWebhooksByOrgID(org.ID)
	if err != nil {
		ctx.Error(500, "GetWebhooksByOrgID", err)
		return
	}
	hooks := make([]*api.Hook, len(orgHooks))
	for i, hook := range orgHooks {
		hooks[i] = convert.ToHook(org.HomeLink(), hook)
	}
	ctx.JSON(200, hooks)
}

// GetHook get an organization's hook by id
func GetHook(ctx *context.APIContext) {
	org := ctx.Org.Organization
	hookID := ctx.ParamsInt64(":id")
	hook, err := utils.GetOrgHook(ctx, org.ID, hookID)
	if err != nil {
		return
	}
	ctx.JSON(200, convert.ToHook(org.HomeLink(), hook))
}

// CreateHook create a hook for an organization
func CreateHook(ctx *context.APIContext, form api.CreateHookOption) {
	if !utils.CheckCreateHookOption(ctx, &form) {
		return
	}
	utils.AddOrgHook(ctx, &form)
}

// EditHook modify a hook of a repository
func EditHook(ctx *context.APIContext, form api.EditHookOption) {
	hookID := ctx.ParamsInt64(":id")
	utils.EditOrgHook(ctx, &form, hookID)
}

// DeleteHook delete a hook of an organization
func DeleteHook(ctx *context.APIContext) {
	org := ctx.Org.Organization
	hookID := ctx.ParamsInt64(":id")
	if err := models.DeleteWebhookByOrgID(org.ID, hookID); err != nil {
		if models.IsErrWebhookNotExist(err) {
			ctx.Status(404)
		} else {
			ctx.Error(500, "DeleteWebhookByOrgID", err)
		}
		return
	}
	ctx.Status(204)
}
