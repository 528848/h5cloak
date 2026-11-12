package opssvc

import (
	"context"

	"cloak/internal/domain/ops"
)

// NameCount 名称 + 计数，用于 Top 榜。
type NameCount struct {
	Name  string
	Count int64
}

// DigestData 日报 / 问答用的概况数据。
type DigestData struct {
	Visits      int64
	Real        int64
	Review      int64
	Block       int64
	First       int64
	Return      int64
	Handoff     int64
	Conversions int64
	Amount      float64
	TopChannels []NameCount
	TopCodes    []NameCount
	OpenAlerts  int64
}

// Digest 汇总某区间的概况，供 Telegram 日报与问答复用。
func (s *Service) Digest(ctx context.Context, merchantID, siteID uint64, r Range) (DigestData, error) {
	var d DigestData
	hours, err := s.repo.HourlyRange(ctx, merchantID, siteID, r.From, r.To)
	if err != nil {
		return d, err
	}
	sum := sumHourly(hours)
	d.Visits = sum["visits"]
	d.Real = sum["realTotal"]
	d.Review = sum["review"]
	d.Block = sum["block"]
	d.First = sum["firstReal"]
	d.Return = sum["returnReal"]
	d.Handoff = sum["handoffReal"]
	convN, convAmt, _ := s.repo.ConversionTotals(ctx, merchantID, siteID, r.From, r.To)
	d.Conversions = convN
	d.Amount = convAmt
	for _, kv := range topKV(s, ctx, merchantID, siteID, r, "channel", 3) {
		d.TopChannels = append(d.TopChannels, kv)
	}
	for _, kv := range topKV(s, ctx, merchantID, siteID, r, "code", 3) {
		d.TopCodes = append(d.TopCodes, kv)
	}
	if counts, e := s.repo.AlertCounts(ctx, merchantID); e == nil {
		d.OpenAlerts = counts[ops.AlertOpen]
	}
	return d, nil
}

func topKV(s *Service, ctx context.Context, merchantID, siteID uint64, r Range, col string, n int) []NameCount {
	rows, err := s.repo.Breakdown(ctx, merchantID, siteID, r.From, r.To, col, true, n)
	if err != nil {
		return nil
	}
	out := make([]NameCount, 0, len(rows))
	for _, kv := range rows {
		name := kv.K
		if name == "" {
			name = "(空)"
		}
		out = append(out, NameCount{Name: name, Count: kv.V})
	}
	return out
}
