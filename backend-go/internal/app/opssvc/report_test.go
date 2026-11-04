package opssvc

import (
	"testing"
	"time"

	"cloak/internal/domain/ops"
)

func TestApplyLiveHourDoesNotDoubleCount(t *testing.T) {
	cur := time.Date(2026, 11, 4, 15, 0, 0, 0, time.Local)
	hours := []ops.MetricsHourly{
		{Hour: cur.Add(-time.Hour), Visits: 88, RealTotal: 58, Review: 30},
		{Hour: cur, Visits: 1, RealTotal: 1, ValidationFailed: 2},
	}
	live := map[string]int64{"total": 1, "real": 1, "review": 0, "block": 0, "bypass": 0, "validationFailed": 2, "degraded": 0}

	sum := sumHourly(hours)
	delta := applyLiveHour(sum, hours, cur, live)
	if sum["visits"] != 89 || sum["realTotal"] != 59 || sum["validationFailed"] != 2 {
		t.Fatalf("已聚合与实时一致时不应再加一次: %+v", sum)
	}
	if delta["visits"] != 0 || delta["real"] != 0 {
		t.Fatalf("增量应为 0: %+v", delta)
	}

	live["total"], live["real"] = 4, 3
	sum = sumHourly(hours)
	delta = applyLiveHour(sum, hours, cur, live)
	if sum["visits"] != 92 || sum["realTotal"] != 61 {
		t.Fatalf("实时多于聚合时应只补差值: %+v", sum)
	}
	if delta["visits"] != 3 || delta["real"] != 2 {
		t.Fatalf("增量不对: %+v", delta)
	}
}

func TestApplyLiveHourWithoutAggregateRow(t *testing.T) {
	cur := time.Date(2026, 11, 4, 15, 0, 0, 0, time.Local)
	hours := []ops.MetricsHourly{{Hour: cur.Add(-time.Hour), Visits: 10, RealTotal: 6}}
	live := map[string]int64{"total": 3, "real": 2, "review": 1, "block": 0, "bypass": 0, "validationFailed": 0, "degraded": 0}
	sum := sumHourly(hours)
	delta := applyLiveHour(sum, hours, cur, live)
	if sum["visits"] != 13 || sum["realTotal"] != 8 || sum["review"] != 1 {
		t.Fatalf("没有当前小时行时应整段加上: %+v", sum)
	}
	if delta["visits"] != 3 {
		t.Fatalf("delta %+v", delta)
	}
}

func TestPatchTrendHourlyAndDaily(t *testing.T) {
	cur := time.Date(2026, 11, 4, 15, 0, 0, 0, time.Local)
	hourly := []map[string]any{{"t": "11-04 15:00", "visits": uint64(1), "real": uint64(1), "review": uint64(0), "block": uint64(0)}}
	got := patchTrend(hourly, cur, false, map[string]int64{"visits": 3, "real": 2, "review": 1, "block": 0, "degraded": 0})
	if num64(got[0]["visits"]) != 4 || num64(got[0]["real"]) != 3 || num64(got[0]["review"]) != 1 {
		t.Fatalf("小时点应加上增量: %+v", got[0])
	}

	daily := []map[string]any{{"t": "11-04", "visits": uint64(89), "real": uint64(59), "review": uint64(30), "block": uint64(0)}}
	got = patchTrend(daily, cur, true, map[string]int64{"visits": 2, "real": 2, "review": 0, "block": 0})
	if num64(got[0]["visits"]) != 91 || num64(got[0]["real"]) != 61 {
		t.Fatalf("日点应加上增量: %+v", got[0])
	}

	empty := patchTrend(nil, cur, false, map[string]int64{"visits": 2, "real": 2, "review": 0, "block": 0})
	if len(empty) != 1 || empty[0]["t"] != "11-04 15:00" || num64(empty[0]["visits"]) != 2 {
		t.Fatalf("没有点时应追加当前小时: %+v", empty)
	}
	same := patchTrend(nil, cur, false, map[string]int64{"visits": 0, "real": 0, "review": 0, "block": 0})
	if len(same) != 0 {
		t.Fatalf("增量为 0 时不应追加空点: %+v", same)
	}
}
