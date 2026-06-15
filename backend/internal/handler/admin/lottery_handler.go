package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type LotteryHandler struct {
	lotteryService *service.LotteryService
}

func NewLotteryHandler(lotteryService *service.LotteryService) *LotteryHandler {
	return &LotteryHandler{lotteryService: lotteryService}
}

func (h *LotteryHandler) GetConfig(c *gin.Context) {
	cfg, err := h.lotteryService.GetConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

func (h *LotteryHandler) UpdateConfig(c *gin.Context) {
	var req service.LotteryConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	cfg, err := h.lotteryService.UpdateConfig(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

func (h *LotteryHandler) ListPrizes(c *gin.Context) {
	prizes, err := h.lotteryService.ListPrizes(c.Request.Context(), false)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, prizes)
}

func (h *LotteryHandler) CreatePrize(c *gin.Context) {
	var req service.LotteryPrizeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	prize, err := h.lotteryService.CreatePrize(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, prize)
}

func (h *LotteryHandler) UpdatePrize(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid prize id")
		return
	}
	var req service.LotteryPrizeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	prize, err := h.lotteryService.UpdatePrize(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, prize)
}

func (h *LotteryHandler) DeletePrize(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid prize id")
		return
	}
	if err := h.lotteryService.DeletePrize(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *LotteryHandler) ListRecords(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	userID, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	items, total, err := h.lotteryService.ListAdminDrawRecords(c.Request.Context(), page, pageSize, userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}
