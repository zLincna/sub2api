package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateCreditedBalanceWithBonusRules(t *testing.T) {
	rules := []BalanceRechargeBonusRule{
		{Threshold: 100, Bonus: 10, Enabled: true},
		{Threshold: 200, Bonus: 25, Enabled: true},
		{Threshold: 500, Bonus: 80, Enabled: true},
	}

	require.Equal(t, 99.00, calculateCreditedBalance(99, 1, rules))
	require.Equal(t, 110.00, calculateCreditedBalance(100, 1, rules))
	require.Equal(t, 225.00, calculateCreditedBalance(200, 1, rules))
	require.Equal(t, 580.00, calculateCreditedBalance(500, 1, rules))
	require.Equal(t, 250.00, calculateCreditedBalance(100, 2.4, rules))
}

func TestMatchBalanceRechargeBonusIgnoresDisabledRules(t *testing.T) {
	rules := []BalanceRechargeBonusRule{
		{Threshold: 100, Bonus: 10, Enabled: true},
		{Threshold: 200, Bonus: 99, Enabled: false},
	}

	require.Equal(t, 10.00, matchBalanceRechargeBonus(300, rules))
}

func TestMatchBalanceRechargeBonusUsesHighestThresholdRegardlessOrder(t *testing.T) {
	rules := []BalanceRechargeBonusRule{
		{Threshold: 500, Bonus: 80, Enabled: true},
		{Threshold: 100, Bonus: 10, Enabled: true},
		{Threshold: 200, Bonus: 25, Enabled: true},
	}

	require.Equal(t, 25.00, matchBalanceRechargeBonus(300, rules))
}
