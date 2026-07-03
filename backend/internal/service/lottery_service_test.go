package service

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestLotteryEnsureUserChancesBestEffortGrantsDailyLogin(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	svc := NewLotteryService(db, nil)
	cfg := DefaultLotteryConfig()
	cfg.Enabled = true
	cfg.LoginGrant.Enabled = true
	cfg.LoginGrant.DailyChances = 1
	cfg.SpendGrant.Enabled = false
	cfg.RechargeGrant.Enabled = false

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO lottery_chances")).
		WithArgs(int64(42), LotterySourceDailyLogin, sqlmock.AnyArg(), 1, 0.0, sqlmock.AnyArg(), sqlmock.AnyArg(), "每日登录赠送").
		WillReturnResult(sqlmock.NewResult(1, 1))

	svc.ensureUserChancesBestEffort(context.Background(), 42, cfg)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryEnsureDailyLoginChanceSkipsWhenDisabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := NewLotteryService(db, nil)
	cfg := DefaultLotteryConfig()
	cfg.Enabled = true
	cfg.LoginGrant.Enabled = false

	require.NoError(t, svc.ensureDailyLoginChance(context.Background(), 42, cfg))
	require.NoError(t, mock.ExpectationsWereMet())
}
