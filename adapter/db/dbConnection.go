package db

import (
	"fmt"
	"github.com/nkhoaa96/go-base/adapter/vault"
	"github.com/nkhoaa96/go-base/storage/local"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func GetDBConnection(opts ...gorm.Option) (*gorm.DB, error) {
	password, err := vault.GetSecretValue(local.Getenv("KV_DB_PASSWORD"))
	if err != nil {
		return nil, err
	}
	var (
		host     = local.Getenv("DB_HOST")
		port     = local.Getenv("DB_PORT")
		user     = local.Getenv("DB_USER")
		database = local.Getenv("DB_NAME")
		options  = []gorm.Option{
			&gorm.Config{
				Logger:                 logger.Default.LogMode(logger.Info),
				PrepareStmt:            true,
				SkipDefaultTransaction: true,
			},
		}
		psqlInfo = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, database)
	)

	options = append(options, opts...)
	return gorm.Open(postgres.Open(psqlInfo), opts...)
}
