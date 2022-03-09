package tempodb_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/kubeshop/tracetest/server/go/tracedb/tempodb"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
)

func TestGetTraceByID(t *testing.T) {
	t.Skip("TODO: docker-compose tempo")
	cfg := configgrpc.GRPCClientSettings{
		Endpoint:   "localhost:9095",
		TLSSetting: configtls.TLSClientSetting{Insecure: true},
	}

	db, err := tempodb.New(&cfg)
	assert.NoError(t, err)

	defer db.Close()
	trace, err := db.GetTraceByID(context.Background(), "0194fdc2fa2ffcc041d3ff12045b73c9")
	assert.NoError(t, err)

	buf := bytes.Buffer{}
	m := jsonpb.Marshaler{}
	err = m.Marshal(&buf, trace)
	assert.NoError(t, err)
	fmt.Printf("\n%s\n", buf.String())
}
