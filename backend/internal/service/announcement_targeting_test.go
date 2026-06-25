package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnnouncementTargeting_Matches_EmptyMatchesAll(t *testing.T) {
	var targeting AnnouncementTargeting
	require.True(t, targeting.Matches(0, nil))
	require.True(t, targeting.Matches(123.45, map[int64]struct{}{1: {}}))
}

func TestAnnouncementTargeting_NormalizeAndValidate_RejectsEmptyGroup(t *testing.T) {
	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{
			{AllOf: nil},
		},
	}
	_, err := targeting.NormalizeAndValidate()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAnnouncementInvalidTarget)
}

func TestAnnouncementTargeting_NormalizeAndValidate_RejectsInvalidCondition(t *testing.T) {
	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{
			{
				AllOf: []AnnouncementCondition{
					{Type: "balance", Operator: "between", Value: 10},
				},
			},
		},
	}
	_, err := targeting.NormalizeAndValidate()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAnnouncementInvalidTarget)
}

func TestAnnouncementTargeting_Matches_AndOrSemantics(t *testing.T) {
	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{
			{
				AllOf: []AnnouncementCondition{
					{Type: AnnouncementConditionTypeBalance, Operator: AnnouncementOperatorGTE, Value: 100},
					{Type: AnnouncementConditionTypeSubscription, Operator: AnnouncementOperatorIn, GroupIDs: []int64{10}},
				},
			},
			{
				AllOf: []AnnouncementCondition{
					{Type: AnnouncementConditionTypeBalance, Operator: AnnouncementOperatorLT, Value: 5},
				},
			},
		},
	}

	// 命中第 2 组（balance < 5）
	require.True(t, targeting.Matches(4.99, nil))
	require.False(t, targeting.Matches(5, nil))

	// 命中第 1 组（balance >= 100 AND 订阅 in [10]）
	require.False(t, targeting.Matches(100, map[int64]struct{}{}))
	require.False(t, targeting.Matches(99.9, map[int64]struct{}{10: {}}))
	require.True(t, targeting.Matches(100, map[int64]struct{}{10: {}}))
}

func TestAnnouncementTargeting_Matches_UserCondition(t *testing.T) {
	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{
			{
				AllOf: []AnnouncementCondition{
					{Type: AnnouncementConditionTypeUser, Operator: AnnouncementOperatorIn, UserIDs: []int64{10, 20}},
				},
			},
		},
	}

	require.True(t, targeting.MatchesContext(AnnouncementMatchContext{UserID: 20}))
	require.False(t, targeting.MatchesContext(AnnouncementMatchContext{UserID: 30}))
	require.False(t, targeting.MatchesContext(AnnouncementMatchContext{}))
}

func TestAnnouncementTargeting_Matches_APIKeyGroupCondition(t *testing.T) {
	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{
			{
				AllOf: []AnnouncementCondition{
					{Type: AnnouncementConditionTypeGroup, Operator: AnnouncementOperatorIn, GroupIDs: []int64{100, 200}},
				},
			},
		},
	}

	require.True(t, targeting.MatchesContext(AnnouncementMatchContext{
		APIKeyGroupIDs: map[int64]struct{}{200: {}},
	}))
	require.False(t, targeting.MatchesContext(AnnouncementMatchContext{
		APIKeyGroupIDs: map[int64]struct{}{300: {}},
	}))
	require.False(t, targeting.MatchesContext(AnnouncementMatchContext{
		ActiveSubscriptionGroupIDs: map[int64]struct{}{200: {}},
	}))
}

func TestAnnouncementTargeting_NormalizeAndValidate_AcceptsUserAndGroupConditions(t *testing.T) {
	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{
			{
				AllOf: []AnnouncementCondition{
					{Type: AnnouncementConditionTypeUser, Operator: AnnouncementOperatorIn, UserIDs: []int64{10}},
					{Type: AnnouncementConditionTypeGroup, Operator: AnnouncementOperatorIn, GroupIDs: []int64{20}},
				},
			},
		},
	}

	normalized, err := targeting.NormalizeAndValidate()
	require.NoError(t, err)
	require.Equal(t, []int64{10}, normalized.AnyOf[0].AllOf[0].UserIDs)
	require.Equal(t, []int64{20}, normalized.AnyOf[0].AllOf[1].GroupIDs)
}

func TestAnnouncementTargeting_NormalizeAndValidate_RejectsEmptyUserCondition(t *testing.T) {
	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{
			{
				AllOf: []AnnouncementCondition{
					{Type: AnnouncementConditionTypeUser, Operator: AnnouncementOperatorIn},
				},
			},
		},
	}

	_, err := targeting.NormalizeAndValidate()
	require.ErrorIs(t, err, ErrAnnouncementInvalidTarget)
}
