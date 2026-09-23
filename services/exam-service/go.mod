module github.com/pkm-asia/pmb-backend/services/exam-service

go 1.25.0

replace github.com/pkm-asia/pmb-backend/pkg => ../../pkg

require (
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	gorm.io/gorm v1.31.2
)

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/text v0.20.0 // indirect
)
