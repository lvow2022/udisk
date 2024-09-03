package jwt

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"strings"
	"time"
)

var _ Handler = &LocalJWTHandler{}

type LocalJWTHandler struct {
	cache         *cache.Cache
	signingMethod jwt.SigningMethod
	rcExpiration  time.Duration
}

func NewLocalJWTHandler() Handler {
	// 创建一个新的go-cache实例，默认清理间隔为10分钟
	c := cache.New(5*time.Minute, 10*time.Minute)
	return &LocalJWTHandler{
		cache:         c,
		signingMethod: jwt.SigningMethodHS512,
		rcExpiration:  time.Hour * 24 * 7,
	}
}

func (h *LocalJWTHandler) CheckSession(ctx *gin.Context, ssid string) error {
	// 检查ssid是否存在于cache中
	_, found := h.cache.Get(fmt.Sprintf("users:ssid:%s", ssid))
	if !found {
		return errors.New("token 无效")
	}
	return nil
}

func (h *LocalJWTHandler) ExtractToken(ctx *gin.Context) string {
	authCode := ctx.GetHeader("Authorization")
	if authCode == "" {
		return authCode
	}
	segs := strings.Split(authCode, " ")
	if len(segs) != 2 {
		return ""
	}
	return segs[1]
}

func (h *LocalJWTHandler) SetLoginToken(ctx *gin.Context, uid int64) error {
	ssid := uuid.New().String()

	// 将ssid存储到cache中，设置到期时间
	h.cache.Set(fmt.Sprintf("users:ssid:%s", ssid), struct{}{}, h.rcExpiration)

	err := h.setRefreshToken(ctx, uid, ssid)
	if err != nil {
		return err
	}
	return h.SetJWTToken(ctx, uid, ssid)
}

func (h *LocalJWTHandler) ClearToken(ctx *gin.Context) error {
	ctx.Header("x-jwt-token", "")
	ctx.Header("x-refresh-token", "")
	uc := ctx.MustGet("user").(UserClaims)

	h.cache.Delete(fmt.Sprintf("users:ssid:%s", uc.Ssid))

	return nil
}

func (h *LocalJWTHandler) SetJWTToken(ctx *gin.Context, uid int64, ssid string) error {
	uc := UserClaims{
		Uid:       uid,
		Ssid:      ssid,
		UserAgent: ctx.GetHeader("User-Agent"),
		RegisteredClaims: jwt.RegisteredClaims{
			// 30 分钟过期
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
		},
	}
	token := jwt.NewWithClaims(h.signingMethod, uc)
	tokenStr, err := token.SignedString(JWTKey)
	if err != nil {
		return err
	}
	ctx.Header("x-jwt-token", tokenStr)
	return nil
}

func (h *LocalJWTHandler) setRefreshToken(ctx *gin.Context, uid int64, ssid string) error {
	rc := RefreshClaims{
		Uid:  uid,
		Ssid: ssid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.rcExpiration)),
		},
	}
	token := jwt.NewWithClaims(h.signingMethod, rc)
	tokenStr, err := token.SignedString(RCJWTKey)
	if err != nil {
		return err
	}
	ctx.Header("x-refresh-token", tokenStr)
	return nil
}

var JWTKey = []byte("k6CswdUm77WKcbM68UQUuxVsHSpTCwgK")
var RCJWTKey = []byte("k6CswdUm77WKcbM68UQUuxVsHSpTCwgA")

type RefreshClaims struct {
	jwt.RegisteredClaims
	Uid  int64
	Ssid string
}

type UserClaims struct {
	jwt.RegisteredClaims
	Uid       int64
	Ssid      string
	UserAgent string
}
