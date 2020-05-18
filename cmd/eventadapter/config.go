package main

import (
	"fmt"
	"regexp"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

type Filter struct {
	Key      string `yaml:"key"`
	Value    string `yaml:"value"`
	Negative bool   `yaml:"negative"`
}

type FilterExp struct {
	Filter   `yaml:",inline"`
	ValueExp *regexp.Regexp `yaml:"-"`
}

func (t *FilterExp) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal(&t.Filter)
	if err != nil {
		return err
	}

	t.ValueExp, err = regexp.Compile(t.Value)
	if err != nil {
		return fmt.Errorf("get invalid value %s. ", t.Value)
	}
	return nil
}

func (t FilterExp) IsMatch(val string) bool {
	ok := t.ValueExp.MatchString(val)
	if t.Negative {
		return !ok
	}
	return ok
}

type AdapterConfig struct {
	Filters       []FilterExp    `yaml:"filters"`
	DestListeners []ListenerInfo `yaml:"destListeners"`
}

type ListenerInfo struct {
	EventListenerName string             `yaml:"eventListenerName"`
	EventListenerNs   string             `yaml:"eventListenerNamespace"`
	Bindings          []triggersv1.Param `yaml:"params"`
	ReqHeadFields     map[string]string  `yaml:"reqHeadFields"`
	ReqBodyTemplate   string             `yaml:"reqBodyTemplate"`
}

const ConfigKey = "adapters"

var CurrentConfig = make([]AdapterConfig, 0)

// 用于jsonPath 获取值
type DataEvent struct {
	Context cloudevents.EventContext `json:"context"`
	Data    interface{}              `json:"data"`
}

func GetListenerKey(name, namespace string) string {
	return fmt.Sprintf("%s/%s", name, namespace)
}
