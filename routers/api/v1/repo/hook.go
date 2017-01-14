// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repo

import (
	api "code.gitea.io/sdk/gitea"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/routers/api/v1/convert"
	"code.gitea.io/gitea/routers/api/v1/utils"
)

// ListHooks list all hooks of a repository
// see https://github.com/gogits/go-gogs-client/wiki/Repositories#list-hooks
func ListHooks(ctx *context.APIContext) {
	hooks, err := models.GetWebhooksByRepoID(ctx.Repo.Repository.ID)
	if err != nil {
		ctx.Error(500, "GetWebhooksByRepoID", err)
		return
	}

	apiHooks := make([]*api.Hook, len(hooks))
	for i := range hooks {
		apiHooks[i] = convert.ToHook(ctx.Repo.RepoLink, hooks[i])
	}
	ctx.JSON(200, &apiHooks)
}

// GetHook get a repo's hook by id
func GetHook(ctx *context.APIContext) {
	repo := ctx.Repo
	hookID := ctx.ParamsInt64(":id")
	hook, err := utils.GetRepoHook(ctx, repo.Repository.ID, hookID)
	if err != nil {
		return
	}
	ctx.JSON(200, convert.ToHook(repo.RepoLink, hook))
}

// CreateHook create a hook for a repository
// see https://github.com/gogits/go-gogs-client/wiki/Repositories#create-a-hook
func CreateHook(ctx *context.APIContext, form api.CreateHookOption) {
	if !utils.CheckCreateHookOption(ctx, &form) {
		return
	}
	utils.AddRepoHook(ctx, &form)
}

// EditHook modify a hook of a repository
// see https://github.com/gogits/go-gogs-client/wiki/Repositories#edit-a-hook
func EditHook(ctx *context.APIContext, form api.EditHookOption) {
	hookID := ctx.ParamsInt64(":id")
	utils.EditRepoHook(ctx, &form, hookID)
}

// DeleteHook delete a hook of a repository
func DeleteHook(ctx *context.APIContext) {
	if err := models.DeleteWebhookByRepoID(ctx.Repo.Repository.ID, ctx.ParamsInt64(":id")); err != nil {
		if models.IsErrWebhookNotExist(err) {
			ctx.Status(404)
		} else {
			ctx.Error(500, "DeleteWebhookByRepoID", err)
		}
		return
	}
	ctx.Status(204)
}
