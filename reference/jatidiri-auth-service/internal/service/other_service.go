package service

import (
	"time"

	"github.com/iamhexagoninc/go-jatidiri-app/internal/config"
)

type GenderStatistic struct {
	Male    int64 `json:"male"`
	Female  int64 `json:"female"`
	Unknown int64 `json:"unknown"`
}

type AgeDistribution struct {
	Age   int `json:"age"`
	Total int `json:"total"`
}

type CustomerStatistic struct {
	TotalCustomers  int64             `json:"total_customers"`
	Gender          GenderStatistic   `json:"gender"`
	AgeDistribution []AgeDistribution `json:"age_distribution"`
}

type CustomerUserList struct {
	UserID       uint   `json:"user_id"`
	CustomerID   uint   `json:"customer_id"`
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	PhotoProfile string `json:"photo_profile"`
	Role         string `json:"role"`
}

type CustomerUserPagination struct {
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
	Data  []CustomerUserList `json:"data"`
}

type CustomerUserDetail struct {
	UserID       uint       `json:"user_id"`
	CustomerID   uint       `json:"customer_id"`
	FullName     *string    `json:"full_name"`
	BirthPlace   *string    `json:"birth_place"`
	BirthDate    *time.Time `json:"birth_date"`
	Gender       *int8      `json:"gender"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone"`
	PhotoProfile string     `json:"photo_profile"`
	Role         string     `json:"role"`
	CreatedAt    time.Time  `json:"created_at"`
}

func GetCustomerStatistic() (CustomerStatistic, error) {

	var result CustomerStatistic

	//--------------------------------
	// Total Customer
	//--------------------------------
	if err := config.DB.
		Table("customers").
		Count(&result.TotalCustomers).Error; err != nil {
		return result, err
	}

	//--------------------------------
	// Gender Male
	//--------------------------------
	config.DB.
		Table("customers").
		Where("gender = ?", 1).
		Count(&result.Gender.Male)

	//--------------------------------
	// Gender Female
	//--------------------------------
	config.DB.
		Table("customers").
		Where("gender = ?", 2).
		Count(&result.Gender.Female)

	//--------------------------------
	// Gender Unknown
	//--------------------------------
	config.DB.
		Table("customers").
		Where("gender IS NULL").
		Count(&result.Gender.Unknown)

	//--------------------------------
	// Age Distribution
	//--------------------------------
	var ages []AgeDistribution

	err := config.DB.
		Table("customers").
		Select(`
			EXTRACT(YEAR FROM AGE(birth_date)) as age,
			COUNT(*) as total
		`).
		Where("birth_date IS NOT NULL").
		Group("age").
		Order("age ASC").
		Scan(&ages).Error

	if err != nil {
		return result, err
	}

	result.AgeDistribution = ages

	return result, nil
}

func GetCustomerUsers(page int, limit int, search string) (CustomerUserPagination, error) {

	var result CustomerUserPagination
	var data []CustomerUserList
	var total int64

	offset := (page - 1) * limit

	db := config.DB.
		Table("customers").
		Select(`
			users.id as user_id,
			customers.id as customer_id,
			customers.full_name,
			users.email,
			users.phone,
			users.photo_profile,
			users.role
		`).
		Joins("JOIN users ON users.id = customers.user_id")

	//--------------------------------
	// Search
	//--------------------------------
	if search != "" {
		db = db.Where(`
			customers.full_name ILIKE ? OR
			users.email ILIKE ? OR
			users.phone ILIKE ?
		`, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	//--------------------------------
	// Total Data
	//--------------------------------
	db.Count(&total)

	//--------------------------------
	// Pagination
	//--------------------------------
	err := db.
		Limit(limit).
		Offset(offset).
		Order("customers.created_at DESC").
		Scan(&data).Error

	if err != nil {
		return result, err
	}

	result.Total = total
	result.Page = page
	result.Limit = limit
	result.Data = data

	return result, nil
}

func GetCustomerUserDetail(customerID uint) (CustomerUserDetail, error) {

	var result CustomerUserDetail

	err := config.DB.
		Table("customers").
		Select(`
			users.id as user_id,
			customers.id as customer_id,
			customers.full_name,
			customers.birth_place,
			customers.birth_date,
			customers.gender,
			users.email,
			users.phone,
			users.photo_profile,
			users.role,
			customers.created_at
		`).
		Joins("JOIN users ON users.id = customers.user_id").
		Where("customers.id = ?", customerID).
		Scan(&result).Error

	if err != nil {
		return result, err
	}

	return result, nil
}
