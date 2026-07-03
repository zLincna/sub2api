package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	LotterySourceDailyLogin = "daily_login"
	LotterySourceSpend      = "spend"
	LotterySourceRecharge   = "recharge"

	lotteryDefaultTimezone = "Asia/Shanghai"
)

var (
	ErrLotteryDisabled    = infraerrors.Forbidden("LOTTERY_DISABLED", "抽奖中心暂未开启")
	ErrLotteryNoChance    = infraerrors.Conflict("LOTTERY_NO_CHANCE", "暂无可用抽奖次数")
	ErrLotteryNoPrize     = infraerrors.Conflict("LOTTERY_NO_PRIZE", "暂无可用奖品")
	ErrLotteryInvalidRule = infraerrors.BadRequest("LOTTERY_INVALID_RULE", "抽奖配置不正确")
)

type LotteryThresholdRule struct {
	Amount  float64 `json:"amount"`
	Chances int     `json:"chances"`
}

type LotteryLoginGrantConfig struct {
	Enabled      bool   `json:"enabled"`
	DailyChances int    `json:"daily_chances"`
	ExpiryMode   string `json:"expiry_mode"`
	ExpiryHours  int    `json:"expiry_hours"`
}

type LotteryThresholdGrantConfig struct {
	Enabled     bool                   `json:"enabled"`
	Thresholds  []LotteryThresholdRule `json:"thresholds"`
	ExpiryMode  string                 `json:"expiry_mode"`
	ExpiryHours int                    `json:"expiry_hours"`
}

type LotteryConfig struct {
	Enabled       bool                        `json:"enabled"`
	ButtonEnabled bool                        `json:"button_enabled"`
	Timezone      string                      `json:"timezone"`
	RuleText      string                      `json:"rule_text"`
	LoginGrant    LotteryLoginGrantConfig     `json:"login_grant"`
	SpendGrant    LotteryThresholdGrantConfig `json:"spend_grant"`
	RechargeGrant LotteryThresholdGrantConfig `json:"recharge_grant"`
}

type LotteryPrize struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Probability float64   `json:"probability"`
	DailyStock  int       `json:"daily_stock"`
	DailyUsed   int       `json:"daily_used"`
	TotalStock  int       `json:"total_stock"`
	TotalUsed   int       `json:"total_used"`
	Enabled     bool      `json:"enabled"`
	Color       string    `json:"color"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type LotteryChanceSummary struct {
	Remaining int            `json:"remaining"`
	Granted   int            `json:"granted"`
	Used      int            `json:"used"`
	Expired   int            `json:"expired"`
	BySource  map[string]int `json:"by_source"`
}

type LotteryDrawRecord struct {
	ID               int64     `json:"id"`
	UserID           int64     `json:"user_id"`
	UserEmail        string    `json:"user_email,omitempty"`
	PrizeID          int64     `json:"prize_id"`
	PrizeName        string    `json:"prize_name"`
	PrizeDescription string    `json:"prize_description"`
	Amount           float64   `json:"amount"`
	BalanceBefore    float64   `json:"balance_before"`
	BalanceAfter     float64   `json:"balance_after"`
	SourceType       string    `json:"source_type"`
	CreatedAt        time.Time `json:"created_at"`
}

type LotteryStatus struct {
	Config      LotteryConfig        `json:"config"`
	Prizes      []LotteryPrize       `json:"prizes"`
	Chances     LotteryChanceSummary `json:"chances"`
	RecentDraws []LotteryDrawRecord  `json:"recent_draws"`
	ServerTime  time.Time            `json:"server_time"`
}

type LotteryDrawResult struct {
	Prize            LotteryPrize      `json:"prize"`
	Record           LotteryDrawRecord `json:"record"`
	RemainingChances int               `json:"remaining_chances"`
}

type LotteryPrizeInput struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Probability float64 `json:"probability"`
	DailyStock  int     `json:"daily_stock"`
	TotalStock  int     `json:"total_stock"`
	Enabled     bool    `json:"enabled"`
	Color       string  `json:"color"`
	SortOrder   int     `json:"sort_order"`
}

type LotteryDrawRecordFilters struct {
	UserID     int64
	UserQuery  string
	SourceType string
	StartTime  time.Time
	EndTime    time.Time
}

type LotteryService struct {
	db          *sql.DB
	settingRepo SettingRepository
}

func NewLotteryService(db *sql.DB, settingRepo SettingRepository) *LotteryService {
	return &LotteryService{db: db, settingRepo: settingRepo}
}

func DefaultLotteryConfig() LotteryConfig {
	return LotteryConfig{
		Enabled:       false,
		ButtonEnabled: true,
		Timezone:      lotteryDefaultTimezone,
		RuleText:      "每日登录可获得抽奖次数；消费或充值达到后台配置阶梯后，可额外获得抽奖次数。中奖余额将直接发放到账户余额。",
		LoginGrant: LotteryLoginGrantConfig{
			Enabled:      true,
			DailyChances: 1,
			ExpiryMode:   "end_of_day",
			ExpiryHours:  24,
		},
		SpendGrant: LotteryThresholdGrantConfig{
			Enabled: false,
			Thresholds: []LotteryThresholdRule{
				{Amount: 1, Chances: 1},
				{Amount: 5, Chances: 2},
				{Amount: 10, Chances: 3},
			},
			ExpiryMode:  "end_of_day",
			ExpiryHours: 24,
		},
		RechargeGrant: LotteryThresholdGrantConfig{
			Enabled: false,
			Thresholds: []LotteryThresholdRule{
				{Amount: 10, Chances: 1},
				{Amount: 50, Chances: 3},
				{Amount: 100, Chances: 8},
			},
			ExpiryMode:  "hours",
			ExpiryHours: 168,
		},
	}
}

func (s *LotteryService) GetConfig(ctx context.Context) (LotteryConfig, error) {
	cfg := DefaultLotteryConfig()
	if s == nil || s.settingRepo == nil {
		return cfg, nil
	}
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyLotteryEnabled,
		SettingKeyLotteryButtonEnabled,
		SettingKeyLotteryTimezone,
		SettingKeyLotteryRuleText,
		SettingKeyLotteryLoginGrantConfig,
		SettingKeyLotterySpendGrantConfig,
		SettingKeyLotteryRechargeGrantConfig,
	})
	if err != nil {
		return cfg, err
	}
	cfg.Enabled = parseBoolSetting(values[SettingKeyLotteryEnabled], cfg.Enabled)
	cfg.ButtonEnabled = parseBoolSetting(values[SettingKeyLotteryButtonEnabled], cfg.ButtonEnabled)
	if tz := strings.TrimSpace(values[SettingKeyLotteryTimezone]); tz != "" {
		cfg.Timezone = tz
	}
	if text := strings.TrimSpace(values[SettingKeyLotteryRuleText]); text != "" {
		cfg.RuleText = text
	}
	decodeJSONSetting(values[SettingKeyLotteryLoginGrantConfig], &cfg.LoginGrant)
	decodeJSONSetting(values[SettingKeyLotterySpendGrantConfig], &cfg.SpendGrant)
	decodeJSONSetting(values[SettingKeyLotteryRechargeGrantConfig], &cfg.RechargeGrant)
	normalizeLotteryConfig(&cfg)
	return cfg, nil
}

func (s *LotteryService) UpdateConfig(ctx context.Context, cfg LotteryConfig) (LotteryConfig, error) {
	normalizeLotteryConfig(&cfg)
	if s == nil || s.settingRepo == nil {
		return cfg, errors.New("nil lottery settings repository")
	}
	loginJSON, _ := json.Marshal(cfg.LoginGrant)
	spendJSON, _ := json.Marshal(cfg.SpendGrant)
	rechargeJSON, _ := json.Marshal(cfg.RechargeGrant)
	if err := s.settingRepo.SetMultiple(ctx, map[string]string{
		SettingKeyLotteryEnabled:             fmt.Sprintf("%t", cfg.Enabled),
		SettingKeyLotteryButtonEnabled:       fmt.Sprintf("%t", cfg.ButtonEnabled),
		SettingKeyLotteryTimezone:            cfg.Timezone,
		SettingKeyLotteryRuleText:            cfg.RuleText,
		SettingKeyLotteryLoginGrantConfig:    string(loginJSON),
		SettingKeyLotterySpendGrantConfig:    string(spendJSON),
		SettingKeyLotteryRechargeGrantConfig: string(rechargeJSON),
	}); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (s *LotteryService) GetStatus(ctx context.Context, userID int64) (*LotteryStatus, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.Enabled {
		s.ensureUserChancesBestEffort(ctx, userID, cfg)
	}
	prizes, err := s.ListPrizes(ctx, true)
	if err != nil {
		return nil, err
	}
	chances, err := s.GetChanceSummary(ctx, userID)
	if err != nil {
		return nil, err
	}
	records, _, err := s.ListUserDrawRecords(ctx, userID, 1, 10)
	if err != nil {
		return nil, err
	}
	return &LotteryStatus{
		Config:      cfg,
		Prizes:      prizes,
		Chances:     chances,
		RecentDraws: records,
		ServerTime:  time.Now(),
	}, nil
}

func (s *LotteryService) GrantDailyLogin(ctx context.Context, userID int64) error {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return err
	}
	return s.ensureDailyLoginChance(ctx, userID, cfg)
}

func (s *LotteryService) ensureDailyLoginChance(ctx context.Context, userID int64, cfg LotteryConfig) error {
	if !cfg.Enabled || !cfg.LoginGrant.Enabled || cfg.LoginGrant.DailyChances <= 0 {
		return nil
	}
	loc := lotteryLocation(cfg.Timezone)
	now := time.Now().In(loc)
	grantDate := localDate(now)
	expiresAt := expiryFor(cfg.LoginGrant.ExpiryMode, cfg.LoginGrant.ExpiryHours, now, loc)
	return s.grantChance(ctx, userID, LotterySourceDailyLogin, "login:"+grantDate, cfg.LoginGrant.DailyChances, 0, grantDate, expiresAt, "每日登录赠送")
}

func (s *LotteryService) GrantRechargeForOrder(ctx context.Context, orderID, userID int64, amount float64, completedAt time.Time) error {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return err
	}
	if !cfg.Enabled || !cfg.RechargeGrant.Enabled {
		return nil
	}
	chances := matchingThresholdChances(amount, cfg.RechargeGrant.Thresholds)
	if chances <= 0 {
		return nil
	}
	loc := lotteryLocation(cfg.Timezone)
	localCompleted := completedAt.In(loc)
	grantDate := localDate(localCompleted)
	expiresAt := expiryFor(cfg.RechargeGrant.ExpiryMode, cfg.RechargeGrant.ExpiryHours, localCompleted, loc)
	sourceKey := fmt.Sprintf("recharge:%d", orderID)
	return s.grantChance(ctx, userID, LotterySourceRecharge, sourceKey, chances, amount, grantDate, expiresAt, "充值赠送")
}

func (s *LotteryService) Draw(ctx context.Context, userID int64) (*LotteryDrawResult, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, ErrLotteryDisabled
	}
	s.ensureUserChancesBestEffort(ctx, userID, cfg)

	loc := lotteryLocation(cfg.Timezone)
	now := time.Now()
	dayStart, dayEnd := dayRange(now.In(loc), loc)

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var chanceID int64
	var sourceType string
	err = tx.QueryRowContext(ctx, `
		SELECT id, source_type
		FROM lottery_chances
		WHERE user_id = $1
		  AND total_count > used_count
		  AND expires_at > NOW()
		ORDER BY expires_at ASC, id ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, userID).Scan(&chanceID, &sourceType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLotteryNoChance
		}
		return nil, err
	}

	prizes, err := queryAvailablePrizes(ctx, tx, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}
	if len(prizes) == 0 {
		return nil, ErrLotteryNoPrize
	}
	prize, err := pickWeightedPrize(prizes)
	if err != nil {
		return nil, err
	}

	var balanceBefore float64
	if err = tx.QueryRowContext(ctx, `SELECT balance FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&balanceBefore); err != nil {
		return nil, err
	}
	balanceAfter := math.Round((balanceBefore+prize.Amount)*100000000) / 100000000

	if _, err = tx.ExecContext(ctx, `UPDATE lottery_chances SET used_count = used_count + 1, updated_at = NOW() WHERE id = $1`, chanceID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE lottery_prizes SET total_used = total_used + 1, daily_used = $2, updated_at = NOW() WHERE id = $1`, prize.ID, prize.DailyUsed+1); err != nil {
		return nil, err
	}
	if prize.Amount > 0 {
		if _, err = tx.ExecContext(ctx, `UPDATE users SET balance = $2, updated_at = NOW() WHERE id = $1`, userID, balanceAfter); err != nil {
			return nil, err
		}
		if err = insertLotteryRedeemHistory(ctx, tx, userID, prize.Amount, prize.Name); err != nil {
			return nil, err
		}
	}
	snapshot, _ := json.Marshal(map[string]any{
		"prize_probability": prize.Probability,
		"prize_description": prize.Description,
		"source_type":       sourceType,
	})
	var record LotteryDrawRecord
	err = tx.QueryRowContext(ctx, `
		INSERT INTO lottery_draw_records (user_id, chance_id, prize_id, prize_name, prize_description, amount, balance_before, balance_after, source_type, config_snapshot, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, NOW())
		RETURNING id, user_id, prize_id, prize_name, prize_description, amount, balance_before, balance_after, source_type, created_at
	`, userID, chanceID, prize.ID, prize.Name, prize.Description, prize.Amount, balanceBefore, balanceAfter, sourceType, string(snapshot)).
		Scan(&record.ID, &record.UserID, &record.PrizeID, &record.PrizeName, &record.PrizeDescription, &record.Amount, &record.BalanceBefore, &record.BalanceAfter, &record.SourceType, &record.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil

	remaining, _ := s.countRemainingChances(ctx, userID)
	prize.DailyUsed++
	prize.TotalUsed++
	return &LotteryDrawResult{Prize: prize, Record: record, RemainingChances: remaining}, nil
}

func (s *LotteryService) GetChanceSummary(ctx context.Context, userID int64) (LotteryChanceSummary, error) {
	summary := LotteryChanceSummary{BySource: map[string]int{}}
	rows, err := s.db.QueryContext(ctx, `
		SELECT source_type,
		       COALESCE(SUM(total_count), 0)::int,
		       COALESCE(SUM(used_count), 0)::int,
		       COALESCE(SUM(CASE WHEN expires_at <= NOW() THEN GREATEST(total_count - used_count, 0) ELSE 0 END), 0)::int,
		       COALESCE(SUM(CASE WHEN expires_at > NOW() THEN GREATEST(total_count - used_count, 0) ELSE 0 END), 0)::int
		FROM lottery_chances
		WHERE user_id = $1
		GROUP BY source_type
	`, userID)
	if err != nil {
		return summary, err
	}
	defer rows.Close()
	for rows.Next() {
		var source string
		var granted, used, expired, remaining int
		if err := rows.Scan(&source, &granted, &used, &expired, &remaining); err != nil {
			return summary, err
		}
		summary.Granted += granted
		summary.Used += used
		summary.Expired += expired
		summary.Remaining += remaining
		summary.BySource[source] = remaining
	}
	return summary, rows.Err()
}

func (s *LotteryService) ListPrizes(ctx context.Context, enabledOnly bool) ([]LotteryPrize, error) {
	cfg, _ := s.GetConfig(ctx)
	loc := lotteryLocation(cfg.Timezone)
	dayStart, dayEnd := dayRange(time.Now().In(loc), loc)
	query := `
		SELECT p.id, p.name, p.description, p.amount, p.probability, p.daily_stock,
		       COALESCE(d.today_used, 0)::int AS daily_used,
		       p.total_stock, p.total_used, p.enabled, p.color, p.sort_order, p.created_at, p.updated_at
		FROM lottery_prizes p
		LEFT JOIN (
			SELECT prize_id, COUNT(*) AS today_used
			FROM lottery_draw_records
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY prize_id
		) d ON d.prize_id = p.id`
	if enabledOnly {
		query += ` WHERE p.enabled = TRUE`
	}
	query += ` ORDER BY p.sort_order ASC, p.id ASC`
	rows, err := s.db.QueryContext(ctx, query, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLotteryPrizes(rows)
}

func (s *LotteryService) CreatePrize(ctx context.Context, input LotteryPrizeInput) (*LotteryPrize, error) {
	normalizePrizeInput(&input)
	var prize LotteryPrize
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO lottery_prizes (name, description, amount, probability, daily_stock, total_stock, enabled, color, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, name, description, amount, probability, daily_stock, daily_used, total_stock, total_used, enabled, color, sort_order, created_at, updated_at
	`, input.Name, input.Description, input.Amount, input.Probability, input.DailyStock, input.TotalStock, input.Enabled, input.Color, input.SortOrder).
		Scan(&prize.ID, &prize.Name, &prize.Description, &prize.Amount, &prize.Probability, &prize.DailyStock, &prize.DailyUsed, &prize.TotalStock, &prize.TotalUsed, &prize.Enabled, &prize.Color, &prize.SortOrder, &prize.CreatedAt, &prize.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &prize, nil
}

func (s *LotteryService) UpdatePrize(ctx context.Context, id int64, input LotteryPrizeInput) (*LotteryPrize, error) {
	normalizePrizeInput(&input)
	var prize LotteryPrize
	err := s.db.QueryRowContext(ctx, `
		UPDATE lottery_prizes
		SET name = $2, description = $3, amount = $4, probability = $5, daily_stock = $6, total_stock = $7,
		    enabled = $8, color = $9, sort_order = $10, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, description, amount, probability, daily_stock, daily_used, total_stock, total_used, enabled, color, sort_order, created_at, updated_at
	`, id, input.Name, input.Description, input.Amount, input.Probability, input.DailyStock, input.TotalStock, input.Enabled, input.Color, input.SortOrder).
		Scan(&prize.ID, &prize.Name, &prize.Description, &prize.Amount, &prize.Probability, &prize.DailyStock, &prize.DailyUsed, &prize.TotalStock, &prize.TotalUsed, &prize.Enabled, &prize.Color, &prize.SortOrder, &prize.CreatedAt, &prize.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.NotFound("LOTTERY_PRIZE_NOT_FOUND", "奖品不存在")
		}
		return nil, err
	}
	return &prize, nil
}

func (s *LotteryService) DeletePrize(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM lottery_prizes WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return infraerrors.NotFound("LOTTERY_PRIZE_NOT_FOUND", "奖品不存在")
	}
	return nil
}

func (s *LotteryService) ListUserDrawRecords(ctx context.Context, userID int64, page, pageSize int) ([]LotteryDrawRecord, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lottery_draw_records WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, prize_id, prize_name, prize_description, amount, balance_before, balance_after, source_type, created_at
		FROM lottery_draw_records
		WHERE user_id = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3
	`, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	records, err := scanDrawRecords(rows, false)
	return records, total, err
}

func (s *LotteryService) ListAdminDrawRecords(ctx context.Context, page, pageSize int, filters LotteryDrawRecordFilters) ([]LotteryDrawRecord, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	whereParts := make([]string, 0, 5)
	args := make([]any, 0, 7)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filters.UserID > 0 {
		whereParts = append(whereParts, "r.user_id = "+addArg(filters.UserID))
	}
	if query := strings.TrimSpace(filters.UserQuery); query != "" {
		placeholder := addArg("%" + strings.ToLower(query) + "%")
		whereParts = append(whereParts, "(LOWER(COALESCE(u.email, '')) LIKE "+placeholder+" OR CAST(r.user_id AS TEXT) LIKE "+placeholder+")")
	}
	if sourceType := strings.TrimSpace(filters.SourceType); sourceType != "" {
		whereParts = append(whereParts, "r.source_type = "+addArg(sourceType))
	}
	if !filters.StartTime.IsZero() {
		whereParts = append(whereParts, "r.created_at >= "+addArg(filters.StartTime))
	}
	if !filters.EndTime.IsZero() {
		whereParts = append(whereParts, "r.created_at <= "+addArg(filters.EndTime))
	}
	where := ""
	if len(whereParts) > 0 {
		where = "WHERE " + strings.Join(whereParts, " AND ")
	}
	var total int64
	countQuery := `
		SELECT COUNT(*)
		FROM lottery_draw_records r
		LEFT JOIN users u ON u.id = r.user_id
		` + where
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	limitPos := len(args) - 1
	offsetPos := len(args)
	query := fmt.Sprintf(`
		SELECT r.id, r.user_id, COALESCE(u.email, ''), r.prize_id, r.prize_name, r.prize_description, r.amount, r.balance_before, r.balance_after, r.source_type, r.created_at
		FROM lottery_draw_records r
		LEFT JOIN users u ON u.id = r.user_id
		%s
		ORDER BY r.id DESC
		LIMIT $%d OFFSET $%d
	`, where, limitPos, offsetPos)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	records, err := scanDrawRecords(rows, true)
	return records, total, err
}

func (s *LotteryService) ensureUserChancesBestEffort(ctx context.Context, userID int64, cfg LotteryConfig) {
	if err := s.ensureDailyLoginChance(ctx, userID, cfg); err != nil {
		logger.LegacyPrintf("service.lottery", "[Lottery] daily login grant failed for user %d: %v", userID, err)
	}
	if err := s.ensureSpendChances(ctx, userID, cfg); err != nil {
		logger.LegacyPrintf("service.lottery", "[Lottery] spend grant failed for user %d: %v", userID, err)
	}
	if err := s.ensureRechargeChances(ctx, userID, cfg); err != nil {
		logger.LegacyPrintf("service.lottery", "[Lottery] recharge grant failed for user %d: %v", userID, err)
	}
}

func (s *LotteryService) ensureSpendChances(ctx context.Context, userID int64, cfg LotteryConfig) error {
	if !cfg.Enabled || !cfg.SpendGrant.Enabled {
		return nil
	}
	loc := lotteryLocation(cfg.Timezone)
	now := time.Now().In(loc)
	start, end := dayRange(now, loc)
	var amount float64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(actual_cost), 0)
		FROM usage_logs
		WHERE user_id = $1 AND created_at >= $2 AND created_at < $3
	`, userID, start, end).Scan(&amount); err != nil {
		return err
	}
	chances := matchingThresholdChances(amount, cfg.SpendGrant.Thresholds)
	if chances <= 0 {
		return nil
	}
	grantDate := localDate(now)
	expiresAt := expiryFor(cfg.SpendGrant.ExpiryMode, cfg.SpendGrant.ExpiryHours, now, loc)
	return s.grantChance(ctx, userID, LotterySourceSpend, "spend:"+grantDate, chances, amount, grantDate, expiresAt, "消费赠送")
}

func (s *LotteryService) ensureRechargeChances(ctx context.Context, userID int64, cfg LotteryConfig) error {
	if !cfg.Enabled || !cfg.RechargeGrant.Enabled {
		return nil
	}
	loc := lotteryLocation(cfg.Timezone)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, pay_amount, COALESCE(completed_at, paid_at, created_at)
		FROM payment_orders
		WHERE user_id = $1
		  AND order_type = 'balance'
		  AND status = 'COMPLETED'
		  AND COALESCE(completed_at, paid_at, created_at) >= NOW() - INTERVAL '30 days'
		ORDER BY id DESC
	`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var orderID int64
		var amount float64
		var completedAt time.Time
		if err := rows.Scan(&orderID, &amount, &completedAt); err != nil {
			return err
		}
		chances := matchingThresholdChances(amount, cfg.RechargeGrant.Thresholds)
		if chances <= 0 {
			continue
		}
		localCompleted := completedAt.In(loc)
		grantDate := localDate(localCompleted)
		expiresAt := expiryFor(cfg.RechargeGrant.ExpiryMode, cfg.RechargeGrant.ExpiryHours, localCompleted, loc)
		if err := s.grantChance(ctx, userID, LotterySourceRecharge, fmt.Sprintf("recharge:%d", orderID), chances, amount, grantDate, expiresAt, "充值赠送"); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *LotteryService) grantChance(ctx context.Context, userID int64, sourceType, sourceKey string, count int, sourceAmount float64, grantDate string, expiresAt time.Time, notes string) error {
	if s == nil || s.db == nil || userID <= 0 || count <= 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO lottery_chances (user_id, source_type, source_key, total_count, used_count, source_amount, grant_date, expires_at, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 0, $5, $6::date, $7, $8, NOW(), NOW())
		ON CONFLICT (user_id, source_type, source_key)
		DO UPDATE SET
			total_count = GREATEST(lottery_chances.total_count, EXCLUDED.total_count),
			source_amount = GREATEST(lottery_chances.source_amount, EXCLUDED.source_amount),
			expires_at = GREATEST(lottery_chances.expires_at, EXCLUDED.expires_at),
			notes = EXCLUDED.notes,
			updated_at = NOW()
	`, userID, sourceType, sourceKey, count, sourceAmount, grantDate, expiresAt, notes)
	return err
}

func (s *LotteryService) countRemainingChances(ctx context.Context, userID int64) (int, error) {
	var remaining int
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(GREATEST(total_count - used_count, 0)), 0)::int
		FROM lottery_chances
		WHERE user_id = $1 AND expires_at > NOW()
	`, userID).Scan(&remaining)
	return remaining, err
}

func queryAvailablePrizes(ctx context.Context, tx *sql.Tx, dayStart, dayEnd time.Time) ([]LotteryPrize, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT p.id, p.name, p.description, p.amount, p.probability, p.daily_stock,
		       COALESCE(d.today_used, 0)::int AS daily_used,
		       p.total_stock, p.total_used, p.enabled, p.color, p.sort_order, p.created_at, p.updated_at
		FROM lottery_prizes p
		LEFT JOIN (
			SELECT prize_id, COUNT(*) AS today_used
			FROM lottery_draw_records
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY prize_id
		) d ON d.prize_id = p.id
		WHERE p.enabled = TRUE
		  AND p.probability > 0
		  AND (p.total_stock <= 0 OR p.total_used < p.total_stock)
		  AND (p.daily_stock <= 0 OR COALESCE(d.today_used, 0) < p.daily_stock)
		ORDER BY p.sort_order ASC, p.id ASC
		FOR UPDATE OF p
	`, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLotteryPrizes(rows)
}

func scanLotteryPrizes(rows *sql.Rows) ([]LotteryPrize, error) {
	prizes := []LotteryPrize{}
	for rows.Next() {
		var p LotteryPrize
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Amount, &p.Probability, &p.DailyStock, &p.DailyUsed, &p.TotalStock, &p.TotalUsed, &p.Enabled, &p.Color, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prizes = append(prizes, p)
	}
	return prizes, rows.Err()
}

func scanDrawRecords(rows *sql.Rows, withUser bool) ([]LotteryDrawRecord, error) {
	records := []LotteryDrawRecord{}
	for rows.Next() {
		var r LotteryDrawRecord
		var err error
		if withUser {
			err = rows.Scan(&r.ID, &r.UserID, &r.UserEmail, &r.PrizeID, &r.PrizeName, &r.PrizeDescription, &r.Amount, &r.BalanceBefore, &r.BalanceAfter, &r.SourceType, &r.CreatedAt)
		} else {
			err = rows.Scan(&r.ID, &r.UserID, &r.PrizeID, &r.PrizeName, &r.PrizeDescription, &r.Amount, &r.BalanceBefore, &r.BalanceAfter, &r.SourceType, &r.CreatedAt)
		}
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func pickWeightedPrize(prizes []LotteryPrize) (LotteryPrize, error) {
	total := 0.0
	for _, p := range prizes {
		if p.Probability > 0 {
			total += p.Probability
		}
	}
	if total <= 0 {
		return LotteryPrize{}, ErrLotteryNoPrize
	}
	scale := int64(math.Round(total * 1000000))
	n, err := rand.Int(rand.Reader, big.NewInt(scale))
	if err != nil {
		return LotteryPrize{}, err
	}
	target := float64(n.Int64()) / 1000000
	running := 0.0
	for _, p := range prizes {
		running += p.Probability
		if target < running {
			return p, nil
		}
	}
	return prizes[len(prizes)-1], nil
}

func insertLotteryRedeemHistory(ctx context.Context, tx *sql.Tx, userID int64, amount float64, prizeName string) error {
	code, err := randomLotteryRedeemCode()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO redeem_codes (code, type, value, status, used_by, used_at, notes, created_at, validity_days)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6, NOW(), 0)
	`, code, RedeemTypeLotteryBalance, amount, StatusUsed, userID, "抽奖中奖："+prizeName)
	return err
}

func randomLotteryRedeemCode() (string, error) {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "LOT" + strings.ToUpper(hex.EncodeToString(b)), nil
}

func parseBoolSetting(raw string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}

func decodeJSONSetting(raw string, out any) {
	if strings.TrimSpace(raw) == "" {
		return
	}
	_ = json.Unmarshal([]byte(raw), out)
}

func normalizeLotteryConfig(cfg *LotteryConfig) {
	if strings.TrimSpace(cfg.Timezone) == "" {
		cfg.Timezone = lotteryDefaultTimezone
	}
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		cfg.Timezone = lotteryDefaultTimezone
	}
	if strings.TrimSpace(cfg.RuleText) == "" {
		cfg.RuleText = DefaultLotteryConfig().RuleText
	}
	if cfg.LoginGrant.DailyChances < 0 {
		cfg.LoginGrant.DailyChances = 0
	}
	normalizeExpiry(&cfg.LoginGrant.ExpiryMode, &cfg.LoginGrant.ExpiryHours)
	normalizeThresholdConfig(&cfg.SpendGrant)
	normalizeThresholdConfig(&cfg.RechargeGrant)
}

func normalizeThresholdConfig(cfg *LotteryThresholdGrantConfig) {
	normalizeExpiry(&cfg.ExpiryMode, &cfg.ExpiryHours)
	out := make([]LotteryThresholdRule, 0, len(cfg.Thresholds))
	for _, rule := range cfg.Thresholds {
		if rule.Amount <= 0 || rule.Chances <= 0 {
			continue
		}
		out = append(out, rule)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Amount < out[j].Amount })
	cfg.Thresholds = out
}

func normalizeExpiry(mode *string, hours *int) {
	switch strings.TrimSpace(*mode) {
	case "hours", "end_of_day":
	default:
		*mode = "end_of_day"
	}
	if *hours <= 0 {
		*hours = 24
	}
	if *hours > 8760 {
		*hours = 8760
	}
}

func normalizePrizeInput(input *LotteryPrizeInput) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		input.Name = "余额奖励"
	}
	if len([]rune(input.Name)) > 100 {
		input.Name = string([]rune(input.Name)[:100])
	}
	input.Description = strings.TrimSpace(input.Description)
	if len([]rune(input.Description)) > 500 {
		input.Description = string([]rune(input.Description)[:500])
	}
	if input.Amount < 0 {
		input.Amount = 0
	}
	if input.Probability < 0 {
		input.Probability = 0
	}
	if input.DailyStock < 0 {
		input.DailyStock = 0
	}
	if input.TotalStock < 0 {
		input.TotalStock = 0
	}
	if strings.TrimSpace(input.Color) == "" {
		input.Color = "#f59e0b"
	}
}

func matchingThresholdChances(amount float64, rules []LotteryThresholdRule) int {
	chances := 0
	for _, rule := range rules {
		if amount+1e-9 >= rule.Amount && rule.Chances > chances {
			chances = rule.Chances
		}
	}
	return chances
}

func lotteryLocation(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation(lotteryDefaultTimezone)
	}
	if loc == nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return loc
}

func localDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func dayRange(t time.Time, loc *time.Location) (time.Time, time.Time) {
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 0, 1)
}

func expiryFor(mode string, hours int, base time.Time, loc *time.Location) time.Time {
	if mode == "hours" {
		return base.Add(time.Duration(hours) * time.Hour)
	}
	_, end := dayRange(base, loc)
	return end
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
