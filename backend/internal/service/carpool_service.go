package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/account"
	"github.com/Wei-Shaw/sub2api/ent/carpoolnoticeversion"
	"github.com/Wei-Shaw/sub2api/ent/carpoolparticipant"
	"github.com/Wei-Shaw/sub2api/ent/carpoolsession"
	"github.com/Wei-Shaw/sub2api/ent/carpoolvehicletype"
	"github.com/Wei-Shaw/sub2api/ent/carpoolvoucher"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/usagelog"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	CarpoolSessionRecruiting   = "recruiting"
	CarpoolSessionFull         = "full"
	CarpoolSessionProvisioning = "provisioning"
	CarpoolSessionActive       = "active"
	CarpoolSessionFailed       = "failed"
	CarpoolSessionCancelled    = "cancelled"
	CarpoolSessionEnded        = "ended"

	CarpoolParticipantPendingPayment  = "pending_payment"
	CarpoolParticipantPaid            = "paid"
	CarpoolParticipantActive          = "active"
	CarpoolParticipantRefundPending   = "refund_pending"
	CarpoolParticipantRefundedBalance = "refunded_balance"
	CarpoolParticipantRefundedGateway = "refunded_gateway"
	CarpoolParticipantCancelled       = "cancelled"

	CarpoolRefundBalance = "balance"
	CarpoolRefundGateway = "gateway"

	defaultCarpoolRefundWaitHours = 2
)

type CarpoolService struct {
	entClient      *dbent.Client
	paymentService *PaymentService
	settingService *SettingService
}

func NewCarpoolService(entClient *dbent.Client) *CarpoolService {
	return &CarpoolService{entClient: entClient}
}

func (s *CarpoolService) SetPaymentService(paymentService *PaymentService) {
	s.paymentService = paymentService
}

func (s *CarpoolService) SetSettingService(settingService *SettingService) {
	s.settingService = settingService
}

type CarpoolVehicleTypeInput struct {
	Product             string   `json:"product"`
	PlanTier            string   `json:"plan_tier"`
	Multiplier          string   `json:"multiplier"`
	Name                string   `json:"name"`
	SeatCount           int      `json:"seat_count"`
	TotalPrice          float64  `json:"total_price"`
	UnitPrice           float64  `json:"unit_price"`
	ServiceDays         int      `json:"service_days"`
	RefundWaitHours     int      `json:"refund_wait_hours"`
	CompletedBaseCount  int      `json:"completed_base_count"`
	Enabled             bool     `json:"enabled"`
	SupportRevenuePool  bool     `json:"support_revenue_pool"`
	RequireStaticIP     bool     `json:"require_static_ip"`
	WaitDurationOptions []int    `json:"wait_duration_options"`
	RefundMethods       []string `json:"refund_methods"`
	Description         string   `json:"description"`
	SortOrder           int      `json:"sort_order"`
}

type CarpoolNoticeInput struct {
	Title     string `json:"title"`
	ContentMD string `json:"content_md"`
	Active    bool   `json:"active"`
}

type CarpoolJoinInput struct {
	VehicleTypeID   int64  `json:"vehicle_type_id"`
	NoticeVersionID int64  `json:"notice_version_id"`
	NoticeAccepted  bool   `json:"notice_accepted"`
	PaymentType     string `json:"payment_type"`
	ReturnURL       string `json:"return_url"`
	PaymentSource   string `json:"payment_source"`
	OpenID          string `json:"openid"`
	ClientIP        string `json:"-"`
	IsMobile        bool   `json:"-"`
	IsWeChatBrowser bool   `json:"-"`
	SrcHost         string `json:"-"`
	SrcURL          string `json:"-"`
	Locale          string `json:"-"`
}

type CarpoolJoinResponse struct {
	Participant *dbent.CarpoolParticipant `json:"participant"`
	Order       *CreateOrderResponse      `json:"order"`
}

type CarpoolRefundRequestInput struct {
	RefundMethod string `json:"refund_method"`
}

type CarpoolSessionProvisionInput struct {
	Status        string         `json:"status"`
	AccountInfo   map[string]any `json:"account_info"`
	ProxyInfo     map[string]any `json:"proxy_info"`
	Communication map[string]any `json:"communication"`
	AdminNotes    string         `json:"admin_notes"`
}

type CarpoolVoucherInput struct {
	FileURL     string `json:"file_url"`
	FileName    string `json:"file_name"`
	Description string `json:"description"`
}

type CarpoolAdminUsageSummary struct {
	RequestCount    int     `json:"request_count"`
	TotalTokens     int64   `json:"total_tokens"`
	TotalCost       float64 `json:"total_cost"`
	TotalActualCost float64 `json:"total_actual_cost"`
}

type CarpoolAdminParticipantRow struct {
	Participant *dbent.CarpoolParticipant `json:"participant"`
	Usage       CarpoolAdminUsageSummary  `json:"usage"`
	User        map[string]any            `json:"user,omitempty"`
}

type CarpoolAdminSessionRow struct {
	Session      *dbent.CarpoolSession        `json:"session"`
	Participants []CarpoolAdminParticipantRow `json:"participants"`
	Usage        CarpoolAdminUsageSummary     `json:"usage"`
}

type CarpoolUserUsageWindows struct {
	FiveHour CarpoolAdminUsageSummary `json:"five_hour"`
	SevenDay CarpoolAdminUsageSummary `json:"seven_day"`
}

type CarpoolAccountWindowUsage struct {
	AccountID        int64                    `json:"account_id"`
	AccountName      string                   `json:"account_name"`
	Window           string                   `json:"window"`
	Utilization      float64                  `json:"utilization"`
	ResetsAt         *time.Time               `json:"resets_at,omitempty"`
	RemainingSeconds int                      `json:"remaining_seconds"`
	Usage            CarpoolAdminUsageSummary `json:"usage"`
}

type CarpoolUserMemberUsage struct {
	ParticipantID int64                    `json:"participant_id"`
	UserID        int64                    `json:"user_id"`
	DisplayName   string                   `json:"display_name"`
	Initial       string                   `json:"initial"`
	AvatarURL     string                   `json:"avatar_url,omitempty"`
	IsSelf        bool                     `json:"is_self"`
	Status        string                   `json:"status"`
	Usage         CarpoolAdminUsageSummary `json:"usage"`
	Windows       CarpoolUserUsageWindows  `json:"windows"`
}

type CarpoolUserDetail struct {
	Participant    *dbent.CarpoolParticipant   `json:"participant"`
	Session        *dbent.CarpoolSession       `json:"session"`
	Members        []CarpoolUserMemberUsage    `json:"members"`
	TotalUsage     CarpoolAdminUsageSummary    `json:"total_usage"`
	TotalWindows   CarpoolUserUsageWindows     `json:"total_windows"`
	AccountWindows []CarpoolAccountWindowUsage `json:"account_windows"`
}

type CarpoolAdminManagementSummary struct {
	CompletedSessions int64            `json:"completed_sessions"`
	PaidMembers       int64            `json:"paid_members"`
	ActiveMembers     int64            `json:"active_members"`
	TotalPaidAmount   float64          `json:"total_paid_amount"`
	TotalTokens       int64            `json:"total_tokens"`
	TotalActualCost   float64          `json:"total_actual_cost"`
	ByStatus          map[string]int64 `json:"by_status"`
	BySegment         []map[string]any `json:"by_segment"`
}

type CarpoolAdminManagementResponse struct {
	Summary  CarpoolAdminManagementSummary `json:"summary"`
	Items    []CarpoolAdminSessionRow      `json:"items"`
	Total    int                           `json:"total"`
	Page     int                           `json:"page"`
	PageSize int                           `json:"page_size"`
}

func (s *CarpoolService) ListVehicleTypes(ctx context.Context, enabledOnly bool) ([]*dbent.CarpoolVehicleType, error) {
	q := s.entClient.CarpoolVehicleType.Query().Order(
		dbent.Asc(carpoolvehicletype.FieldProduct),
		dbent.Asc(carpoolvehicletype.FieldPlanTier),
		dbent.Asc(carpoolvehicletype.FieldMultiplier),
		dbent.Asc(carpoolvehicletype.FieldSortOrder),
		dbent.Asc(carpoolvehicletype.FieldSeatCount),
		dbent.Asc(carpoolvehicletype.FieldID),
	)
	if enabledOnly {
		q.Where(carpoolvehicletype.EnabledEQ(true))
	}
	return q.All(ctx)
}

func (s *CarpoolService) GetCurrentCards(ctx context.Context) ([]map[string]any, error) {
	types, err := s.ListVehicleTypes(ctx, true)
	if err != nil {
		return nil, err
	}
	completedCounts, err := s.completedSessionCountsByVehicleType(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(types))
	for _, vt := range types {
		session, err := s.getOrCreateRecruitingSession(ctx, vt)
		if err != nil {
			return nil, err
		}
		completedCount := completedCounts[vt.ID]
		out = append(out, map[string]any{
			"vehicle_type":            vt,
			"session":                 session,
			"paid_count":              session.PaidCount,
			"seat_count":              session.SeatCount,
			"completed_count":         completedCount,
			"display_completed_count": vt.CompletedBaseCount + completedCount,
			"real_completed_count":    completedCount,
			"refund_available_at":     time.Now().Add(time.Duration(vt.RefundWaitHours) * time.Hour),
			"refund_wait_hours":       vt.RefundWaitHours,
			"completed_base_count":    vt.CompletedBaseCount,
		})
	}
	return out, nil
}

func (s *CarpoolService) GetCurrentNotice(ctx context.Context) (*dbent.CarpoolNoticeVersion, error) {
	n, err := s.entClient.CarpoolNoticeVersion.Query().
		Where(carpoolnoticeversion.ActiveEQ(true)).
		Order(dbent.Desc(carpoolnoticeversion.FieldVersion), dbent.Desc(carpoolnoticeversion.FieldID)).
		First(ctx)
	if err == nil {
		return n, nil
	}
	if !dbent.IsNotFound(err) {
		return nil, err
	}
	now := time.Now()
	return s.entClient.CarpoolNoticeVersion.Create().
		SetTitle("拼车用户须知").
		SetContentMd(defaultCarpoolNotice()).
		SetVersion(1).
		SetActive(true).
		SetPublishedAt(now).
		Save(ctx)
}

func (s *CarpoolService) Join(ctx context.Context, userID int64, input CarpoolJoinInput) (*CarpoolJoinResponse, error) {
	if s.paymentService == nil {
		return nil, infraerrors.InternalServer("CARPOOL_PAYMENT_UNAVAILABLE", "carpool payment is unavailable")
	}
	if userID <= 0 {
		return nil, infraerrors.Unauthorized("UNAUTHORIZED", "unauthorized")
	}
	if !input.NoticeAccepted {
		return nil, infraerrors.BadRequest("CARPOOL_NOTICE_REQUIRED", "please read and accept the carpool notice")
	}
	vt, err := s.entClient.CarpoolVehicleType.Get(ctx, input.VehicleTypeID)
	if err != nil {
		return nil, infraerrors.NotFound("CARPOOL_TYPE_NOT_FOUND", "carpool type not found")
	}
	if !vt.Enabled {
		return nil, infraerrors.Forbidden("CARPOOL_TYPE_DISABLED", "carpool type is disabled")
	}
	notice, err := s.GetCurrentNotice(ctx)
	if err != nil {
		return nil, err
	}
	if input.NoticeVersionID != notice.ID {
		return nil, infraerrors.BadRequest("CARPOOL_NOTICE_VERSION_MISMATCH", "please accept the current carpool notice")
	}
	refundWaitHours := normalizeRefundWaitHours(vt.RefundWaitHours)
	amount := vt.UnitPrice
	if amount <= 0 && vt.SeatCount > 0 {
		amount = math.Round((vt.TotalPrice/float64(vt.SeatCount))*100) / 100
	}
	if amount <= 0 {
		return nil, infraerrors.BadRequest("CARPOOL_INVALID_PRICE", "carpool price is invalid")
	}
	now := time.Now()
	participant, err := s.entClient.CarpoolParticipant.Create().
		SetVehicleTypeID(vt.ID).
		SetUserID(userID).
		SetStatus(CarpoolParticipantPendingPayment).
		SetAmount(amount).
		SetWaitUntil(now.Add(time.Duration(refundWaitHours) * time.Hour)).
		SetRefundMethod(CarpoolRefundBalance).
		SetNoticeVersionID(notice.ID).
		SetNoticeAcceptedAt(now).
		SetNoticeAcceptIP(strings.TrimSpace(input.ClientIP)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create carpool participant: %w", err)
	}
	order, err := s.paymentService.CreateOrder(ctx, CreateOrderRequest{
		UserID:          userID,
		Amount:          amount,
		PaymentType:     input.PaymentType,
		OpenID:          input.OpenID,
		ClientIP:        input.ClientIP,
		IsMobile:        input.IsMobile,
		IsWeChatBrowser: input.IsWeChatBrowser,
		SrcHost:         input.SrcHost,
		SrcURL:          input.SrcURL,
		ReturnURL:       input.ReturnURL,
		PaymentSource:   input.PaymentSource,
		OrderType:       payment.OrderTypeCarpool,
		Locale:          input.Locale,
	})
	if err != nil {
		_ = s.entClient.CarpoolParticipant.UpdateOneID(participant.ID).SetStatus(CarpoolParticipantCancelled).Exec(ctx)
		return nil, err
	}
	if _, err := s.entClient.CarpoolParticipant.UpdateOneID(participant.ID).SetPaymentOrderID(order.OrderID).Save(ctx); err != nil {
		return nil, fmt.Errorf("bind payment order: %w", err)
	}
	participant, _ = s.entClient.CarpoolParticipant.Get(ctx, participant.ID)
	return &CarpoolJoinResponse{Participant: participant, Order: order}, nil
}

func (s *CarpoolService) RequestRefund(ctx context.Context, userID, participantID int64, input CarpoolRefundRequestInput) (*dbent.CarpoolParticipant, error) {
	if userID <= 0 {
		return nil, infraerrors.Unauthorized("UNAUTHORIZED", "unauthorized")
	}
	p, err := s.entClient.CarpoolParticipant.Query().
		Where(carpoolparticipant.IDEQ(participantID), carpoolparticipant.UserIDEQ(userID)).
		WithVehicleType().
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("CARPOOL_PARTICIPANT_NOT_FOUND", "carpool record not found")
		}
		return nil, err
	}
	if p.Status != CarpoolParticipantPaid {
		return nil, infraerrors.BadRequest("CARPOOL_REFUND_STATUS_INVALID", "only waiting paid carpool records can request refund")
	}
	if p.SessionID != nil {
		session, err := s.entClient.CarpoolSession.Get(ctx, *p.SessionID)
		if err == nil && session.Status != CarpoolSessionRecruiting {
			return nil, infraerrors.BadRequest("CARPOOL_REFUND_SESSION_NOT_WAITING", "carpool has already succeeded or is being provisioned")
		}
		if err != nil && !dbent.IsNotFound(err) {
			return nil, err
		}
	}
	now := time.Now()
	if now.Before(p.WaitUntil) {
		return nil, infraerrors.BadRequest("CARPOOL_REFUND_NOT_AVAILABLE", "refund is not available yet")
	}
	allowed := []string{CarpoolRefundBalance, CarpoolRefundGateway}
	if p.Edges.VehicleType != nil && len(p.Edges.VehicleType.RefundMethods) > 0 {
		allowed = p.Edges.VehicleType.RefundMethods
	}
	method := normalizeRefundMethod(input.RefundMethod, allowed)
	processing, err := s.entClient.CarpoolParticipant.UpdateOneID(p.ID).
		Where(carpoolparticipant.StatusEQ(CarpoolParticipantPaid)).
		SetStatus(CarpoolParticipantRefundPending).
		SetRefundMethod(method).
		Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.Conflict("CARPOOL_REFUND_CONFLICT", "carpool status changed")
		}
		return nil, err
	}
	if p.SessionID != nil {
		if err := s.recalculateSessionPaidCount(ctx, *p.SessionID); err != nil {
			return nil, err
		}
	}
	updated, err := s.executeParticipantRefund(ctx, processing)
	if err != nil {
		_, _ = s.entClient.CarpoolParticipant.UpdateOneID(p.ID).
			Where(carpoolparticipant.StatusEQ(CarpoolParticipantRefundPending)).
			SetStatus(CarpoolParticipantPaid).
			Save(ctx)
		if p.SessionID != nil {
			_ = s.recalculateSessionPaidCount(ctx, *p.SessionID)
		}
		return nil, err
	}
	return updated, nil
}

func (s *CarpoolService) HandlePaymentCompleted(ctx context.Context, orderID int64) error {
	o, err := s.entClient.PaymentOrder.Get(ctx, orderID)
	if err != nil {
		return err
	}
	if o.OrderType != payment.OrderTypeCarpool {
		return nil
	}
	p, err := s.entClient.CarpoolParticipant.Query().Where(carpoolparticipant.PaymentOrderIDEQ(orderID)).Only(ctx)
	if err != nil {
		return fmt.Errorf("find carpool participant for order %d: %w", orderID, err)
	}
	if p.Status == CarpoolParticipantPaid || p.Status == CarpoolParticipantActive {
		return s.markOrderCompleted(ctx, o)
	}
	vt, err := s.entClient.CarpoolVehicleType.Get(ctx, p.VehicleTypeID)
	if err != nil {
		return err
	}
	session, err := s.getOrCreateRecruitingSession(ctx, vt)
	if err != nil {
		return err
	}
	now := time.Now()
	_, err = s.entClient.CarpoolParticipant.UpdateOneID(p.ID).
		SetSessionID(session.ID).
		SetStatus(CarpoolParticipantPaid).
		SetPaidAt(now).
		SetJoinedAt(now).
		Save(ctx)
	if err != nil {
		return err
	}
	paidCount, err := s.entClient.CarpoolParticipant.Query().
		Where(carpoolparticipant.SessionIDEQ(session.ID), carpoolparticipant.StatusIn(CarpoolParticipantPaid, CarpoolParticipantActive)).
		Count(ctx)
	if err != nil {
		return err
	}
	up := s.entClient.CarpoolSession.UpdateOneID(session.ID).SetPaidCount(paidCount)
	if paidCount >= session.SeatCount {
		up.SetStatus(CarpoolSessionFull).SetFilledAt(now)
	}
	if _, err := up.Save(ctx); err != nil {
		return err
	}
	if paidCount >= session.SeatCount {
		s.notifyCarpoolFullAdmins(ctx, session.ID)
		if _, err := s.getOrCreateRecruitingSession(ctx, vt); err != nil {
			return err
		}
	}
	return s.markOrderCompleted(ctx, o)
}

func (s *CarpoolService) markOrderCompleted(ctx context.Context, o *dbent.PaymentOrder) error {
	if o.Status == OrderStatusCompleted {
		return nil
	}
	now := time.Now()
	_, err := s.entClient.PaymentOrder.Update().
		Where(paymentorder.IDEQ(o.ID), paymentorder.StatusIn(OrderStatusPaid, OrderStatusRecharging)).
		SetStatus(OrderStatusCompleted).
		SetCompletedAt(now).
		Save(ctx)
	return err
}

func (s *CarpoolService) executeParticipantRefund(ctx context.Context, p *dbent.CarpoolParticipant) (*dbent.CarpoolParticipant, error) {
	if p == nil {
		return nil, infraerrors.BadRequest("CARPOOL_PARTICIPANT_REQUIRED", "carpool record is required")
	}
	if p.PaymentOrderID == nil || *p.PaymentOrderID <= 0 {
		return nil, infraerrors.BadRequest("CARPOOL_REFUND_ORDER_MISSING", "payment order is missing")
	}
	if s.paymentService == nil {
		return nil, infraerrors.InternalServer("CARPOOL_PAYMENT_UNAVAILABLE", "carpool payment is unavailable")
	}
	method := normalizeRefundMethod(p.RefundMethod, []string{CarpoolRefundBalance, CarpoolRefundGateway})
	switch method {
	case CarpoolRefundGateway:
		return s.executeParticipantGatewayRefund(ctx, p)
	default:
		return s.executeParticipantBalanceRefund(ctx, p)
	}
}

func (s *CarpoolService) executeParticipantBalanceRefund(ctx context.Context, p *dbent.CarpoolParticipant) (*dbent.CarpoolParticipant, error) {
	orderID := *p.PaymentOrderID
	if s.paymentService.userRepo == nil {
		return nil, infraerrors.InternalServer("USER_REPOSITORY_UNAVAILABLE", "user repository is unavailable")
	}
	if err := s.paymentService.userRepo.UpdateBalance(ctx, p.UserID, p.Amount); err != nil {
		return nil, fmt.Errorf("refund carpool to balance: %w", err)
	}
	now := time.Now()
	_, err := s.entClient.PaymentOrder.Update().
		Where(paymentorder.IDEQ(orderID), paymentorder.StatusIn(OrderStatusCompleted, OrderStatusPaid, OrderStatusRefundFailed)).
		SetStatus(OrderStatusRefunded).
		SetRefundAmount(p.Amount).
		SetRefundReason("carpool refund to balance").
		SetRefundAt(now).
		Save(ctx)
	if err != nil {
		_ = s.paymentService.userRepo.UpdateBalance(ctx, p.UserID, -p.Amount)
		return nil, fmt.Errorf("mark carpool balance refund order: %w", err)
	}
	s.paymentService.writeAuditLog(ctx, orderID, "CARPOOL_REFUNDED_TO_BALANCE", fmt.Sprintf("user:%d", p.UserID), map[string]any{"amount": p.Amount, "participantID": p.ID})
	return s.entClient.CarpoolParticipant.UpdateOneID(p.ID).
		Where(carpoolparticipant.StatusEQ(CarpoolParticipantRefundPending)).
		SetStatus(CarpoolParticipantRefundedBalance).
		SetRefundedAt(now).
		Save(ctx)
}

func (s *CarpoolService) executeParticipantGatewayRefund(ctx context.Context, p *dbent.CarpoolParticipant) (*dbent.CarpoolParticipant, error) {
	orderID := *p.PaymentOrderID
	plan, earlyResult, err := s.paymentService.PrepareRefund(ctx, orderID, p.Amount, "carpool refund to gateway", false, false)
	if err != nil {
		return nil, err
	}
	if earlyResult != nil && !earlyResult.Success {
		return nil, infraerrors.BadRequest("CARPOOL_REFUND_GATEWAY_NOT_READY", earlyResult.Warning)
	}
	if plan == nil {
		return nil, infraerrors.InternalServer("CARPOOL_REFUND_PLAN_MISSING", "refund plan is missing")
	}
	result, err := s.paymentService.ExecuteRefund(ctx, plan)
	if err != nil {
		return nil, err
	}
	if result != nil && !result.Success {
		return nil, infraerrors.InternalServer("CARPOOL_REFUND_GATEWAY_FAILED", result.Warning)
	}
	now := time.Now()
	s.paymentService.writeAuditLog(ctx, orderID, "CARPOOL_REFUNDED_TO_GATEWAY", fmt.Sprintf("user:%d", p.UserID), map[string]any{"amount": p.Amount, "participantID": p.ID})
	return s.entClient.CarpoolParticipant.UpdateOneID(p.ID).
		Where(carpoolparticipant.StatusEQ(CarpoolParticipantRefundPending)).
		SetStatus(CarpoolParticipantRefundedGateway).
		SetRefundedAt(now).
		Save(ctx)
}

func (s *CarpoolService) MyParticipations(ctx context.Context, userID int64) ([]*dbent.CarpoolParticipant, error) {
	return s.entClient.CarpoolParticipant.Query().
		Where(
			carpoolparticipant.UserIDEQ(userID),
			carpoolparticipant.StatusIn(
				CarpoolParticipantPaid,
				CarpoolParticipantActive,
				CarpoolParticipantRefundPending,
				CarpoolParticipantRefundedBalance,
				CarpoolParticipantRefundedGateway,
			),
		).
		WithSession(func(q *dbent.CarpoolSessionQuery) { q.WithVehicleType().WithVouchers() }).
		WithVehicleType().
		Order(dbent.Desc(carpoolparticipant.FieldCreatedAt)).
		All(ctx)
}

func (s *CarpoolService) MyParticipationDetail(ctx context.Context, userID, participantID int64) (*CarpoolUserDetail, error) {
	if userID <= 0 {
		return nil, infraerrors.Unauthorized("UNAUTHORIZED", "unauthorized")
	}
	participant, err := s.entClient.CarpoolParticipant.Query().
		Where(
			carpoolparticipant.IDEQ(participantID),
			carpoolparticipant.UserIDEQ(userID),
			carpoolparticipant.StatusIn(
				CarpoolParticipantPaid,
				CarpoolParticipantActive,
				CarpoolParticipantRefundPending,
				CarpoolParticipantRefundedBalance,
				CarpoolParticipantRefundedGateway,
			),
		).
		WithVehicleType().
		WithSession(func(q *dbent.CarpoolSessionQuery) { q.WithVehicleType().WithVouchers() }).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("CARPOOL_PARTICIPANT_NOT_FOUND", "carpool record not found")
		}
		return nil, err
	}
	session := participant.Edges.Session
	if session == nil || participant.SessionID == nil {
		return &CarpoolUserDetail{
			Participant: participant,
			Session:     session,
			Members:     []CarpoolUserMemberUsage{},
		}, nil
	}

	members, err := s.entClient.CarpoolParticipant.Query().
		Where(
			carpoolparticipant.SessionIDEQ(session.ID),
			carpoolparticipant.StatusIn(CarpoolParticipantPaid, CarpoolParticipantActive),
		).
		WithUser().
		Order(dbent.Asc(carpoolparticipant.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	subscriptionGroupID := numberFromMap(session.AccountInfo, "subscription_group_id")
	accountPoolGroupID := numberFromMap(session.AccountInfo, "account_pool_group_id")
	if accountPoolGroupID <= 0 {
		accountPoolGroupID = subscriptionGroupID
	}
	detail := &CarpoolUserDetail{
		Participant:    participant,
		Session:        session,
		Members:        make([]CarpoolUserMemberUsage, 0, len(members)),
		AccountWindows: []CarpoolAccountWindowUsage{},
	}
	for _, member := range members {
		usage, err := s.carpoolParticipantUsageForGroup(ctx, session, member, subscriptionGroupID, 0)
		if err != nil {
			return nil, err
		}
		fiveHour, err := s.carpoolParticipantUsageForGroup(ctx, session, member, subscriptionGroupID, 5*time.Hour)
		if err != nil {
			return nil, err
		}
		sevenDay, err := s.carpoolParticipantUsageForGroup(ctx, session, member, subscriptionGroupID, 7*24*time.Hour)
		if err != nil {
			return nil, err
		}
		row := CarpoolUserMemberUsage{
			ParticipantID: member.ID,
			UserID:        member.UserID,
			DisplayName:   maskedCarpoolMemberName(member.Edges.User),
			Initial:       carpoolMemberInitial(member.Edges.User),
			IsSelf:        member.UserID == userID,
			Status:        member.Status,
			Usage:         usage,
			Windows: CarpoolUserUsageWindows{
				FiveHour: fiveHour,
				SevenDay: sevenDay,
			},
		}
		if avatarURL, err := s.userAvatarURL(ctx, member.UserID); err == nil {
			row.AvatarURL = avatarURL
		}
		detail.Members = append(detail.Members, row)
		detail.TotalUsage = addCarpoolUsage(detail.TotalUsage, usage)
		detail.TotalWindows.FiveHour = addCarpoolUsage(detail.TotalWindows.FiveHour, fiveHour)
		detail.TotalWindows.SevenDay = addCarpoolUsage(detail.TotalWindows.SevenDay, sevenDay)
	}

	if accountPoolGroupID > 0 {
		accountWindows, err := s.carpoolAccountWindows(ctx, accountPoolGroupID)
		if err != nil {
			return nil, err
		}
		detail.AccountWindows = accountWindows
	}
	return detail, nil
}

func (s *CarpoolService) AdminListSessions(ctx context.Context, page, pageSize int, status string) ([]*dbent.CarpoolSession, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	q := s.entClient.CarpoolSession.Query().WithVehicleType().WithVouchers().WithParticipants(func(pq *dbent.CarpoolParticipantQuery) {
		pq.WithUser()
	})
	if strings.TrimSpace(status) != "" {
		q.Where(carpoolsession.StatusEQ(strings.TrimSpace(status)))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	items, err := q.Order(dbent.Desc(carpoolsession.FieldCreatedAt)).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	return items, total, err
}

func (s *CarpoolService) AdminManagement(ctx context.Context, page, pageSize int, status string) (*CarpoolAdminManagementResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	status = strings.TrimSpace(status)
	buildQuery := func() *dbent.CarpoolSessionQuery {
		q := s.entClient.CarpoolSession.Query().WithVehicleType().WithVouchers().WithParticipants(func(pq *dbent.CarpoolParticipantQuery) {
			pq.WithUser().Order(dbent.Asc(carpoolparticipant.FieldID))
		})
		if status != "" && status != "all" {
			q.Where(carpoolsession.StatusEQ(status))
		} else {
			q.Where(carpoolsession.StatusIn(carpoolManagementStatuses()...))
		}
		return q
	}

	total, err := buildQuery().Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := buildQuery().
		Order(dbent.Desc(carpoolsession.FieldFilledAt), dbent.Desc(carpoolsession.FieldCreatedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.adminSessionRows(ctx, items)
	if err != nil {
		return nil, err
	}

	allSessions, err := buildQuery().Order(dbent.Desc(carpoolsession.FieldFilledAt), dbent.Desc(carpoolsession.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	allRows, err := s.adminSessionRows(ctx, allSessions)
	if err != nil {
		return nil, err
	}
	return &CarpoolAdminManagementResponse{
		Summary:  carpoolManagementSummary(allRows),
		Items:    rows,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *CarpoolService) adminSessionRows(ctx context.Context, sessions []*dbent.CarpoolSession) ([]CarpoolAdminSessionRow, error) {
	rows := make([]CarpoolAdminSessionRow, 0, len(sessions))
	for _, session := range sessions {
		row := CarpoolAdminSessionRow{Session: session, Participants: []CarpoolAdminParticipantRow{}}
		participants := session.Edges.Participants
		for _, participant := range participants {
			usage, err := s.carpoolParticipantUsage(ctx, session, participant)
			if err != nil {
				return nil, err
			}
			participantRow := CarpoolAdminParticipantRow{
				Participant: participant,
				Usage:       usage,
			}
			if participant.Edges.User != nil {
				participantRow.User = map[string]any{
					"id":       participant.Edges.User.ID,
					"email":    participant.Edges.User.Email,
					"username": participant.Edges.User.Username,
				}
			}
			row.Participants = append(row.Participants, participantRow)
			row.Usage.RequestCount += usage.RequestCount
			row.Usage.TotalTokens += usage.TotalTokens
			row.Usage.TotalCost += usage.TotalCost
			row.Usage.TotalActualCost += usage.TotalActualCost
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *CarpoolService) carpoolParticipantUsage(ctx context.Context, session *dbent.CarpoolSession, participant *dbent.CarpoolParticipant) (CarpoolAdminUsageSummary, error) {
	start := carpoolUsageStartTime(session, participant)
	return s.carpoolUsageAggregate(ctx, []predicateBuilder{
		func(q *dbent.UsageLogQuery) *dbent.UsageLogQuery {
			return q.Where(usagelog.UserIDEQ(participant.UserID))
		},
		func(q *dbent.UsageLogQuery) *dbent.UsageLogQuery { return q.Where(usagelog.CreatedAtGTE(start)) },
	})
}

type predicateBuilder func(*dbent.UsageLogQuery) *dbent.UsageLogQuery

func (s *CarpoolService) carpoolParticipantUsageForGroup(ctx context.Context, session *dbent.CarpoolSession, participant *dbent.CarpoolParticipant, groupID int64, window time.Duration) (CarpoolAdminUsageSummary, error) {
	start := carpoolUsageStartTime(session, participant)
	if window > 0 {
		windowStart := time.Now().Add(-window)
		if windowStart.After(start) {
			start = windowStart
		}
	}
	builders := []predicateBuilder{
		func(q *dbent.UsageLogQuery) *dbent.UsageLogQuery {
			return q.Where(usagelog.UserIDEQ(participant.UserID))
		},
		func(q *dbent.UsageLogQuery) *dbent.UsageLogQuery { return q.Where(usagelog.CreatedAtGTE(start)) },
	}
	if groupID > 0 {
		builders = append(builders, func(q *dbent.UsageLogQuery) *dbent.UsageLogQuery { return q.Where(usagelog.GroupIDEQ(groupID)) })
	}
	return s.carpoolUsageAggregate(ctx, builders)
}

func (s *CarpoolService) carpoolAccountUsage(ctx context.Context, accountID int64, start time.Time) (CarpoolAdminUsageSummary, error) {
	return s.carpoolUsageAggregate(ctx, []predicateBuilder{
		func(q *dbent.UsageLogQuery) *dbent.UsageLogQuery { return q.Where(usagelog.AccountIDEQ(accountID)) },
		func(q *dbent.UsageLogQuery) *dbent.UsageLogQuery { return q.Where(usagelog.CreatedAtGTE(start)) },
	})
}

func (s *CarpoolService) carpoolUsageAggregate(ctx context.Context, builders []predicateBuilder) (CarpoolAdminUsageSummary, error) {
	type usageAggregate struct {
		RequestCount    int     `json:"request_count"`
		TotalCost       float64 `json:"total_cost"`
		TotalActualCost float64 `json:"total_actual_cost"`
	}
	var out []usageAggregate
	q := s.entClient.UsageLog.Query()
	for _, builder := range builders {
		q = builder(q)
	}
	err := q.
		Aggregate(
			dbent.As(dbent.Count(), "request_count"),
			dbent.As(dbent.Sum(usagelog.FieldTotalCost), "total_cost"),
			dbent.As(dbent.Sum(usagelog.FieldActualCost), "total_actual_cost"),
		).
		Scan(ctx, &out)
	if err != nil {
		return CarpoolAdminUsageSummary{}, err
	}
	if len(out) == 0 {
		return CarpoolAdminUsageSummary{}, nil
	}
	var tokens []struct {
		InputTokens         int64 `json:"input_tokens"`
		OutputTokens        int64 `json:"output_tokens"`
		CacheCreationTokens int64 `json:"cache_creation_tokens"`
		CacheReadTokens     int64 `json:"cache_read_tokens"`
	}
	tokenQ := s.entClient.UsageLog.Query()
	for _, builder := range builders {
		tokenQ = builder(tokenQ)
	}
	if err := tokenQ.
		Aggregate(
			dbent.As(dbent.Sum(usagelog.FieldInputTokens), "input_tokens"),
			dbent.As(dbent.Sum(usagelog.FieldOutputTokens), "output_tokens"),
			dbent.As(dbent.Sum(usagelog.FieldCacheCreationTokens), "cache_creation_tokens"),
			dbent.As(dbent.Sum(usagelog.FieldCacheReadTokens), "cache_read_tokens"),
		).
		Scan(ctx, &tokens); err != nil {
		return CarpoolAdminUsageSummary{}, err
	}
	var totalTokens int64
	if len(tokens) > 0 {
		totalTokens = tokens[0].InputTokens + tokens[0].OutputTokens + tokens[0].CacheCreationTokens + tokens[0].CacheReadTokens
	}
	return CarpoolAdminUsageSummary{
		RequestCount:    out[0].RequestCount,
		TotalTokens:     totalTokens,
		TotalCost:       out[0].TotalCost,
		TotalActualCost: out[0].TotalActualCost,
	}, nil
}

func (s *CarpoolService) carpoolAccountWindows(ctx context.Context, groupID int64) ([]CarpoolAccountWindowUsage, error) {
	accounts, err := s.entClient.Account.Query().
		Where(
			account.DeletedAtIsNil(),
			account.StatusEQ(StatusActive),
			account.SchedulableEQ(true),
			account.HasGroupsWith(group.IDEQ(groupID)),
		).
		Order(dbent.Asc(account.FieldPriority), dbent.Asc(account.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]CarpoolAccountWindowUsage, 0, len(accounts)*2)
	for _, acc := range accounts {
		five := accountWindowFromExtra(acc, "5h", now)
		five.Usage, err = s.carpoolAccountUsage(ctx, acc.ID, now.Add(-5*time.Hour))
		if err != nil {
			return nil, err
		}
		out = append(out, five)
		seven := accountWindowFromExtra(acc, "7d", now)
		seven.Usage, err = s.carpoolAccountUsage(ctx, acc.ID, now.Add(-7*24*time.Hour))
		if err != nil {
			return nil, err
		}
		out = append(out, seven)
	}
	return out, nil
}

func carpoolUsageStartTime(session *dbent.CarpoolSession, participant *dbent.CarpoolParticipant) time.Time {
	if session != nil && session.ServiceStartedAt != nil && !session.ServiceStartedAt.IsZero() {
		return *session.ServiceStartedAt
	}
	if participant.PaidAt != nil && !participant.PaidAt.IsZero() {
		return *participant.PaidAt
	}
	if participant.JoinedAt != nil && !participant.JoinedAt.IsZero() {
		return *participant.JoinedAt
	}
	return participant.CreatedAt
}

func accountWindowFromExtra(acc *dbent.Account, window string, now time.Time) CarpoolAccountWindowUsage {
	out := CarpoolAccountWindowUsage{
		Window:           window,
		RemainingSeconds: 0,
	}
	if acc == nil || acc.Extra == nil {
		return out
	}
	out.AccountID = acc.ID
	out.AccountName = acc.Name
	prefix := "codex_" + window
	out.Utilization = parseExtraFloat64(acc.Extra[prefix+"_used_percent"])
	resetAt := parseExtraTime(acc.Extra[prefix+"_reset_at"])
	if resetAt.IsZero() {
		if seconds := parseExtraFloat64(acc.Extra[prefix+"_reset_after_seconds"]); seconds > 0 {
			resetAt = now.Add(time.Duration(seconds) * time.Second)
		}
	}
	if !resetAt.IsZero() {
		out.ResetsAt = &resetAt
		if resetAt.After(now) {
			out.RemainingSeconds = int(resetAt.Sub(now).Seconds())
		}
	}
	if out.Utilization < 0 {
		out.Utilization = 0
	}
	return out
}

func addCarpoolUsage(a, b CarpoolAdminUsageSummary) CarpoolAdminUsageSummary {
	a.RequestCount += b.RequestCount
	a.TotalTokens += b.TotalTokens
	a.TotalCost += b.TotalCost
	a.TotalActualCost += b.TotalActualCost
	return a
}

func numberFromMap(input map[string]any, key string) int64 {
	if input == nil {
		return 0
	}
	return int64(parseExtraFloat64(input[key]))
}

func (s *CarpoolService) userAvatarURL(ctx context.Context, userID int64) (string, error) {
	rows, err := s.entClient.QueryContext(ctx, `SELECT url FROM user_avatars WHERE user_id = $1`, userID)
	if err != nil {
		return "", err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return "", rows.Err()
	}
	var url string
	if err := rows.Scan(&url); err != nil {
		return "", err
	}
	return strings.TrimSpace(url), rows.Err()
}

func maskedCarpoolMemberName(u *dbent.User) string {
	if u == nil {
		return "拼车成员"
	}
	if username := strings.TrimSpace(u.Username); username != "" {
		return username
	}
	return maskLoginAccount(u.Email)
}

func carpoolMemberInitial(u *dbent.User) string {
	name := maskedCarpoolMemberName(u)
	for _, r := range name {
		if r != '*' && r != '@' && r != '.' && r != ' ' {
			return strings.ToUpper(string(r))
		}
	}
	return "U"
}

func maskLoginAccount(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "拼车成员"
	}
	if at := strings.Index(value, "@"); at > 0 {
		local := value[:at]
		domain := value[at:]
		runes := []rune(local)
		if len(runes) <= 2 {
			return string(runes[:1]) + "***" + domain
		}
		return string(runes[:2]) + "***" + domain
	}
	runes := []rune(value)
	if len(runes) <= 4 {
		return string(runes[:1]) + "***"
	}
	return string(runes[:3]) + "***" + string(runes[len(runes)-2:])
}

func carpoolManagementStatuses() []string {
	return []string{CarpoolSessionFull, CarpoolSessionProvisioning, CarpoolSessionActive, CarpoolSessionEnded}
}

func carpoolManagementSummary(rows []CarpoolAdminSessionRow) CarpoolAdminManagementSummary {
	summary := CarpoolAdminManagementSummary{
		ByStatus:  map[string]int64{},
		BySegment: []map[string]any{},
	}
	segmentMap := map[string]map[string]any{}
	for _, row := range rows {
		if row.Session == nil {
			continue
		}
		summary.CompletedSessions++
		summary.ByStatus[row.Session.Status]++
		summary.TotalTokens += row.Usage.TotalTokens
		summary.TotalActualCost += row.Usage.TotalActualCost
		for _, participant := range row.Participants {
			if participant.Participant == nil {
				continue
			}
			if participant.Participant.Status == CarpoolParticipantPaid || participant.Participant.Status == CarpoolParticipantActive {
				summary.PaidMembers++
				summary.TotalPaidAmount += participant.Participant.Amount
			}
			if participant.Participant.Status == CarpoolParticipantActive {
				summary.ActiveMembers++
			}
		}
		vt := row.Session.Edges.VehicleType
		segment := "未分组"
		if vt != nil {
			segment = strings.TrimSpace(strings.Join([]string{productDisplayName(vt.Product), tierDisplayName(vt.PlanTier), strings.ToUpper(vt.Multiplier)}, " "))
		}
		if _, ok := segmentMap[segment]; !ok {
			segmentMap[segment] = map[string]any{"label": segment, "sessions": int64(0), "paid_members": int64(0), "amount": float64(0)}
		}
		segmentMap[segment]["sessions"] = segmentMap[segment]["sessions"].(int64) + 1
		segmentMap[segment]["paid_members"] = segmentMap[segment]["paid_members"].(int64) + int64(row.Session.PaidCount)
		segmentMap[segment]["amount"] = segmentMap[segment]["amount"].(float64) + sumSessionParticipantAmount(row.Participants)
	}
	for _, item := range segmentMap {
		summary.BySegment = append(summary.BySegment, item)
	}
	return summary
}

func sumSessionParticipantAmount(participants []CarpoolAdminParticipantRow) float64 {
	var total float64
	for _, participant := range participants {
		if participant.Participant != nil && (participant.Participant.Status == CarpoolParticipantPaid || participant.Participant.Status == CarpoolParticipantActive) {
			total += participant.Participant.Amount
		}
	}
	return total
}

func (s *CarpoolService) AdminOverview(ctx context.Context) (map[string]any, error) {
	statuses := []string{CarpoolSessionRecruiting, CarpoolSessionFull, CarpoolSessionProvisioning, CarpoolSessionActive}
	counts := map[string]int{}
	for _, st := range statuses {
		c, err := s.entClient.CarpoolSession.Query().Where(carpoolsession.StatusEQ(st)).Count(ctx)
		if err != nil {
			return nil, err
		}
		counts[st] = c
	}
	refunds, err := s.entClient.CarpoolParticipant.Query().Where(carpoolparticipant.StatusEQ(CarpoolParticipantRefundPending)).Count(ctx)
	if err != nil {
		return nil, err
	}
	completedByVehicleType, err := s.completedSessionCountsByVehicleType(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"sessions": counts, "refund_pending": refunds, "completed_by_vehicle_type": completedByVehicleType}, nil
}

func (s *CarpoolService) AdminCreateVehicleType(ctx context.Context, input CarpoolVehicleTypeInput) (*dbent.CarpoolVehicleType, error) {
	input = normalizeVehicleInput(input)
	return s.entClient.CarpoolVehicleType.Create().
		SetProduct(input.Product).
		SetPlanTier(input.PlanTier).
		SetMultiplier(input.Multiplier).
		SetName(input.Name).
		SetSeatCount(input.SeatCount).
		SetTotalPrice(input.TotalPrice).
		SetUnitPrice(input.UnitPrice).
		SetServiceDays(input.ServiceDays).
		SetRefundWaitHours(input.RefundWaitHours).
		SetCompletedBaseCount(input.CompletedBaseCount).
		SetEnabled(input.Enabled).
		SetSupportRevenuePool(input.SupportRevenuePool).
		SetRequireStaticIP(input.RequireStaticIP).
		SetWaitDurationOptions(input.WaitDurationOptions).
		SetRefundMethods(input.RefundMethods).
		SetDescription(input.Description).
		SetSortOrder(input.SortOrder).
		Save(ctx)
}

func (s *CarpoolService) AdminUpdateVehicleType(ctx context.Context, id int64, input CarpoolVehicleTypeInput) (*dbent.CarpoolVehicleType, error) {
	input = normalizeVehicleInput(input)
	return s.entClient.CarpoolVehicleType.UpdateOneID(id).
		SetProduct(input.Product).
		SetPlanTier(input.PlanTier).
		SetMultiplier(input.Multiplier).
		SetName(input.Name).
		SetSeatCount(input.SeatCount).
		SetTotalPrice(input.TotalPrice).
		SetUnitPrice(input.UnitPrice).
		SetServiceDays(input.ServiceDays).
		SetRefundWaitHours(input.RefundWaitHours).
		SetCompletedBaseCount(input.CompletedBaseCount).
		SetEnabled(input.Enabled).
		SetSupportRevenuePool(input.SupportRevenuePool).
		SetRequireStaticIP(input.RequireStaticIP).
		SetWaitDurationOptions(input.WaitDurationOptions).
		SetRefundMethods(input.RefundMethods).
		SetDescription(input.Description).
		SetSortOrder(input.SortOrder).
		Save(ctx)
}

func (s *CarpoolService) AdminDeleteVehicleType(ctx context.Context, id int64) error {
	return s.entClient.CarpoolVehicleType.DeleteOneID(id).Exec(ctx)
}

func (s *CarpoolService) AdminListNotices(ctx context.Context) ([]*dbent.CarpoolNoticeVersion, error) {
	return s.entClient.CarpoolNoticeVersion.Query().Order(dbent.Desc(carpoolnoticeversion.FieldVersion), dbent.Desc(carpoolnoticeversion.FieldID)).All(ctx)
}

func (s *CarpoolService) AdminCreateNotice(ctx context.Context, input CarpoolNoticeInput) (*dbent.CarpoolNoticeVersion, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "拼车用户须知"
	}
	maxVersion, _ := s.entClient.CarpoolNoticeVersion.Query().Aggregate(dbent.Max(carpoolnoticeversion.FieldVersion)).Int(ctx)
	if input.Active {
		if err := s.entClient.CarpoolNoticeVersion.Update().SetActive(false).Exec(ctx); err != nil {
			return nil, err
		}
	}
	create := s.entClient.CarpoolNoticeVersion.Create().
		SetTitle(title).
		SetContentMd(strings.TrimSpace(input.ContentMD)).
		SetVersion(maxVersion + 1).
		SetActive(input.Active)
	if input.Active {
		create.SetPublishedAt(time.Now())
	}
	return create.Save(ctx)
}

func (s *CarpoolService) AdminProvisionSession(ctx context.Context, id int64, input CarpoolSessionProvisionInput) (*dbent.CarpoolSession, error) {
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = CarpoolSessionProvisioning
	}
	up := s.entClient.CarpoolSession.UpdateOneID(id).
		SetStatus(status).
		SetAccountInfo(nonNilMap(input.AccountInfo)).
		SetProxyInfo(nonNilMap(input.ProxyInfo)).
		SetCommunication(nonNilMap(input.Communication)).
		SetAdminNotes(strings.TrimSpace(input.AdminNotes))
	now := time.Now()
	if status == CarpoolSessionActive {
		up.SetProvisionedAt(now).SetServiceStartedAt(now)
	}
	session, err := up.Save(ctx)
	if err != nil {
		return nil, err
	}
	if status == CarpoolSessionActive {
		s.notifyCarpoolActiveUsers(ctx, session.ID)
	}
	return session, nil
}

func (s *CarpoolService) AdminCreateVoucher(ctx context.Context, sessionID, uploadedBy int64, input CarpoolVoucherInput) (*dbent.CarpoolVoucher, error) {
	if sessionID <= 0 {
		return nil, infraerrors.BadRequest("CARPOOL_SESSION_REQUIRED", "carpool session is required")
	}
	fileURL := strings.TrimSpace(input.FileURL)
	if fileURL == "" {
		return nil, infraerrors.BadRequest("CARPOOL_VOUCHER_FILE_REQUIRED", "voucher image is required")
	}
	fileName := strings.TrimSpace(input.FileName)
	if fileName == "" {
		fileName = "拼车交付凭证"
	}
	description := strings.TrimSpace(input.Description)
	_, err := s.entClient.CarpoolSession.Get(ctx, sessionID)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("CARPOOL_SESSION_NOT_FOUND", "carpool session not found")
		}
		return nil, err
	}
	create := s.entClient.CarpoolVoucher.Create().
		SetSessionID(sessionID).
		SetFileURL(fileURL).
		SetFileName(fileName).
		SetUploadedBy(uploadedBy)
	if description != "" {
		create.SetDescription(description)
	}
	return create.Save(ctx)
}

func (s *CarpoolService) AdminDeleteVoucher(ctx context.Context, voucherID int64) error {
	if voucherID <= 0 {
		return infraerrors.BadRequest("CARPOOL_VOUCHER_REQUIRED", "carpool voucher is required")
	}
	return s.entClient.CarpoolVoucher.DeleteOneID(voucherID).Exec(ctx)
}

func (s *CarpoolService) AdminListVouchers(ctx context.Context, sessionID int64) ([]*dbent.CarpoolVoucher, error) {
	if sessionID <= 0 {
		return nil, infraerrors.BadRequest("CARPOOL_SESSION_REQUIRED", "carpool session is required")
	}
	return s.entClient.CarpoolVoucher.Query().
		Where(carpoolvoucher.SessionIDEQ(sessionID)).
		Order(dbent.Desc(carpoolvoucher.FieldCreatedAt)).
		All(ctx)
}

func (s *CarpoolService) notifyCarpoolFullAdmins(ctx context.Context, sessionID int64) {
	if s == nil || s.settingService == nil || sessionID <= 0 {
		return
	}
	settings, err := s.settingService.GetAllSettings(ctx)
	if err != nil {
		slog.Warn("carpool admin full sms skipped: load settings failed", "session_id", sessionID, "error", err)
		return
	}
	if settings == nil || !settings.CarpoolAdminFullSMSNotifyEnabled {
		return
	}
	templateCode := strings.TrimSpace(settings.CarpoolAdminFullSMSTemplateCode)
	phones := parseSMSPhoneList(settings.CarpoolAdminFullSMSPhones)
	if templateCode == "" || len(phones) == 0 {
		return
	}
	session, err := s.entClient.CarpoolSession.Query().
		Where(carpoolsession.IDEQ(sessionID)).
		WithVehicleType().
		Only(ctx)
	if err != nil {
		slog.Warn("carpool admin full sms skipped: load session failed", "session_id", sessionID, "error", err)
		return
	}
	if hasSMSNotifyMark(session.AccountInfo, "admin_full_sms_sent_at") {
		return
	}
	cfg, err := s.settingService.GetAliyunSMSConfig(ctx)
	if err != nil {
		slog.Warn("carpool admin full sms skipped: aliyun sms not configured", "session_id", sessionID, "error", err)
		return
	}
	params := carpoolSMSTemplateParams(session, nil)
	sent := 0
	for _, phone := range phones {
		if err := sendAliyunTemplateSMS(ctx, cfg, AliyunTemplateSMSInput{
			Phone:        phone,
			TemplateCode: templateCode,
			Params:       params,
			OutID:        fmt.Sprintf("carpool-full-%d-%s", session.ID, randomPhoneVerificationHex(6)),
		}); err != nil {
			slog.Warn("carpool admin full sms failed", "session_id", session.ID, "phone", phone, "error", err)
			continue
		}
		sent++
	}
	if sent > 0 {
		s.markCarpoolSMSNotified(ctx, session.ID, "admin_full_sms_sent_at")
	}
}

func (s *CarpoolService) notifyCarpoolActiveUsers(ctx context.Context, sessionID int64) {
	if s == nil || s.settingService == nil || sessionID <= 0 {
		return
	}
	settings, err := s.settingService.GetAllSettings(ctx)
	if err != nil {
		slog.Warn("carpool active user sms skipped: load settings failed", "session_id", sessionID, "error", err)
		return
	}
	if settings == nil || !settings.CarpoolUserActiveSMSNotifyEnabled {
		return
	}
	templateCode := strings.TrimSpace(settings.CarpoolUserActiveSMSTemplateCode)
	if templateCode == "" {
		return
	}
	session, err := s.entClient.CarpoolSession.Query().
		Where(carpoolsession.IDEQ(sessionID)).
		WithVehicleType().
		Only(ctx)
	if err != nil {
		slog.Warn("carpool active user sms skipped: load session failed", "session_id", sessionID, "error", err)
		return
	}
	if hasSMSNotifyMark(session.AccountInfo, "user_active_sms_sent_at") {
		return
	}
	cfg, err := s.settingService.GetAliyunSMSConfig(ctx)
	if err != nil {
		slog.Warn("carpool active user sms skipped: aliyun sms not configured", "session_id", session.ID, "error", err)
		return
	}
	participants, err := s.entClient.CarpoolParticipant.Query().
		Where(
			carpoolparticipant.SessionIDEQ(session.ID),
			carpoolparticipant.StatusIn(CarpoolParticipantPaid, CarpoolParticipantActive),
		).
		WithUser().
		All(ctx)
	if err != nil {
		slog.Warn("carpool active user sms skipped: load participants failed", "session_id", session.ID, "error", err)
		return
	}
	sent := 0
	for _, p := range participants {
		if p == nil || p.Edges.User == nil {
			continue
		}
		phone, err := NormalizeMainlandPhone(p.Edges.User.Phone)
		if err != nil {
			continue
		}
		if err := sendAliyunTemplateSMS(ctx, cfg, AliyunTemplateSMSInput{
			Phone:        phone,
			TemplateCode: templateCode,
			Params:       carpoolSMSTemplateParams(session, p.Edges.User),
			OutID:        fmt.Sprintf("carpool-active-%d-%d-%s", session.ID, p.UserID, randomPhoneVerificationHex(6)),
		}); err != nil {
			slog.Warn("carpool active user sms failed", "session_id", session.ID, "user_id", p.UserID, "error", err)
			continue
		}
		sent++
	}
	if sent > 0 {
		s.markCarpoolSMSNotified(ctx, session.ID, "user_active_sms_sent_at")
	}
}

func (s *CarpoolService) markCarpoolSMSNotified(ctx context.Context, sessionID int64, key string) {
	session, err := s.entClient.CarpoolSession.Get(ctx, sessionID)
	if err != nil {
		slog.Warn("carpool sms mark skipped: load session failed", "session_id", sessionID, "key", key, "error", err)
		return
	}
	info := nonNilMap(session.AccountInfo)
	info[key] = time.Now().Format(time.RFC3339)
	if _, err := s.entClient.CarpoolSession.UpdateOneID(sessionID).SetAccountInfo(info).Save(ctx); err != nil {
		slog.Warn("carpool sms mark failed", "session_id", sessionID, "key", key, "error", err)
	}
}

func hasSMSNotifyMark(info map[string]any, key string) bool {
	if info == nil {
		return false
	}
	value, ok := info[key]
	return ok && strings.TrimSpace(fmt.Sprint(value)) != ""
}

func parseSMSPhoneList(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	seen := map[string]struct{}{}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		phone, err := NormalizeMainlandPhone(field)
		if err != nil {
			continue
		}
		if _, ok := seen[phone]; ok {
			continue
		}
		seen[phone] = struct{}{}
		out = append(out, phone)
	}
	return out
}

func carpoolSMSTemplateParams(session *dbent.CarpoolSession, u *dbent.User) map[string]string {
	params := map[string]string{
		"session_no":   "",
		"vehicle_name": "",
		"seat_count":   "",
		"paid_count":   "",
		"filled_at":    "",
		"group_name":   "",
		"service_days": "",
		"user_name":    "",
	}
	if session == nil {
		return params
	}
	params["session_no"] = session.SessionNo
	params["seat_count"] = strconv.Itoa(session.SeatCount)
	params["paid_count"] = strconv.Itoa(session.PaidCount)
	if session.FilledAt != nil {
		params["filled_at"] = session.FilledAt.Format("2006-01-02 15:04")
	}
	if vt := session.Edges.VehicleType; vt != nil {
		params["vehicle_name"] = vt.Name
		params["service_days"] = strconv.Itoa(vt.ServiceDays)
	}
	if groupName := strings.TrimSpace(fmt.Sprint(session.AccountInfo["subscription_group_name"])); groupName != "" {
		params["group_name"] = groupName
	}
	if u != nil {
		params["user_name"] = maskedCarpoolMemberName(u)
	}
	return params
}

func (s *CarpoolService) completedSessionCountsByVehicleType(ctx context.Context) (map[int64]int, error) {
	rows, err := s.entClient.CarpoolSession.Query().
		Where(carpoolsession.StatusIn(CarpoolSessionFull, CarpoolSessionProvisioning, CarpoolSessionActive, CarpoolSessionEnded)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	counts := make(map[int64]int, len(rows))
	for _, row := range rows {
		counts[row.VehicleTypeID]++
	}
	return counts, nil
}

func (s *CarpoolService) recalculateSessionPaidCount(ctx context.Context, sessionID int64) error {
	if sessionID <= 0 {
		return nil
	}
	count, err := s.entClient.CarpoolParticipant.Query().
		Where(carpoolparticipant.SessionIDEQ(sessionID), carpoolparticipant.StatusIn(CarpoolParticipantPaid, CarpoolParticipantActive)).
		Count(ctx)
	if err != nil {
		return err
	}
	_, err = s.entClient.CarpoolSession.UpdateOneID(sessionID).SetPaidCount(count).Save(ctx)
	return err
}

func (s *CarpoolService) getOrCreateRecruitingSession(ctx context.Context, vt *dbent.CarpoolVehicleType) (*dbent.CarpoolSession, error) {
	session, err := s.entClient.CarpoolSession.Query().
		Where(carpoolsession.VehicleTypeIDEQ(vt.ID), carpoolsession.StatusEQ(CarpoolSessionRecruiting)).
		Order(dbent.Asc(carpoolsession.FieldID)).
		First(ctx)
	if err == nil {
		return session, nil
	}
	if !dbent.IsNotFound(err) {
		return nil, err
	}
	now := time.Now()
	session, err = s.entClient.CarpoolSession.Create().
		SetVehicleTypeID(vt.ID).
		SetSessionNo(fmt.Sprintf("CP-%d-%d", vt.ID, now.Unix())).
		SetStatus(CarpoolSessionRecruiting).
		SetSeatCount(vt.SeatCount).
		SetPaidCount(0).
		SetStartedAt(now).
		Save(ctx)
	if err != nil {
		if isUniqueOrRace(err) {
			return s.entClient.CarpoolSession.Query().
				Where(carpoolsession.VehicleTypeIDEQ(vt.ID), carpoolsession.StatusEQ(CarpoolSessionRecruiting)).
				Order(dbent.Asc(carpoolsession.FieldID)).
				First(ctx)
		}
		return nil, err
	}
	return session, nil
}

func normalizeVehicleInput(input CarpoolVehicleTypeInput) CarpoolVehicleTypeInput {
	input.Product = normalizeCarpoolCode(input.Product, "openai")
	input.PlanTier = normalizeCarpoolCode(input.PlanTier, "pro")
	input.Multiplier = normalizeCarpoolCode(input.Multiplier, "20x")
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		input.Name = fmt.Sprintf("%d人车", input.SeatCount)
	}
	if input.SeatCount <= 0 {
		input.SeatCount = 2
	}
	if input.ServiceDays <= 0 {
		input.ServiceDays = 30
	}
	if input.RefundWaitHours <= 0 {
		input.RefundWaitHours = defaultCarpoolRefundWaitHours
	}
	if input.CompletedBaseCount < 0 {
		input.CompletedBaseCount = 0
	}
	if input.UnitPrice <= 0 && input.TotalPrice > 0 {
		input.UnitPrice = math.Round((input.TotalPrice/float64(input.SeatCount))*100) / 100
	}
	input.WaitDurationOptions = []int{input.RefundWaitHours}
	if len(input.RefundMethods) == 0 {
		input.RefundMethods = []string{CarpoolRefundBalance, CarpoolRefundGateway}
	}
	return input
}

func normalizeCarpoolCode(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func productDisplayName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "openai":
		return "OpenAI"
	case "claudecode", "claude_code":
		return "ClaudeCode"
	default:
		if strings.TrimSpace(value) == "" {
			return "OpenAI"
		}
		return strings.TrimSpace(value)
	}
}

func tierDisplayName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pro":
		return "Pro"
	case "plus":
		return "Plus"
	default:
		if strings.TrimSpace(value) == "" {
			return "Pro"
		}
		return strings.TrimSpace(value)
	}
}

func normalizeRefundWaitHours(waitHours int) int {
	if waitHours <= 0 {
		return defaultCarpoolRefundWaitHours
	}
	return waitHours
}

func normalizeRefundMethod(method string, allowed []string) string {
	method = strings.TrimSpace(method)
	if method == "" {
		method = CarpoolRefundBalance
	}
	if len(allowed) == 0 {
		return method
	}
	for _, item := range allowed {
		if item == method {
			return method
		}
	}
	return allowed[0]
}

func nonNilMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

func isUniqueOrRace(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || err == sql.ErrNoRows
}

func defaultCarpoolNotice() string {
	return `# 拼车用户须知

请在支付前完整阅读并确认以下规则：

1. 拼车为多人共同等待成团，人满后由管理员采购和交付。
2. 每种车会配置固定可退款时间；到达该时间后，您可在“我的拼车”中自行发起退款，未发起则继续等待成团。
3. 发车后的账号、代理、使用方式和沟通方式以管理员交付信息为准。
4. 中转投入计划属于第二阶段能力，必须经过车内成员正式投票确认后才会执行。
5. 请勿将车内敏感信息泄露给非本车成员。
`
}

func (s *CarpoolService) UserBriefsForSession(ctx context.Context, sessionID int64) (map[int64]map[string]any, error) {
	participants, err := s.entClient.CarpoolParticipant.Query().
		Where(carpoolparticipant.SessionIDEQ(sessionID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	userIDs := make([]int64, 0, len(participants))
	for _, p := range participants {
		userIDs = append(userIDs, p.UserID)
	}
	users, err := s.entClient.User.Query().Where(user.IDIn(userIDs...)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64]map[string]any{}
	for _, u := range users {
		out[u.ID] = map[string]any{"id": u.ID, "email": carpoolMaskEmail(u.Email), "username": u.Username}
	}
	return out, nil
}

func carpoolMaskEmail(email string) string {
	email = strings.TrimSpace(email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	name := parts[0]
	if len(name) <= 2 {
		return name[:1] + "***@" + parts[1]
	}
	return name[:2] + "***@" + parts[1]
}
