package base

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	generic_gorm "github.com/harryosmar/generic-gorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TablerWithPrimaryKey interface {
	TableName() string
	PrimaryKey() string
}

type Paginator struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

type OrderBy struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // asc, desc
}

func (o OrderBy) String() string {
	if o.Field != "" && (o.Direction == "asc" || o.Direction == "desc") {
		return fmt.Sprintf("%s %s", o.Field, o.Direction)
	}

	return ""
}

type BaseGorm[T TablerWithPrimaryKey, PkType string | int64 | int32 | int | uint] struct {
	db *gorm.DB
}

func NewBaseGorm[T TablerWithPrimaryKey, PkType string | int64 | int32 | int | uint](db *gorm.DB) *BaseGorm[T, PkType] {
	return &BaseGorm[T, PkType]{db: db}
}

func (o *BaseGorm[T, PkType]) Detail(ctx context.Context, id PkType) (*T, error) {
	var (
		db       = o.db.WithContext(ctx)
		row      T
		logEntry = generic_gorm.GetLoggerFromContext(ctx)
		err      error
	)

	defer func() {
		if err != nil {
			logEntry.Error(err)
		}
	}()

	db = db.
		Table(row.TableName()).
		Where(
			fmt.Sprintf(
				"%s = ?",
				row.PrimaryKey(),
			),
			id,
		)

	if err = db.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &row, nil
}

type Where struct {
	Name             string      `json:"name"`
	IsLike           bool        `json:"is_like"`             // use "%keyword%" : WHERE name LIKE '%ware%'
	IsFullTextSearch bool        `json:"is_full_text_search"` // use "*keyword*" : WHERE MATCH(name) AGAINST ('*ware*' IN BOOLEAN MODE) : To fully optimize this, create index "FULLTEXT KEY `idx_fulltext_columName` (`columName`)", read also about stopwords https://dev.mysql.com/doc/refman/8.4/en/fulltext-stopwords.html
	Value            interface{} `json:"value"`
}

// UnmarshalJSON Custom for the Where struct
func (w *Where) UnmarshalJSON(data []byte) error {
	type Alias Where
	aux := &struct {
		IsFullTextSearch string `json:"is_full_text_search"` // Treat as string initially
		IsLike           string `json:"is_like"`             // Treat as string initially
		*Alias
	}{
		Alias: (*Alias)(w), // Embed the original struct
	}

	// Unmarshal JSON into the temporary struct
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Convert string "1" to boolean true for IsFullTextSearch
	w.IsFullTextSearch = aux.IsFullTextSearch == "1" || aux.IsFullTextSearch == "true"
	w.IsLike = aux.IsLike == "1" || aux.IsLike == "true"

	return nil
}

func (c *Where) String() string {
	whereSql := fmt.Sprintf("%s = ?", c.Name)
	if c.IsFullTextSearch {
		whereSql = fmt.Sprintf("MATCH(%s) AGAINST (? IN BOOLEAN MODE)", c.Name)
	} else if c.IsLike {
		whereSql = fmt.Sprintf("%s LIKE ?", c.Name)
	}

	return whereSql
}

func (o *BaseGorm[T, PkType]) Wheres(ctx context.Context, wheres []Where) (*T, error) {
	var (
		logEntry = generic_gorm.GetLoggerFromContext(ctx)
		row      T
		db       = o.db.WithContext(ctx).Table(row.TableName())
		err      error
	)

	defer func() {
		if err != nil {
			logEntry.Error(err)
		}
	}()

	for _, v := range wheres {
		if v.IsLike {
			v.Value = fmt.Sprintf("%%%s%%", v.Value)
		}
		db.Where(v.String(), v.Value)
	}

	if err = db.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &row, nil
}

func (o *BaseGorm[T, PkType]) WheresList(ctx context.Context, orders []OrderBy, wheres []Where) ([]T, error) {
	var (
		logEntry = generic_gorm.GetLoggerFromContext(ctx)
		e        T
		db       = o.db.WithContext(ctx).Table(e.TableName())
		rows     []T
		err      error
	)

	defer func() {
		if err != nil {
			logEntry.Error(err)
		}
	}()

	for _, v := range wheres {
		if v.IsLike {
			v.Value = fmt.Sprintf("%%%s%%", v.Value)
		}
		db.Where(v.String(), v.Value)
	}

	for _, order := range orders {
		orderByStr := order.String()
		if orderByStr != "" {
			db.Order(orderByStr)
		}
	}

	if err = db.Find(&rows).Error; err != nil {
		return rows, err
	}

	return rows, nil
}

func (o *BaseGorm[T, PkType]) List(ctx context.Context, page int, pageSize int, orders []OrderBy, wheres []Where) ([]T, *Paginator, error) {
	var (
		logEntry  = generic_gorm.GetLoggerFromContext(ctx)
		e         T
		db        = o.db.WithContext(ctx).Table(e.TableName())
		rows      []T
		count     int64
		err       error
		paginator = &Paginator{
			Page:    page,
			PerPage: pageSize,
			Total:   0,
		}
	)

	defer func() {
		if err != nil {
			logEntry.Error(err)
		}
	}()

	for _, v := range wheres {
		if v.IsLike {
			v.Value = fmt.Sprintf("%%%s%%", v.Value)
		}
		db.Where(v.String(), v.Value)
	}

	for _, order := range orders {
		orderByStr := order.String()
		if orderByStr != "" {
			db.Order(orderByStr)
		}
	}

	if err = db.Count(&count).Error; err != nil {
		return rows, nil, err
	}

	paginator.Total = int(count)
	if count == 0 {
		return rows, paginator, nil
	}

	if err = db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return rows, paginator, err
	}

	return rows, paginator, nil
}

func (o *BaseGorm[T, PkType]) Create(ctx context.Context, row *T) (*T, error) {
	var (
		e        T
		db       = o.db.WithContext(ctx).Table(e.TableName())
		logEntry = generic_gorm.GetLoggerFromContext(ctx)
		err      error
	)

	defer func() {
		if err != nil {
			logEntry.Error(err)
		}
	}()

	// cannot handle upsert will get err Duplicate entry
	if err = db.Create(row).Error; err != nil {
		return nil, err
	}

	return row, nil
}

func (o *BaseGorm[T, PkType]) DB(ctx context.Context) *gorm.DB {
	return o.db.WithContext(ctx)
}

func (o *BaseGorm[T, PkType]) CreateMultiple(ctx context.Context, rows []*T) ([]*T, int64, error) {
	var (
		rowsAffected int64
	)

	if len(rows) == 0 {
		return rows, rowsAffected, nil
	}

	var (
		e        T
		db       = o.db.WithContext(ctx).Table(e.TableName())
		logEntry = generic_gorm.GetLoggerFromContext(ctx)
		err      error
	)

	defer func() {
		if err != nil {
			logEntry.Error(err)
		}
	}()

	result := db.Create(rows)
	err = result.Error
	rowsAffected = result.RowsAffected

	return rows, rowsAffected, err
}

func (o *BaseGorm[T, PkType]) Update(ctx context.Context, row *T, updatedColumns []string) (int64, error) {
	var (
		e        T
		db       = o.db.WithContext(ctx).Table(e.TableName())
		logEntry = generic_gorm.GetLoggerFromContext(ctx)
		err      error
	)

	defer func() {
		if err != nil {
			logEntry.Error(err)
		}
	}()

	if len(updatedColumns) > 0 {
		db = db.Select(updatedColumns)
	}

	// Use the model to get the correct table and add WHERE clause for the primary key
	result := db.Model(row).Updates(row)
	err = result.Error

	return result.RowsAffected, err
}

func (o *BaseGorm[T, PkType]) UpdateWhere(ctx context.Context, wheres []Where, values map[string]interface{}) (int64, error) {
	var (
		e        T
		db       = o.db.WithContext(ctx).Table(e.TableName())
		logEntry = generic_gorm.GetLoggerFromContext(ctx)
		err      error
	)

	defer func() {
		if err != nil {
			logEntry.Error(err)
		}
	}()

	// Build where clauses
	for _, v := range wheres {
		if v.IsLike {
			v.Value = fmt.Sprintf("%%%s%%", v.Value)
		}
		db.Where(v.String(), v.Value)
	}

	// Execute update
	result := db.Updates(values)
	err = result.Error

	return result.RowsAffected, err
}

func (o *BaseGorm[T, PkType]) Upsert(ctx context.Context, row *T, onConflictUpdatedColumns []string) (int64, error) {
	var (
		e        T
		db       = o.db.WithContext(ctx).Table(e.TableName())
		logEntry = generic_gorm.GetLoggerFromContext(ctx)
		err      error
	)

	defer func() {
		if err != nil {
			logEntry.Error(err)
		}
	}()

	result := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{},
		DoUpdates: clause.AssignmentColumns(onConflictUpdatedColumns),
	}).Create(&row)

	return result.RowsAffected, result.Error
}

type ListCustomCallback = func(*gorm.DB) *gorm.DB

func (o *BaseGorm[T, PkType]) ListCustom(ctx context.Context, page int, pageSize int, orders []OrderBy, wheres []Where, customCallback ListCustomCallback) ([]T, *Paginator, error) {
	var (
		logEntry  = generic_gorm.GetLoggerFromContext(ctx)
		db        = o.db.WithContext(ctx)
		rows      []T
		count     int64
		err       error
		paginator = &Paginator{
			Page:    page,
			PerPage: pageSize,
			Total:   0,
		}
	)

	defer func() {
		if err != nil {
			logEntry.Error(err)
		}
	}()

	db = customCallback(db)

	for _, v := range wheres {
		if v.IsLike {
			v.Value = fmt.Sprintf("%%%s%%", v.Value)
		}
		db.Where(v.String(), v.Value)
	}

	for _, order := range orders {
		orderByStr := order.String()
		if orderByStr != "" {
			db.Order(orderByStr)
		}
	}

	if err = db.Count(&count).Error; err != nil {
		return rows, nil, err
	}

	paginator.Total = int(count)
	if count == 0 {
		return rows, paginator, nil
	}

	if err = db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return rows, paginator, err
	}

	return rows, paginator, nil
}

func (o *BaseGorm[T, PkType]) Association(ctx context.Context, model *T, field string) *gorm.Association {
	return o.db.WithContext(ctx).Model(model).Association(field)
}

func (o *BaseGorm[T, PkType]) AppendAssociation(ctx context.Context, model *T, field string, values interface{}) error {
	return o.Association(ctx, model, field).Append(values)
}

func (o *BaseGorm[T, PkType]) ReplaceAssociation(ctx context.Context, model *T, field string, values interface{}) error {
	return o.Association(ctx, model, field).Replace(values)
}

func (o *BaseGorm[T, PkType]) DeleteAssociation(ctx context.Context, model *T, field string, values interface{}) error {
	return o.Association(ctx, model, field).Delete(values)
}

func (o *BaseGorm[T, PkType]) ClearAssociation(ctx context.Context, model *T, field string) error {
	return o.Association(ctx, model, field).Clear()
}

func (o *BaseGorm[T, PkType]) CountAssociation(ctx context.Context, model *T, field string) int64 {
	return o.Association(ctx, model, field).Count()
}

func (o *BaseGorm[T, PkType]) FindAssociation(ctx context.Context, model *T, field string, dest interface{}) error {
	return o.Association(ctx, model, field).Find(dest)
}
