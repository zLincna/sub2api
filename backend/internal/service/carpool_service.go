package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"math/big"
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
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
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

	CarpoolRevenueStatusDisabled       = "disabled"
	CarpoolRevenueStatusActive         = "active"
	CarpoolRevenueStatusPausedByUser   = "paused_by_user"
	CarpoolRevenueStatusPausedByAdmin  = "paused_by_admin"
	CarpoolRevenueStatusExpired        = "expired"
	CarpoolRevenueStatusRiskHold       = "risk_hold"
	CarpoolRevenueRecordStatusPending  = "pending"
	CarpoolRevenueRecordStatusSettled  = "settled"
	CarpoolRevenueRecordStatusFrozen   = "frozen"
	CarpoolRevenueRecordStatusReversed = "reversed"

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

type CarpoolRevenueConfig struct {
	Enabled                 bool      `json:"enabled"`
	UserShareRatio          float64   `json:"user_share_ratio"`
	PlatformShareRatio      float64   `json:"platform_share_ratio"`
	MinWithdrawAmount       float64   `json:"min_withdraw_amount"`
	WithdrawCooldownMinutes int       `json:"withdraw_cooldown_minutes"`
	SettlementCycle         string    `json:"settlement_cycle"`
	FreezeMinutes           int       `json:"freeze_minutes"`
	AllowUserWithdraw       bool      `json:"allow_user_withdraw"`
	PriorityPolicy          string    `json:"priority_policy"`
	RiskNotes               string    `json:"risk_notes"`
	GatewayDispatchEnabled  bool      `json:"gateway_dispatch_enabled"`
	GatewayShadowMode       bool      `json:"gateway_shadow_mode"`
	GatewayTrafficPercent   float64   `json:"gateway_traffic_percent"`
	GatewayAllowedGroupIDs  string    `json:"gateway_allowed_group_ids"`
	GatewayAllowedModels    string    `json:"gateway_allowed_models"`
	GatewayMinRemainRatio   float64   `json:"gateway_min_remain_ratio"`
	GatewayMaxDailyQuota    float64   `json:"gateway_max_daily_quota"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type CarpoolRevenueConfigInput struct {
	Enabled                 bool    `json:"enabled"`
	UserShareRatio          float64 `json:"user_share_ratio"`
	PlatformShareRatio      float64 `json:"platform_share_ratio"`
	MinWithdrawAmount       float64 `json:"min_withdraw_amount"`
	WithdrawCooldownMinutes int     `json:"withdraw_cooldown_minutes"`
	SettlementCycle         string  `json:"settlement_cycle"`
	FreezeMinutes           int     `json:"freeze_minutes"`
	AllowUserWithdraw       bool    `json:"allow_user_withdraw"`
	PriorityPolicy          string  `json:"priority_policy"`
	RiskNotes               string  `json:"risk_notes"`
	GatewayDispatchEnabled  bool    `json:"gateway_dispatch_enabled"`
	GatewayShadowMode       bool    `json:"gateway_shadow_mode"`
	GatewayTrafficPercent   float64 `json:"gateway_traffic_percent"`
	GatewayAllowedGroupIDs  string  `json:"gateway_allowed_group_ids"`
	GatewayAllowedModels    string  `json:"gateway_allowed_models"`
	GatewayMinRemainRatio   float64 `json:"gateway_min_remain_ratio"`
	GatewayMaxDailyQuota    float64 `json:"gateway_max_daily_quota"`
}

type CarpoolRevenueContribution struct {
	ID                  int64      `json:"id"`
	ParticipantID       int64      `json:"participant_id"`
	UserID              int64      `json:"user_id"`
	SessionID           int64      `json:"session_id"`
	VehicleTypeID       int64      `json:"vehicle_type_id"`
	SubscriptionID      *int64     `json:"subscription_id,omitempty"`
	SubscriptionGroupID int64      `json:"subscription_group_id"`
	Enabled             bool       `json:"enabled"`
	EnabledAt           *time.Time `json:"enabled_at,omitempty"`
	DisabledAt          *time.Time `json:"disabled_at,omitempty"`
	ShareRatio          float64    `json:"share_ratio"`
	Status              string     `json:"status"`
	PausedReason        string     `json:"paused_reason"`
	LastSettledAt       *time.Time `json:"last_settled_at,omitempty"`
	Notes               string     `json:"notes"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type CarpoolRevenueSummary struct {
	TotalRevenue        float64 `json:"total_revenue"`
	PendingRevenue      float64 `json:"pending_revenue"`
	FrozenRevenue       float64 `json:"frozen_revenue"`
	AvailableRevenue    float64 `json:"available_revenue"`
	WithdrawnRevenue    float64 `json:"withdrawn_revenue"`
	QuotaCost           float64 `json:"quota_cost"`
	RequestCount        int64   `json:"request_count"`
	PlatformShareAmount float64 `json:"platform_share_amount"`
}

type CarpoolRevenueRecord struct {
	ID                  int64      `json:"id"`
	ContributionID      int64      `json:"contribution_id"`
	ParticipantID       int64      `json:"participant_id"`
	SessionID           int64      `json:"session_id"`
	UserID              int64      `json:"user_id"`
	SubscriptionGroupID int64      `json:"subscription_group_id"`
	SubscriptionID      *int64     `json:"subscription_id,omitempty"`
	APIKeyID            *int64     `json:"api_key_id,omitempty"`
	UsageLogID          *int64     `json:"usage_log_id,omitempty"`
	RequestUserID       *int64     `json:"request_user_id,omitempty"`
	RequestAPIKeyID     *int64     `json:"request_api_key_id,omitempty"`
	RequestAccountID    *int64     `json:"request_account_id,omitempty"`
	RequestGroupID      *int64     `json:"request_group_id,omitempty"`
	DispatchMode        string     `json:"dispatch_mode"`
	DecisionReason      string     `json:"decision_reason"`
	RequestID           string     `json:"request_id"`
	RequestCount        int        `json:"request_count"`
	QuotaCost           float64    `json:"quota_cost"`
	GrossRevenue        float64    `json:"gross_revenue"`
	UpstreamCost        float64    `json:"upstream_cost"`
	NetRevenue          float64    `json:"net_revenue"`
	UserShareAmount     float64    `json:"user_share_amount"`
	PlatformShareAmount float64    `json:"platform_share_amount"`
	SettlementPeriod    string     `json:"settlement_period"`
	Status              string     `json:"status"`
	OccurredAt          time.Time  `json:"occurred_at"`
	SettledAt           *time.Time `json:"settled_at,omitempty"`
	Notes               string     `json:"notes"`
	CreatedAt           time.Time  `json:"created_at"`
}

type CarpoolRevenueWithdrawal struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	ParticipantID   *int64     `json:"participant_id,omitempty"`
	SessionID       *int64     `json:"session_id,omitempty"`
	Amount          float64    `json:"amount"`
	AvailableBefore float64    `json:"available_before"`
	AvailableAfter  float64    `json:"available_after"`
	BalanceBefore   *float64   `json:"balance_before,omitempty"`
	BalanceAfter    *float64   `json:"balance_after,omitempty"`
	Status          string     `json:"status"`
	RequestedAt     time.Time  `json:"requested_at"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	FailureReason   string     `json:"failure_reason"`
	CreatedAt       time.Time  `json:"created_at"`
}

type CarpoolRevenueDetail struct {
	AvailableReason string                      `json:"available_reason"`
	Contribution    *CarpoolRevenueContribution `json:"contribution,omitempty"`
	Summary         CarpoolRevenueSummary       `json:"summary"`
	Config          *CarpoolRevenueConfig       `json:"config,omitempty"`
	Records         []CarpoolRevenueRecord      `json:"records"`
	Withdrawals     []CarpoolRevenueWithdrawal  `json:"withdrawals"`
}

type CarpoolRevenueWithdrawInput struct {
	Amount float64 `json:"amount"`
}

type CarpoolRevenueContributionAdminRow struct {
	Contribution CarpoolRevenueContribution `json:"contribution"`
	Summary      CarpoolRevenueSummary      `json:"summary"`
	User         map[string]any             `json:"user,omitempty"`
	Session      map[string]any             `json:"session,omitempty"`
	VehicleType  map[string]any             `json:"vehicle_type,omitempty"`
}

type CarpoolRevenueAdminListResponse struct {
	Items    []CarpoolRevenueContributionAdminRow `json:"items"`
	Total    int                                  `json:"total"`
	Page     int                                  `json:"page"`
	PageSize int                                  `json:"page_size"`
}

type CarpoolRevenueRecordInput struct {
	ContributionID      int64   `json:"contribution_id"`
	GrossRevenue        float64 `json:"gross_revenue"`
	UpstreamCost        float64 `json:"upstream_cost"`
	NetRevenue          float64 `json:"net_revenue"`
	UserShareAmount     float64 `json:"user_share_amount"`
	PlatformShareAmount float64 `json:"platform_share_amount"`
	QuotaCost           float64 `json:"quota_cost"`
	RequestCount        int     `json:"request_count"`
	RequestID           string  `json:"request_id"`
	SettlementPeriod    string  `json:"settlement_period"`
	OccurredAt          string  `json:"occurred_at"`
	Notes               string  `json:"notes"`
}

type CarpoolRevenueGatewayDispatchInput struct {
	RequestUserID   int64
	RequestAPIKeyID int64
	OriginalGroupID *int64
	RequestedModel  string
	RequestPlatform string
	RequestKey      string
	ExcludedUserID  int64
}

type CarpoolRevenueGatewayDispatchDecision struct {
	Mode                string  `json:"mode"`
	Reason              string  `json:"reason"`
	ContributionID      int64   `json:"contribution_id"`
	ParticipantID       int64   `json:"participant_id"`
	SessionID           int64   `json:"session_id"`
	UserID              int64   `json:"user_id"`
	SubscriptionID      int64   `json:"subscription_id"`
	SubscriptionGroupID int64   `json:"subscription_group_id"`
	ShareRatio          float64 `json:"share_ratio"`
	OriginalGroupID     int64   `json:"original_group_id"`
	RequestUserID       int64   `json:"request_user_id"`
	RequestAPIKeyID     int64   `json:"request_api_key_id"`
	Routed              bool    `json:"routed"`
}

func (d *CarpoolRevenueGatewayDispatchDecision) RoutingGroupID(original *int64) *int64 {
	if d == nil || !d.Routed || d.SubscriptionGroupID <= 0 || d.Mode != "real" {
		return original
	}
	return &d.SubscriptionGroupID
}

func (d *CarpoolRevenueGatewayDispatchDecision) ShouldSettle() bool {
	return d != nil && d.Mode == "real" && d.Routed && d.ContributionID > 0 && d.SubscriptionID > 0
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
	subscriptionGroupID, accountPoolGroupID, err := s.carpoolAssignedGroupIDs(ctx, participant, session)
	if err != nil {
		return nil, err
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

func (s *CarpoolService) GetRevenueConfig(ctx context.Context) (*CarpoolRevenueConfig, error) {
	if _, err := s.entClient.ExecContext(ctx, `INSERT INTO carpool_revenue_configs (id) VALUES (1) ON CONFLICT (id) DO NOTHING`); err != nil {
		return nil, err
	}
	cfg := &CarpoolRevenueConfig{}
	err := s.scanOne(ctx, `
		SELECT enabled, user_share_ratio, platform_share_ratio, min_withdraw_amount, withdraw_cooldown_minutes,
		       settlement_cycle, freeze_minutes, allow_user_withdraw, priority_policy, risk_notes,
		       gateway_dispatch_enabled, gateway_shadow_mode, gateway_traffic_percent, gateway_allowed_group_ids,
		       gateway_allowed_models, gateway_min_remain_ratio, gateway_max_daily_quota, updated_at
		FROM carpool_revenue_configs
		WHERE id = 1
	`, nil,
		&cfg.Enabled,
		&cfg.UserShareRatio,
		&cfg.PlatformShareRatio,
		&cfg.MinWithdrawAmount,
		&cfg.WithdrawCooldownMinutes,
		&cfg.SettlementCycle,
		&cfg.FreezeMinutes,
		&cfg.AllowUserWithdraw,
		&cfg.PriorityPolicy,
		&cfg.RiskNotes,
		&cfg.GatewayDispatchEnabled,
		&cfg.GatewayShadowMode,
		&cfg.GatewayTrafficPercent,
		&cfg.GatewayAllowedGroupIDs,
		&cfg.GatewayAllowedModels,
		&cfg.GatewayMinRemainRatio,
		&cfg.GatewayMaxDailyQuota,
		&cfg.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *CarpoolService) AdminUpdateRevenueConfig(ctx context.Context, input CarpoolRevenueConfigInput) (*CarpoolRevenueConfig, error) {
	userShare := input.UserShareRatio
	if userShare < 0 {
		userShare = 0
	}
	if userShare > 1 {
		userShare = 1
	}
	platformShare := input.PlatformShareRatio
	if platformShare < 0 {
		platformShare = 0
	}
	if platformShare > 1 {
		platformShare = 1
	}
	if userShare == 0 && platformShare == 0 {
		userShare = 0.7
		platformShare = 0.3
	}
	minWithdraw := input.MinWithdrawAmount
	if minWithdraw < 0 {
		minWithdraw = 0
	}
	cycle := strings.TrimSpace(input.SettlementCycle)
	if cycle == "" {
		cycle = "manual"
	}
	priorityPolicy := strings.TrimSpace(input.PriorityPolicy)
	if priorityPolicy == "" {
		priorityPolicy = "user_first"
	}
	if input.WithdrawCooldownMinutes < 0 {
		input.WithdrawCooldownMinutes = 0
	}
	if input.FreezeMinutes < 0 {
		input.FreezeMinutes = 0
	}
	trafficPercent := input.GatewayTrafficPercent
	if trafficPercent < 0 {
		trafficPercent = 0
	}
	if trafficPercent > 100 {
		trafficPercent = 100
	}
	minRemainRatio := input.GatewayMinRemainRatio
	if minRemainRatio < 0 {
		minRemainRatio = 0
	}
	if minRemainRatio > 0.95 {
		minRemainRatio = 0.95
	}
	maxDailyQuota := input.GatewayMaxDailyQuota
	if maxDailyQuota < 0 {
		maxDailyQuota = 0
	}
	if _, err := s.entClient.ExecContext(ctx, `
		INSERT INTO carpool_revenue_configs (
			id, enabled, user_share_ratio, platform_share_ratio, min_withdraw_amount,
			withdraw_cooldown_minutes, settlement_cycle, freeze_minutes, allow_user_withdraw,
			priority_policy, risk_notes, gateway_dispatch_enabled, gateway_shadow_mode,
			gateway_traffic_percent, gateway_allowed_group_ids, gateway_allowed_models,
			gateway_min_remain_ratio, gateway_max_daily_quota, updated_at
		)
		VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW())
		ON CONFLICT (id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			user_share_ratio = EXCLUDED.user_share_ratio,
			platform_share_ratio = EXCLUDED.platform_share_ratio,
			min_withdraw_amount = EXCLUDED.min_withdraw_amount,
			withdraw_cooldown_minutes = EXCLUDED.withdraw_cooldown_minutes,
			settlement_cycle = EXCLUDED.settlement_cycle,
			freeze_minutes = EXCLUDED.freeze_minutes,
			allow_user_withdraw = EXCLUDED.allow_user_withdraw,
			priority_policy = EXCLUDED.priority_policy,
			risk_notes = EXCLUDED.risk_notes,
			gateway_dispatch_enabled = EXCLUDED.gateway_dispatch_enabled,
			gateway_shadow_mode = EXCLUDED.gateway_shadow_mode,
			gateway_traffic_percent = EXCLUDED.gateway_traffic_percent,
			gateway_allowed_group_ids = EXCLUDED.gateway_allowed_group_ids,
			gateway_allowed_models = EXCLUDED.gateway_allowed_models,
			gateway_min_remain_ratio = EXCLUDED.gateway_min_remain_ratio,
			gateway_max_daily_quota = EXCLUDED.gateway_max_daily_quota,
			updated_at = NOW()
	`, input.Enabled, userShare, platformShare, minWithdraw, input.WithdrawCooldownMinutes, cycle, input.FreezeMinutes, input.AllowUserWithdraw, priorityPolicy, strings.TrimSpace(input.RiskNotes), input.GatewayDispatchEnabled, input.GatewayShadowMode, trafficPercent, normalizeCSV(input.GatewayAllowedGroupIDs), normalizeCSV(input.GatewayAllowedModels), minRemainRatio, maxDailyQuota); err != nil {
		return nil, err
	}
	return s.GetRevenueConfig(ctx)
}

func (s *CarpoolService) MyRevenueDetail(ctx context.Context, userID, participantID int64) (*CarpoolRevenueDetail, error) {
	_, participant, session, vt, groupID, sub, reason, err := s.resolveRevenueEligibility(ctx, userID, participantID, false)
	if err != nil {
		return nil, err
	}
	cfg, _ := s.GetRevenueConfig(ctx)
	detail := &CarpoolRevenueDetail{
		AvailableReason: reason,
		Config:          cfg,
		Records:         []CarpoolRevenueRecord{},
		Withdrawals:     []CarpoolRevenueWithdrawal{},
	}
	contribution, err := s.getContributionByParticipant(ctx, participantID)
	if err != nil && !errorsIsSQLNoRows(err) {
		return nil, err
	}
	if contribution == nil && participant != nil && session != nil && vt != nil && sub != nil {
		contribution = &CarpoolRevenueContribution{
			ParticipantID:       participant.ID,
			UserID:              userID,
			SessionID:           session.ID,
			VehicleTypeID:       vt.ID,
			SubscriptionID:      &sub.ID,
			SubscriptionGroupID: groupID,
			Enabled:             false,
			ShareRatio:          revenueConfigUserShare(cfg),
			Status:              CarpoolRevenueStatusDisabled,
		}
	}
	if contribution != nil {
		if contribution.Enabled && contribution.Status == CarpoolRevenueStatusActive {
			detail.AvailableReason = "available"
		} else if detail.AvailableReason == "available" && contribution.Status != "" {
			detail.AvailableReason = contribution.Status
		}
	}
	detail.Contribution = contribution
	detail.Summary, err = s.revenueSummary(ctx, userID, participantID)
	if err != nil {
		return nil, err
	}
	detail.Records, err = s.listRevenueRecords(ctx, userID, participantID, 20)
	if err != nil {
		return nil, err
	}
	detail.Withdrawals, err = s.listRevenueWithdrawals(ctx, userID, participantID, 20)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (s *CarpoolService) EnableMyRevenue(ctx context.Context, userID, participantID int64) (*CarpoolRevenueDetail, error) {
	cfg, participant, session, vt, groupID, sub, _, err := s.resolveRevenueEligibility(ctx, userID, participantID, true)
	if err != nil {
		return nil, err
	}
	shareRatio := revenueConfigUserShare(cfg)
	var id int64
	err = s.scanOne(ctx, `
		INSERT INTO carpool_revenue_contributions (
			participant_id, user_id, session_id, vehicle_type_id, subscription_id, subscription_group_id,
			enabled, enabled_at, disabled_at, share_ratio, status, paused_reason, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, TRUE, NOW(), NULL, $7, $8, '', NOW())
		ON CONFLICT (participant_id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			session_id = EXCLUDED.session_id,
			vehicle_type_id = EXCLUDED.vehicle_type_id,
			subscription_id = EXCLUDED.subscription_id,
			subscription_group_id = EXCLUDED.subscription_group_id,
			enabled = TRUE,
			enabled_at = COALESCE(carpool_revenue_contributions.enabled_at, NOW()),
			disabled_at = NULL,
			share_ratio = EXCLUDED.share_ratio,
			status = EXCLUDED.status,
			paused_reason = '',
			updated_at = NOW()
		RETURNING id
	`, []any{participant.ID, userID, session.ID, vt.ID, sub.ID, groupID, shareRatio, CarpoolRevenueStatusActive}, &id)
	if err != nil {
		return nil, err
	}
	return s.MyRevenueDetail(ctx, userID, participantID)
}

func (s *CarpoolService) DisableMyRevenue(ctx context.Context, userID, participantID int64) (*CarpoolRevenueDetail, error) {
	res, err := s.entClient.ExecContext(ctx, `
		UPDATE carpool_revenue_contributions
		SET enabled = FALSE, disabled_at = NOW(), status = $1, updated_at = NOW()
		WHERE participant_id = $2 AND user_id = $3
	`, CarpoolRevenueStatusPausedByUser, participantID, userID)
	if err != nil {
		return nil, err
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		_, _, _, _, _, _, _, err := s.resolveRevenueEligibility(ctx, userID, participantID, false)
		if err != nil {
			return nil, err
		}
	}
	return s.MyRevenueDetail(ctx, userID, participantID)
}

func (s *CarpoolService) WithdrawMyRevenue(ctx context.Context, userID, participantID int64, input CarpoolRevenueWithdrawInput) (*CarpoolRevenueWithdrawal, error) {
	cfg, err := s.GetRevenueConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil || !cfg.Enabled || !cfg.AllowUserWithdraw {
		return nil, infraerrors.Forbidden("CARPOOL_REVENUE_WITHDRAW_DISABLED", "carpool revenue withdraw is disabled")
	}
	amount := math.Round(input.Amount*100000000) / 100000000
	if amount <= 0 {
		return nil, infraerrors.BadRequest("CARPOOL_REVENUE_WITHDRAW_AMOUNT_INVALID", "withdraw amount must be greater than zero")
	}
	if amount < cfg.MinWithdrawAmount {
		return nil, infraerrors.BadRequest("CARPOOL_REVENUE_WITHDRAW_AMOUNT_TOO_SMALL", "withdraw amount is below minimum")
	}
	if participantID > 0 {
		if _, _, _, _, _, _, _, err := s.resolveRevenueEligibility(ctx, userID, participantID, false); err != nil {
			return nil, err
		}
	}
	var pID any
	if participantID > 0 {
		pID = participantID
	}
	if cfg.WithdrawCooldownMinutes > 0 {
		var lastRequestedAt sql.NullTime
		if err := s.scanOne(ctx, `
			SELECT MAX(requested_at)
			FROM carpool_revenue_withdrawals
			WHERE user_id = $1
			  AND status = 'completed'
			  AND ($2::bigint IS NULL OR participant_id = $2::bigint)
		`, []any{userID, pID}, &lastRequestedAt); err != nil {
			return nil, err
		}
		if lastRequestedAt.Valid && time.Since(lastRequestedAt.Time) < time.Duration(cfg.WithdrawCooldownMinutes)*time.Minute {
			return nil, infraerrors.BadRequest("CARPOOL_REVENUE_WITHDRAW_COOLDOWN", "withdraw is in cooldown")
		}
	}
	var w CarpoolRevenueWithdrawal
	var participantNull sql.NullInt64
	var sessionNull sql.NullInt64
	var balanceBefore sql.NullFloat64
	var balanceAfter sql.NullFloat64
	var processedAt sql.NullTime
	err = s.scanOne(ctx, `
		WITH lock AS (
			SELECT pg_advisory_xact_lock($1::bigint)
		),
		totals AS (
			SELECT COALESCE(SUM(user_share_amount), 0)::numeric AS total
			FROM carpool_revenue_records, lock
			WHERE user_id = $1
			  AND status = 'settled'
			  AND ($3::bigint IS NULL OR participant_id = $3::bigint)
		),
		withdrawn AS (
			SELECT COALESCE(SUM(amount), 0)::numeric AS total
			FROM carpool_revenue_withdrawals, lock
			WHERE user_id = $1
			  AND status = 'completed'
			  AND ($3::bigint IS NULL OR participant_id = $3::bigint)
		),
		availability AS (
			SELECT GREATEST(totals.total - withdrawn.total, 0)::numeric AS available
			FROM totals, withdrawn
		),
		updated_user AS (
			UPDATE users
			SET balance = balance + $2,
			    updated_at = NOW()
			FROM availability
			WHERE users.id = $1
			  AND users.deleted_at IS NULL
			  AND $2 > 0
			  AND availability.available >= $2
			RETURNING users.balance - $2 AS balance_before, users.balance AS balance_after, availability.available AS available_before
		),
		inserted AS (
			INSERT INTO carpool_revenue_withdrawals (
				user_id, participant_id, session_id, amount, available_before, available_after,
				balance_before, balance_after, status, requested_at, processed_at, created_at, updated_at
			)
			SELECT
				$1,
				$3::bigint,
				(SELECT session_id FROM carpool_participants WHERE id = $3::bigint),
				$2,
				available_before,
				available_before - $2,
				balance_before,
				balance_after,
				'completed',
				NOW(),
				NOW(),
				NOW(),
				NOW()
			FROM updated_user
			RETURNING id, user_id, participant_id, session_id, amount, available_before, available_after,
			          balance_before, balance_after, status, requested_at, processed_at, failure_reason, created_at
		)
		SELECT id, user_id, participant_id, session_id, amount, available_before, available_after,
		       balance_before, balance_after, status, requested_at, processed_at, failure_reason, created_at
		FROM inserted
	`, []any{userID, amount, pID},
		&w.ID,
		&w.UserID,
		&participantNull,
		&sessionNull,
		&w.Amount,
		&w.AvailableBefore,
		&w.AvailableAfter,
		&balanceBefore,
		&balanceAfter,
		&w.Status,
		&w.RequestedAt,
		&processedAt,
		&w.FailureReason,
		&w.CreatedAt,
	)
	if errorsIsSQLNoRows(err) {
		return nil, infraerrors.BadRequest("CARPOOL_REVENUE_WITHDRAW_INSUFFICIENT", "available revenue is insufficient")
	}
	if err != nil {
		return nil, err
	}
	w.ParticipantID = int64PtrFromNull(participantNull)
	w.SessionID = int64PtrFromNull(sessionNull)
	w.BalanceBefore = float64PtrFromNull(balanceBefore)
	w.BalanceAfter = float64PtrFromNull(balanceAfter)
	w.ProcessedAt = timePtrFromNull(processedAt)
	return &w, nil
}

func (s *CarpoolService) AdminListRevenueContributions(ctx context.Context, page, pageSize int, status string) (*CarpoolRevenueAdminListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	status = strings.TrimSpace(status)
	args := []any{}
	where := ""
	if status != "" && status != "all" {
		args = append(args, status)
		where = fmt.Sprintf("WHERE c.status = $%d", len(args))
	}
	countQuery := "SELECT COUNT(*) FROM carpool_revenue_contributions c " + where
	var total int
	if err := s.scanOne(ctx, countQuery, args, &total); err != nil {
		return nil, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.entClient.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			c.id, c.participant_id, c.user_id, c.session_id, c.vehicle_type_id, c.subscription_id, c.subscription_group_id,
			c.enabled, c.enabled_at, c.disabled_at, c.share_ratio, c.status, c.paused_reason, c.last_settled_at, c.notes, c.created_at, c.updated_at,
			COALESCE(u.email, ''), COALESCE(u.username, ''), COALESCE(s.session_no, ''), COALESCE(s.status, ''), COALESCE(v.name, ''),
			COALESCE(SUM(CASE WHEN r.status = 'settled' THEN r.user_share_amount ELSE 0 END), 0) AS settled,
			COALESCE(SUM(CASE WHEN r.status = 'pending' THEN r.user_share_amount ELSE 0 END), 0) AS pending,
			COALESCE(SUM(CASE WHEN r.status = 'frozen' THEN r.user_share_amount ELSE 0 END), 0) AS frozen,
			COALESCE(SUM(CASE WHEN r.status = 'settled' THEN r.quota_cost ELSE 0 END), 0) AS quota_cost,
			COALESCE(SUM(CASE WHEN r.status = 'settled' THEN r.request_count ELSE 0 END), 0) AS request_count,
			COALESCE(SUM(CASE WHEN r.status = 'settled' THEN r.platform_share_amount ELSE 0 END), 0) AS platform_share,
			COALESCE((
				SELECT SUM(w.amount) FROM carpool_revenue_withdrawals w
				WHERE w.user_id = c.user_id AND w.participant_id = c.participant_id AND w.status = 'completed'
			), 0) AS withdrawn
		FROM carpool_revenue_contributions c
		LEFT JOIN users u ON u.id = c.user_id
		LEFT JOIN carpool_sessions s ON s.id = c.session_id
		LEFT JOIN carpool_vehicle_types v ON v.id = c.vehicle_type_id
		LEFT JOIN carpool_revenue_records r ON r.contribution_id = c.id
		%s
		GROUP BY c.id, u.email, u.username, s.session_no, s.status, v.name
		ORDER BY c.updated_at DESC, c.id DESC
		LIMIT $%d OFFSET $%d
	`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []CarpoolRevenueContributionAdminRow{}
	for rows.Next() {
		var row CarpoolRevenueContributionAdminRow
		var email, username, sessionNo, sessionStatus, vehicleName string
		var settled, withdrawn float64
		if err := scanRevenueContribution(rows, &row.Contribution, &email, &username, &sessionNo, &sessionStatus, &vehicleName, &settled, &row.Summary.PendingRevenue, &row.Summary.FrozenRevenue, &row.Summary.QuotaCost, &row.Summary.RequestCount, &row.Summary.PlatformShareAmount, &withdrawn); err != nil {
			return nil, err
		}
		row.Summary.TotalRevenue = settled
		row.Summary.WithdrawnRevenue = withdrawn
		row.Summary.AvailableRevenue = math.Max(0, settled-withdrawn)
		row.User = map[string]any{"id": row.Contribution.UserID, "email": email, "username": username}
		row.Session = map[string]any{"id": row.Contribution.SessionID, "session_no": sessionNo, "status": sessionStatus}
		row.VehicleType = map[string]any{"id": row.Contribution.VehicleTypeID, "name": vehicleName}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &CarpoolRevenueAdminListResponse{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *CarpoolService) AdminPauseRevenueContribution(ctx context.Context, id int64, reason string) (*CarpoolRevenueContribution, error) {
	if id <= 0 {
		return nil, infraerrors.BadRequest("CARPOOL_REVENUE_CONTRIBUTION_REQUIRED", "contribution id is required")
	}
	res, err := s.entClient.ExecContext(ctx, `
		UPDATE carpool_revenue_contributions
		SET enabled = FALSE, disabled_at = NOW(), status = $1, paused_reason = $2, updated_at = NOW()
		WHERE id = $3
	`, CarpoolRevenueStatusPausedByAdmin, strings.TrimSpace(reason), id)
	if err != nil {
		return nil, err
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return nil, infraerrors.NotFound("CARPOOL_REVENUE_CONTRIBUTION_NOT_FOUND", "contribution not found")
	}
	return s.getContributionByID(ctx, id)
}

func (s *CarpoolService) AdminResumeRevenueContribution(ctx context.Context, id int64) (*CarpoolRevenueContribution, error) {
	if id <= 0 {
		return nil, infraerrors.BadRequest("CARPOOL_REVENUE_CONTRIBUTION_REQUIRED", "contribution id is required")
	}
	var participantID, userID int64
	if err := s.scanOne(ctx, `SELECT participant_id, user_id FROM carpool_revenue_contributions WHERE id = $1`, []any{id}, &participantID, &userID); err != nil {
		if errorsIsSQLNoRows(err) {
			return nil, infraerrors.NotFound("CARPOOL_REVENUE_CONTRIBUTION_NOT_FOUND", "contribution not found")
		}
		return nil, err
	}
	if _, _, _, _, _, _, _, err := s.resolveRevenueEligibility(ctx, userID, participantID, true); err != nil {
		return nil, err
	}
	res, err := s.entClient.ExecContext(ctx, `
		UPDATE carpool_revenue_contributions
		SET enabled = TRUE, enabled_at = COALESCE(enabled_at, NOW()), disabled_at = NULL, status = $1, paused_reason = '', updated_at = NOW()
		WHERE id = $2
	`, CarpoolRevenueStatusActive, id)
	if err != nil {
		return nil, err
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return nil, infraerrors.NotFound("CARPOOL_REVENUE_CONTRIBUTION_NOT_FOUND", "contribution not found")
	}
	return s.getContributionByID(ctx, id)
}

func (s *CarpoolService) AdminCreateRevenueRecord(ctx context.Context, input CarpoolRevenueRecordInput) (*CarpoolRevenueRecord, error) {
	contribution, err := s.getContributionByID(ctx, input.ContributionID)
	if err != nil {
		return nil, err
	}
	if input.UserShareAmount == 0 && input.NetRevenue != 0 {
		input.UserShareAmount = math.Round(input.NetRevenue*contribution.ShareRatio*100000000) / 100000000
	}
	if input.PlatformShareAmount == 0 {
		input.PlatformShareAmount = math.Round((input.NetRevenue-input.UserShareAmount)*100000000) / 100000000
	}
	occurredAt := time.Now()
	if strings.TrimSpace(input.OccurredAt) != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(input.OccurredAt)); parseErr == nil {
			occurredAt = parsed
		}
	}
	var record CarpoolRevenueRecord
	err = s.scanOne(ctx, `
		INSERT INTO carpool_revenue_records (
			contribution_id, participant_id, session_id, user_id, subscription_group_id, request_id,
			request_count, quota_cost, gross_revenue, upstream_cost, net_revenue, user_share_amount,
			platform_share_amount, settlement_period, status, occurred_at, settled_at, notes, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 'settled', $15, NOW(), $16, NOW(), NOW())
		RETURNING id, contribution_id, participant_id, session_id, user_id, subscription_group_id, api_key_id, usage_log_id,
		          request_id, request_count, quota_cost, gross_revenue, upstream_cost, net_revenue, user_share_amount,
		          platform_share_amount, settlement_period, status, occurred_at, settled_at, notes, created_at
	`, []any{contribution.ID, contribution.ParticipantID, contribution.SessionID, contribution.UserID, contribution.SubscriptionGroupID, strings.TrimSpace(input.RequestID), input.RequestCount, input.QuotaCost, input.GrossRevenue, input.UpstreamCost, input.NetRevenue, input.UserShareAmount, input.PlatformShareAmount, strings.TrimSpace(input.SettlementPeriod), occurredAt, strings.TrimSpace(input.Notes)},
		&record.ID, &record.ContributionID, &record.ParticipantID, &record.SessionID, &record.UserID, &record.SubscriptionGroupID,
		newNullInt64Scanner(&record.APIKeyID), newNullInt64Scanner(&record.UsageLogID), &record.RequestID, &record.RequestCount, &record.QuotaCost,
		&record.GrossRevenue, &record.UpstreamCost, &record.NetRevenue, &record.UserShareAmount, &record.PlatformShareAmount,
		&record.SettlementPeriod, &record.Status, &record.OccurredAt, newNullTimeScanner(&record.SettledAt), &record.Notes, &record.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *CarpoolService) PlanRevenueGatewayDispatch(ctx context.Context, input CarpoolRevenueGatewayDispatchInput) (*CarpoolRevenueGatewayDispatchDecision, error) {
	if s == nil || s.entClient == nil {
		return nil, nil
	}
	cfg, err := s.GetRevenueConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil || !cfg.Enabled || !cfg.GatewayDispatchEnabled {
		return nil, nil
	}
	originalGroupID := derefGroupID(input.OriginalGroupID)
	if originalGroupID <= 0 {
		return nil, nil
	}
	if !csvAllowsInt64(cfg.GatewayAllowedGroupIDs, originalGroupID) {
		return nil, nil
	}
	if !modelPatternsAllow(cfg.GatewayAllowedModels, input.RequestedModel) {
		return nil, nil
	}

	mode := "real"
	routed := true
	if cfg.GatewayShadowMode {
		mode = "shadow"
		routed = false
	} else if !percentHit(cfg.GatewayTrafficPercent) {
		return nil, nil
	}

	minRemainRatio := cfg.GatewayMinRemainRatio
	if minRemainRatio < 0 {
		minRemainRatio = 0
	}
	if minRemainRatio > 0.95 {
		minRemainRatio = 0.95
	}
	platform := strings.TrimSpace(input.RequestPlatform)
	var platformArg any
	if platform != "" {
		platformArg = platform
	}
	var decision CarpoolRevenueGatewayDispatchDecision
	err = s.scanOne(ctx, `
		WITH daily_contribution AS (
			SELECT contribution_id, COALESCE(SUM(quota_cost), 0)::numeric AS quota
			FROM carpool_revenue_records
			WHERE occurred_at >= NOW() - INTERVAL '24 hours'
			  AND status IN ('settled', 'pending', 'frozen')
			GROUP BY contribution_id
		)
		SELECT
			c.id, c.participant_id, c.session_id, c.user_id, c.subscription_id, c.subscription_group_id, c.share_ratio
		FROM carpool_revenue_contributions c
		JOIN user_subscriptions us ON us.id = c.subscription_id
		JOIN groups g ON g.id = c.subscription_group_id
		LEFT JOIN daily_contribution dc ON dc.contribution_id = c.id
		WHERE c.enabled = TRUE
		  AND c.status = $1
		  AND c.subscription_id IS NOT NULL
		  AND c.user_id <> $2
		  AND us.deleted_at IS NULL
		  AND us.status = $3
		  AND us.expires_at > NOW()
		  AND g.deleted_at IS NULL
		  AND g.status = $4
		  AND g.subscription_type = $5
		  AND ($6::text IS NULL OR g.platform = $6::text)
		  AND (
			g.daily_limit_usd IS NULL OR g.daily_limit_usd <= 0 OR
			(CASE WHEN us.daily_window_start IS NULL OR us.daily_window_start + INTERVAL '24 hours' <= NOW() THEN 0 ELSE us.daily_usage_usd END) < g.daily_limit_usd * (1::numeric - $7::numeric)
		  )
		  AND (
			g.weekly_limit_usd IS NULL OR g.weekly_limit_usd <= 0 OR
			(CASE WHEN us.weekly_window_start IS NULL OR us.weekly_window_start + INTERVAL '7 days' <= NOW() THEN 0 ELSE us.weekly_usage_usd END) < g.weekly_limit_usd * (1::numeric - $7::numeric)
		  )
		  AND (
			g.monthly_limit_usd IS NULL OR g.monthly_limit_usd <= 0 OR
			(CASE WHEN us.monthly_window_start IS NULL OR us.monthly_window_start + INTERVAL '1 month' <= NOW() THEN 0 ELSE us.monthly_usage_usd END) < g.monthly_limit_usd * (1::numeric - $7::numeric)
		  )
		  AND ($8::numeric <= 0 OR COALESCE(dc.quota, 0) < $8::numeric)
		ORDER BY COALESCE(c.last_settled_at, c.enabled_at, c.created_at) ASC, c.id ASC
		LIMIT 1
	`, []any{CarpoolRevenueStatusActive, input.RequestUserID, SubscriptionStatusActive, StatusActive, SubscriptionTypeSubscription, platformArg, minRemainRatio, cfg.GatewayMaxDailyQuota},
		&decision.ContributionID,
		&decision.ParticipantID,
		&decision.SessionID,
		&decision.UserID,
		&decision.SubscriptionID,
		&decision.SubscriptionGroupID,
		&decision.ShareRatio,
	)
	if errorsIsSQLNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	decision.Mode = mode
	decision.Reason = mode + "_candidate"
	decision.OriginalGroupID = originalGroupID
	decision.RequestUserID = input.RequestUserID
	decision.RequestAPIKeyID = input.RequestAPIKeyID
	decision.Routed = routed
	return &decision, nil
}

func (s *CarpoolService) SettleRevenueGatewayUsage(ctx context.Context, decision *CarpoolRevenueGatewayDispatchDecision, usageLog *UsageLog, cost *CostBreakdown) error {
	if s == nil || s.entClient == nil || !decision.ShouldSettle() || usageLog == nil || cost == nil {
		return nil
	}
	quotaCost := math.Round(cost.ActualCost*10000000000) / 10000000000
	if quotaCost <= 0 {
		return nil
	}
	cfg, err := s.GetRevenueConfig(ctx)
	if err != nil {
		return err
	}
	if cfg == nil || !cfg.Enabled || !cfg.GatewayDispatchEnabled {
		return nil
	}
	userShareRatio := decision.ShareRatio
	if userShareRatio <= 0 {
		userShareRatio = revenueConfigUserShare(cfg)
	}
	if userShareRatio > 1 {
		userShareRatio = 1
	}
	upstreamCost := cost.TotalCost
	if usageLog.AccountStatsCost != nil {
		upstreamCost = *usageLog.AccountStatsCost
	} else if usageLog.AccountRateMultiplier != nil {
		upstreamCost = cost.TotalCost * *usageLog.AccountRateMultiplier
	}
	grossRevenue := cost.ActualCost
	netRevenue := grossRevenue - upstreamCost
	if netRevenue < 0 {
		netRevenue = 0
	}
	userShare := math.Round(netRevenue*userShareRatio*100000000) / 100000000
	platformShare := math.Round((netRevenue-userShare)*100000000) / 100000000
	requestGroupID := int64(0)
	if usageLog.GroupID != nil {
		requestGroupID = *usageLog.GroupID
	}
	requestID := strings.TrimSpace(usageLog.RequestID)
	if requestID == "" {
		requestID = "generated:" + generateRequestID()
	}
	minRemainRatio := cfg.GatewayMinRemainRatio
	if minRemainRatio < 0 {
		minRemainRatio = 0
	}
	if minRemainRatio > 0.95 {
		minRemainRatio = 0.95
	}

	var recordID int64
	err = s.scanOne(ctx, `
		WITH candidate AS (
			SELECT
				c.id, c.participant_id, c.session_id, c.user_id, c.subscription_id, c.subscription_group_id, c.share_ratio,
				us.daily_window_start, us.weekly_window_start, us.monthly_window_start,
				CASE WHEN us.daily_window_start IS NULL OR us.daily_window_start + INTERVAL '24 hours' <= NOW() THEN 0 ELSE us.daily_usage_usd END AS daily_usage,
				CASE WHEN us.weekly_window_start IS NULL OR us.weekly_window_start + INTERVAL '7 days' <= NOW() THEN 0 ELSE us.weekly_usage_usd END AS weekly_usage,
				CASE WHEN us.monthly_window_start IS NULL OR us.monthly_window_start + INTERVAL '1 month' <= NOW() THEN 0 ELSE us.monthly_usage_usd END AS monthly_usage,
				g.daily_limit_usd, g.weekly_limit_usd, g.monthly_limit_usd
			FROM carpool_revenue_contributions c
			JOIN user_subscriptions us ON us.id = c.subscription_id
			JOIN groups g ON g.id = c.subscription_group_id
			WHERE c.id = $1
			  AND c.enabled = TRUE
			  AND c.status = $2
			  AND c.subscription_id = $3
			  AND us.deleted_at IS NULL
			  AND us.status = $4
			  AND us.expires_at > NOW()
			  AND g.deleted_at IS NULL
			  AND g.status = $5
			FOR UPDATE OF us, c
		),
		daily_contribution AS (
			SELECT COALESCE(SUM(quota_cost), 0)::numeric AS quota
			FROM carpool_revenue_records
			WHERE contribution_id = $1
			  AND occurred_at >= NOW() - INTERVAL '24 hours'
			  AND status IN ('settled', 'pending', 'frozen')
		),
		eligible AS (
			SELECT candidate.*
			FROM candidate, daily_contribution
			WHERE ($21::numeric <= 0 OR daily_contribution.quota + $6 <= $21::numeric)
			  AND (daily_limit_usd IS NULL OR daily_limit_usd <= 0 OR daily_usage + $6 <= daily_limit_usd * (1::numeric - $20::numeric))
			  AND (weekly_limit_usd IS NULL OR weekly_limit_usd <= 0 OR weekly_usage + $6 <= weekly_limit_usd * (1::numeric - $20::numeric))
			  AND (monthly_limit_usd IS NULL OR monthly_limit_usd <= 0 OR monthly_usage + $6 <= monthly_limit_usd * (1::numeric - $20::numeric))
		),
		updated_sub AS (
			UPDATE user_subscriptions us
			SET
				daily_usage_usd = CASE WHEN us.daily_window_start IS NULL OR us.daily_window_start + INTERVAL '24 hours' <= NOW() THEN $6 ELSE us.daily_usage_usd + $6 END,
				weekly_usage_usd = CASE WHEN us.weekly_window_start IS NULL OR us.weekly_window_start + INTERVAL '7 days' <= NOW() THEN $6 ELSE us.weekly_usage_usd + $6 END,
				monthly_usage_usd = CASE WHEN us.monthly_window_start IS NULL OR us.monthly_window_start + INTERVAL '1 month' <= NOW() THEN $6 ELSE us.monthly_usage_usd + $6 END,
				daily_window_start = CASE WHEN us.daily_window_start IS NULL OR us.daily_window_start + INTERVAL '24 hours' <= NOW() THEN NOW() ELSE us.daily_window_start END,
				weekly_window_start = CASE WHEN us.weekly_window_start IS NULL OR us.weekly_window_start + INTERVAL '7 days' <= NOW() THEN NOW() ELSE us.weekly_window_start END,
				monthly_window_start = CASE WHEN us.monthly_window_start IS NULL OR us.monthly_window_start + INTERVAL '1 month' <= NOW() THEN NOW() ELSE us.monthly_window_start END,
				updated_at = NOW()
			FROM eligible e
			WHERE us.id = e.subscription_id
			RETURNING e.*, e.monthly_usage AS quota_before,
				CASE WHEN us.monthly_window_start IS NULL OR us.monthly_window_start + INTERVAL '1 month' <= NOW() THEN $6 ELSE us.monthly_usage_usd END AS quota_after
		),
		inserted AS (
			INSERT INTO carpool_revenue_records (
				contribution_id, participant_id, session_id, user_id, subscription_group_id, subscription_id,
				api_key_id, usage_log_id, request_user_id, request_api_key_id, request_account_id, request_group_id,
				dispatch_mode, decision_reason, request_id, request_count, quota_cost, quota_before, quota_after,
				gross_revenue, upstream_cost, net_revenue, user_share_amount, platform_share_amount,
				settlement_period, status, occurred_at, settled_at, notes, created_at, updated_at
			)
			SELECT
				id, participant_id, session_id, user_id, subscription_group_id, subscription_id,
				$7, $8, $9, $10, $11, NULLIF($12, 0),
				$13, $14, $15, 1, $6, quota_before, quota_after,
				$16, $17, $18, $19, $22,
				to_char(NOW(), 'YYYY-MM-DD'), $23, NOW(), NOW(), $24, NOW(), NOW()
			FROM updated_sub
			ON CONFLICT (contribution_id, request_id) WHERE request_id <> '' DO NOTHING
			RETURNING id
		),
		touched AS (
			UPDATE carpool_revenue_contributions
			SET last_settled_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND EXISTS (SELECT 1 FROM inserted)
			RETURNING id
		)
		SELECT id FROM inserted
	`, []any{
		decision.ContributionID,
		CarpoolRevenueStatusActive,
		decision.SubscriptionID,
		SubscriptionStatusActive,
		StatusActive,
		quotaCost,
		usageLog.APIKeyID,
		nil,
		usageLog.UserID,
		decision.RequestAPIKeyID,
		usageLog.AccountID,
		requestGroupID,
		decision.Mode,
		decision.Reason,
		requestID,
		grossRevenue,
		upstreamCost,
		netRevenue,
		userShare,
		minRemainRatio,
		cfg.GatewayMaxDailyQuota,
		platformShare,
		CarpoolRevenueRecordStatusSettled,
		"网关自动结算",
	}, &recordID)
	if errorsIsSQLNoRows(err) {
		return nil
	}
	return err
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

func (s *CarpoolService) carpoolAssignedGroupIDs(ctx context.Context, participant *dbent.CarpoolParticipant, session *dbent.CarpoolSession) (int64, int64, error) {
	if session == nil {
		return 0, 0, nil
	}
	subscriptionGroupID := numberFromMap(session.AccountInfo, "subscription_group_id")
	accountPoolGroupID := numberFromMap(session.AccountInfo, "account_pool_group_id")
	if subscriptionGroupID <= 0 && participant != nil {
		contribution, err := s.getContributionByParticipant(ctx, participant.ID)
		if err != nil && !errorsIsSQLNoRows(err) {
			return 0, 0, err
		}
		if contribution != nil && contribution.SubscriptionGroupID > 0 {
			subscriptionGroupID = contribution.SubscriptionGroupID
		}
	}
	if accountPoolGroupID <= 0 {
		accountPoolGroupID = subscriptionGroupID
	}
	return subscriptionGroupID, accountPoolGroupID, nil
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

func (s *CarpoolService) resolveRevenueEligibility(ctx context.Context, userID, participantID int64, strict bool) (*CarpoolRevenueConfig, *dbent.CarpoolParticipant, *dbent.CarpoolSession, *dbent.CarpoolVehicleType, int64, *dbent.UserSubscription, string, error) {
	if userID <= 0 {
		return nil, nil, nil, nil, 0, nil, "unauthorized", infraerrors.Unauthorized("UNAUTHORIZED", "unauthorized")
	}
	if participantID <= 0 {
		return nil, nil, nil, nil, 0, nil, "invalid_participant", infraerrors.BadRequest("CARPOOL_PARTICIPANT_REQUIRED", "carpool record is required")
	}
	participant, err := s.entClient.CarpoolParticipant.Query().
		Where(carpoolparticipant.IDEQ(participantID), carpoolparticipant.UserIDEQ(userID)).
		WithVehicleType().
		WithSession(func(q *dbent.CarpoolSessionQuery) { q.WithVehicleType() }).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil, nil, nil, 0, nil, "not_found", infraerrors.NotFound("CARPOOL_PARTICIPANT_NOT_FOUND", "carpool record not found")
		}
		return nil, nil, nil, nil, 0, nil, "load_failed", err
	}
	session := participant.Edges.Session
	vt := participant.Edges.VehicleType
	if vt == nil && session != nil {
		vt = session.Edges.VehicleType
	}
	cfg, err := s.GetRevenueConfig(ctx)
	if err != nil {
		return nil, participant, session, vt, 0, nil, "config_error", err
	}
	fail := func(code, msg string, err error) (*CarpoolRevenueConfig, *dbent.CarpoolParticipant, *dbent.CarpoolSession, *dbent.CarpoolVehicleType, int64, *dbent.UserSubscription, string, error) {
		if strict {
			return cfg, participant, session, vt, 0, nil, code, err
		}
		return cfg, participant, session, vt, 0, nil, code, nil
	}
	if cfg == nil || !cfg.Enabled {
		return fail("global_disabled", "carpool revenue pool is disabled", infraerrors.Forbidden("CARPOOL_REVENUE_DISABLED", "carpool revenue pool is disabled"))
	}
	if vt == nil || !vt.SupportRevenuePool {
		return fail("vehicle_unsupported", "this carpool type does not support revenue pool", infraerrors.Forbidden("CARPOOL_REVENUE_UNSUPPORTED", "this carpool type does not support revenue pool"))
	}
	if session == nil || participant.SessionID == nil {
		return fail("waiting_session", "carpool session is not assigned yet", infraerrors.BadRequest("CARPOOL_REVENUE_SESSION_MISSING", "carpool session is not assigned yet"))
	}
	if session.Status != CarpoolSessionActive {
		return fail("session_not_active", "carpool is not active yet", infraerrors.BadRequest("CARPOOL_REVENUE_SESSION_NOT_ACTIVE", "carpool is not active yet"))
	}
	if participant.Status != CarpoolParticipantPaid && participant.Status != CarpoolParticipantActive {
		return fail("participant_not_active", "carpool record cannot join revenue pool", infraerrors.BadRequest("CARPOOL_REVENUE_PARTICIPANT_INVALID", "carpool record cannot join revenue pool"))
	}
	groupID, _, err := s.carpoolAssignedGroupIDs(ctx, participant, session)
	if err != nil {
		return cfg, participant, session, vt, 0, nil, "subscription_group_error", err
	}
	if groupID <= 0 {
		return fail("subscription_group_missing", "subscription group is not assigned yet", infraerrors.BadRequest("CARPOOL_REVENUE_GROUP_MISSING", "subscription group is not assigned yet"))
	}
	sub, err := s.entClient.UserSubscription.Query().
		Where(
			usersubscription.UserIDEQ(userID),
			usersubscription.GroupIDEQ(groupID),
			usersubscription.StatusEQ(SubscriptionStatusActive),
			usersubscription.ExpiresAtGT(time.Now()),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return fail("subscription_missing", "active subscription is not available", infraerrors.BadRequest("CARPOOL_REVENUE_SUBSCRIPTION_MISSING", "active subscription is not available"))
		}
		return cfg, participant, session, vt, groupID, nil, "subscription_error", err
	}
	return cfg, participant, session, vt, groupID, sub, "available", nil
}

func (s *CarpoolService) getContributionByParticipant(ctx context.Context, participantID int64) (*CarpoolRevenueContribution, error) {
	return s.getContribution(ctx, `WHERE participant_id = $1`, participantID)
}

func (s *CarpoolService) getContributionByID(ctx context.Context, id int64) (*CarpoolRevenueContribution, error) {
	return s.getContribution(ctx, `WHERE id = $1`, id)
}

func (s *CarpoolService) getContribution(ctx context.Context, where string, arg any) (*CarpoolRevenueContribution, error) {
	var contribution CarpoolRevenueContribution
	err := s.scanOne(ctx, `
		SELECT id, participant_id, user_id, session_id, vehicle_type_id, subscription_id, subscription_group_id,
		       enabled, enabled_at, disabled_at, share_ratio, status, paused_reason, last_settled_at, notes, created_at, updated_at
		FROM carpool_revenue_contributions
		`+where, []any{arg},
		&contribution.ID,
		&contribution.ParticipantID,
		&contribution.UserID,
		&contribution.SessionID,
		&contribution.VehicleTypeID,
		newNullInt64Scanner(&contribution.SubscriptionID),
		&contribution.SubscriptionGroupID,
		&contribution.Enabled,
		newNullTimeScanner(&contribution.EnabledAt),
		newNullTimeScanner(&contribution.DisabledAt),
		&contribution.ShareRatio,
		&contribution.Status,
		&contribution.PausedReason,
		newNullTimeScanner(&contribution.LastSettledAt),
		&contribution.Notes,
		&contribution.CreatedAt,
		&contribution.UpdatedAt,
	)
	if err != nil {
		if errorsIsSQLNoRows(err) {
			return nil, err
		}
		return nil, err
	}
	return &contribution, nil
}

func (s *CarpoolService) revenueSummary(ctx context.Context, userID, participantID int64) (CarpoolRevenueSummary, error) {
	var participantArg any
	if participantID > 0 {
		participantArg = participantID
	}
	var summary CarpoolRevenueSummary
	var settled, withdrawn float64
	err := s.scanOne(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'settled' THEN user_share_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'pending' THEN user_share_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'frozen' THEN user_share_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'settled' THEN quota_cost ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'settled' THEN request_count ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'settled' THEN platform_share_amount ELSE 0 END), 0),
			COALESCE((
				SELECT SUM(amount)
				FROM carpool_revenue_withdrawals w
				WHERE w.user_id = $1
				  AND w.status = 'completed'
				  AND ($2::bigint IS NULL OR w.participant_id = $2::bigint)
			), 0)
		FROM carpool_revenue_records r
		WHERE r.user_id = $1
		  AND ($2::bigint IS NULL OR r.participant_id = $2::bigint)
	`, []any{userID, participantArg},
		&settled,
		&summary.PendingRevenue,
		&summary.FrozenRevenue,
		&summary.QuotaCost,
		&summary.RequestCount,
		&summary.PlatformShareAmount,
		&withdrawn,
	)
	if err != nil {
		return summary, err
	}
	summary.TotalRevenue = settled
	summary.WithdrawnRevenue = withdrawn
	summary.AvailableRevenue = math.Max(0, settled-withdrawn)
	return summary, nil
}

func (s *CarpoolService) listRevenueRecords(ctx context.Context, userID, participantID int64, limit int) ([]CarpoolRevenueRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var participantArg any
	if participantID > 0 {
		participantArg = participantID
	}
	rows, err := s.entClient.QueryContext(ctx, `
		SELECT id, contribution_id, participant_id, session_id, user_id, subscription_group_id, api_key_id, usage_log_id,
		       request_id, request_count, quota_cost, gross_revenue, upstream_cost, net_revenue, user_share_amount,
		       platform_share_amount, settlement_period, status, occurred_at, settled_at, notes, created_at
		FROM carpool_revenue_records
		WHERE user_id = $1
		  AND ($2::bigint IS NULL OR participant_id = $2::bigint)
		ORDER BY occurred_at DESC, id DESC
		LIMIT $3
	`, userID, participantArg, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	records := []CarpoolRevenueRecord{}
	for rows.Next() {
		var record CarpoolRevenueRecord
		if err := rows.Scan(
			&record.ID, &record.ContributionID, &record.ParticipantID, &record.SessionID, &record.UserID, &record.SubscriptionGroupID,
			newNullInt64Scanner(&record.APIKeyID), newNullInt64Scanner(&record.UsageLogID), &record.RequestID, &record.RequestCount, &record.QuotaCost,
			&record.GrossRevenue, &record.UpstreamCost, &record.NetRevenue, &record.UserShareAmount, &record.PlatformShareAmount,
			&record.SettlementPeriod, &record.Status, &record.OccurredAt, newNullTimeScanner(&record.SettledAt), &record.Notes, &record.CreatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *CarpoolService) listRevenueWithdrawals(ctx context.Context, userID, participantID int64, limit int) ([]CarpoolRevenueWithdrawal, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var participantArg any
	if participantID > 0 {
		participantArg = participantID
	}
	rows, err := s.entClient.QueryContext(ctx, `
		SELECT id, user_id, participant_id, session_id, amount, available_before, available_after,
		       balance_before, balance_after, status, requested_at, processed_at, failure_reason, created_at
		FROM carpool_revenue_withdrawals
		WHERE user_id = $1
		  AND ($2::bigint IS NULL OR participant_id = $2::bigint)
		ORDER BY requested_at DESC, id DESC
		LIMIT $3
	`, userID, participantArg, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	withdrawals := []CarpoolRevenueWithdrawal{}
	for rows.Next() {
		var item CarpoolRevenueWithdrawal
		var participantNull sql.NullInt64
		var sessionNull sql.NullInt64
		var balanceBefore sql.NullFloat64
		var balanceAfter sql.NullFloat64
		var processedAt sql.NullTime
		if err := rows.Scan(
			&item.ID, &item.UserID, &participantNull, &sessionNull, &item.Amount, &item.AvailableBefore, &item.AvailableAfter,
			&balanceBefore, &balanceAfter, &item.Status, &item.RequestedAt, &processedAt, &item.FailureReason, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.ParticipantID = int64PtrFromNull(participantNull)
		item.SessionID = int64PtrFromNull(sessionNull)
		item.BalanceBefore = float64PtrFromNull(balanceBefore)
		item.BalanceAfter = float64PtrFromNull(balanceAfter)
		item.ProcessedAt = timePtrFromNull(processedAt)
		withdrawals = append(withdrawals, item)
	}
	return withdrawals, rows.Err()
}

type scanRow interface {
	Scan(dest ...any) error
}

func (s *CarpoolService) scanOne(ctx context.Context, query string, args []any, dest ...any) error {
	rows, err := s.entClient.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	if err := rows.Scan(dest...); err != nil {
		return err
	}
	return rows.Err()
}

func scanRevenueContribution(row scanRow, c *CarpoolRevenueContribution, email, username, sessionNo, sessionStatus, vehicleName *string, settled, pending, frozen, quotaCost *float64, requestCount *int64, platformShare, withdrawn *float64) error {
	return row.Scan(
		&c.ID,
		&c.ParticipantID,
		&c.UserID,
		&c.SessionID,
		&c.VehicleTypeID,
		newNullInt64Scanner(&c.SubscriptionID),
		&c.SubscriptionGroupID,
		&c.Enabled,
		newNullTimeScanner(&c.EnabledAt),
		newNullTimeScanner(&c.DisabledAt),
		&c.ShareRatio,
		&c.Status,
		&c.PausedReason,
		newNullTimeScanner(&c.LastSettledAt),
		&c.Notes,
		&c.CreatedAt,
		&c.UpdatedAt,
		email,
		username,
		sessionNo,
		sessionStatus,
		vehicleName,
		settled,
		pending,
		frozen,
		quotaCost,
		requestCount,
		platformShare,
		withdrawn,
	)
}

type nullInt64Scanner struct {
	target **int64
}

func newNullInt64Scanner(target **int64) sql.Scanner {
	return &nullInt64Scanner{target: target}
}

func (s *nullInt64Scanner) Scan(value any) error {
	var v sql.NullInt64
	if err := v.Scan(value); err != nil {
		return err
	}
	*s.target = int64PtrFromNull(v)
	return nil
}

type nullTimeScanner struct {
	target **time.Time
}

func newNullTimeScanner(target **time.Time) sql.Scanner {
	return &nullTimeScanner{target: target}
}

func (s *nullTimeScanner) Scan(value any) error {
	var v sql.NullTime
	if err := v.Scan(value); err != nil {
		return err
	}
	*s.target = timePtrFromNull(v)
	return nil
}

func int64PtrFromNull(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	out := v.Int64
	return &out
}

func float64PtrFromNull(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	out := v.Float64
	return &out
}

func timePtrFromNull(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	out := v.Time
	return &out
}

func errorsIsSQLNoRows(err error) bool {
	return err == sql.ErrNoRows || dbent.IsNotFound(err)
}

func revenueConfigUserShare(cfg *CarpoolRevenueConfig) float64 {
	if cfg == nil || cfg.UserShareRatio <= 0 {
		return 0.7
	}
	if cfg.UserShareRatio > 1 {
		return 1
	}
	return cfg.UserShareRatio
}

func normalizeCSV(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == '，' || r == ';' || r == '；'
	})
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		normalized = append(normalized, item)
	}
	return strings.Join(normalized, ",")
}

func csvAllowsInt64(csv string, value int64) bool {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return true
	}
	target := strconv.FormatInt(value, 10)
	for _, item := range strings.Split(normalizeCSV(csv), ",") {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}

func modelPatternsAllow(patterns string, model string) bool {
	patterns = strings.TrimSpace(patterns)
	if patterns == "" {
		return true
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	for _, pattern := range strings.Split(normalizeCSV(patterns), ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if matchModelPattern(pattern, model) {
			return true
		}
	}
	return false
}

func percentHit(percent float64) bool {
	if percent <= 0 {
		return false
	}
	if percent >= 100 {
		return true
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return false
	}
	return float64(n.Int64()) < percent*10000
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
	if phone := strings.TrimSpace(u.Phone); phone != "" {
		return maskLoginAccount(phone)
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
	existing, err := s.entClient.CarpoolSession.Query().
		Where(carpoolsession.IDEQ(id)).
		WithVehicleType().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	up := s.entClient.CarpoolSession.UpdateOneID(id).
		SetStatus(status).
		SetAccountInfo(nonNilMap(input.AccountInfo)).
		SetProxyInfo(nonNilMap(input.ProxyInfo)).
		SetCommunication(nonNilMap(input.Communication)).
		SetAdminNotes(strings.TrimSpace(input.AdminNotes))
	now := time.Now()
	if status == CarpoolSessionActive {
		serviceStartedAt := now
		if existing.ServiceStartedAt != nil && !existing.ServiceStartedAt.IsZero() {
			serviceStartedAt = *existing.ServiceStartedAt
		} else {
			up.SetServiceStartedAt(serviceStartedAt)
		}
		if existing.ProvisionedAt == nil || existing.ProvisionedAt.IsZero() {
			up.SetProvisionedAt(now)
		}
		serviceDays := int(numberFromMap(input.AccountInfo, "subscription_validity_days"))
		if serviceDays <= 0 && existing.Edges.VehicleType != nil {
			serviceDays = existing.Edges.VehicleType.ServiceDays
		}
		if serviceDays <= 0 {
			serviceDays = 30
		}
		up.SetServiceEndedAt(serviceStartedAt.AddDate(0, 0, serviceDays))
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
4. 中转投入计划按车类型开放；如本车支持，发车后用户可自主选择是否将自己的订阅额度投入中转，不影响其他成员。
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
