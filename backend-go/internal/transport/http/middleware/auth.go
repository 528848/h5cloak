package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"cloak/internal/auth"
	"cloak/internal/domain/identity"
	"cloak/internal/rbac"
	"cloak/internal/transport/http/response"
)

// context 键。
const (
	CtxClaims = "auth.claims"
	CtxUser   = "auth.user"
	CtxPerms  = "auth.perms"
)

// Authenticator 解析 Bearer token，加载用户与权限并放入 context。
type Authenticator struct {
	tokens *auth.TokenManager
	users  identity.UserRepository
	rbac   *rbac.Service
}

func NewAuthenticator(tokens *auth.TokenManager, users identity.UserRepository, r *rbac.Service) *Authenticator {
	return &Authenticator{tokens: tokens, users: users, rbac: r}
}

// Required 必须登录。
func (a *Authenticator) Required() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearer(c.GetHeader("Authorization"))
		if raw == "" {
			response.Error(c, http.StatusUnauthorized, "未登录或登录已过期")
			return
		}
		claims, err := a.tokens.Parse(c.Request.Context(), raw)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "登录已过期，请重新登录")
			return
		}
		user, err := a.users.FindByID(c.Request.Context(), claims.UserID)
		if err != nil || user == nil || user.Status != identity.StatusEnabled {
			response.Error(c, http.StatusUnauthorized, "账号不可用，请重新登录")
			return
		}
		if user.Portal() != claims.Portal {
			response.Error(c, http.StatusForbidden, "账号门户不匹配")
			return
		}
		perms, err := a.rbac.Permissions(c.Request.Context(), user)
		if err != nil {
			response.Fail(c, "加载权限失败")
			return
		}
		c.Set(CtxClaims, claims)
		c.Set(CtxUser, user)
		c.Set(CtxPerms, perms)
		c.Next()
	}
}

// RequirePortal 路由组所属门户必须与 token 门户一致。
func RequirePortal(portal string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := Claims(c)
		if claims == nil || claims.Portal != portal {
			response.Error(c, http.StatusForbidden, "无权访问该门户接口")
			return
		}
		c.Next()
	}
}

// RequirePerm 校验权限标识。
func RequirePerm(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		perms, _ := c.Get(CtxPerms)
		list, _ := perms.([]string)
		if !rbac.HasPermission(list, perm) {
			response.Error(c, http.StatusForbidden, "没有操作权限")
			return
		}
		c.Next()
	}
}

// Claims 取当前 token 载荷。
func Claims(c *gin.Context) *auth.Claims {
	v, _ := c.Get(CtxClaims)
	cl, _ := v.(*auth.Claims)
	return cl
}

// User 取当前用户。
func User(c *gin.Context) *identity.User {
	v, _ := c.Get(CtxUser)
	u, _ := v.(*identity.User)
	return u
}

func bearer(h string) string {
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}
