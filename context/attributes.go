package context

import (
	"fmt"
	"strings"
)

type Attributes map[string]string

func (a *Attributes) String() string {
	if a == nil {
		return ""
	}

	var sb strings.Builder
	for k, v := range *a {
		sb.WriteString(fmt.Sprintf(",\"%s\":\"%s\"", k, v))
	}

	return sb.String()
}
