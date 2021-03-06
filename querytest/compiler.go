package querytest

import (
	"context"

	"github.com/influxdata/flux"
	"github.com/influxdata/flux/stdlib/csv"
	"github.com/influxdata/flux/stdlib/influxdata/influxdb"
	"github.com/influxdata/flux/stdlib/influxdata/influxdb/v1"
)

// FromCSVCompiler wraps a compiler and replaces all From operations with FromCSV
type FromCSVCompiler struct {
	flux.Compiler
	InputFile string
}

// FromInfluxJSONCompiler wraps a compiler and replaces all From operations with FromJSON
type FromInfluxJSONCompiler struct {
	flux.Compiler
	InputFile string
}

func (c FromCSVCompiler) Compile(ctx context.Context) (*flux.Spec, error) {
	spec, err := c.Compiler.Compile(ctx)
	if err != nil {
		return nil, err
	}
	ReplaceFromSpec(spec, c.InputFile)
	return spec, nil
}

func (c FromInfluxJSONCompiler) Compile(ctx context.Context) (*flux.Spec, error) {
	spec, err := c.Compiler.Compile(ctx)
	if err != nil {
		return nil, err
	}
	ReplaceFromWithFromInfluxJSONSpec(spec, c.InputFile)
	return spec, nil
}

func ReplaceFromSpec(q *flux.Spec, csvSrc string) {
	for _, op := range q.Operations {
		if op.Spec.Kind() == influxdb.FromKind {
			op.Spec = &csv.FromCSVOpSpec{
				File: csvSrc,
			}
		}
	}
}

func ReplaceFromWithFromInfluxJSONSpec(q *flux.Spec, jsonSrc string) {
	for _, op := range q.Operations {
		if op.Spec.Kind() == influxdb.FromKind {
			op.Spec = &v1.FromInfluxJSONOpSpec{
				File: jsonSrc,
			}
		}
	}
}
