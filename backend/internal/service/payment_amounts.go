package service

import (
	"math"
	"sort"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/shopspring/decimal"
)

const defaultBalanceRechargeMultiplier = 1.0

func normalizeBalanceRechargeMultiplier(multiplier float64) float64 {
	if math.IsNaN(multiplier) || math.IsInf(multiplier, 0) || multiplier <= 0 {
		return defaultBalanceRechargeMultiplier
	}
	return multiplier
}

func calculateCreditedBalance(paymentAmount, multiplier float64, bonusRules []BalanceRechargeBonusRule) float64 {
	base := decimal.NewFromFloat(paymentAmount).
		Mul(decimal.NewFromFloat(normalizeBalanceRechargeMultiplier(multiplier))).
		Round(2)
	bonus := decimal.NewFromFloat(matchBalanceRechargeBonus(paymentAmount, bonusRules))
	return base.Add(bonus).Round(2).InexactFloat64()
}

func matchBalanceRechargeBonus(paymentAmount float64, rules []BalanceRechargeBonusRule) float64 {
	if paymentAmount <= 0 || len(rules) == 0 {
		return 0
	}
	var matchedThreshold float64
	var matchedBonus float64
	for _, rule := range rules {
		if !rule.Enabled || rule.Threshold <= 0 || rule.Bonus <= 0 {
			continue
		}
		if paymentAmount >= rule.Threshold && rule.Threshold >= matchedThreshold {
			matchedThreshold = rule.Threshold
			matchedBonus = rule.Bonus
		}
	}
	return decimal.NewFromFloat(matchedBonus).Round(2).InexactFloat64()
}

func sortBalanceRechargeBonusRules(rules []BalanceRechargeBonusRule) {
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].Threshold == rules[j].Threshold {
			return rules[i].Bonus < rules[j].Bonus
		}
		return rules[i].Threshold < rules[j].Threshold
	})
}

func calculateGatewayRefundAmount(orderAmount, payAmount, refundAmount float64, currency string) float64 {
	if orderAmount <= 0 || payAmount <= 0 || refundAmount <= 0 {
		return 0
	}
	fractionDigits := int32(payment.CurrencyMaxFractionDigits(currency))
	if math.Abs(refundAmount-orderAmount) <= paymentAmountToleranceForCurrency(currency) {
		return decimal.NewFromFloat(payAmount).Round(fractionDigits).InexactFloat64()
	}
	return decimal.NewFromFloat(payAmount).
		Mul(decimal.NewFromFloat(refundAmount)).
		Div(decimal.NewFromFloat(orderAmount)).
		Round(fractionDigits).
		InexactFloat64()
}
