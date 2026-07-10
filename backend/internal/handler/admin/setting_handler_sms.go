package admin

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type sendTestSMSRequest struct {
	Phone                         string `json:"phone"`
	Type                          string `json:"type"`
	AliyunSMSAccessKeyID          string `json:"aliyun_sms_access_key_id"`
	AliyunSMSAccessKeySecret      string `json:"aliyun_sms_access_key_secret"`
	AliyunSMSSignName             string `json:"aliyun_sms_sign_name"`
	AliyunSMSRegistrationMode     string `json:"aliyun_sms_registration_mode"`
	AliyunSMSVerifyCodeSignName   string `json:"aliyun_sms_verify_code_sign_name"`
	AliyunSMSVerifyCodeTemplate   string `json:"aliyun_sms_verify_code_template_code"`
	AliyunSMSVerifyCodeStaticJSON string `json:"aliyun_sms_verify_code_static_params"`
	AliyunSMSTemplateCode         string `json:"aliyun_sms_template_code"`
	AliyunSMSTemplateParamKey     string `json:"aliyun_sms_template_param_key"`
	AliyunSMSTemplateStaticJSON   string `json:"aliyun_sms_template_static_params"`
	CarpoolAdminFullSMSTemplate   string `json:"carpool_admin_full_sms_template_code"`
	CarpoolUserActiveSMSTemplate  string `json:"carpool_user_active_sms_template_code"`
}

func (h *SettingHandler) GetModelMarketplaceConfig(c *gin.Context) {
	config, err := h.settingService.GetModelMarketplaceConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, config)
}

func (h *SettingHandler) UpdateModelMarketplaceConfig(c *gin.Context) {
	var req service.ModelMarketplaceConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	config, err := h.settingService.UpdateModelMarketplaceConfig(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, config)
}

func (h *SettingHandler) SendTestSMS(c *gin.Context) {
	var req sendTestSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	templateCode, err := h.settingService.SendAliyunTestSMS(c.Request.Context(), service.AliyunTestSMSOptions{
		Type:                           service.AliyunTestSMSType(strings.TrimSpace(req.Type)),
		Phone:                          req.Phone,
		AccessKeyID:                    req.AliyunSMSAccessKeyID,
		AccessKeySecret:                req.AliyunSMSAccessKeySecret,
		SignName:                       req.AliyunSMSSignName,
		RegistrationMode:               req.AliyunSMSRegistrationMode,
		VerifyCodeSignName:             req.AliyunSMSVerifyCodeSignName,
		VerifyCodeTemplateCode:         req.AliyunSMSVerifyCodeTemplate,
		VerifyCodeStaticJSON:           req.AliyunSMSVerifyCodeStaticJSON,
		RegistrationTemplateCode:       req.AliyunSMSTemplateCode,
		RegistrationTemplateParamKey:   req.AliyunSMSTemplateParamKey,
		RegistrationTemplateStaticJSON: req.AliyunSMSTemplateStaticJSON,
		CarpoolAdminFullTemplateCode:   req.CarpoolAdminFullSMSTemplate,
		CarpoolUserActiveTemplateCode:  req.CarpoolUserActiveSMSTemplate,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"message":       "Test SMS sent successfully",
		"template_code": templateCode,
	})
}
