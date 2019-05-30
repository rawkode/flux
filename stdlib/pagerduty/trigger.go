package pagerduty

import (
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/influxdata/flux"
	"github.com/influxdata/flux/semantic"
)

const (
	TriggerPagerDutyKind = "triggerPagerDuty"
	DefaultTimeout       = 1 * time.Second
)

type Severity string

const (
	CRITICAL Severity = "Critical"
	ERROR    Severity = "Error"
	WARNING  Severity = "Warning"
	INFO     Severity = "Info"
)

type TriggerPagerDutyOpSpec struct {
	Token      string   `json:"token"`
	RoutingKey string   `json:"routingKey"`
	Summary    string   `json:"summary"`
	Source     string   `json:"source"`
	Severity   Severity `json:"severity"`
	dedupKey   string   `json:"dedupKey"`
	component  string   `json:"component"`
	group      string   `json:"group"`
	class      string   `json:"class"`
	links      []string `json:"links"`
}

// DefaultTriggerPagerDutyUserAgent is the default user agent used by TriggerPagerDuty
var DefaultTriggerPagerDutyUserAgent = "fluxd/dev"

func newToHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		},
	}
}

var triggerPagerDutyKeepAliveClient = newToHTTPClient()

func init() {
	triggerPagerDutySignature := flux.FunctionSignature(
		map[string]semantic.PolyType{
			"token":      semantic.String,
			"routingKey": semantic.String,
			"summary":    semantic.String,
			"source":     semantic.String,
			"severity":   semantic.String,
			"dedupKey":   semantic.String,
			"component":  semantic.String,
			"group":      semantic.String,
			"class":      semantic.String,
			"links":      semantic.NewArrayPolyType(semantic.String),
		},
		[]string{"token", "routingKey", "summary", "source", "severity"},
	)

	flux.RegisterPackageValue("pagerduty", "trigger", flux.FunctionValueWithSideEffect(TriggerPagerDutyKind, createTriggerPagerDutyOpSpec, triggerPagerDutySignature))
	flux.RegisterOpSpec(TriggerPagerDutyKind, func() flux.OperationSpec { return &TriggerPagerDutyOpSpec{} })
	// plan.RegisterProcedureSpecWithSideEffect(ToHTTPKind, newToHTTPProcedure, ToHTTPKind)
	// execute.RegisterTransformation(ToHTTPKind, createToHTTPTransformation)
}

// ReadArgs loads a flux.Arguments into TriggerPagerDutyOpSpec
func (o *TriggerPagerDutyOpSpec) ReadArgs(args flux.Arguments) error {
	var err error

	o.Token, _, err = args.GetString("token")
	if err != nil {
		return err
	}

	o.RoutingKey, _, err = args.GetString("routingKey")
	if err != nil {
		return err
	}

	o.Severity, _, err = args.GetString("severity")
	if err != nil {
		return err
	}

	return err
}

func createTriggerPagerDutyOpSpec(args flux.Arguments, a *flux.Administration) (flux.OperationSpec, error) {
	if err := a.AddParentFromArgs(args); err != nil {
		return nil, err
	}
	s := new(TriggerPagerDutyOpSpec)
	if err := s.ReadArgs(args); err != nil {
		return nil, err
	}
	return s, nil
}
