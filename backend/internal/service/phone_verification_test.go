package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildSMSParams(t *testing.T) {
	params := buildSMSParams(map[string]string{"site": "ZemraAI"}, "code", "987977")

	require.Equal(t, "ZemraAI", params["site"])
	require.Equal(t, "987977", params["code"])
}

func TestBuildSMSParamsDefaultCodeKey(t *testing.T) {
	params := buildSMSParams(nil, "", "123456")

	require.Equal(t, "123456", params["code"])
}

func TestAliyunPercentEncode(t *testing.T) {
	require.Equal(t, "a%20b%2Ac~", aliyunPercentEncode("a b*c~"))
}

func TestRandomPhoneVerificationCode(t *testing.T) {
	code := randomPhoneVerificationCode(6)

	require.Len(t, code, 6)
	for _, r := range code {
		require.True(t, r >= '0' && r <= '9')
	}
}
