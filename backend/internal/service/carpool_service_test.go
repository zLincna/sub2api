//go:build unit

package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/carpoolsession"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestCarpoolPaymentCompletionFillsSessionAndCreatesNextQueue(t *testing.T) {
	ctx := context.Background()
	client := newCarpoolTestClient(t)

	user1 := client.User.Create().SetEmail("a@example.com").SetPasswordHash("x").SaveX(ctx)
	user2 := client.User.Create().SetEmail("b@example.com").SetPasswordHash("x").SaveX(ctx)
	vt := client.CarpoolVehicleType.Create().
		SetName("2人车").
		SetSeatCount(2).
		SetTotalPrice(1300).
		SetUnitPrice(650).
		SetEnabled(true).
		SaveX(ctx)
	notice := client.CarpoolNoticeVersion.Create().SetTitle("须知").SetContentMd("read").SetVersion(1).SetActive(true).SaveX(ctx)
	order1 := client.PaymentOrder.Create().
		SetUserID(user1.ID).
		SetUserEmail(user1.Email).
		SetUserName("").
		SetAmount(650).
		SetPayAmount(650).
		SetRechargeCode("CP1").
		SetOutTradeNo("cp1").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade1").
		SetOrderType(payment.OrderTypeCarpool).
		SetStatus(OrderStatusPaid).
		SetClientIP("127.0.0.1").
		SetSrcHost("test").
		SetExpiresAt(carpoolTestFuture()).
		SaveX(ctx)
	order2 := client.PaymentOrder.Create().
		SetUserID(user2.ID).
		SetUserEmail(user2.Email).
		SetUserName("").
		SetAmount(650).
		SetPayAmount(650).
		SetRechargeCode("CP2").
		SetOutTradeNo("cp2").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade2").
		SetOrderType(payment.OrderTypeCarpool).
		SetStatus(OrderStatusPaid).
		SetClientIP("127.0.0.1").
		SetSrcHost("test").
		SetExpiresAt(carpoolTestFuture()).
		SaveX(ctx)
	client.CarpoolParticipant.Create().
		SetVehicleTypeID(vt.ID).
		SetUserID(user1.ID).
		SetPaymentOrderID(order1.ID).
		SetStatus(CarpoolParticipantPendingPayment).
		SetAmount(650).
		SetWaitUntil(carpoolTestFuture()).
		SetNoticeVersionID(notice.ID).
		SaveX(ctx)
	client.CarpoolParticipant.Create().
		SetVehicleTypeID(vt.ID).
		SetUserID(user2.ID).
		SetPaymentOrderID(order2.ID).
		SetStatus(CarpoolParticipantPendingPayment).
		SetAmount(650).
		SetWaitUntil(carpoolTestFuture()).
		SetNoticeVersionID(notice.ID).
		SaveX(ctx)

	svc := NewCarpoolService(client)
	require.NoError(t, svc.HandlePaymentCompleted(ctx, order1.ID))
	require.NoError(t, svc.HandlePaymentCompleted(ctx, order2.ID))

	fullCount := client.CarpoolSession.Query().Where(carpoolsession.VehicleTypeIDEQ(vt.ID), carpoolsession.StatusEQ(CarpoolSessionFull)).CountX(ctx)
	recruitingCount := client.CarpoolSession.Query().Where(carpoolsession.VehicleTypeIDEQ(vt.ID), carpoolsession.StatusEQ(CarpoolSessionRecruiting)).CountX(ctx)
	require.Equal(t, 1, fullCount)
	require.Equal(t, 1, recruitingCount)

	management, err := svc.AdminManagement(ctx, 1, 20, "")
	require.NoError(t, err)
	require.Equal(t, 1, management.Total)
	require.Len(t, management.Items, 1)
	require.Equal(t, CarpoolSessionFull, management.Items[0].Session.Status)
	require.Equal(t, int64(1), management.Summary.CompletedSessions)
	require.Equal(t, int64(2), management.Summary.PaidMembers)
}

func TestCarpoolPaymentCompletionKeepsWaitingAfterRefundTime(t *testing.T) {
	ctx := context.Background()
	client := newCarpoolTestClient(t)

	user1 := client.User.Create().SetEmail("late@example.com").SetPasswordHash("x").SaveX(ctx)
	vt := client.CarpoolVehicleType.Create().
		SetName("2人车").
		SetSeatCount(2).
		SetTotalPrice(1300).
		SetUnitPrice(650).
		SetRefundWaitHours(1).
		SetEnabled(true).
		SaveX(ctx)
	notice := client.CarpoolNoticeVersion.Create().SetTitle("须知").SetContentMd("read").SetVersion(1).SetActive(true).SaveX(ctx)
	order := client.PaymentOrder.Create().
		SetUserID(user1.ID).
		SetUserEmail(user1.Email).
		SetUserName("").
		SetAmount(650).
		SetPayAmount(650).
		SetRechargeCode("CP-LATE").
		SetOutTradeNo("cp-late").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-late").
		SetOrderType(payment.OrderTypeCarpool).
		SetStatus(OrderStatusPaid).
		SetClientIP("127.0.0.1").
		SetSrcHost("test").
		SetExpiresAt(carpoolTestFuture()).
		SaveX(ctx)
	client.CarpoolParticipant.Create().
		SetVehicleTypeID(vt.ID).
		SetUserID(user1.ID).
		SetPaymentOrderID(order.ID).
		SetStatus(CarpoolParticipantPendingPayment).
		SetAmount(650).
		SetWaitUntil(time.Now().Add(-time.Minute)).
		SetNoticeVersionID(notice.ID).
		SaveX(ctx)

	svc := NewCarpoolService(client)
	require.NoError(t, svc.HandlePaymentCompleted(ctx, order.ID))

	participant := client.CarpoolParticipant.Query().OnlyX(ctx)
	require.Equal(t, CarpoolParticipantPaid, participant.Status)
	require.NotNil(t, participant.SessionID)

	session := client.CarpoolSession.GetX(ctx, *participant.SessionID)
	require.Equal(t, CarpoolSessionRecruiting, session.Status)
	require.Equal(t, 1, session.PaidCount)
}

func carpoolTestFuture() time.Time {
	return time.Now().Add(time.Hour)
}

func newCarpoolTestClient(t *testing.T) *dbent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", "file:carpool_payment_completion?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}
