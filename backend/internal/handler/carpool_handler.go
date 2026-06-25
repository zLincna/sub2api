package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type CarpoolHandler struct {
	carpoolService *service.CarpoolService
}

func NewCarpoolHandler(carpoolService *service.CarpoolService) *CarpoolHandler {
	return &CarpoolHandler{carpoolService: carpoolService}
}

func (h *CarpoolHandler) ListCards(c *gin.Context) {
	cards, err := h.carpoolService.GetCurrentCards(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cards)
}

func (h *CarpoolHandler) CurrentNotice(c *gin.Context) {
	notice, err := h.carpoolService.GetCurrentNotice(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, notice)
}

func (h *CarpoolHandler) Join(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	var req service.CarpoolJoinInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.ClientIP = c.ClientIP()
	req.IsMobile = isMobile(c)
	req.IsWeChatBrowser = isWeChatBrowser(c)
	if strings.EqualFold(strings.TrimSpace(req.PaymentType), "alipay") {
		req.IsMobile = false
	}
	req.SrcHost = c.Request.Host
	req.SrcURL = c.Request.Referer()
	req.Locale = c.GetHeader("Accept-Language")
	if strings.TrimSpace(req.PaymentSource) == "" {
		req.PaymentSource = "carpool"
	}
	result, err := h.carpoolService.Join(c.Request.Context(), subject.UserID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *CarpoolHandler) My(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	items, err := h.carpoolService.MyParticipations(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CarpoolHandler) MyDetail(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid carpool record id")
		return
	}
	detail, err := h.carpoolService.MyParticipationDetail(c.Request.Context(), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, detail)
}

func (h *CarpoolHandler) RequestRefund(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid carpool record id")
		return
	}
	var req service.CarpoolRefundRequestInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.carpoolService.RequestRefund(c.Request.Context(), subject.UserID, id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}
