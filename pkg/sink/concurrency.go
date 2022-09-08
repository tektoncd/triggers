package sink

import (
	"fmt"
	"strings"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
)

const concurrencyLabelKey = "concurrency"

func GetConcurrencyKey(c *triggersv1.Concurrency, params []triggersv1.Param) string {
	if c == nil {
		return ""
	}
	out := c.Key
	for _, param := range params {
		// Assume the param is valid
		paramVariable := fmt.Sprintf("$(params.%s)", param.Name)
		out = strings.ReplaceAll(out, paramVariable, param.Value)
	}
	return out
}
