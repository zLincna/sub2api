package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
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
var ErrPhoneVerifyCooldown = infraerrors.TooManyRequests("PHONE_VERIFY_COOLDOWN", "phone verification code was sent too frequently")

const (
	AliyunSMSRegistrationModeVerifyCode = "verify_code"
	AliyunSMSRegistrationModeTemplate   = "template"
)

type AliyunSMSConfig struct {
	AccessKeyID            string
	AccessKeySecret        string
	SignName               string
	RegistrationMode       string
	VerifyCodeSignName     string
	VerifyCodeTemplateCode string
	VerifyCodeStaticParams map[string]string
	TemplateCode           string
	TemplateParamKey       string
	TemplateStaticParams   map[string]string
	SchemeName             string
	ValidTimeSeconds       int
	IntervalSeconds        int
}

type AliyunTemplateSMSInput struct {
	Phone        string
	TemplateCode string
	Params       map[string]string
	OutID        string
}

type AliyunTestSMSType string

const (
	AliyunTestSMSRegistration      AliyunTestSMSType = "registration"
	AliyunTestSMSCarpoolAdminFull  AliyunTestSMSType = "carpool_admin_full"
	AliyunTestSMSCarpoolUserActive AliyunTestSMSType = "carpool_user_active"
)

type AliyunTestSMSOptions struct {
	Type                           AliyunTestSMSType
	Phone                          string
	AccessKeyID                    string
	AccessKeySecret                string
	SignName                       string
	RegistrationMode               string
	VerifyCodeSignName             string
	VerifyCodeTemplateCode         string
	VerifyCodeStaticJSON           string
	RegistrationTemplateCode       string
	RegistrationTemplateParamKey   string
	RegistrationTemplateStaticJSON string
	CarpoolAdminFullTemplateCode   string
	CarpoolUserActiveTemplateCode  string
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
	code := ""
	mode := normalizeAliyunSMSRegistrationMode(cfg.RegistrationMode)
	switch mode {
	case AliyunSMSRegistrationModeTemplate:
		code = randomPhoneVerificationCode(6)
		if err := sendAliyunTemplateSMS(ctx, cfg, AliyunTemplateSMSInput{
			Phone:        normalizedPhone,
			TemplateCode: cfg.TemplateCode,
			Params:       buildSMSParams(cfg.TemplateStaticParams, cfg.TemplateParamKey, code),
			OutID:        outID,
		}); err != nil {
			return nil, fmt.Errorf("send phone verify code: %w", err)
		}
	default:
		aliyunOutID, err := sendAliyunSMSVerifyCode(ctx, cfg, normalizedPhone, outID)
		if err != nil {
			return nil, fmt.Errorf("send phone verify code: %w", err)
		}
		if aliyunOutID != "" {
			outID = aliyunOutID
		}
	}

	expiresAt := now.Add(time.Duration(cfg.ValidTimeSeconds) * time.Second)
	cooldownUntil := now.Add(time.Duration(cfg.IntervalSeconds) * time.Second)

	s.smsChallengeMu.Lock()
	s.smsChallenges[normalizedPhone] = smsChallenge{
		OutID:         outID,
		Code:          code,
		Mode:          mode,
		ExpiresAt:     expiresAt,
		CooldownUntil: cooldownUntil,
	}
	s.smsChallengeMu.Unlock()

	return &SendPhoneVerifyCodeResult{Countdown: cfg.IntervalSeconds, ExpiresAt: expiresAt}, nil
}

func (s *AuthService) VerifyPhoneRegistrationCode(ctx context.Context, phone, code string) error {
	return s.verifyPhoneRegistrationCode(ctx, phone, code, true)
}

func (s *AuthService) CheckPhoneRegistrationCode(ctx context.Context, phone, code string) error {
	return s.verifyPhoneRegistrationCode(ctx, phone, code, false)
}

func (s *AuthService) ConsumePhoneRegistrationCode(ctx context.Context, phone string) {
	normalizedPhone, err := NormalizeMainlandPhone(phone)
	if err != nil {
		return
	}
	s.smsChallengeMu.Lock()
	delete(s.smsChallenges, normalizedPhone)
	s.smsChallengeMu.Unlock()
}

func (s *AuthService) verifyPhoneRegistrationCode(ctx context.Context, phone, code string, consume bool) error {
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

	mode := normalizeAliyunSMSRegistrationMode(challenge.Mode)
	if mode == AliyunSMSRegistrationModeTemplate {
		if challenge.Code == "" || challenge.Code != code {
			return ErrPhoneVerifyInvalid
		}
	} else {
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
	}

	if consume {
		s.smsChallengeMu.Lock()
		delete(s.smsChallenges, normalizedPhone)
		s.smsChallengeMu.Unlock()
	}
	return nil
}

func (s *AuthService) GetAliyunSMSConfig(ctx context.Context) (*AliyunSMSConfig, error) {
	if s == nil || s.settingService == nil || s.settingService.settingRepo == nil {
		return nil, ErrAliyunSMSNotConfigured
	}
	cfg, err := s.settingService.GetAliyunSMSConfig(ctx)
	if err != nil {
		return nil, err
	}
	mode := normalizeAliyunSMSRegistrationMode(cfg.RegistrationMode)
	if mode == AliyunSMSRegistrationModeVerifyCode && (strings.TrimSpace(cfg.VerifyCodeSignName) == "" || strings.TrimSpace(cfg.VerifyCodeTemplateCode) == "") {
		return nil, ErrAliyunSMSNotConfigured
	}
	if mode == AliyunSMSRegistrationModeTemplate && strings.TrimSpace(cfg.TemplateCode) == "" {
		return nil, ErrAliyunSMSNotConfigured
	}
	return cfg, nil
}

func (s *SettingService) GetAliyunSMSConfig(ctx context.Context) (*AliyunSMSConfig, error) {
	if s == nil || s.settingRepo == nil {
		return nil, ErrAliyunSMSNotConfigured
	}
	keys := []string{
		SettingKeyAliyunSMSAccessKeyID,
		SettingKeyAliyunSMSAccessKeySecret,
		SettingKeyAliyunSMSSignName,
		SettingKeyAliyunSMSRegistrationMode,
		SettingKeyAliyunSMSVerifyCodeSignName,
		SettingKeyAliyunSMSVerifyCodeTemplateCode,
		SettingKeyAliyunSMSVerifyCodeStaticJSON,
		SettingKeyAliyunSMSTemplateCode,
		SettingKeyAliyunSMSTemplateParamKey,
		SettingKeyAliyunSMSTemplateStaticJSON,
		SettingKeyAliyunSMSSchemeName,
		SettingKeyAliyunSMSValidTimeSeconds,
		SettingKeyAliyunSMSIntervalSeconds,
	}
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return nil, err
	}

	cfg := &AliyunSMSConfig{
		AccessKeyID:            firstNonEmpty(settings[SettingKeyAliyunSMSAccessKeyID], getenv("ALIYUN_ACCESS_KEY_ID"), getenv("ALIBABA_CLOUD_ACCESS_KEY_ID")),
		AccessKeySecret:        firstNonEmpty(settings[SettingKeyAliyunSMSAccessKeySecret], getenv("ALIYUN_ACCESS_KEY_SECRET"), getenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET")),
		SignName:               firstNonEmpty(settings[SettingKeyAliyunSMSSignName], getenv("ALIYUN_SMS_SIGN_NAME")),
		RegistrationMode:       normalizeAliyunSMSRegistrationMode(firstNonEmpty(settings[SettingKeyAliyunSMSRegistrationMode], getenv("ALIYUN_SMS_REGISTRATION_MODE"))),
		VerifyCodeSignName:     firstNonEmpty(settings[SettingKeyAliyunSMSVerifyCodeSignName], getenv("ALIYUN_SMS_VERIFY_CODE_SIGN_NAME")),
		VerifyCodeTemplateCode: firstNonEmpty(settings[SettingKeyAliyunSMSVerifyCodeTemplateCode], getenv("ALIYUN_SMS_VERIFY_CODE_TEMPLATE_CODE")),
		TemplateCode:           firstNonEmpty(settings[SettingKeyAliyunSMSTemplateCode], getenv("ALIYUN_SMS_TEMPLATE_CODE")),
		TemplateParamKey:       firstNonEmpty(settings[SettingKeyAliyunSMSTemplateParamKey], getenv("ALIYUN_SMS_TEMPLATE_PARAM_KEY"), "code"),
		SchemeName:             firstNonEmpty(settings[SettingKeyAliyunSMSSchemeName], getenv("ALIYUN_SMS_SCHEME_NAME")),
		ValidTimeSeconds:       parsePositiveIntOrDefault(firstNonEmpty(settings[SettingKeyAliyunSMSValidTimeSeconds], getenv("ALIYUN_SMS_VALID_TIME_SECONDS")), 300),
		IntervalSeconds:        parsePositiveIntOrDefault(firstNonEmpty(settings[SettingKeyAliyunSMSIntervalSeconds], getenv("ALIYUN_SMS_INTERVAL_SECONDS")), 60),
	}
	staticParams, err := parseSMSStaticParams(firstNonEmpty(settings[SettingKeyAliyunSMSTemplateStaticJSON], getenv("ALIYUN_SMS_TEMPLATE_STATIC_PARAMS"), "{}"))
	if err != nil {
		return nil, err
	}
	cfg.TemplateStaticParams = staticParams
	verifyCodeStaticParams, err := parseSMSStaticParams(firstNonEmpty(settings[SettingKeyAliyunSMSVerifyCodeStaticJSON], getenv("ALIYUN_SMS_VERIFY_CODE_STATIC_PARAMS"), "{}"))
	if err != nil {
		return nil, err
	}
	cfg.VerifyCodeStaticParams = verifyCodeStaticParams

	if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" {
		return nil, ErrAliyunSMSNotConfigured
	}
	return cfg, nil
}

func (s *SettingService) SendAliyunTestSMS(ctx context.Context, opts AliyunTestSMSOptions) (string, error) {
	if s == nil {
		return "", ErrAliyunSMSNotConfigured
	}
	normalizedPhone, err := NormalizeMainlandPhone(opts.Phone)
	if err != nil {
		return "", err
	}
	cfg, err := s.GetAliyunSMSConfig(ctx)
	if err != nil {
		cfg = &AliyunSMSConfig{
			TemplateParamKey:     "code",
			TemplateStaticParams: map[string]string{},
			ValidTimeSeconds:     300,
			IntervalSeconds:      60,
		}
	}
	if value := strings.TrimSpace(opts.AccessKeyID); value != "" {
		cfg.AccessKeyID = value
	}
	if value := strings.TrimSpace(opts.AccessKeySecret); value != "" {
		cfg.AccessKeySecret = value
	}
	if value := strings.TrimSpace(opts.SignName); value != "" {
		cfg.SignName = value
	}
	if value := strings.TrimSpace(opts.RegistrationMode); value != "" {
		cfg.RegistrationMode = normalizeAliyunSMSRegistrationMode(value)
	}
	if value := strings.TrimSpace(opts.VerifyCodeSignName); value != "" {
		cfg.VerifyCodeSignName = value
	}
	if value := strings.TrimSpace(opts.VerifyCodeTemplateCode); value != "" {
		cfg.VerifyCodeTemplateCode = value
	}
	if value := strings.TrimSpace(opts.VerifyCodeStaticJSON); value != "" {
		staticParams, err := parseSMSStaticParams(value)
		if err != nil {
			return "", err
		}
		cfg.VerifyCodeStaticParams = staticParams
	}
	if value := strings.TrimSpace(opts.RegistrationTemplateParamKey); value != "" {
		cfg.TemplateParamKey = value
	}
	if value := strings.TrimSpace(opts.RegistrationTemplateStaticJSON); value != "" {
		staticParams, err := parseSMSStaticParams(value)
		if err != nil {
			return "", err
		}
		cfg.TemplateStaticParams = staticParams
	}
	settings, err := s.GetAllSettings(ctx)
	if err != nil {
		return "", err
	}
	siteName := "ZemraAI"
	if settings != nil && strings.TrimSpace(settings.SiteName) != "" {
		siteName = strings.TrimSpace(settings.SiteName)
	}
	templateCode := firstNonEmpty(opts.RegistrationTemplateCode, cfg.TemplateCode)
	cfg.TemplateCode = templateCode
	params := buildSMSParams(cfg.TemplateStaticParams, cfg.TemplateParamKey, randomPhoneVerificationCode(6))
	switch opts.Type {
	case AliyunTestSMSRegistration:
		if normalizeAliyunSMSRegistrationMode(cfg.RegistrationMode) == AliyunSMSRegistrationModeVerifyCode {
			templateCode = firstNonEmpty(cfg.VerifyCodeTemplateCode, templateCode)
			if templateCode == "" || strings.TrimSpace(cfg.VerifyCodeSignName) == "" {
				return "", ErrAliyunSMSNotConfigured
			}
			_, err := sendAliyunSMSVerifyCode(ctx, cfg, normalizedPhone, fmt.Sprintf("sub2api-test-%s-%s", opts.Type, randomPhoneVerificationHex(6)))
			if err != nil {
				return "", err
			}
			return templateCode, nil
		}
		if templateCode == "" {
			return "", ErrAliyunSMSNotConfigured
		}
	case AliyunTestSMSCarpoolAdminFull:
		templateCode = strings.TrimSpace(opts.CarpoolAdminFullTemplateCode)
		if templateCode == "" && settings != nil {
			templateCode = settings.CarpoolAdminFullSMSTemplateCode
		}
		params = sampleCarpoolTemplateParams(siteName)
	case AliyunTestSMSCarpoolUserActive:
		templateCode = strings.TrimSpace(opts.CarpoolUserActiveTemplateCode)
		if templateCode == "" && settings != nil {
			templateCode = settings.CarpoolUserActiveSMSTemplateCode
		}
		params = sampleCarpoolTemplateParams(siteName)
	default:
		return "", infraerrors.BadRequest("INVALID_SMS_TEST_TYPE", "invalid sms test type")
	}
	if templateCode == "" {
		return "", ErrAliyunSMSNotConfigured
	}
	if err := sendAliyunTemplateSMS(ctx, cfg, AliyunTemplateSMSInput{
		Phone:        normalizedPhone,
		TemplateCode: templateCode,
		Params:       params,
		OutID:        fmt.Sprintf("sub2api-test-%s-%s", opts.Type, randomPhoneVerificationHex(6)),
	}); err != nil {
		return "", err
	}
	return templateCode, nil
}

func sampleCarpoolTemplateParams(siteName string) map[string]string {
	now := time.Now()
	if strings.TrimSpace(siteName) == "" {
		siteName = "ZemraAI"
	}
	return map[string]string{
		"site_name":          siteName,
		"code":               "123456",
		"min":                "5",
		"session_no":         "CP-TEST-001",
		"vehicle_name":       "Codex 20x Pro 2人车",
		"seat_count":         "2",
		"paid_count":         "2",
		"filled_at":          now.Format("2006-01-02 15:04"),
		"group_name":         "测试订阅分组",
		"service_started_at": now.Format("2006-01-02 15:04"),
		"service_ended_at":   now.Add(30 * 24 * time.Hour).Format("2006-01-02 15:04"),
		"service_days":       "30",
		"user_name":          "测试用户",
	}
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

func buildSMSParams(staticParams map[string]string, codeKey, code string) map[string]string {
	templateParams := make(map[string]string, len(staticParams)+1)
	for k, v := range staticParams {
		templateParams[k] = v
	}
	if strings.TrimSpace(codeKey) == "" {
		codeKey = "code"
	}
	templateParams[codeKey] = code
	return templateParams
}

func normalizeAliyunSMSRegistrationMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case AliyunSMSRegistrationModeTemplate:
		return AliyunSMSRegistrationModeTemplate
	default:
		return AliyunSMSRegistrationModeVerifyCode
	}
}

func sendAliyunSMSVerifyCode(ctx context.Context, cfg *AliyunSMSConfig, phone, outID string) (string, error) {
	params := aliyunSMSBaseParams(cfg, "SendSmsVerifyCode")
	signName := firstNonEmpty(cfg.VerifyCodeSignName, cfg.SignName)
	templateCode := firstNonEmpty(cfg.VerifyCodeTemplateCode, cfg.TemplateCode)
	templateParams := make(map[string]string, len(cfg.VerifyCodeStaticParams)+1)
	for k, v := range cfg.VerifyCodeStaticParams {
		templateParams[k] = v
	}
	templateParamKey := strings.TrimSpace(cfg.TemplateParamKey)
	if templateParamKey == "" {
		templateParamKey = "code"
	}
	templateParams[templateParamKey] = "##code##"
	templateJSON, _ := json.Marshal(templateParams)
	params.Set("PhoneNumber", phone)
	params.Set("CountryCode", "86")
	params.Set("SignName", signName)
	params.Set("TemplateCode", templateCode)
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
		return "", err
	}
	if !resp.ok() || !strings.EqualFold(resp.responseCode(), "OK") {
		return "", aliyunSMSError(resp.responseCode(), resp.responseMessage(), "")
	}
	return firstNonEmpty(resp.Model.OutID, resp.LowerModel.OutID), nil
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
		return false, aliyunSMSError(resp.responseCode(), resp.responseMessage(), "")
	}
	return strings.EqualFold(resp.verifyResult(), "PASS"), nil
}

type aliyunSMSResponse struct {
	Success bool   `json:"Success"`
	Code    string `json:"Code"`
	Message string `json:"Message"`
	Model   struct {
		BizID        string `json:"BizId"`
		OutID        string `json:"OutId"`
		VerifyResult string `json:"VerifyResult"`
	} `json:"Model"`
	LowerSuccess bool   `json:"success"`
	LowerCode    string `json:"code"`
	LowerMessage string `json:"message"`
	LowerModel   struct {
		BizID        string `json:"bizId"`
		OutID        string `json:"outId"`
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
	params.Set("RegionId", "cn-shanghai")
	params.Set("Version", "2017-05-25")
	params.Set("Action", action)
	params.Set("AccessKeyId", cfg.AccessKeyID)
	params.Set("SignatureMethod", "HMAC-SHA1")
	params.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	params.Set("SignatureVersion", "1.0")
	params.Set("SignatureNonce", randomPhoneVerificationHex(16))
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
		pairs = append(pairs, aliyunPercentEncode(key)+"="+aliyunPercentEncode(params.Get(key)))
	}
	stringToSign := strings.ToUpper(method) + "&" + aliyunPercentEncode("/") + "&" + aliyunPercentEncode(strings.Join(pairs, "&"))
	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func sendAliyunTemplateSMS(ctx context.Context, cfg *AliyunSMSConfig, input AliyunTemplateSMSInput) error {
	if cfg == nil || strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.AccessKeySecret) == "" || strings.TrimSpace(cfg.SignName) == "" {
		return ErrAliyunSMSNotConfigured
	}
	phone := strings.TrimSpace(input.Phone)
	templateCode := strings.TrimSpace(input.TemplateCode)
	if phone == "" || templateCode == "" {
		return ErrAliyunSMSNotConfigured
	}
	params := input.Params
	if params == nil {
		params = map[string]string{}
	}
	templateJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal aliyun sms template params: %w", err)
	}
	values := map[string]string{
		"PhoneNumbers":  phone,
		"SignName":      cfg.SignName,
		"TemplateCode":  templateCode,
		"TemplateParam": string(templateJSON),
	}
	if outID := strings.TrimSpace(input.OutID); outID != "" {
		values["OutId"] = outID
	}
	canonicalQuery := aliyunCanonicalQuery(values)
	headers := map[string]string{
		"host":                  "dysmsapi.aliyuncs.com",
		"x-acs-action":          "SendSms",
		"x-acs-content-sha256":  aliyunSHA256Hex(""),
		"x-acs-date":            time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"x-acs-signature-nonce": randomPhoneVerificationHex(16),
		"x-acs-version":         "2017-05-25",
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://dysmsapi.aliyuncs.com/?"+canonicalQuery, bytes.NewBufferString(""))
	if err != nil {
		return err
	}
	req.Host = headers["host"]
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Authorization", aliyunV3Authorization(http.MethodPost, "/", canonicalQuery, headers, cfg.AccessKeyID, cfg.AccessKeySecret))
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
	var result struct {
		Code      string `json:"Code"`
		Message   string `json:"Message"`
		RequestID string `json:"RequestId"`
		BizID     string `json:"BizId"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode aliyun sms response: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(result.Code), "OK") {
		return aliyunSMSError(result.Code, result.Message, result.RequestID)
	}
	return nil
}

func aliyunCanonicalQuery(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, aliyunPercentEncode(k)+"="+aliyunPercentEncode(values[k]))
	}
	return strings.Join(parts, "&")
}

func aliyunV3Authorization(method, canonicalURI, canonicalQuery string, headers map[string]string, accessKeyID, secret string) string {
	signedHeaders := aliyunSignedHeaders(headers)
	canonicalHeaders := aliyunCanonicalHeaders(headers)
	hashedPayload := strings.TrimSpace(headers["x-acs-content-sha256"])
	canonicalRequest := strings.ToUpper(method) + "\n" +
		canonicalURI + "\n" +
		canonicalQuery + "\n" +
		canonicalHeaders + "\n" +
		signedHeaders + "\n" +
		hashedPayload
	stringToSign := "ACS3-HMAC-SHA256\n" + aliyunSHA256Hex(canonicalRequest)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(stringToSign))
	signature := hex.EncodeToString(mac.Sum(nil))
	return "ACS3-HMAC-SHA256 Credential=" + strings.TrimSpace(accessKeyID) + ",SignedHeaders=" + signedHeaders + ",Signature=" + signature
}

func aliyunCanonicalHeaders(headers map[string]string) string {
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, strings.ToLower(strings.TrimSpace(k)))
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(strings.TrimSpace(headers[k]))
		b.WriteByte('\n')
	}
	return b.String()
}

func aliyunSignedHeaders(headers map[string]string) string {
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, strings.ToLower(strings.TrimSpace(k)))
	}
	sort.Strings(keys)
	return strings.Join(keys, ";")
}

func aliyunSHA256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func aliyunPercentEncode(value string) string {
	escaped := url.QueryEscape(value)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	escaped = strings.ReplaceAll(escaped, "*", "%2A")
	escaped = strings.ReplaceAll(escaped, "%7E", "~")
	return escaped
}

func aliyunSMSError(code, message, detail string) error {
	code = strings.TrimSpace(code)
	message = strings.TrimSpace(message)
	detail = strings.TrimSpace(detail)
	switch strings.ToLower(code) {
	case "biz.frequency", "isv.business_limit_control":
		return fmt.Errorf("发送过于频繁，请稍后再试（阿里云限制：%s）", firstNonEmpty(code, message, detail))
	}
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

func randomPhoneVerificationHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return fmt.Sprintf("%x", buf)
}

func randomPhoneVerificationCode(length int) string {
	if length <= 0 {
		length = 6
	}
	var b strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
		}
		b.WriteByte(byte('0' + n.Int64()))
	}
	return b.String()
}
