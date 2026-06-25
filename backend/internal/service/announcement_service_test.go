package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type announcementRepoStub struct {
	item   *Announcement
	active []Announcement
}

func (s *announcementRepoStub) Create(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (s *announcementRepoStub) GetByID(_ context.Context, _ int64) (*Announcement, error) {
	if s.item == nil {
		return nil, ErrAnnouncementNotFound
	}
	return s.item, nil
}

func (s *announcementRepoStub) Update(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (*announcementRepoStub) Delete(context.Context, int64) error {
	return nil
}

func (*announcementRepoStub) List(context.Context, pagination.PaginationParams, AnnouncementListFilters) ([]Announcement, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *announcementRepoStub) ListActive(_ context.Context, _ time.Time) ([]Announcement, error) {
	return s.active, nil
}

type announcementUserRepoStub struct {
	UserRepository
	users map[int64]*User
}

func (s *announcementUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	if s.users == nil {
		return nil, ErrUserNotFound
	}
	user, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

type announcementUserSubRepoStub struct {
	UserSubscriptionRepository
	activeByUser map[int64][]UserSubscription
}

func (s *announcementUserSubRepoStub) ListActiveByUserID(_ context.Context, userID int64) ([]UserSubscription, error) {
	if s.activeByUser == nil {
		return nil, nil
	}
	return s.activeByUser[userID], nil
}

type announcementAPIKeyGroupRepoStub struct {
	groupIDsByUser map[int64][]int64
}

func (s *announcementAPIKeyGroupRepoStub) ListDistinctGroupIDsByUserID(_ context.Context, userID int64) ([]int64, error) {
	if s.groupIDsByUser == nil {
		return nil, nil
	}
	return s.groupIDsByUser[userID], nil
}

type announcementReadRepoStub struct {
	AnnouncementReadRepository
}

func (s *announcementReadRepoStub) GetReadMapByUser(context.Context, int64, []int64) (map[int64]time.Time, error) {
	return map[int64]time.Time{}, nil
}

func TestAnnouncementServiceCreateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{}
	svc := NewAnnouncementService(repo, nil, nil, nil, nil)
	now := time.Unix(1776790020, 0)

	_, err := svc.Create(context.Background(), &CreateAnnouncementInput{
		Title:      "公告",
		Content:    "内容",
		Status:     AnnouncementStatusActive,
		NotifyMode: AnnouncementNotifyModePopup,
		StartsAt:   &now,
		EndsAt:     &now,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}

func TestAnnouncementServiceUpdateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{
		item: &Announcement{
			ID:         1,
			Title:      "公告",
			Content:    "内容",
			Status:     AnnouncementStatusActive,
			NotifyMode: AnnouncementNotifyModePopup,
		},
	}
	svc := NewAnnouncementService(repo, nil, nil, nil, nil)
	now := time.Unix(1776790020, 0)
	startsAt := &now
	endsAt := &now

	_, err := svc.Update(context.Background(), 1, &UpdateAnnouncementInput{
		StartsAt: &startsAt,
		EndsAt:   &endsAt,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}

func TestAnnouncementServiceListForUserMatchesUserAndAPIKeyGroupConditions(t *testing.T) {
	now := time.Now().Add(-time.Hour)
	repo := &announcementRepoStub{
		active: []Announcement{
			{
				ID:         1,
				Title:      "group notice",
				Content:    "content",
				Status:     AnnouncementStatusActive,
				NotifyMode: AnnouncementNotifyModeSilent,
				Targeting: AnnouncementTargeting{AnyOf: []AnnouncementConditionGroup{{
					AllOf: []AnnouncementCondition{{
						Type:     AnnouncementConditionTypeGroup,
						Operator: AnnouncementOperatorIn,
						GroupIDs: []int64{88},
					}},
				}}},
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:         2,
				Title:      "user notice",
				Content:    "content",
				Status:     AnnouncementStatusActive,
				NotifyMode: AnnouncementNotifyModeSilent,
				Targeting: AnnouncementTargeting{AnyOf: []AnnouncementConditionGroup{{
					AllOf: []AnnouncementCondition{{
						Type:     AnnouncementConditionTypeUser,
						Operator: AnnouncementOperatorIn,
						UserIDs:  []int64{42},
					}},
				}}},
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:         3,
				Title:      "other group notice",
				Content:    "content",
				Status:     AnnouncementStatusActive,
				NotifyMode: AnnouncementNotifyModeSilent,
				Targeting: AnnouncementTargeting{AnyOf: []AnnouncementConditionGroup{{
					AllOf: []AnnouncementCondition{{
						Type:     AnnouncementConditionTypeGroup,
						Operator: AnnouncementOperatorIn,
						GroupIDs: []int64{99},
					}},
				}}},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	svc := NewAnnouncementService(
		repo,
		&announcementReadRepoStub{},
		&announcementUserRepoStub{users: map[int64]*User{
			42: {ID: 42, Email: "u@example.com", Balance: 10},
		}},
		&announcementUserSubRepoStub{},
		&announcementAPIKeyGroupRepoStub{groupIDsByUser: map[int64][]int64{
			42: {88},
		}},
	)

	items, err := svc.ListForUser(context.Background(), 42, false)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, int64(2), items[0].Announcement.ID)
	require.Equal(t, int64(1), items[1].Announcement.ID)
}
