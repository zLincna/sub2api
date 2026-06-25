package admin

import (
	"strconv"

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

func (h *CarpoolHandler) Overview(c *gin.Context) {
	data, err := h.carpoolService.AdminOverview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}

func (h *CarpoolHandler) ListVehicleTypes(c *gin.Context) {
	items, err := h.carpoolService.ListVehicleTypes(c.Request.Context(), false)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CarpoolHandler) CreateVehicleType(c *gin.Context) {
	var req service.CarpoolVehicleTypeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.carpoolService.AdminCreateVehicleType(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

func (h *CarpoolHandler) UpdateVehicleType(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid vehicle type id")
		return
	}
	var req service.CarpoolVehicleTypeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.carpoolService.AdminUpdateVehicleType(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *CarpoolHandler) DeleteVehicleType(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid vehicle type id")
		return
	}
	if err := h.carpoolService.AdminDeleteVehicleType(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *CarpoolHandler) ListSessions(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.carpoolService.AdminListSessions(c.Request.Context(), page, pageSize, c.Query("status"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, int64(total), page, pageSize)
}

func (h *CarpoolHandler) Management(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	data, err := h.carpoolService.AdminManagement(c.Request.Context(), page, pageSize, c.Query("status"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}

func (h *CarpoolHandler) ProvisionSession(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid session id")
		return
	}
	var req service.CarpoolSessionProvisionInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.carpoolService.AdminProvisionSession(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *CarpoolHandler) ListVouchers(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid session id")
		return
	}
	items, err := h.carpoolService.AdminListVouchers(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CarpoolHandler) CreateVoucher(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid session id")
		return
	}
	var req service.CarpoolVoucherInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	var uploadedBy int64
	if subject, ok := servermiddleware.GetAuthSubjectFromContext(c); ok {
		uploadedBy = subject.UserID
	}
	item, err := h.carpoolService.AdminCreateVoucher(c.Request.Context(), id, uploadedBy, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

func (h *CarpoolHandler) DeleteVoucher(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid voucher id")
		return
	}
	if err := h.carpoolService.AdminDeleteVoucher(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *CarpoolHandler) ListNotices(c *gin.Context) {
	items, err := h.carpoolService.AdminListNotices(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CarpoolHandler) CreateNotice(c *gin.Context) {
	var req service.CarpoolNoticeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.carpoolService.AdminCreateNotice(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}
