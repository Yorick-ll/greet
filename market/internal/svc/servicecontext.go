package svc

import (
	"fmt"
	"greet/market/internal/config"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ServiceContext struct {
	Config config.Config
	DB     *gorm.DB
}

func NewServiceContext(c config.Config) *ServiceContext {
	masterDsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		c.Mysql.Master.Username, c.Mysql.Master.Password, c.Mysql.Master.Path, c.Mysql.Master.Port, c.Mysql.Master.Dbname)
	db, err := gorm.Open(mysql.Open(masterDsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		panic(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}

	sqlDB.SetMaxOpenConns(c.Mysql.Master.MaxOpenConns)
	sqlDB.SetMaxIdleConns(c.Mysql.Master.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	if err = sqlDB.Ping(); err != nil {
		panic(err)
	}

	return &ServiceContext{
		Config: c,
		DB:     db,
	}
}
