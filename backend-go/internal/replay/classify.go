package replay

import "strings"

// 差异分类。前五类是「可解释的差异」，unexplained 需要人工看。
const (
	ClassMatch         = "match"               // 判定一致
	ClassNeedsToken    = "needs_token"         // 原请求依赖访客令牌或交接令牌（回访/交接），回放没有 Cookie
	ClassGlobal        = "global_override"     // 全局开关改写了结果
	ClassQuota         = "quota_override"      // 商户额度策略改写了结果
	ClassSiteNotServe  = "site_not_serving"    // 站点被停用或商户未启用
	ClassReconstructed = "param_reconstructed" // 参数值是导出时合成的，归因校验结果不可比
	ClassHostUnbound   = "host_unbound"        // 网关没有该域名（服务端接入或域名已解绑）
	ClassTransport     = "transport_error"     // 请求失败或响应不是判定 JSON
	ClassUnexplained   = "unexplained"         // 以上都不是：需要排查
)

// Explained 该类差异是否已有解释。
func Explained(class string) bool {
	switch class {
	case ClassMatch, ClassNeedsToken, ClassGlobal, ClassQuota, ClassSiteNotServe, ClassReconstructed, ClassHostUnbound:
		return true
	}
	return false
}

// Classify 对比期望与观察到的判定，给出类别与一句话原因。
func Classify(c Case, o Observed) (string, string) {
	exp := c.Expected
	sameContent := exp.Content == o.Content
	sameTouch := exp.TouchType == "" || exp.TouchType == o.TouchType
	sameAttr := exp.Attributed == o.Attributed && (!exp.Attributed || sameChannel(exp, o))
	if sameContent && sameTouch && sameAttr {
		return ClassMatch, ""
	}
	if step := findStep(o.Trace, "global"); step != "" {
		return ClassGlobal, "全局开关：" + step
	}
	if step := findStep(o.Trace, "quota"); step != "" {
		return ClassQuota, "额度策略：" + step
	}
	if step := findStep(o.Trace, "site"); strings.HasPrefix(step, "not_serving") {
		return ClassSiteNotServe, "站点未提供服务：" + step
	}
	// 路径规则与原请求不同：站点配置变了，必须人工确认，优先于历史改写的解释
	if o.PathRule != "" && o.PathRule != exp.PathRule {
		return ClassUnexplained, "路径规则变化：" + nonEmpty(exp.PathRule, "无") + " → " + o.PathRule
	}
	for _, ov := range c.Overrides {
		switch {
		case strings.HasPrefix(ov, "global:"):
			return ClassGlobal, "原请求当时被全局开关改写（" + ov + "），现已解除"
		case strings.HasPrefix(ov, "quota:"):
			return ClassQuota, "原请求当时被额度策略改写（" + ov + "），现已解除"
		case strings.HasPrefix(ov, "site:"):
			return ClassSiteNotServe, "原请求当时站点未提供服务（" + ov + "），现已恢复"
		}
	}
	if exp.TouchType == "return" || exp.TouchType == "handoff" || strings.HasPrefix(exp.TouchType, "handoff") {
		return ClassNeedsToken, "原请求为 " + exp.TouchType + "，依赖访客或交接令牌"
	}
	if len(c.Reconstructed) > 0 && (len(o.ValidationErrors) > 0 || !sameAttr) {
		return ClassReconstructed, "合成参数 " + strings.Join(c.Reconstructed, ",") + " 的校验结果与原请求不同"
	}
	if !sameContent {
		return ClassUnexplained, "内容 " + exp.Content + " → " + o.Content
	}
	if !sameTouch {
		return ClassUnexplained, "触点 " + exp.TouchType + " → " + o.TouchType
	}
	return ClassUnexplained, "归因 " + describeAttr(exp.Attributed, exp.Channel, exp.Code) + " → " + describeAttr(o.Attributed, o.Channel, o.Code)
}

func sameChannel(exp Expected, o Observed) bool {
	if exp.Channel != "" && exp.Channel != o.Channel {
		return false
	}
	if exp.Code != "" && exp.Code != o.Code {
		return false
	}
	return true
}

func findStep(trace []Step, step string) string {
	for _, s := range trace {
		if s.Step == step {
			return s.Result
		}
	}
	return ""
}

func describeAttr(attributed bool, channel, code string) string {
	if !attributed {
		return "未归因"
	}
	s := channel
	if code != "" {
		s += "(" + code + ")"
	}
	if s == "" {
		s = "已归因"
	}
	return s
}
