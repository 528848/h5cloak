package admin

import (
	"github.com/gin-gonic/gin"

	"cloak/internal/transport/http/response"
)

type dictItem struct {
	DictLabel string `json:"dictLabel"`
	DictValue string `json:"dictValue"`
	ListClass string `json:"listClass"`
	Status    string `json:"status"`
}

// 内置字典。业务状态由前端 constants.js 维护，这里只保留若依基础组件依赖的字典。
var dicts = map[string][]dictItem{
	"sys_user_sex": {
		{"男", "0", "default", "0"}, {"女", "1", "default", "0"}, {"未知", "2", "default", "0"},
	},
	"sys_normal_disable": {
		{"正常", "0", "primary", "0"}, {"停用", "1", "danger", "0"},
	},
	"sys_yes_no": {
		{"是", "Y", "primary", "0"}, {"否", "N", "danger", "0"},
	},
}

// GET /system/dict/data/type/:type
func dictByType(c *gin.Context) {
	items := dicts[c.Param("type")]
	if items == nil {
		items = []dictItem{}
	}
	response.Success(c, items, "查询成功")
}
