package service

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/iamhexagoninc/go-jatidiri-app/internal/config"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/model"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UpdateProfileRequest struct {
	FullName   string `json:"full_name" binding:"required"`
	Phone      string `json:"phone"`
	BirthPlace string `json:"birth_place" binding:"required"`
	BirthDate  string `json:"birth_date" binding:"required"`
	Gender     string `json:"gender" binding:"required"`
}

type UpdateProfileRequestById struct {
	CustomerId uint   `json:"customer_id" binding:"required"`
	FullName   string `json:"full_name" binding:"required"`
	Phone      string `json:"phone"`
	BirthPlace string `json:"birth_place" binding:"required"`
	BirthDate  string `json:"birth_date" binding:"required"`
	Gender     string `json:"gender" binding:"required"`
}

type CustomerResponse struct {
	ID         uint      `json:"id"`
	FullName   string    `json:"full_name"`
	Email      string    `json:"email"`
	BirthPlace string    `json:"birth_place"`
	BirthDate  time.Time `json:"birth_date"`
	Gender     int8      `json:"gender"`
	UserID     uint      `json:"user_id"`
	Role       string    `json:"role"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type UserResponse struct {
	ID       uint              `json:"id"`
	Email    string            `json:"email"`
	Role     string            `json:"role"`
	Phone    string            `json:"phone"`
	Customer *CustomerResponse `json:"customer,omitempty"`
}

type LoginResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	Message      string        `json:"message"`
	User         *UserResponse `json:"user"`
}

type PhotoProfileRequest struct {
	PhotoProfile string
}

type CreateCustomerImportInput struct {
	UserID    uint
	FullName  string
	BirthDate time.Time
}

func Register(email, password string, role string, isImport bool) (*LoginResponse, error) {
	var existing model.User
	err := config.DB.Where("email = ?", email).First(&existing).Error
	if err == nil {
		return nil, errors.New("email already registered")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("failed to check existing user")
	}

	if role != "students" && role != "teachers" && role != "regulers" && role != "workers" && role != "families" && role != "psychology" && role != "marketing" && role != "other" {
		return nil, errors.New("invalid role specified")
	}

	cost := bcrypt.DefaultCost
	if v := os.Getenv("BCRYPT_COST"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= bcrypt.MinCost && parsed <= bcrypt.MaxCost {
			cost = parsed
		}
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// =====================
	// OTP / Auto Verify
	// =====================
	var verifiedAt *time.Time = nil
	var otp *int = nil
	var otpExpired *time.Time = nil

	if isImport {
		now := time.Now()
		verifiedAt = &now
	} else {
		rand.Seed(time.Now().UnixNano())
		code := rand.Intn(900000) + 100000
		exp := time.Now().Add(5 * time.Minute)

		otp = &code
		otpExpired = &exp
	}

	user := model.User{
		Email:             email,
		Password:          string(hashed),
		Role:              "customer",
		EmailVerifiedAt:   verifiedAt, // 🔥 AUTO VERIFIED IF IMPORT
		EmailOtp:          otp,
		EmailOtpExpiresAt: otpExpired,
		EmailOtpAttempt:   new(int16),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := config.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	customer := model.Customer{
		UserID:    user.ID,
		Role:      string(role),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := config.DB.Create(&customer).Error; err != nil {
		return nil, errors.New("failed to create customer record")
	}

	user.Customer = customer

	// Kirim OTP via email
	if !isImport && otp != nil {
		go func() {
			if err := utils.SendOtpEmail(email, *otp); err != nil {
				log.Println("Gagal kirim OTP:", err)
			}
		}()
	}

	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	if err := config.SaveRefreshToken(user.ID, refreshToken, 7*24*time.Hour); err != nil {
		return nil, errors.New("failed to save refresh token")
	}

	// Siapkan user response
	fullName := ""
	if user.Customer.FullName != nil {
		fullName = *user.Customer.FullName
	}

	resp := &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Message:      "login success",
		User: &UserResponse{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
			Phone: user.Phone,
			Customer: &CustomerResponse{
				ID:        user.Customer.ID,
				FullName:  fullName,
				UserID:    user.Customer.UserID,
				Role:      user.Customer.Role,
				CreatedAt: user.Customer.CreatedAt,
				UpdatedAt: user.Customer.UpdatedAt,
			},
		},
	}

	return resp, nil
}

func VerifyEmail(email string, otp int) error {
	var user model.User
	if err := config.DB.Preload("Customer").Where("email = ?", email).First(&user).Error; err != nil {
		return errors.New("user not found")
	}

	if user.EmailVerifiedAt != nil {
		return errors.New("email already verified")
	}

	if user.EmailOtp != nil && *user.EmailOtp != otp {
		return errors.New("invalid otp code")
	}

	if user.EmailOtpExpiresAt != nil && time.Now().After(*user.EmailOtpExpiresAt) {
		return errors.New("otp code has expired")
	}

	if user.EmailOtpAttempt != nil && *user.EmailOtpAttempt >= 3 {
		return errors.New("maximum otp attempts exceeded")
	}

	user.EmailVerifiedAt = utils.TimePtr(time.Now())
	user.EmailOtp = nil
	user.EmailOtpExpiresAt = nil
	user.EmailOtpAttempt = nil

	if err := config.DB.Save(&user).Error; err != nil {
		return errors.New("failed to verify email")
	}
	return nil
}

func SendOtpToEmail(email string) error {
	var user model.User
	if err := config.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return errors.New("user not found")
	}

	// Generate 6 digit OTP
	rand.Seed(time.Now().UnixNano())
	otp := rand.Intn(900000) + 100000
	expiration := time.Now().Add(5 * time.Minute)
	attempt := int16(0)

	user.EmailOtp = &otp
	user.EmailOtpExpiresAt = &expiration
	user.EmailOtpAttempt = &attempt

	if err := config.DB.Save(&user).Error; err != nil {
		return errors.New("failed to save OTP")
	}

	if err := utils.SendOtpEmail(email, otp); err != nil {
		return errors.New("failed to send OTP email")
	}

	return nil
}

func UpdateCustomerProfile(userID uint, req UpdateProfileRequest) error {
	var customer model.Customer
	var user model.User

	// Pastikan customer milik user ditemukan
	if err := config.DB.Where("user_id = ?", userID).First(&customer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("customer not found")
		}
		return err
	}

	// Konversi string BirthDate ke time.Time
	birthDate, err := time.Parse("2006-01-02", req.BirthDate)
	if err != nil {
		return errors.New("invalid birth date format, use YYYY-MM-DD")
	}

	// Update informasi customer
	customer.FullName = &req.FullName
	customer.BirthPlace = &req.BirthPlace
	customer.BirthDate = &birthDate
	genderInt, err := strconv.ParseInt(req.Gender, 10, 8)
	if err != nil {
		return errors.New("invalid gender value")
	}
	gender := int8(genderInt)
	customer.Gender = &gender
	customer.UpdatedAt = time.Now()

	// Simpan perubahan
	if err := config.DB.Save(&customer).Error; err != nil {
		return errors.New("failed to update customer profile")
	}

	// Pastikan user ditemukan
	if err := config.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	// Update informasi user
	user.Phone = req.Phone
	user.UpdatedAt = time.Now()

	// Simpan perubahan
	if err := config.DB.Save(&user).Error; err != nil {
		return errors.New("failed to update user profile")
	}

	return nil
}

func UpdateCustomerProfileById(id uint, req UpdateProfileRequestById) error {
	var customer model.Customer
	var user model.User

	// =========================
	// 1. Cari customer by ID
	// =========================
	if err := config.DB.Where("id = ?", id).First(&customer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("customer not found")
		}
		return err
	}

	// =========================
	// 2. Parse tanggal lahir
	// =========================
	birthDate, err := time.Parse("2006-01-02", req.BirthDate)
	if err != nil {
		return errors.New("invalid birth date format, use YYYY-MM-DD")
	}

	// =========================
	// 3. Update customer
	// =========================
	customer.FullName = &req.FullName
	customer.BirthPlace = &req.BirthPlace
	customer.BirthDate = &birthDate

	genderInt, err := strconv.ParseInt(req.Gender, 10, 8)
	if err != nil {
		return errors.New("invalid gender value")
	}
	gender := int8(genderInt)
	customer.Gender = &gender

	customer.UpdatedAt = time.Now()

	if err := config.DB.Save(&customer).Error; err != nil {
		return errors.New("failed to update customer profile")
	}

	// =========================
	// 4. Ambil user dari relasi
	// =========================
	if err := config.DB.Where("id = ?", customer.UserID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	// =========================
	// 5. Update user
	// =========================
	user.Phone = req.Phone
	user.UpdatedAt = time.Now()

	if err := config.DB.Save(&user).Error; err != nil {
		return errors.New("failed to update user profile")
	}

	return nil
}

func UpdateCustomerRole(userID uint, role string) error {
	var customer model.Customer

	// Pastikan customer milik user ditemukan
	if err := config.DB.Where("user_id = ?", userID).First(&customer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("customer not found")
		}
		return err
	}

	// Update informasi customer
	customer.Role = role
	customer.UpdatedAt = time.Now()

	// Simpan perubahan
	if err := config.DB.Save(&customer).Error; err != nil {
		return errors.New("failed to change customer role")
	}

	return nil
}

func UpdatePhotoProfile(userID uint, req PhotoProfileRequest) (*model.User, error) {
	var user model.User

	if err := config.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	user.PhotoProfile = req.PhotoProfile
	user.UpdatedAt = time.Now()

	if err := config.DB.Save(&user).Error; err != nil {
		return nil, errors.New("failed to change profile")
	}

	return &user, nil
}

func Login(email, password string) (*LoginResponse, error) {
	var user model.User
	// if err := config.DB.Preload("Customer").Where("email = ?", email).First(&user).Error; err != nil {
	// 	return nil, errors.New("user not found")
	// }

	if err := config.DB.
		Preload("Customer", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC").Limit(1)
		}).
		Where("email = ?", email).
		First(&user).Error; err != nil {
		return nil, errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid password")
	}

	// Cek apakah email sudah diverifikasi
	if user.EmailVerifiedAt == nil {
		return nil, errors.New("Email not verified")
	}

	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	if err := config.SaveRefreshToken(user.ID, refreshToken, 7*24*time.Hour); err != nil {
		return nil, errors.New("failed to save refresh token")
	}

	// Siapkan user response
	fullName := ""
	if user.Customer.FullName != nil {
		fullName = *user.Customer.FullName
	}
	birthPlace := ""
	if user.Customer.BirthPlace != nil {
		birthPlace = *user.Customer.BirthPlace
	}
	gender := int8(0)
	if user.Customer.Gender != nil {
		gender = *user.Customer.Gender
	}
	birthDate := time.Time{}
	if user.Customer.BirthDate != nil {
		birthDate = *user.Customer.BirthDate
	}

	resp := &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Message:      "login success",
		User: &UserResponse{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
			Phone: user.Phone,
			Customer: &CustomerResponse{
				ID:         user.Customer.ID,
				FullName:   fullName,
				BirthPlace: birthPlace,
				BirthDate:  birthDate,
				Gender:     gender,
				UserID:     user.Customer.UserID,
				Role:       user.Customer.Role,
				CreatedAt:  user.Customer.CreatedAt,
				UpdatedAt:  user.Customer.UpdatedAt,
			},
		},
	}

	return resp, nil
}

func GoogleLoginCallback(googleID, email, name, picture, googleAccessToken string) (string, string, *model.User, error) {
	var user model.User

	err := config.DB.Where("google_id = ? OR email = ?", googleID, email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = model.User{
			Email:             email,
			PhotoProfile:      picture,
			Role:              "customer",
			GoogleID:          &googleID,
			GoogleAccessToken: &googleAccessToken,
			EmailVerifiedAt:   utils.TimePtr(time.Now()),
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if err := config.DB.Create(&user).Error; err != nil {
			return "", "", nil, err
		}

		customer := model.Customer{
			UserID:    user.ID,
			FullName:  utils.StringPtr(name),
			Role:      "students",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_ = config.DB.Create(&customer)
	} else if err != nil {
		return "", "", nil, err
	} else {
		user.GoogleAccessToken = &googleAccessToken
		user.PhotoProfile = picture
		user.EmailVerifiedAt = utils.TimePtr(time.Now())
		user.UpdatedAt = time.Now()
		config.DB.Save(&user)
	}

	accessJWT, err := utils.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return "", "", nil, err
	}

	refreshJWT, err := utils.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		return "", "", nil, err
	}

	if err := config.SaveRefreshToken(user.ID, refreshJWT, 7*24*time.Hour); err != nil {
		return "", "", nil, err
	}

	return accessJWT, refreshJWT, &user, nil
}

func MeService(userID uint) (*model.User, error) {
	var user model.User

	if err := config.DB.
		Preload("Customer", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC").Limit(1)
		}).
		Where("id = ?", userID).
		First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func GetCustomerByUserID(userID uint) (*model.Customer, error) {
	var customer model.Customer

	if err := config.DB.
		Where("user_id = ?", userID).
		First(&customer).Error; err != nil {
		return nil, fmt.Errorf("customer tidak ditemukan")
	}

	return &customer, nil
}

func CreateCustomerFromImport(
	userID uint,
	fullName string,
	birthDate time.Time,
) (*model.Customer, error) {

	customer := model.Customer{
		UserID:    userID,
		FullName:  &fullName,
		BirthDate: &birthDate,
		Role:      "students",
	}

	if err := config.DB.Create(&customer).Error; err != nil {
		return nil, err
	}

	return &customer, nil
}

func GetCustomersByIDs(ids []uint) ([]CustomerResponse, error) {
	if len(ids) == 0 {
		return nil, errors.New("customer_ids kosong")
	}

	var customers []model.Customer
	if err := config.DB.
		Where("id IN ?", ids).
		Find(&customers).Error; err != nil {
		return nil, err
	}

	var result []CustomerResponse
	for _, c := range customers {
		if c.FullName == nil || c.BirthDate == nil {
			continue
		}

		result = append(result, CustomerResponse{
			ID:        c.ID,
			FullName:  *c.FullName,
			Email:     getEmailByUserID(c.UserID),
			BirthDate: *c.BirthDate,
		})
	}

	return result, nil
}

func getEmailByUserID(userID uint) string {
	var u struct {
		Email string
	}
	config.DB.Table("users").
		Select("email").
		Where("id = ?", userID).
		Scan(&u)

	return u.Email
}

func ForgotPassword(email string) error {
	var user model.User

	if err := config.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return errors.New("email tidak ditemukan")
	}

	// Generate token reset
	token := utils.GenerateRandomString(32)

	expired := time.Now().Add(30 * time.Minute)

	user.ResetPasswordToken = &token
	user.ResetPasswordExpiresAt = &expired

	if err := config.DB.Save(&user).Error; err != nil {
		return errors.New("gagal menyimpan token reset password")
	}

	// Link reset password
	resetLink := fmt.Sprintf(
		"%s/reset-password?token=%s",
		os.Getenv("FRONTEND_URL"),
		token,
	)

	// Kirim email
	if err := utils.SendResetPasswordEmail(email, resetLink); err != nil {
		return errors.New("gagal mengirim email reset password")
	}

	return nil
}

func ResetPassword(token string, newPassword string) error {
	var user model.User

	if err := config.DB.
		Where("reset_password_token = ?", token).
		First(&user).Error; err != nil {
		return errors.New("token tidak valid")
	}

	if user.ResetPasswordExpiresAt != nil && time.Now().After(*user.ResetPasswordExpiresAt) {
		return errors.New("token reset password telah expired")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("gagal hash password")
	}

	user.Password = string(hashed)
	user.ResetPasswordToken = nil
	user.ResetPasswordExpiresAt = nil

	if err := config.DB.Save(&user).Error; err != nil {
		return errors.New("gagal reset password")
	}

	return nil
}
