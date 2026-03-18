package prometheusx

import (
	"context"
	"time"

	"github.com/grafana/sobek"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/prometheus", new(RootModule))
}

type RootModule struct{}

func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &ModuleInstance{vu: vu}
}

type ModuleInstance struct {
	vu modules.VU
}

func (m *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]any{
			"NewClient": m.NewClientCtor,
		},
	}
}

func (m *ModuleInstance) NewClientCtor(call sobek.ConstructorCall) *sobek.Object {
	rt := m.vu.Runtime()

	var cfg map[string]any
	if len(call.Arguments) > 0 && !common.IsNullish(call.Argument(0)) {
		if err := rt.ExportTo(call.Argument(0), &cfg); err != nil {
			common.Throw(rt, err)
		}
	}

	address := "http://localhost:9090"
	if v, ok := cfg["address"].(string); ok && v != "" {
		address = v
	}

	client, err := api.NewClient(api.Config{Address: address})
	if err != nil {
		common.Throw(rt, err)
	}

	api := v1.NewAPI(client)
	pc := &PromClient{api: api}

	obj := rt.NewObject()
	_ = obj.Set("query", pc.Query)
	_ = obj.Set("evaluateThreshold", pc.EvaluateThreshold)

	return obj
}

type PromClient struct {
	api v1.API
}

func (p *PromClient) Query(query string) float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, _, err := p.api.Query(ctx, query, time.Now())
	if err != nil {
		panic(err)
	}

	switch res.Type() {
	case model.ValScalar:
		s := res.(*model.Scalar)
		return float64(s.Value)

	case model.ValVector:
		v := res.(model.Vector)
		if len(v) == 0 {
			return 0
		}
		return float64(v[0].Value)

	default:
		panic("unsupported type")
	}
}

func (p *PromClient) EvaluateThreshold(query string, threshold float64) bool {
	v := p.Query(query)
	return v > threshold
}
