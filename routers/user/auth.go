// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package user

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/go-macaron/captcha"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/auth"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

const (
	// tplSignIn template for sign in page
	tplSignIn base.TplName = "user/auth/signin"
	// tplSignUp template path for sign up page
	tplSignUp base.TplName = "user/auth/signup"
	// TplActivate template path for activate user
	TplActivate       base.TplName = "user/auth/activate"
	tplForgotPassword base.TplName = "user/auth/forgot_passwd"
	tplResetPassword  base.TplName = "user/auth/reset_passwd"
	tplTwofa          base.TplName = "user/auth/twofa"
	tplTwofaScratch   base.TplName = "user/auth/twofa_scratch"
)

// AutoSignIn reads cookie and try to auto-login.
func AutoSignIn(ctx *context.Context) (bool, error) {
	if !models.HasEngine {
		return false, nil
	}

	uname := ctx.GetCookie(setting.CookieUserName)
	if len(uname) == 0 {
		return false, nil
	}

	isSucceed := false
	defer func() {
		if !isSucceed {
			log.Trace("auto-login cookie cleared: %s", uname)
			ctx.SetCookie(setting.CookieUserName, "", -1, setting.AppSubURL)
			ctx.SetCookie(setting.CookieRememberName, "", -1, setting.AppSubURL)
		}
	}()

	u, err := models.GetUserByName(uname)
	if err != nil {
		if !models.IsErrUserNotExist(err) {
			return false, fmt.Errorf("GetUserByName: %v", err)
		}
		return false, nil
	}

	if val, _ := ctx.GetSuperSecureCookie(
		base.EncodeMD5(u.Rands+u.Passwd), setting.CookieRememberName); val != u.Name {
		return false, nil
	}

	isSucceed = true
	ctx.Session.Set("uid", u.ID)
	ctx.Session.Set("uname", u.Name)
	ctx.SetCookie(setting.CSRFCookieName, "", -1, setting.AppSubURL)
	return true, nil
}

func checkAutoLogin(ctx *context.Context) bool {
	// Check auto-login.
	isSucceed, err := AutoSignIn(ctx)
	if err != nil {
		ctx.Handle(500, "AutoSignIn", err)
		return true
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
		return true
	}

	return false
}

// SignIn render sign in page
func SignIn(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("sign_in")

	// Check auto-login.
	if checkAutoLogin(ctx) {
		return
	}

	ctx.HTML(200, tplSignIn)
}

// SignInPost response for sign in request
func SignInPost(ctx *context.Context, form auth.SignInForm) {
	ctx.Data["Title"] = ctx.Tr("sign_in")

	if ctx.HasError() {
		ctx.HTML(200, tplSignIn)
		return
	}

	u, err := models.UserSignIn(form.UserName, form.Password)
	if err != nil {
		if models.IsErrUserNotExist(err) {
			ctx.RenderWithErr(ctx.Tr("form.username_password_incorrect"), tplSignIn, &form)
		} else {
			ctx.Handle(500, "UserSignIn", err)
		}
		return
	}

	// If this user is enrolled in 2FA, we can't sign the user in just yet.
	// Instead, redirect them to the 2FA authentication page.
	_, err = models.GetTwoFactorByUID(u.ID)
	if err != nil {
		if models.IsErrTwoFactorNotEnrolled(err) {
			handleSignIn(ctx, u, form.Remember)
		} else {
			ctx.Handle(500, "UserSignIn", err)
		}
		return
	}

	// User needs to use 2FA, save data and redirect to 2FA page.
	ctx.Session.Set("twofaUid", u.ID)
	ctx.Session.Set("twofaRemember", form.Remember)
	ctx.Redirect(setting.AppSubURL + "/user/two_factor")
}

// TwoFactor shows the user a two-factor authentication page.
func TwoFactor(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("twofa")

	// Check auto-login.
	if checkAutoLogin(ctx) {
		return
	}

	// Ensure user is in a 2FA session.
	if ctx.Session.Get("twofaUid") == nil {
		ctx.Handle(500, "UserSignIn", errors.New("not in 2FA session"))
		return
	}

	ctx.HTML(200, tplTwofa)
}

// TwoFactorPost validates a user's two-factor authentication token.
func TwoFactorPost(ctx *context.Context, form auth.TwoFactorAuthForm) {
	ctx.Data["Title"] = ctx.Tr("twofa")

	// Ensure user is in a 2FA session.
	idSess := ctx.Session.Get("twofaUid")
	if idSess == nil {
		ctx.Handle(500, "UserSignIn", errors.New("not in 2FA session"))
		return
	}

	id := idSess.(int64)
	twofa, err := models.GetTwoFactorByUID(id)
	if err != nil {
		ctx.Handle(500, "UserSignIn", err)
		return
	}

	// Validate the passcode with the stored TOTP secret.
	ok, err := twofa.ValidateTOTP(form.Passcode)
	if err != nil {
		ctx.Handle(500, "UserSignIn", err)
		return
	}

	if ok {
		remember := ctx.Session.Get("twofaRemember").(bool)
		u, err := models.GetUserByID(id)
		if err != nil {
			ctx.Handle(500, "UserSignIn", err)
			return
		}

		handleSignIn(ctx, u, remember)
		return
	}

	ctx.RenderWithErr(ctx.Tr("auth.twofa_passcode_incorrect"), tplTwofa, auth.TwoFactorAuthForm{})
}

// TwoFactorScratch shows the scratch code form for two-factor authentication.
func TwoFactorScratch(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("twofa_scratch")

	// Check auto-login.
	if checkAutoLogin(ctx) {
		return
	}

	// Ensure user is in a 2FA session.
	if ctx.Session.Get("twofaUid") == nil {
		ctx.Handle(500, "UserSignIn", errors.New("not in 2FA session"))
		return
	}

	ctx.HTML(200, tplTwofaScratch)
}

// TwoFactorScratchPost validates and invalidates a user's two-factor scratch token.
func TwoFactorScratchPost(ctx *context.Context, form auth.TwoFactorScratchAuthForm) {
	ctx.Data["Title"] = ctx.Tr("twofa_scratch")

	// Ensure user is in a 2FA session.
	idSess := ctx.Session.Get("twofaUid")
	if idSess == nil {
		ctx.Handle(500, "UserSignIn", errors.New("not in 2FA session"))
		return
	}

	id := idSess.(int64)
	twofa, err := models.GetTwoFactorByUID(id)
	if err != nil {
		ctx.Handle(500, "UserSignIn", err)
		return
	}

	// Validate the passcode with the stored TOTP secret.
	if twofa.VerifyScratchToken(form.Token) {
		// Invalidate the scratch token.
		twofa.ScratchToken = ""
		if err = models.UpdateTwoFactor(twofa); err != nil {
			ctx.Handle(500, "UserSignIn", err)
			return
		}

		remember := ctx.Session.Get("twofaRemember").(bool)
		u, err := models.GetUserByID(id)
		if err != nil {
			ctx.Handle(500, "UserSignIn", err)
			return
		}

		handleSignInFull(ctx, u, remember, false)
		ctx.Flash.Info(ctx.Tr("auth.twofa_scratch_used"))
		ctx.Redirect(setting.AppSubURL + "/user/settings/two_factor")
		return
	}

	ctx.RenderWithErr(ctx.Tr("auth.twofa_scratch_token_incorrect"), tplTwofaScratch, auth.TwoFactorScratchAuthForm{})
}

// This handles the final part of the sign-in process of the user.
func handleSignIn(ctx *context.Context, u *models.User, remember bool) {
	handleSignInFull(ctx, u, remember, true)
}

func handleSignInFull(ctx *context.Context, u *models.User, remember bool, obeyRedirect bool) {
	if remember {
		days := 86400 * setting.LogInRememberDays
		ctx.SetCookie(setting.CookieUserName, u.Name, days, setting.AppSubURL)
		ctx.SetSuperSecureCookie(base.EncodeMD5(u.Rands+u.Passwd),
			setting.CookieRememberName, u.Name, days, setting.AppSubURL)
	}

	ctx.Session.Delete("twofaUid")
	ctx.Session.Delete("twofaRemember")
	ctx.Session.Set("uid", u.ID)
	ctx.Session.Set("uname", u.Name)

	// Clear whatever CSRF has right now, force to generate a new one
	ctx.SetCookie(setting.CSRFCookieName, "", -1, setting.AppSubURL)

	// Register last login
	u.SetLastLogin()
	if err := models.UpdateUser(u); err != nil {
		ctx.Handle(500, "UpdateUser", err)
		return
	}

	if redirectTo, _ := url.QueryUnescape(ctx.GetCookie("redirect_to")); len(redirectTo) > 0 {
		ctx.SetCookie("redirect_to", "", -1, setting.AppSubURL)
		if obeyRedirect {
			ctx.Redirect(redirectTo)
		}
		return
	}

	if obeyRedirect {
		ctx.Redirect(setting.AppSubURL + "/")
	}
}

// SignOut sign out from login status
func SignOut(ctx *context.Context) {
	ctx.Session.Delete("uid")
	ctx.Session.Delete("uname")
	ctx.Session.Delete("socialId")
	ctx.Session.Delete("socialName")
	ctx.Session.Delete("socialEmail")
	ctx.SetCookie(setting.CookieUserName, "", -1, setting.AppSubURL)
	ctx.SetCookie(setting.CookieRememberName, "", -1, setting.AppSubURL)
	ctx.SetCookie(setting.CSRFCookieName, "", -1, setting.AppSubURL)
	ctx.Redirect(setting.AppSubURL + "/")
}

// SignUp render the register page
func SignUp(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("sign_up")

	ctx.Data["EnableCaptcha"] = setting.Service.EnableCaptcha

	if setting.Service.DisableRegistration {
		ctx.Data["DisableRegistration"] = true
		ctx.HTML(200, tplSignUp)
		return
	}

	ctx.HTML(200, tplSignUp)
}

// SignUpPost response for sign up information submission
func SignUpPost(ctx *context.Context, cpt *captcha.Captcha, form auth.RegisterForm) {
	ctx.Data["Title"] = ctx.Tr("sign_up")

	ctx.Data["EnableCaptcha"] = setting.Service.EnableCaptcha

	if setting.Service.DisableRegistration {
		ctx.Error(403)
		return
	}

	if ctx.HasError() {
		ctx.HTML(200, tplSignUp)
		return
	}

	if setting.Service.EnableCaptcha && !cpt.VerifyReq(ctx.Req) {
		ctx.Data["Err_Captcha"] = true
		ctx.RenderWithErr(ctx.Tr("form.captcha_incorrect"), tplSignUp, &form)
		return
	}

	if form.Password != form.Retype {
		ctx.Data["Err_Password"] = true
		ctx.RenderWithErr(ctx.Tr("form.password_not_match"), tplSignUp, &form)
		return
	}
	if len(form.Password) < setting.MinPasswordLength {
		ctx.Data["Err_Password"] = true
		ctx.RenderWithErr(ctx.Tr("auth.password_too_short", setting.MinPasswordLength), tplSignUp, &form)
		return
	}

	u := &models.User{
		Name:     form.UserName,
		Email:    form.Email,
		Passwd:   form.Password,
		IsActive: !setting.Service.RegisterEmailConfirm,
	}
	if err := models.CreateUser(u); err != nil {
		switch {
		case models.IsErrUserAlreadyExist(err):
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("form.username_been_taken"), tplSignUp, &form)
		case models.IsErrEmailAlreadyUsed(err):
			ctx.Data["Err_Email"] = true
			ctx.RenderWithErr(ctx.Tr("form.email_been_used"), tplSignUp, &form)
		case models.IsErrNameReserved(err):
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("user.form.name_reserved", err.(models.ErrNameReserved).Name), tplSignUp, &form)
		case models.IsErrNamePatternNotAllowed(err):
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("user.form.name_pattern_not_allowed", err.(models.ErrNamePatternNotAllowed).Pattern), tplSignUp, &form)
		default:
			ctx.Handle(500, "CreateUser", err)
		}
		return
	}
	log.Trace("Account created: %s", u.Name)

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
		ctx.Data["Hours"] = setting.Service.ActiveCodeLives / 60
		ctx.HTML(200, TplActivate)

		if err := ctx.Cache.Put("MailResendLimit_"+u.LowerName, u.LowerName, 180); err != nil {
			log.Error(4, "Set cache(MailResendLimit) fail: %v", err)
		}
		return
	}

	ctx.Redirect(setting.AppSubURL + "/user/login")
}

// Activate render activate user page
func Activate(ctx *context.Context) {
	code := ctx.Query("code")
	if len(code) == 0 {
		ctx.Data["IsActivatePage"] = true
		if ctx.User.IsActive {
			ctx.Error(404)
			return
		}
		// Resend confirmation email.
		if setting.Service.RegisterEmailConfirm {
			if ctx.Cache.IsExist("MailResendLimit_" + ctx.User.LowerName) {
				ctx.Data["ResendLimited"] = true
			} else {
				ctx.Data["Hours"] = setting.Service.ActiveCodeLives / 60
				models.SendActivateAccountMail(ctx.Context, ctx.User)

				if err := ctx.Cache.Put("MailResendLimit_"+ctx.User.LowerName, ctx.User.LowerName, 180); err != nil {
					log.Error(4, "Set cache(MailResendLimit) fail: %v", err)
				}
			}
		} else {
			ctx.Data["ServiceNotEnabled"] = true
		}
		ctx.HTML(200, TplActivate)
		return
	}

	// Verify code.
	if user := models.VerifyUserActiveCode(code); user != nil {
		user.IsActive = true
		var err error
		if user.Rands, err = models.GetUserSalt(); err != nil {
			ctx.Handle(500, "UpdateUser", err)
			return
		}
		if err := models.UpdateUser(user); err != nil {
			if models.IsErrUserNotExist(err) {
				ctx.Error(404)
			} else {
				ctx.Handle(500, "UpdateUser", err)
			}
			return
		}

		log.Trace("User activated: %s", user.Name)

		ctx.Session.Set("uid", user.ID)
		ctx.Session.Set("uname", user.Name)
		ctx.Redirect(setting.AppSubURL + "/")
		return
	}

	ctx.Data["IsActivateFailed"] = true
	ctx.HTML(200, TplActivate)
}

// ActivateEmail render the activate email page
func ActivateEmail(ctx *context.Context) {
	code := ctx.Query("code")
	emailStr := ctx.Query("email")

	// Verify code.
	if email := models.VerifyActiveEmailCode(code, emailStr); email != nil {
		if err := email.Activate(); err != nil {
			ctx.Handle(500, "ActivateEmail", err)
		}

		log.Trace("Email activated: %s", email.Email)
		ctx.Flash.Success(ctx.Tr("settings.add_email_success"))
	}

	ctx.Redirect(setting.AppSubURL + "/user/settings/email")
	return
}

// ForgotPasswd render the forget pasword page
func ForgotPasswd(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("auth.forgot_password")

	if setting.MailService == nil {
		ctx.Data["IsResetDisable"] = true
		ctx.HTML(200, tplForgotPassword)
		return
	}

	ctx.Data["IsResetRequest"] = true
	ctx.HTML(200, tplForgotPassword)
}

// ForgotPasswdPost response for forget password request
func ForgotPasswdPost(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("auth.forgot_password")

	if setting.MailService == nil {
		ctx.Handle(403, "ForgotPasswdPost", nil)
		return
	}
	ctx.Data["IsResetRequest"] = true

	email := ctx.Query("email")
	ctx.Data["Email"] = email

	u, err := models.GetUserByEmail(email)
	if err != nil {
		if models.IsErrUserNotExist(err) {
			ctx.Data["Hours"] = setting.Service.ActiveCodeLives / 60
			ctx.Data["IsResetSent"] = true
			ctx.HTML(200, tplForgotPassword)
			return
		}

		ctx.Handle(500, "user.ResetPasswd(check existence)", err)
		return
	}

	if !u.IsLocal() {
		ctx.Data["Err_Email"] = true
		ctx.RenderWithErr(ctx.Tr("auth.non_local_account"), tplForgotPassword, nil)
		return
	}

	if ctx.Cache.IsExist("MailResendLimit_" + u.LowerName) {
		ctx.Data["ResendLimited"] = true
		ctx.HTML(200, tplForgotPassword)
		return
	}

	models.SendResetPasswordMail(ctx.Context, u)
	if err = ctx.Cache.Put("MailResendLimit_"+u.LowerName, u.LowerName, 180); err != nil {
		log.Error(4, "Set cache(MailResendLimit) fail: %v", err)
	}

	ctx.Data["Hours"] = setting.Service.ActiveCodeLives / 60
	ctx.Data["IsResetSent"] = true
	ctx.HTML(200, tplForgotPassword)
}

// ResetPasswd render the reset password page
func ResetPasswd(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("auth.reset_password")

	code := ctx.Query("code")
	if len(code) == 0 {
		ctx.Error(404)
		return
	}
	ctx.Data["Code"] = code
	ctx.Data["IsResetForm"] = true
	ctx.HTML(200, tplResetPassword)
}

// ResetPasswdPost response from reset password request
func ResetPasswdPost(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("auth.reset_password")

	code := ctx.Query("code")
	if len(code) == 0 {
		ctx.Error(404)
		return
	}
	ctx.Data["Code"] = code

	if u := models.VerifyUserActiveCode(code); u != nil {
		// Validate password length.
		passwd := ctx.Query("password")
		if len(passwd) < setting.MinPasswordLength {
			ctx.Data["IsResetForm"] = true
			ctx.Data["Err_Password"] = true
			ctx.RenderWithErr(ctx.Tr("auth.password_too_short", setting.MinPasswordLength), tplResetPassword, nil)
			return
		}

		u.Passwd = passwd
		var err error
		if u.Rands, err = models.GetUserSalt(); err != nil {
			ctx.Handle(500, "UpdateUser", err)
			return
		}
		if u.Salt, err = models.GetUserSalt(); err != nil {
			ctx.Handle(500, "UpdateUser", err)
			return
		}
		u.EncodePasswd()
		if err := models.UpdateUser(u); err != nil {
			ctx.Handle(500, "UpdateUser", err)
			return
		}

		log.Trace("User password reset: %s", u.Name)
		ctx.Redirect(setting.AppSubURL + "/user/login")
		return
	}

	ctx.Data["IsResetFailed"] = true
	ctx.HTML(200, tplResetPassword)
}
