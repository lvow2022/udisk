package web

import (
	regexp "github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	"github.com/lvow2022/udisk/internel/domain"
	"github.com/lvow2022/udisk/internel/service"
	"net/http"
)

const (
	emailRegexPattern = "^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$"
	// 和上面比起来，用 ` 看起来就比较清爽
	passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$`
	bizLogin             = "login"
)

type UserHandler struct {
	emailRegex     *regexp.Regexp
	passwordRexExp *regexp.Regexp
	userSvc        service.UserService
}

func NewUserHandler(userSvc service.UserService) *UserHandler {
	return &UserHandler{
		emailRegex:     regexp.MustCompile(emailRegexPattern, regexp.None),
		passwordRexExp: regexp.MustCompile(passwordRegexPattern, regexp.None),
		userSvc:        userSvc,
	}
}

func (h *UserHandler) RegisterRoutes(server *gin.Engine) {
	ug := server.Group("/users")
	ug.POST("/signup", h.SignUp)
	ug.POST("/signin", h.SignIn)
	ug.POST("/signout", h.SignOut)
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

	err = h.userSvc.Signup(ctx, domain.User{
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

func (h *UserHandler) SignIn(context *gin.Context) {

}

func (h *UserHandler) SignOut(context *gin.Context) {

}
