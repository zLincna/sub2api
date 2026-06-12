package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrAliyunSMSNotConfigured = infraerrors.ServiceUnavailable("ALIYUN_SMS_NOT_CONFIGURED", "aliyun sms service not configured")

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
	Countdown int `json:"countdown"`
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

	outID := "sub2api-register-" + randomHex(16)
	if err := sendAliyunSMSVerifyCode(ctx, cfg, normalizedPhone, outID); err != nil {
		return nil, fmt.Errorf("send phone verify code: %w", err)
	}

	s.smsChallengeMu.Lock()
	s.smsChallenges[normalizedPhone] = smsChallenge{
		OutID:     outID,
		ExpiresAt: time.Now().Add(time.Duration(cfg.ValidTimeSeconds) * time.Second),
	}
	s.smsChallengeMu.Unlock()

	return &SendPhoneVerifyCodeResult{Countdown: cfg.IntervalSeconds}, nil
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

func sendAliyunSMSVerifyCode(ctx context.Context, cfg *AliyunSMSConfig, phone, outID string) error {
	params := aliyunSMSBaseParams(cfg, "SendSmsVerifyCode")
	templateParams := make(map[string]string, len(cfg.TemplateStaticParams)+1)
	for k, v := range cfg.TemplateStaticParams {
		templateParams[k] = v
	}
	templateParams[cfg.TemplateParamKey] = "##code##"
	templateJSON, _ := json.Marshal(templateParams)
	params.Set("PhoneNumber", phone)
	params.Set("CountryCode", "86")
	params.Set("SignName", cfg.SignName)
	params.Set("TemplateCode", cfg.TemplateCode)
	params.Set("TemplateParam", string(templateJSON))
	params.Set("ValidTime", strconv.Itoa(cfg.ValidTimeSeconds))
	params.Set("Interval", strconv.Itoa(cfg.IntervalSeconds))
	params.Set("OutId", outID)
	params.Set("CodeLength", "6")
	params.Set("CodeType", "1")
	params.Set("DuplicatePolicy", "1")
	params.Set("ReturnVerifyCode", "false")
	if cfg.SchemeName != "" {
		params.Set("SchemeName", cfg.SchemeName)
	}
	var resp aliyunSMSResponse
	if err := aliyunSMSRPC(ctx, cfg, params, &resp); err != nil {
		return err
	}
	if !resp.ok() || !strings.EqualFold(resp.responseCode(), "OK") {
		return fmt.Errorf("%s: %s", resp.responseCode(), resp.responseMessage())
	}
	return nil
}

func checkAliyunSMSVerifyCode(ctx context.Context, cfg *AliyunSMSConfig, phone, code, outID string) (bool, error) {
	params := aliyunSMSBaseParams(cfg, "CheckSmsVerifyCode")
	params.Set("PhoneNumber", phone)
	params.Set("CountryCode", "86")
	params.Set("VerifyCode", code)
	params.Set("OutId", outID)
	params.Set("CaseAuthPolicy", "1")
	if cfg.SchemeName != "" {
		params.Set("SchemeName", cfg.SchemeName)
	}
	var resp aliyunSMSResponse
	if err := aliyunSMSRPC(ctx, cfg, params, &resp); err != nil {
		return false, err
	}
	if !resp.ok() || !strings.EqualFold(resp.responseCode(), "OK") {
		return false, fmt.Errorf("%s: %s", resp.responseCode(), resp.responseMessage())
	}
	return strings.EqualFold(resp.verifyResult(), "PASS"), nil
}

type aliyunSMSResponse struct {
	Success bool   `json:"Success"`
	Code    string `json:"Code"`
	Message string `json:"Message"`
	Model   struct {
		VerifyResult string `json:"VerifyResult"`
	} `json:"Model"`
	LowerSuccess bool   `json:"success"`
	LowerCode    string `json:"code"`
	LowerMessage string `json:"message"`
	LowerModel   struct {
		VerifyResult string `json:"verifyResult"`
	} `json:"model"`
}

func (r aliyunSMSResponse) ok() bool {
	return r.Success || r.LowerSuccess
}

func (r aliyunSMSResponse) responseCode() string {
	return firstNonEmpty(r.Code, r.LowerCode)
}

func (r aliyunSMSResponse) responseMessage() string {
	return firstNonEmpty(r.Message, r.LowerMessage)
}

func (r aliyunSMSResponse) verifyResult() string {
	return firstNonEmpty(r.Model.VerifyResult, r.LowerModel.VerifyResult)
}

func aliyunSMSBaseParams(cfg *AliyunSMSConfig, action string) url.Values {
	params := url.Values{}
	params.Set("Format", "JSON")
	params.Set("Version", "2017-05-25")
	params.Set("Action", action)
	params.Set("AccessKeyId", cfg.AccessKeyID)
	params.Set("SignatureMethod", "HMAC-SHA1")
	params.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	params.Set("SignatureVersion", "1.0")
	params.Set("SignatureNonce", randomHex(16))
	return params
}

func aliyunSMSRPC(ctx context.Context, cfg *AliyunSMSConfig, params url.Values, out any) error {
	params.Set("Signature", aliyunSignature("POST", params, cfg.AccessKeySecret+"&"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://dypnsapi.aliyuncs.com/", strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("aliyun sms http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return err
	}
	return nil
}

func aliyunSignature(method string, params url.Values, secret string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, percentEncode(key)+"="+percentEncode(params.Get(key)))
	}
	stringToSign := method + "&%2F&" + percentEncode(strings.Join(pairs, "&"))
	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func percentEncode(value string) string {
	encoded := url.QueryEscape(value)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return fmt.Sprintf("%x", buf)
}
