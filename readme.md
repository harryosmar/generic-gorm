# Gorm(Golang ORM) with generic features


## Create entities

```go
package main

import (
	"context"
	"github.com/harryosmar/generic-gorm/base"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type DummyEntities struct {
	Id     int64  `json:"id" gorm:"column:id"`
	Field1 string `json:"field_1" gorm:"column:field_1"`
}

func (t DummyEntities) TableName() string {
	return "dummy"
}

func (t DummyEntities) PrimaryKey() string {
	return "id"
}

type MySQLDummyRepository struct {
	*base.BaseGorm[DummyEntities, int64]
}

func NewMySQLDummyRepository(db *gorm.DB) *MySQLDummyRepository {
	return &MySQLDummyRepository{
		base.NewBaseGorm[DummyEntities, int64](db),
	}
}


// Create MySQLDummyRepository with inherited methods from ./base/core.go :
```