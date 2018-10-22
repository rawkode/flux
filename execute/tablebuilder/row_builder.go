package tablebuilder

import (
	"github.com/influxdata/flux"
	"github.com/influxdata/flux/execute"
	"github.com/influxdata/flux/values"
)

// NewRowBuilder creates a new table builder that is focused on constructing tables
// with row-based algorithms. In general, algorithms should prioritize using the
// ColumnBuilder with New, but some algorithms are row-based.
func NewRowBuilder(a *execute.Allocator) *RowBuilder {
	panic("implement me")
}

type RowBuilder struct{}

// WithGroupKey will use the given group key for this table. If group key entries have
// already been added, this will append or replace the values within the constructed group key.
// See AddKeyValue for details of how the group key is constructed.
func (b *RowBuilder) WithGroupKey(key flux.GroupKey) *RowBuilder {
	panic("implement me")
}

// AddKeyValue will add an additional column to the table and mark it as part of the group key.
// The column will not be modifiable as group keys remain consistent within the table.
// The column type is automatically inferred from the value.
func (b *RowBuilder) AddKeyValue(key string, value values.Value) error {
	panic("implement me")
}

// AddColumn will add a new column with the given type. If the column has already been
// added with a conflicting type, then this will return an error.
func (b *RowBuilder) AddColumn(key string, typ flux.ColType) error {
	panic("implement me")
}

// AppendMap will read the mapping of key/value pairs and add them as an additional row
// within the table at the appropriate index. If the Value is not of the correct type,
// this will return an error.
func (b *RowBuilder) AppendMap(m map[string]values.Value) error {
	panic("implement me")
}

// Build validates the table is constructed correctly and will return a flux.Table.
func (b *RowBuilder) Build() (flux.Table, error) {
	panic("implement me")
}
