package service

import (
	"fmt"
	"testing"

	dypnsapi "github.com/alibabacloud-go/dypnsapi-20170525/v3/client"
	"github.com/alibabacloud-go/tea/dara"
	teautil "github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/require"
)

func TestParseAliyunSMSVerifyResult(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		pass, err := parseAliyunSMSVerifyResult(&dypnsapi.CheckSmsVerifyCodeResponseBody{
			Success: dara.Bool(true),
			Code:    dara.String("OK"),
			Model: &dypnsapi.CheckSmsVerifyCodeResponseBodyModel{
				VerifyResult: dara.String("PASS"),
			},
		})
		require.NoError(t, err)
		require.True(t, pass)
	})

	t.Run("unknown verify result is invalid code", func(t *testing.T) {
		pass, err := parseAliyunSMSVerifyResult(&dypnsapi.CheckSmsVerifyCodeResponseBody{
			Success: dara.Bool(true),
			Code:    dara.String("OK"),
			Model: &dypnsapi.CheckSmsVerifyCodeResponseBodyModel{
				VerifyResult: dara.String("UNKNOWN"),
			},
		})
		require.NoError(t, err)
		require.False(t, pass)
	})

	t.Run("unknown response code is invalid code", func(t *testing.T) {
		pass, err := parseAliyunSMSVerifyResult(&dypnsapi.CheckSmsVerifyCodeResponseBody{
			Success: dara.Bool(false),
			Code:    dara.String("UNKNOWN"),
			Message: dara.String("UNKNOWN"),
		})
		require.NoError(t, err)
		require.False(t, pass)
	})

	t.Run("other aliyun failure remains provider error", func(t *testing.T) {
		pass, err := parseAliyunSMSVerifyResult(&dypnsapi.CheckSmsVerifyCodeResponseBody{
			Success: dara.Bool(false),
			Code:    dara.String("InvalidAccessKeyId.NotFound"),
			Message: dara.String("access key not found"),
		})
		require.Error(t, err)
		require.False(t, pass)
		require.Contains(t, err.Error(), "InvalidAccessKeyId.NotFound")
	})
}

func TestIsAliyunSMSInvalidVerifyError(t *testing.T) {
	t.Run("tea sdk validate fail", func(t *testing.T) {
		err := teautil.NewSDKError(map[string]interface{}{
			"statusCode": 400,
			"code":       "isv.ValidateFail",
			"message":    "验证失败",
			"data": map[string]interface{}{
				"Code":    "isv.ValidateFail",
				"Message": "验证失败",
			},
		})
		require.True(t, isAliyunSMSInvalidVerifyError(err))
	})

	t.Run("dara sdk validate fail", func(t *testing.T) {
		err := dara.NewSDKError(map[string]interface{}{
			"statusCode": 400,
			"code":       "isv.ValidateFail",
			"message":    "验证失败",
		})
		require.True(t, isAliyunSMSInvalidVerifyError(err))
	})

	t.Run("wrapped validate fail", func(t *testing.T) {
		err := fmt.Errorf("check failed: %w", teautil.NewSDKError(map[string]interface{}{
			"statusCode": 400,
			"code":       "isv.ValidateFail",
			"message":    "验证失败",
		}))
		require.True(t, isAliyunSMSInvalidVerifyError(err))
	})

	t.Run("provider configuration error remains system error", func(t *testing.T) {
		err := teautil.NewSDKError(map[string]interface{}{
			"statusCode": 400,
			"code":       "InvalidAccessKeyId.NotFound",
			"message":    "access key not found",
		})
		require.False(t, isAliyunSMSInvalidVerifyError(err))
	})
}
