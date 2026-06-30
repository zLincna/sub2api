package admin

import (
	"strconv"
	"strings"
	"time"

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
	filters := service.LotteryDrawRecordFilters{
		UserID:     userID,
		UserQuery:  strings.TrimSpace(c.Query("user_query")),
		SourceType: strings.TrimSpace(c.Query("source_type")),
	}
	if start := parseLotteryTimeQuery(c.Query("start_time")); !start.IsZero() {
		filters.StartTime = start
	}
	if end := parseLotteryTimeQuery(c.Query("end_time")); !end.IsZero() {
		filters.EndTime = end
	}
	items, total, err := h.lotteryService.ListAdminDrawRecords(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func parseLotteryTimeQuery(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t
		}
	}
	return time.Time{}
}
