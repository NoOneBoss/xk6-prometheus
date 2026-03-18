package prometheusx

import (
	"context"
	"fmt"
	_ "net/http"
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
	// build API client
	address := "http://localhost:9090"
	if v, ok := cfg["address"].(string); ok && v != "" {
		address = v
	}
	client, err := api.NewClient(api.Config{Address: address})
	if err != nil {
		common.Throw(rt, fmt.Errorf("failed to create prom client: %w", err))
	}
	promAPI := v1.NewAPI(client)

	pc := &PromClient{api: promAPI}
	obj := rt.ToValue(pc).ToObject(rt)
	return obj
}

type PromClient struct {
	api v1.API
}

func (p *PromClient) Query(query string) (*QueryResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, warnings, err := p.api.Query(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("prometheus query error: %w", err)
	}
	_ = warnings // ignore for now
	qr := &QueryResult{value: res}
	return qr, nil
}

func (p *PromClient) QueryRange(query string, start, end time.Time, step time.Duration) (model.Value, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	r := v1.Range{Start: start, End: end, Step: step}
	res, warnings, err := p.api.QueryRange(ctx, query, r)
	if err != nil {
		return nil, fmt.Errorf("prometheus range query error: %w", err)
	}
	_ = warnings
	return res, nil
}

func (p *PromClient) EvaluateThreshold(query string, threshold float64) (bool, error) {
	qr, err := p.Query(query)
	if err != nil {
		return false, err
	}
	v, err := qr.AsNumber()
	if err != nil {
		return false, nil
	}
	return v > threshold, nil
}

type QueryResult struct {
	value model.Value
}

func (qr *QueryResult) AsNumber() (float64, error) {
	switch qr.value.Type() {
	case model.ValScalar:
		s, ok := qr.value.(*model.Scalar)
		if !ok {
			return 0, fmt.Errorf("invalid scalar type")
		}
		return float64(s.Value), nil
	case model.ValVector:
		v, ok := qr.value.(model.Vector)
		if !ok {
			return 0, fmt.Errorf("invalid vector type")
		}
		if len(v) == 0 {
			return 0, fmt.Errorf("empty vector")
		}
		return float64(v[0].Value), nil
	default:
		return 0, fmt.Errorf("unsupported result type: %s", qr.value.Type().String())
	}
}

func (qr *QueryResult) AsSamples() ([]map[string]any, error) {
	if qr.value.Type() != model.ValVector {
		return nil, fmt.Errorf("AsSamples: not a vector result")
	}
	v, ok := qr.value.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("invalid vector type")
	}
	out := make([]map[string]any, 0, len(v))
	for _, s := range v {
		m := make(map[string]any)
		met := make(map[string]string)
		for k, val := range s.Metric {
			met[string(k)] = string(val)
		}
		m["metric"] = met
		m["value"] = float64(s.Value)
		out = append(out, m)
	}
	return out, nil
}

var _ = func() any { return &PromClient{} }
var _ = func() any { return &QueryResult{} }
