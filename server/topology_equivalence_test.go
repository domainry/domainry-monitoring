package server_test

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"

	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	"github.com/domainry/domainry-monitoring-sdk/remote"
	monitoringmodule "github.com/domainry/domainry-monitoring/module"
	monitoringserver "github.com/domainry/domainry-monitoring/server"
)

func TestModuleAndSaaSTopologiesProduceEquivalentSnapshots(t *testing.T) {
	application := monitoringsdk.ApplicationRef{RuntimeID: "runtime-1"}
	moduleBinding, err := monitoringmodule.NewFactory(monitoringmodule.Options{}).OpenModule(t.Context(), application, remoteHost{})
	if err != nil {
		t.Fatal(err)
	}
	service := httptest.NewServer(monitoringserver.New(monitoringserver.Options{BearerToken: "secret"}).Routes())
	defer service.Close()
	remoteBinding, err := remote.NewFactory(remote.Config{Endpoint: service.URL, Token: "secret", Client: service.Client()}).OpenSaaS(t.Context(), application, remoteHost{})
	if err != nil {
		t.Fatal(err)
	}
	if moduleHealth, remoteHealth := normalize(moduleBinding.Health(t.Context())), normalize(remoteBinding.Health(t.Context())); !reflect.DeepEqual(moduleHealth, remoteHealth) {
		t.Fatalf("health mismatch\nmodule=%#v\nsaas=%#v", moduleHealth, remoteHealth)
	}
	moduleMetrics, remoteMetrics := normalize(moduleBinding.Metrics(t.Context())), normalize(remoteBinding.Metrics(t.Context()))
	if !reflect.DeepEqual(moduleMetrics, remoteMetrics) {
		t.Fatalf("metrics mismatch\nmodule=%#v\nsaas=%#v", moduleMetrics, remoteMetrics)
	}
}

func normalize(value map[string]any) map[string]any {
	payload, _ := json.Marshal(value)
	result := map[string]any{}
	_ = json.Unmarshal(payload, &result)
	return result
}
