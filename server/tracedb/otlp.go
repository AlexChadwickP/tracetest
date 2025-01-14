package tracedb

import (
	"context"
	"strings"

	"github.com/kubeshop/tracetest/server/model"
	"github.com/kubeshop/tracetest/server/traces"
)

type OTLPTraceDB struct {
	db model.RunRepository
}

func newCollectorDB(repository model.RunRepository) (TraceDB, error) {
	return &OTLPTraceDB{
		db: repository,
	}, nil
}

func (tdb *OTLPTraceDB) Close() error {
	// No need to implement this
	return nil
}

// GetTraceByID implements TraceDB
func (tdb *OTLPTraceDB) GetTraceByID(ctx context.Context, id string) (traces.Trace, error) {
	run, err := tdb.db.GetRunByTraceID(ctx, traces.DecodeTraceID(id))
	if err != nil && strings.Contains(err.Error(), "record not found") {
		return traces.Trace{}, ErrTraceNotFound
	}

	if run.Trace == nil {
		return traces.Trace{}, ErrTraceNotFound
	}

	return *run.Trace, nil
}

var _ TraceDB = &OTLPTraceDB{}
