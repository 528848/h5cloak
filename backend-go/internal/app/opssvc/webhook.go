package opssvc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"go.uber.org/zap"

	"cloak/internal/domain/ops"
	"cloak/internal/pkg/notifier"
)

// WebhookEvents 可订阅事件。
var WebhookEvents = []string{
	ops.AlertOriginUnhealthy, ops.AlertOriginUnhealthy + ".resolved",
	ops.AlertDomainUnhealthy, ops.AlertDomainUnhealthy + ".resolved",
	ops.AlertCertExpiring, ops.AlertCertExpiring + ".resolved",
	ops.AlertQuotaThreshold, ops.AlertReviewSpike, ops.AlertCodeAbnormal, ops.AlertValidationSpike, ops.AlertDegraded,
	"export.done", "webhook.test",
}

// EndpointView 输出。
type EndpointView struct {
	ID              uint64   `json:"id"`
	Name            string   `json:"name"`
	URL             string   `json:"url"`
	SecretHint      string   `json:"secretHint"`
	Events          []string `json:"events"`
	Enabled         bool     `json:"enabled"`
	LastStatus      int      `json:"lastStatus"`
	LastDeliveredAt string   `json:"lastDeliveredAt"`
	FailureCount    uint32   `json:"failureCount"`
	CreatedAt       string   `json:"createdAt"`
}

func endpointView(e ops.WebhookEndpoint) EndpointView {
	v := EndpointView{ID: e.ID, Name: e.Name, URL: e.URL, Events: jsonList(e.Events), Enabled: e.Enabled, LastStatus: e.LastStatus,
		FailureCount: e.FailureCount, CreatedAt: e.CreatedAt.Format(layout)}
	if len(e.Secret) > 8 {
		v.SecretHint = e.Secret[:8] + "…"
	}
	if e.LastDeliveredAt != nil {
		v.LastDeliveredAt = e.LastDeliveredAt.Format(layout)
	}
	return v
}

func (s *Service) Endpoints(ctx context.Context, merchantID uint64) ([]EndpointView, error) {
	rows, err := s.repo.ListEndpoints(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	out := make([]EndpointView, 0, len(rows))
	for _, r := range rows {
		out = append(out, endpointView(r))
	}
	return out, nil
}

// EndpointInput 新建 / 编辑。
type EndpointInput struct {
	Name    string
	URL     string
	Events  []string
	Enabled *bool
}

// SaveEndpoint id=0 新建；返回视图与（仅新建时）明文密钥。
func (s *Service) SaveEndpoint(ctx context.Context, merchantID, id uint64, in EndpointInput) (EndpointView, string, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.URL = strings.TrimSpace(in.URL)
	if in.Name == "" || len(in.Name) > 128 {
		return EndpointView{}, "", errf(400, "名称必填，不超过 128 字")
	}
	if err := s.webhook.ValidateURL(in.URL); err != nil {
		return EndpointView{}, "", errf(400, err.Error())
	}
	valid := map[string]bool{}
	for _, e := range WebhookEvents {
		valid[e] = true
	}
	events := make([]string, 0, len(in.Events))
	for _, e := range in.Events {
		if valid[e] {
			events = append(events, e)
		}
	}
	ej, _ := json.Marshal(events)
	var row *ops.WebhookEndpoint
	secret := ""
	if id == 0 {
		list, _ := s.repo.ListEndpoints(ctx, merchantID)
		if len(list) >= 10 {
			return EndpointView{}, "", errf(400, "最多 10 个 Webhook")
		}
		buf := make([]byte, 24)
		_, _ = rand.Read(buf)
		secret = "whsec_" + hex.EncodeToString(buf)
		row = &ops.WebhookEndpoint{MerchantID: merchantID, Secret: secret, Enabled: true, CreatedAt: s.now()}
	} else {
		var err error
		row, err = s.repo.FindEndpoint(ctx, merchantID, id)
		if err != nil {
			return EndpointView{}, "", errf(404, "Webhook 不存在")
		}
	}
	row.Name, row.URL, row.Events = in.Name, in.URL, ej
	if in.Enabled != nil {
		row.Enabled = *in.Enabled
	}
	row.UpdatedAt = s.now()
	if err := s.repo.SaveEndpoint(ctx, row); err != nil {
		return EndpointView{}, "", err
	}
	return endpointView(*row), secret, nil
}

// RotateEndpointSecret 重置密钥。
func (s *Service) RotateEndpointSecret(ctx context.Context, merchantID, id uint64) (string, error) {
	row, err := s.repo.FindEndpoint(ctx, merchantID, id)
	if err != nil {
		return "", errf(404, "Webhook 不存在")
	}
	buf := make([]byte, 24)
	_, _ = rand.Read(buf)
	row.Secret = "whsec_" + hex.EncodeToString(buf)
	row.UpdatedAt = s.now()
	return row.Secret, s.repo.SaveEndpoint(ctx, row)
}

func (s *Service) DeleteEndpoint(ctx context.Context, merchantID, id uint64) error {
	return s.repo.DeleteEndpoint(ctx, merchantID, id)
}

// EnqueueWebhook 给商户所有订阅该事件的端点各建一条投递。返回入队数。
func (s *Service) EnqueueWebhook(ctx context.Context, merchantID uint64, event, eventID string, data map[string]any) (int, error) {
	eps, err := s.repo.EnabledEndpoints(ctx, merchantID)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, ep := range eps {
		subs := jsonList(ep.Events)
		if len(subs) > 0 && !contains(subs, event) {
			continue
		}
		payload, _ := json.Marshal(map[string]any{
			"id": eventID, "event": event, "merchantId": merchantID, "createdAt": s.now().UTC().Format(time.RFC3339), "data": data,
		})
		d := &ops.WebhookDelivery{MerchantID: merchantID, EndpointID: ep.ID, Event: event, EventID: eventID, Payload: payload,
			MaxAttempts: uint32(s.maxAttempts()), Status: ops.DeliveryPending, NextAttemptAt: s.now(), CreatedAt: s.now()}
		if err := s.repo.CreateDelivery(ctx, d); err != nil {
			s.log.Warn("enqueue webhook", zap.Error(err))
			continue
		}
		n++
	}
	return n, nil
}

func (s *Service) maxAttempts() int {
	if s.attemptSrc != nil {
		s.attemptMu.Lock()
		defer s.attemptMu.Unlock()
		if s.attemptN == 0 || time.Since(s.attemptAt) > 15*time.Second {
			if n := s.attemptSrc(context.Background()); n > 0 {
				s.attemptN = n
				s.attemptAt = time.Now()
			}
		}
		if s.attemptN > 0 {
			return s.attemptN
		}
	}
	if s.cfg.Webhook.MaxAttempts > 0 {
		return s.cfg.Webhook.MaxAttempts
	}
	return 5
}

// SendTest 立即发送一次测试事件（同步）。
func (s *Service) SendTest(ctx context.Context, merchantID, id uint64) (map[string]any, error) {
	ep, err := s.repo.FindEndpoint(ctx, merchantID, id)
	if err != nil {
		return nil, errf(404, "Webhook 不存在")
	}
	eventID := "test-" + hex.EncodeToString(randBytes(8))
	payload, _ := json.Marshal(map[string]any{
		"id": eventID, "event": "webhook.test", "merchantId": merchantID, "createdAt": s.now().UTC().Format(time.RFC3339),
		"data": map[string]any{"message": "斗篷系统 Webhook 测试", "endpointId": ep.ID},
	})
	d := &ops.WebhookDelivery{MerchantID: merchantID, EndpointID: ep.ID, Event: "webhook.test", EventID: eventID, Payload: payload,
		MaxAttempts: 1, Status: ops.DeliveryPending, NextAttemptAt: s.now(), CreatedAt: s.now()}
	if err := s.repo.CreateDelivery(ctx, d); err != nil {
		return nil, err
	}
	s.deliver(ctx, d, ep)
	return deliveryView(*d), nil
}

// Redeliver 手动重发（新建一条投递并立即执行）。
func (s *Service) Redeliver(ctx context.Context, merchantID, deliveryID uint64) (map[string]any, error) {
	old, err := s.repo.FindDelivery(ctx, merchantID, deliveryID)
	if err != nil {
		return nil, errf(404, "投递记录不存在")
	}
	ep, err := s.repo.FindEndpoint(ctx, merchantID, old.EndpointID)
	if err != nil {
		return nil, errf(404, "Webhook 已删除")
	}
	d := &ops.WebhookDelivery{MerchantID: merchantID, EndpointID: ep.ID, Event: old.Event, EventID: old.EventID, Payload: old.Payload,
		MaxAttempts: 1, Status: ops.DeliveryPending, NextAttemptAt: s.now(), CreatedAt: s.now()}
	if err := s.repo.CreateDelivery(ctx, d); err != nil {
		return nil, err
	}
	s.deliver(ctx, d, ep)
	return deliveryView(*d), nil
}

// deliver 执行一次投递并写回状态。
func (s *Service) deliver(ctx context.Context, d *ops.WebhookDelivery, ep *ops.WebhookEndpoint) {
	d.Attempt++
	res := s.webhook.Deliver(ctx, ep.URL, ep.Secret, d.Event, d.EventID, d.ID, d.Payload)
	d.ResponseStatus = res.Status
	d.ResponseBody = res.Body
	d.LastError = res.Err
	now := s.now()
	ok := res.Err == ""
	if ok {
		d.Status = ops.DeliverySuccess
		d.DeliveredAt = &now
	} else if d.Attempt >= d.MaxAttempts {
		d.Status = ops.DeliveryFailed
	} else {
		d.Status = ops.DeliveryPending
		d.NextAttemptAt = now.Add(notifier.Backoff(int(d.Attempt)))
	}
	_ = s.repo.UpdateDelivery(ctx, d)
	_ = s.repo.UpdateEndpointStatus(ctx, ep.ID, res.Status, ok, now)
}

// ProcessDue 投递到期记录（worker 周期调用）。
func (s *Service) ProcessDue(ctx context.Context, limitN int) int {
	rows, err := s.repo.DueDeliveries(ctx, s.now(), limitN)
	if err != nil {
		s.log.Warn("due deliveries", zap.Error(err))
		return 0
	}
	n := 0
	cache := map[uint64]*ops.WebhookEndpoint{}
	for i := range rows {
		d := &rows[i]
		ep, ok := cache[d.EndpointID]
		if !ok {
			ep, err = s.repo.FindEndpoint(ctx, d.MerchantID, d.EndpointID)
			if err != nil || !ep.Enabled {
				d.Status = ops.DeliveryFailed
				d.LastError = "endpoint_disabled"
				_ = s.repo.UpdateDelivery(ctx, d)
				continue
			}
			cache[d.EndpointID] = ep
		}
		s.deliver(ctx, d, ep)
		n++
	}
	return n
}

func (s *Service) Deliveries(ctx context.Context, merchantID, endpointID uint64, status string, page, size int) ([]map[string]any, int64, error) {
	rows, total, err := s.repo.ListDeliveries(ctx, merchantID, endpointID, status, page, size)
	if err != nil {
		return nil, 0, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, deliveryView(r))
	}
	return out, total, nil
}

func (s *Service) Delivery(ctx context.Context, merchantID, id uint64) (map[string]any, error) {
	d, err := s.repo.FindDelivery(ctx, merchantID, id)
	if err != nil {
		return nil, errf(404, "投递记录不存在")
	}
	v := deliveryView(*d)
	v["payload"] = json.RawMessage(nonEmpty(d.Payload))
	return v, nil
}

func deliveryView(d ops.WebhookDelivery) map[string]any {
	v := map[string]any{
		"id": d.ID, "endpointId": d.EndpointID, "event": d.Event, "eventId": d.EventID, "attempt": d.Attempt, "maxAttempts": d.MaxAttempts,
		"status": d.Status, "responseStatus": d.ResponseStatus, "responseBody": d.ResponseBody, "lastError": d.LastError,
		"nextAttemptAt": d.NextAttemptAt.Format(layout), "createdAt": d.CreatedAt.Format(layout),
	}
	if d.DeliveredAt != nil {
		v["deliveredAt"] = d.DeliveredAt.Format(layout)
	}
	return v
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}
