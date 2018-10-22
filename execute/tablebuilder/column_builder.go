package tablebuilder

import (
	"github.com/influxdata/flux"
	"github.com/influxdata/flux/execute"
	"github.com/influxdata/flux/values"
)

// New creates a new table builder for generating tables in a columnar method.
func New(a *execute.Allocator) *ColumnBuilder {
	panic("implement me")
}

type ColumnBuilder struct{}

// WithGroupKey will use the given group key for this table. If group key entries have
// already been added, this will append or replace the values within the constructed group key.
// See AddKeyValue for details of how the group key is constructed.
func (b *ColumnBuilder) WithGroupKey(key flux.GroupKey) *ColumnBuilder {
	panic("implement me")
}

type FloatColumn struct {
	// Name is the name of this column.
	Name string

	// Index is the column index of this column.
	Index int
}

func (c *FloatColumn) Append(value float64) {
	panic("implement me")
}

// AddKeyValue will add an additional column to the table and mark it as part of the group key.
// The column will not be modifiable as group keys remain consistent within the table.
// The column type is automatically inferred from the value.
func (b *ColumnBuilder) AddKeyValue(key string, value values.Value) error {
	panic("implement me")
}

// AddFloatColumn will create a new float column that is not part of the group key.
// If the column has already been added or is part of the group key, then this will fail with
// an error. The column will be passed to the function so it can be constructed.
func (b *ColumnBuilder) AddFloatColumn(name string, fn func(c *FloatColumn) error) error {
	panic("implement me")
}

// Build will validate the table is consistent and will return a flux.Table if it is.
func (b *ColumnBuilder) Build() (flux.Table, error) {
	panic("implement me")
}
