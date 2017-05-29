// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package user

import (
	"fmt"
	"net/url"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/auth"
	"code.gitea.io/gitea/modules/auth/openid"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"github.com/go-macaron/captcha"
)

const (
	tplSignInOpenID base.TplName = "user/auth/signin_openid"
	tplConnectOID   base.TplName = "user/auth/signup_openid_connect"
	tplSignUpOID    base.TplName = "user/auth/signup_openid_register"
)

// SignInOpenID render sign in page
func SignInOpenID(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("sign_in")

	if ctx.Query("openid.return_to") != "" {
		signInOpenIDVerify(ctx)
		return
	}

	// Check auto-login.
	isSucceed, err := AutoSignIn(ctx)
	if err != nil {
		ctx.Handle(500, "AutoSignIn", err)
		return
	}

	redirectTo := ctx.Query("redirect_to")
	if len(redirectTo) > 0 {
		ctx.SetCookie("redirect_to", redirectTo, 0, setting.AppSubURL)
	} else {
		redirectTo, _ = url.QueryUnescape(ctx.GetCookie("redirect_to"))
	}

	if isSucceed {
		if len(redirectTo) > 0 {
			ctx.SetCookie("redirect_to", "", -1, setting.AppSubURL)
			ctx.Redirect(redirectTo)
		} else {
			ctx.Redirect(setting.AppSubURL + "/")
		}
		return
	}

	ctx.Data["PageIsSignIn"] = true
	ctx.Data["PageIsLoginOpenID"] = true
	ctx.HTML(200, tplSignInOpenID)
}

// Check if the given OpenID URI is allowed by blacklist/whitelist
func allowedOpenIDURI(uri string) (err error) {

	// In case a Whitelist is present, URI must be in it
	// in order to be accepted
	if len(setting.Service.OpenIDWhitelist) != 0 {
		for _, pat := range setting.Service.OpenIDWhitelist {
			if pat.MatchString(uri) {
				return nil // pass
			}
		}
		// must match one of this or be refused
		return fmt.Errorf("URI not allowed by whitelist")
	}

	// A blacklist match expliclty forbids
	for _, pat := range setting.Service.OpenIDBlacklist {
		if pat.MatchString(uri) {
			return fmt.Errorf("URI forbidden by blacklist")
		}
	}

	return nil
}

// SignInOpenIDPost response for openid sign in request
func SignInOpenIDPost(ctx *context.Context, form auth.SignInOpenIDForm) {
	ctx.Data["Title"] = ctx.Tr("sign_in")
	ctx.Data["PageIsSignIn"] = true
	ctx.Data["PageIsLoginOpenID"] = true

	if ctx.HasError() {
		ctx.HTML(200, tplSignInOpenID)
		return
	}

	id, err := openid.Normalize(form.Openid)
	if err != nil {
		ctx.RenderWithErr(err.Error(), tplSignInOpenID, &form)
		return
	}
	form.Openid = id

	log.Trace("OpenID uri: " + id)

	err = allowedOpenIDURI(id)
	if err != nil {
		ctx.RenderWithErr(err.Error(), tplSignInOpenID, &form)
		return
	}

	redirectTo := setting.AppURL + "user/login/openid"
	url, err := openid.RedirectURL(id, redirectTo, setting.AppURL)
	if err != nil {
		ctx.RenderWithErr(err.Error(), tplSignInOpenID, &form)
		return
	}

	// Request optional nickname and email info
	// NOTE: change to `openid.sreg.required` to require it
	url += "&openid.ns.sreg=http%3A%2F%2Fopenid.net%2Fextensions%2Fsreg%2F1.1"
	url += "&openid.sreg.optional=nickname%2Cemail"

	log.Trace("Form-passed openid-remember: %s", form.Remember)
	ctx.Session.Set("openid_signin_remember", form.Remember)

	ctx.Redirect(url)
}

// signInOpenIDVerify handles response from OpenID provider
func signInOpenIDVerify(ctx *context.Context) {

	log.Trace("Incoming call to: " + ctx.Req.Request.URL.String())

	fullURL := setting.AppURL + ctx.Req.Request.URL.String()[1:]
	log.Trace("Full URL: " + fullURL)

	var id, err = openid.Verify(fullURL)
	if err != nil {
		ctx.RenderWithErr(err.Error(), tplSignInOpenID, &auth.SignInOpenIDForm{
			Openid: id,
		})
		return
	}

	log.Trace("Verified ID: " + id)

	/* Now we should seek for the user and log him in, or prompt
	 * to register if not found */

	u, _ := models.GetUserByOpenID(id)
	if err != nil {
		if !models.IsErrUserNotExist(err) {
			ctx.RenderWithErr(err.Error(), tplSignInOpenID, &auth.SignInOpenIDForm{
				Openid: id,
			})
			return
		}
	}
	if u != nil {
		log.Trace("User exists, logging in")
		remember, _ := ctx.Session.Get("openid_signin_remember").(bool)
		log.Trace("Session stored openid-remember: %s", remember)
		handleSignIn(ctx, u, remember)
		return
	}

	log.Trace("User with openid " + id + " does not exist, should connect or register")

	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		ctx.RenderWithErr(err.Error(), tplSignInOpenID, &auth.SignInOpenIDForm{
			Openid: id,
		})
		return
	}
	values, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		ctx.RenderWithErr(err.Error(), tplSignInOpenID, &auth.SignInOpenIDForm{
			Openid: id,
		})
		return
	}
	email := values.Get("openid.sreg.email")
	nickname := values.Get("openid.sreg.nickname")

	log.Trace("User has email=" + email + " and nickname=" + nickname)

	if email != "" {
		u, _ = models.GetUserByEmail(email)
		if err != nil {
			if !models.IsErrUserNotExist(err) {
				ctx.RenderWithErr(err.Error(), tplSignInOpenID, &auth.SignInOpenIDForm{
					Openid: id,
				})
				return
			}
		}
		if u != nil {
			log.Trace("Local user " + u.LowerName + " has OpenID provided email " + email)
		}
	}

	if u == nil && nickname != "" {
		u, _ = models.GetUserByName(nickname)
		if err != nil {
			if !models.IsErrUserNotExist(err) {
				ctx.RenderWithErr(err.Error(), tplSignInOpenID, &auth.SignInOpenIDForm{
					Openid: id,
				})
				return
			}
		}
		if u != nil {
			log.Trace("Local user " + u.LowerName + " has OpenID provided nickname " + nickname)
		}
	}

	ctx.Session.Set("openid_verified_uri", id)

	ctx.Session.Set("openid_determined_email", email)

	if u != nil {
		nickname = u.LowerName
	}

	ctx.Session.Set("openid_determined_username", nickname)

	if u != nil || !setting.Service.EnableOpenIDSignUp {
		ctx.Redirect(setting.AppSubURL + "/user/openid/connect")
	} else {
		ctx.Redirect(setting.AppSubURL + "/user/openid/register")
	}
}

// ConnectOpenID shows a form to connect an OpenID URI to an existing account
func ConnectOpenID(ctx *context.Context) {
	oid, _ := ctx.Session.Get("openid_verified_uri").(string)
	if oid == "" {
		ctx.Redirect(setting.AppSubURL + "/user/login/openid")
		return
	}
	ctx.Data["Title"] = "OpenID connect"
	ctx.Data["PageIsSignIn"] = true
	ctx.Data["PageIsOpenIDConnect"] = true
	ctx.Data["EnableOpenIDSignUp"] = setting.Service.EnableOpenIDSignUp
	ctx.Data["OpenID"] = oid
	userName, _ := ctx.Session.Get("openid_determined_username").(string)
	if userName != "" {
		ctx.Data["user_name"] = userName
	}
	ctx.HTML(200, tplConnectOID)
}

// ConnectOpenIDPost handles submission of a form to connect an OpenID URI to an existing account
func ConnectOpenIDPost(ctx *context.Context, form auth.ConnectOpenIDForm) {
	oid, _ := ctx.Session.Get("openid_verified_uri").(string)
	if oid == "" {
		ctx.Redirect(setting.AppSubURL + "/user/login/openid")
		return
	}
	ctx.Data["Title"] = "OpenID connect"
	ctx.Data["PageIsSignIn"] = true
	ctx.Data["PageIsOpenIDConnect"] = true
	ctx.Data["EnableOpenIDSignUp"] = setting.Service.EnableOpenIDSignUp
	ctx.Data["OpenID"] = oid

	u, err := models.UserSignIn(form.UserName, form.Password)
	if err != nil {
		if models.IsErrUserNotExist(err) {
			ctx.RenderWithErr(ctx.Tr("form.username_password_incorrect"), tplConnectOID, &form)
		} else {
			ctx.Handle(500, "ConnectOpenIDPost", err)
		}
		return
	}

	// add OpenID for the user
	userOID := &models.UserOpenID{UID: u.ID, URI: oid}
	if err = models.AddUserOpenID(userOID); err != nil {
		if models.IsErrOpenIDAlreadyUsed(err) {
			ctx.RenderWithErr(ctx.Tr("form.openid_been_used", oid), tplConnectOID, &form)
			return
		}
		ctx.Handle(500, "AddUserOpenID", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.add_openid_success"))

	remember, _ := ctx.Session.Get("openid_signin_remember").(bool)
	log.Trace("Session stored openid-remember: %s", remember)
	handleSignIn(ctx, u, remember)
}

// RegisterOpenID shows a form to create a new user authenticated via an OpenID URI
func RegisterOpenID(ctx *context.Context) {
	if !setting.Service.EnableOpenIDSignUp {
		ctx.Error(403)
		return
	}
	oid, _ := ctx.Session.Get("openid_verified_uri").(string)
	if oid == "" {
		ctx.Redirect(setting.AppSubURL + "/user/login/openid")
		return
	}
	ctx.Data["Title"] = "OpenID signup"
	ctx.Data["PageIsSignIn"] = true
	ctx.Data["PageIsOpenIDRegister"] = true
	ctx.Data["EnableOpenIDSignUp"] = setting.Service.EnableOpenIDSignUp
	ctx.Data["EnableCaptcha"] = setting.Service.EnableCaptcha
	ctx.Data["OpenID"] = oid
	userName, _ := ctx.Session.Get("openid_determined_username").(string)
	if userName != "" {
		ctx.Data["user_name"] = userName
	}
	email, _ := ctx.Session.Get("openid_determined_email").(string)
	if email != "" {
		ctx.Data["email"] = email
	}
	ctx.HTML(200, tplSignUpOID)
}

// RegisterOpenIDPost handles submission of a form to create a new user authenticated via an OpenID URI
func RegisterOpenIDPost(ctx *context.Context, cpt *captcha.Captcha, form auth.SignUpOpenIDForm) {
	if !setting.Service.EnableOpenIDSignUp {
		ctx.Error(403)
		return
	}
	oid, _ := ctx.Session.Get("openid_verified_uri").(string)
	if oid == "" {
		ctx.Redirect(setting.AppSubURL + "/user/login/openid")
		return
	}

	ctx.Data["Title"] = "OpenID signup"
	ctx.Data["PageIsSignIn"] = true
	ctx.Data["PageIsOpenIDRegister"] = true
	ctx.Data["EnableOpenIDSignUp"] = setting.Service.EnableOpenIDSignUp
	ctx.Data["EnableCaptcha"] = setting.Service.EnableCaptcha
	ctx.Data["OpenID"] = oid

	if setting.Service.EnableCaptcha && !cpt.VerifyReq(ctx.Req) {
		ctx.Data["Err_Captcha"] = true
		ctx.RenderWithErr(ctx.Tr("form.captcha_incorrect"), tplSignUpOID, &form)
		return
	}

	len := setting.MinPasswordLength
	if len < 256 {
		len = 256
	}
	password, err := base.GetRandomString(len)
	if err != nil {
		ctx.RenderWithErr(err.Error(), tplSignUpOID, form)
		return
	}

	// TODO: abstract a finalizeSignUp function ?
	u := &models.User{
		Name:     form.UserName,
		Email:    form.Email,
		Passwd:   password,
		IsActive: !setting.Service.RegisterEmailConfirm,
	}
	if err := models.CreateUser(u); err != nil {
		switch {
		case models.IsErrUserAlreadyExist(err):
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("form.username_been_taken"), tplSignUpOID, &form)
		case models.IsErrEmailAlreadyUsed(err):
			ctx.Data["Err_Email"] = true
			ctx.RenderWithErr(ctx.Tr("form.email_been_used"), tplSignUpOID, &form)
		case models.IsErrNameReserved(err):
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("user.form.name_reserved", err.(models.ErrNameReserved).Name), tplSignUpOID, &form)
		case models.IsErrNamePatternNotAllowed(err):
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("user.form.name_pattern_not_allowed", err.(models.ErrNamePatternNotAllowed).Pattern), tplSignUpOID, &form)
		default:
			ctx.Handle(500, "CreateUser", err)
		}
		return
	}
	log.Trace("Account created: %s", u.Name)

	// add OpenID for the user
	userOID := &models.UserOpenID{UID: u.ID, URI: oid}
	if err = models.AddUserOpenID(userOID); err != nil {
		if models.IsErrOpenIDAlreadyUsed(err) {
			ctx.RenderWithErr(ctx.Tr("form.openid_been_used", oid), tplSignUpOID, &form)
			return
		}
		ctx.Handle(500, "AddUserOpenID", err)
		return
	}

	// Auto-set admin for the only user.
	if models.CountUsers() == 1 {
		u.IsAdmin = true
		u.IsActive = true
		if err := models.UpdateUser(u); err != nil {
			ctx.Handle(500, "UpdateUser", err)
			return
		}
	}

	// Send confirmation email, no need for social account.
	if setting.Service.RegisterEmailConfirm && u.ID > 1 {
		models.SendActivateAccountMail(ctx.Context, u)
		ctx.Data["IsSendRegisterMail"] = true
		ctx.Data["Email"] = u.Email
		ctx.Data["ActiveCodeLives"] = base.MinutesToFriendly(setting.Service.ActiveCodeLives)
		ctx.HTML(200, TplActivate)

		if err := ctx.Cache.Put("MailResendLimit_"+u.LowerName, u.LowerName, 180); err != nil {
			log.Error(4, "Set cache(MailResendLimit) fail: %v", err)
		}
		return
	}

	remember, _ := ctx.Session.Get("openid_signin_remember").(bool)
	log.Trace("Session stored openid-remember: %s", remember)
	handleSignIn(ctx, u, remember)
}
