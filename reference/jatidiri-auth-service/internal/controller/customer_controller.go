package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/config"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/model"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/service"
)

func GetCustomersBulk(c *gin.Context) {
	idsParam := c.Query("ids")

	if idsParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids is required"})
		return
	}

	idStrings := strings.Split(idsParam, ",")
	var ids []uint

	for _, s := range idStrings {
		id, _ := strconv.ParseUint(s, 10, 64)
		ids = append(ids, uint(id))
	}

	var customers []model.Customer
	if err := config.DB.Where("id IN ?", ids).Find(&customers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, customers)
}

func GetCustomerByUserID(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id tidak valid",
		})
		return
	}

	customer, err := service.GetCustomerByUserID(uint(userID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, customer)
}

func GetCustomersInternal(c *gin.Context) {

	// request body
	var req struct {
		CustomerIDs []uint `json:"customer_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	data, err := service.GetCustomersByIDs(req.CustomerIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}

type UserResponse struct {
	UserID       uint   `json:"user_id"`
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	PhotoProfile string `json:"photo_profile"`
}

func GetUserInternal(c *gin.Context) {

	var req struct {
		UserIDs []uint `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var customers []model.Customer

	if err := config.DB.
		Preload("User").
		Where("user_id IN ?", req.UserIDs).
		Find(&customers).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var result []UserResponse

	for _, cst := range customers {
		name := ""
		if cst.FullName != nil {
			name = *cst.FullName
		}

		result = append(result, UserResponse{
			UserID:       cst.UserID,
			FullName:     name,
			Email:        cst.User.Email,
			Phone:        cst.User.Phone,
			PhotoProfile: cst.User.PhotoProfile,
		})
	}

	c.JSON(http.StatusOK, result)
}
