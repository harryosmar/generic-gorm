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
Gorm with generic with methods :
//- (o *BaseGorm[T, PkType]) Detail(ctx context.Context, id PkType) (*T, error)
//- (o *BaseGorm[T, PkType]) Wheres(ctx context.Context, wheres []Where)
//- (o *BaseGorm[T, PkType]) WheresList(ctx context.Context, orders []OrderBy, wheres []Where) ([]T, error)
//- (o *BaseGorm[T, PkType]) List(ctx context.Context, page int, pageSize int, orders []OrderBy, wheres []Where) ([]T, *Paginator, error)
//- (o *BaseGorm[T, PkType]) Create(ctx context.Context, row *T) (*T, error)
//- (o *BaseGorm[T, PkType]) CreateMultiple(ctx context.Context, rows []*T) ([]*T, int64, error)
//- (o *BaseGorm[T, PkType]) Update(ctx context.Context, row *T, updatedColumns []string) (int64, error)
//- (o *BaseGorm[T, PkType]) UpdateWhere(ctx context.Context, wheres []Where, values map[string]interface{}) (int64, error)
//- (o *BaseGorm[T, PkType]) Upsert(ctx context.Context, row *T, onConflictUpdatedColumns []string) (int64, error)
//- (o *BaseGorm[T, PkType]) ListCustom(ctx context.Context, page int, pageSize int, orders []OrderBy, wheres []Where, customCallback ListCustomCallback) ([]T, *Paginator, error)
```