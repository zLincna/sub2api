package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	dypnsapi "github.com/alibabacloud-go/dypnsapi-20170525/v3/client"
	"github.com/alibabacloud-go/tea/dara"
)

var ErrAliyunSMSNotConfigured = infraerrors.ServiceUnavailable("ALIYUN_SMS_NOT_CONFIGURED", "aliyun sms service not configured")
var ErrPhoneVerifyCooldown = infraerrors.TooManyRequests("PHONE_VERIFY_COOLDOWN", "phone verification code was sent too frequently")

type AliyunSMSConfig struct {
	AccessKeyID          string
	AccessKeySecret      string
	SignName             string
	TemplateCode         string
	TemplateParamKey     string
	TemplateStaticParams map[string]string
	SchemeName           string
	ValidTimeSeconds     int
	IntervalSeconds      int
}

type SendPhoneVerifyCodeResult struct {
	Countdown int       `json:"countdown"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NormalizeMainlandPhone(phone string) (string, error) {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	normalized := b.String()
	if normalized == "" {
		return "", ErrPhoneRequired
	}
	if len(normalized) == 13 && strings.HasPrefix(normalized, "86") {
		normalized = normalized[2:]
	}
	if len(normalized) != 11 || normalized[0] != '1' {
		return "", ErrInvalidPhone
	}
	return normalized, nil
}

func (s *AuthService) SendPhoneVerifyCode(ctx context.Context, phone string) (*SendPhoneVerifyCodeResult, error) {
	if s == nil || s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
		return nil, ErrRegDisabled
	}
	if !s.settingService.IsPhoneVerifyEnabled(ctx) {
		return nil, infraerrors.BadRequest("PHONE_VERIFY_DISABLED", "phone verification is disabled")
	}

	normalizedPhone, err := NormalizeMainlandPhone(phone)
	if err != nil {
		return nil, err
	}
	exists, err := s.userRepo.ExistsByPhone(ctx, normalizedPhone)
	if err != nil {
		return nil, ErrServiceUnavailable
	}
	if exists {
		return nil, ErrPhoneExists
	}

	cfg, err := s.GetAliyunSMSConfig(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	s.smsChallengeMu.Lock()
	if challenge, ok := s.smsChallenges[normalizedPhone]; ok {
		if now.After(challenge.ExpiresAt) {
			delete(s.smsChallenges, normalizedPhone)
		} else if now.Before(challenge.CooldownUntil) {
			s.smsChallengeMu.Unlock()
			return nil, ErrPhoneVerifyCooldown
		}
	}
	s.smsChallengeMu.Unlock()

	outID := "sub2api-register-" + randomPhoneVerificationHex(16)
	aliyunOutID, err := sendAliyunSMSVerifyCode(ctx, cfg, normalizedPhone, outID)
	if err != nil {
		return nil, fmt.Errorf("send phone verify code: %w", err)
	}
	if aliyunOutID != "" {
		outID = aliyunOutID
	}

	expiresAt := now.Add(time.Duration(cfg.ValidTimeSeconds) * time.Second)
	cooldownUntil := now.Add(time.Duration(cfg.IntervalSeconds) * time.Second)

	s.smsChallengeMu.Lock()
	s.smsChallenges[normalizedPhone] = smsChallenge{
		OutID:         outID,
		ExpiresAt:     expiresAt,
		CooldownUntil: cooldownUntil,
	}
	s.smsChallengeMu.Unlock()

	return &SendPhoneVerifyCodeResult{Countdown: cfg.IntervalSeconds, ExpiresAt: expiresAt}, nil
}

func (s *AuthService) VerifyPhoneRegistrationCode(ctx context.Context, phone, code string) error {
	normalizedPhone, err := NormalizeMainlandPhone(phone)
	if err != nil {
		return err
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return ErrPhoneVerifyRequired
	}

	s.smsChallengeMu.Lock()
	challenge, ok := s.smsChallenges[normalizedPhone]
	if ok && time.Now().After(challenge.ExpiresAt) {
		delete(s.smsChallenges, normalizedPhone)
		ok = false
	}
	s.smsChallengeMu.Unlock()
	if !ok || challenge.OutID == "" {
		return ErrPhoneVerifyInvalid
	}

	cfg, err := s.GetAliyunSMSConfig(ctx)
	if err != nil {
		return err
	}
	pass, err := checkAliyunSMSVerifyCode(ctx, cfg, normalizedPhone, code, challenge.OutID)
	if err != nil {
		return fmt.Errorf("check phone verify code: %w", err)
	}
	if !pass {
		return ErrPhoneVerifyInvalid
	}

	s.smsChallengeMu.Lock()
	delete(s.smsChallenges, normalizedPhone)
	s.smsChallengeMu.Unlock()
	return nil
}

func (s *AuthService) GetAliyunSMSConfig(ctx context.Context) (*AliyunSMSConfig, error) {
	if s == nil || s.settingService == nil || s.settingService.settingRepo == nil {
		return nil, ErrAliyunSMSNotConfigured
	}
	keys := []string{
		SettingKeyAliyunSMSAccessKeyID,
		SettingKeyAliyunSMSAccessKeySecret,
		SettingKeyAliyunSMSSignName,
		SettingKeyAliyunSMSTemplateCode,
		SettingKeyAliyunSMSTemplateParamKey,
		SettingKeyAliyunSMSTemplateStaticJSON,
		SettingKeyAliyunSMSSchemeName,
		SettingKeyAliyunSMSValidTimeSeconds,
		SettingKeyAliyunSMSIntervalSeconds,
	}
	settings, err := s.settingService.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return nil, err
	}

	cfg := &AliyunSMSConfig{
		AccessKeyID:      firstNonEmpty(settings[SettingKeyAliyunSMSAccessKeyID], getenv("ALIYUN_ACCESS_KEY_ID"), getenv("ALIBABA_CLOUD_ACCESS_KEY_ID")),
		AccessKeySecret:  firstNonEmpty(settings[SettingKeyAliyunSMSAccessKeySecret], getenv("ALIYUN_ACCESS_KEY_SECRET"), getenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET")),
		SignName:         firstNonEmpty(settings[SettingKeyAliyunSMSSignName], getenv("ALIYUN_SMS_SIGN_NAME")),
		TemplateCode:     firstNonEmpty(settings[SettingKeyAliyunSMSTemplateCode], getenv("ALIYUN_SMS_TEMPLATE_CODE")),
		TemplateParamKey: firstNonEmpty(settings[SettingKeyAliyunSMSTemplateParamKey], getenv("ALIYUN_SMS_TEMPLATE_PARAM_KEY"), "code"),
		SchemeName:       firstNonEmpty(settings[SettingKeyAliyunSMSSchemeName], getenv("ALIYUN_SMS_SCHEME_NAME")),
		ValidTimeSeconds: parsePositiveIntOrDefault(firstNonEmpty(settings[SettingKeyAliyunSMSValidTimeSeconds], getenv("ALIYUN_SMS_VALID_TIME_SECONDS")), 300),
		IntervalSeconds:  parsePositiveIntOrDefault(firstNonEmpty(settings[SettingKeyAliyunSMSIntervalSeconds], getenv("ALIYUN_SMS_INTERVAL_SECONDS")), 60),
	}
	staticParams, err := parseSMSStaticParams(firstNonEmpty(settings[SettingKeyAliyunSMSTemplateStaticJSON], getenv("ALIYUN_SMS_TEMPLATE_STATIC_PARAMS"), "{}"))
	if err != nil {
		return nil, err
	}
	cfg.TemplateStaticParams = staticParams

	if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" || cfg.SignName == "" || cfg.TemplateCode == "" {
		return nil, ErrAliyunSMSNotConfigured
	}
	return cfg, nil
}

func getenv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func parseSMSStaticParams(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}, nil
	}
	var values map[string]any
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, infraerrors.BadRequest("INVALID_ALIYUN_SMS_TEMPLATE_STATIC_PARAMS", "aliyun sms template static params must be valid JSON")
	}
	out := make(map[string]string, len(values))
	for k, v := range values {
		if v != nil {
			out[k] = fmt.Sprint(v)
		}
	}
	return out, nil
}

func sendAliyunSMSVerifyCode(ctx context.Context, cfg *AliyunSMSConfig, phone, outID string) (string, error) {
	templateParams := make(map[string]string, len(cfg.TemplateStaticParams)+1)
	for k, v := range cfg.TemplateStaticParams {
		templateParams[k] = v
	}
	templateParams[cfg.TemplateParamKey] = "##code##"
	templateJSON, _ := json.Marshal(templateParams)

	client, err := newAliyunSMSClient(cfg)
	if err != nil {
		return "", err
	}
	request := &dypnsapi.SendSmsVerifyCodeRequest{
		PhoneNumber:      dara.String(phone),
		CountryCode:      dara.String("86"),
		SignName:         dara.String(cfg.SignName),
		TemplateCode:     dara.String(cfg.TemplateCode),
		TemplateParam:    dara.String(string(templateJSON)),
		ValidTime:        dara.Int64(int64(cfg.ValidTimeSeconds)),
		Interval:         dara.Int64(int64(cfg.IntervalSeconds)),
		OutId:            dara.String(outID),
		CodeLength:       dara.Int64(6),
		CodeType:         dara.Int64(1),
		DuplicatePolicy:  dara.Int64(1),
		ReturnVerifyCode: dara.Bool(false),
	}
	if cfg.SchemeName != "" {
		request.SchemeName = dara.String(cfg.SchemeName)
	}

	resp, err := client.SendSmsVerifyCodeWithOptions(request, &dara.RuntimeOptions{
		Autoretry:      dara.Bool(false),
		ConnectTimeout: dara.Int(10000),
		ReadTimeout:    dara.Int(10000),
	})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.Body == nil {
		return "", fmt.Errorf("empty aliyun sms response")
	}
	body := resp.Body
	if body.Success == nil || !dara.BoolValue(body.Success) || !strings.EqualFold(dara.StringValue(body.Code), "OK") {
		return "", aliyunSMSError(dara.StringValue(body.Code), dara.StringValue(body.Message), dara.StringValue(body.AccessDeniedDetail))
	}
	if body.Model == nil {
		return outID, nil
	}
	return firstNonEmpty(dara.StringValue(body.Model.OutId), outID), nil
}

func checkAliyunSMSVerifyCode(ctx context.Context, cfg *AliyunSMSConfig, phone, code, outID string) (bool, error) {
	client, err := newAliyunSMSClient(cfg)
	if err != nil {
		return false, err
	}
	request := &dypnsapi.CheckSmsVerifyCodeRequest{
		PhoneNumber:    dara.String(phone),
		CountryCode:    dara.String("86"),
		VerifyCode:     dara.String(code),
		OutId:          dara.String(outID),
		CaseAuthPolicy: dara.Int64(1),
	}
	if cfg.SchemeName != "" {
		request.SchemeName = dara.String(cfg.SchemeName)
	}

	resp, err := client.CheckSmsVerifyCodeWithOptions(request, &dara.RuntimeOptions{
		Autoretry:      dara.Bool(false),
		ConnectTimeout: dara.Int(10000),
		ReadTimeout:    dara.Int(10000),
	})
	if err != nil {
		return false, err
	}
	if resp == nil || resp.Body == nil {
		return false, fmt.Errorf("empty aliyun sms response")
	}
	return parseAliyunSMSVerifyResult(resp.Body)
}

func newAliyunSMSClient(cfg *AliyunSMSConfig) (*dypnsapi.Client, error) {
	return dypnsapi.NewClient(&openapiutil.Config{
		AccessKeyId:     dara.String(cfg.AccessKeyID),
		AccessKeySecret: dara.String(cfg.AccessKeySecret),
		Endpoint:        dara.String("dypnsapi.aliyuncs.com"),
	})
}

func aliyunSMSError(code, message, detail string) error {
	code = strings.TrimSpace(code)
	message = strings.TrimSpace(message)
	detail = strings.TrimSpace(detail)
	switch {
	case code != "" && message != "":
		if detail != "" {
			return fmt.Errorf("%s: %s (%s)", code, message, detail)
		}
		return fmt.Errorf("%s: %s", code, message)
	case message != "":
		return fmt.Errorf("%s", message)
	case code != "":
		return fmt.Errorf("%s", code)
	case detail != "":
		return fmt.Errorf("%s", detail)
	default:
		return fmt.Errorf("aliyun sms request failed")
	}
}

func parseAliyunSMSVerifyResult(body *dypnsapi.CheckSmsVerifyCodeResponseBody) (bool, error) {
	if body == nil {
		return false, fmt.Errorf("empty aliyun sms response")
	}
	verifyResult := ""
	if body.Model != nil {
		verifyResult = strings.TrimSpace(dara.StringValue(body.Model.VerifyResult))
	}
	if strings.EqualFold(verifyResult, "UNKNOWN") {
		return false, nil
	}
	if body.Success == nil || !dara.BoolValue(body.Success) || !strings.EqualFold(dara.StringValue(body.Code), "OK") {
		code := dara.StringValue(body.Code)
		message := dara.StringValue(body.Message)
		if strings.EqualFold(code, "UNKNOWN") || strings.EqualFold(message, "UNKNOWN") {
			return false, nil
		}
		return false, aliyunSMSError(code, message, dara.StringValue(body.AccessDeniedDetail))
	}
	return strings.EqualFold(verifyResult, "PASS"), nil
}

func randomPhoneVerificationHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return fmt.Sprintf("%x", buf)
}
