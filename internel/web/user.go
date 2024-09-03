package web

import (
	regexp "github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	"github.com/lvow2022/udisk/internel/domain"
	"github.com/lvow2022/udisk/internel/service"
	ijwt "github.com/lvow2022/udisk/internel/web/jwt"
	"net/http"
)

const (
	emailRegexPattern = "^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$"
	// 和上面比起来，用 ` 看起来就比较清爽
	passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$`
	bizLogin             = "login"
)

type UserHandler struct {
	ijwt.Handler
	emailRegex     *regexp.Regexp
	passwordRexExp *regexp.Regexp
	usrSvc         service.UsrService
}

func NewUserHandler(usrSvc service.UsrService, jwtHdl ijwt.Handler) *UserHandler {
	return &UserHandler{
		emailRegex:     regexp.MustCompile(emailRegexPattern, regexp.None),
		passwordRexExp: regexp.MustCompile(passwordRegexPattern, regexp.None),
		usrSvc:         usrSvc,
		Handler:        jwtHdl,
	}
}

func (h *UserHandler) RegisterRoutes(server *gin.Engine) {
	ug := server.Group("/users")
	ug.POST("/signup", h.SignUp)
	ug.POST("/login", h.Login)
	ug.POST("/logout", h.Logout)
}

func (h *UserHandler) SignUp(ctx *gin.Context) {
	type ReqBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Confirm  string `json:"confirm"`
	}

	var reqBody ReqBody
	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		return
	}

	isEmail, err := h.emailRegex.MatchString(reqBody.Email)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	if !isEmail {
		ctx.String(http.StatusOK, "非法的邮箱格式")
		return
	}

	if reqBody.Password != reqBody.Confirm {
		ctx.String(http.StatusOK, "两次输入密码不对")
		return
	}

	isPassword, err := h.passwordRexExp.MatchString(reqBody.Password)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
	}

	if !isPassword {
		ctx.String(http.StatusOK, "密码必须包含字母")
	}

	err = h.usrSvc.Signup(ctx, domain.User{
		Email:    reqBody.Email,
		Password: reqBody.Password,
	})
	switch err {
	case nil:
		ctx.String(http.StatusOK, "注册成功")
	case service.ErrDuplicateEmail:
		ctx.String(http.StatusOK, "邮箱冲突，请换一个")
	default:
		ctx.String(http.StatusOK, "系统错误")
	}
}

func (h *UserHandler) Login(ctx *gin.Context) {
	u, err := h.usrSvc.Login(ctx, req.Email, req.Password)
	switch err {
	case nil:
		err = h.SetLoginToken(ctx, u.Id)
		if err != nil {

		}

	case service.ErrInvalidUserOrPassword:
		return
	default:
		return
	}
}

func (h *UserHandler) Logout(ctx *gin.Context) {
	err := h.ClearToken(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Msg: "退出登录成功"})
}
