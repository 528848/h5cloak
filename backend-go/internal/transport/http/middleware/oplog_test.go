package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDeriveModule(t *testing.T) {
	cases := []struct {
		method, path       string
		module, action, id string
	}{
		{"POST", "/api/merchant/site", "merchant.site", "post", ""},
		{"PUT", "/api/merchant/site/123", "merchant.site", "put", "123"},
		{"PUT", "/api/merchant/site/changeStatus", "merchant.site", "changeStatus", ""},
		{"DELETE", "/api/admin/merchant/9", "admin.merchant", "delete", "9"},
		{"PUT", "/api/system/user/profile/updatePwd", "system.user.profile", "updatePwd", ""},
	}
	for _, c := range cases {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(c.method, c.path, nil)
		m, a, id := deriveModule(ctx)
		if m != c.module || a != c.action || id != c.id {
			t.Errorf("%s %s => %s %s %s, want %s %s %s", c.method, c.path, m, a, id, c.module, c.action, c.id)
		}
	}
}

func TestSummarizeMasksSensitive(t *testing.T) {
	out := summarize([]byte(`{"username":"a","password":"p","newPassword":"q","apiToken":"t"}`))
	var m map[string]any
	_ = json.Unmarshal(out, &m)
	if m["password"] != "***" || m["newPassword"] != "***" || m["apiToken"] != "***" || m["username"] != "a" {
		t.Fatalf("unexpected %s", out)
	}
}
