package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/service"
)

func GetStatistikCustomers(c *gin.Context) {

	result, err := service.GetCustomerStatistic()
	if err != nil {
		fmt.Println("[Customer Statistik ERROR]", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Customer statistic",
		"data":    result,
	})
}

func GetCustomersUsers(c *gin.Context) {

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	search := c.Query("search")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	result, err := service.GetCustomerUsers(page, limit, search)

	if err != nil {
		fmt.Println("[Customer Users ERROR]", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Customer users list",
		"data":    result,
	})
}

func GetCustomerUserDetail(c *gin.Context) {

	customerParam := c.Param("customer_id")

	customerID, err := strconv.ParseUint(customerParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid customer id",
		})
		return
	}

	result, err := service.GetCustomerUserDetail(uint(customerID))

	if err != nil {
		fmt.Println("[Customer Detail ERROR]", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Customer detail",
		"data":    result,
	})
}
