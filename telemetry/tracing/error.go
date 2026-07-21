package tracing

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type strStackTracer interface {
	StackTrace() string
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

type stackTrace struct {
	stackTracer
}

func (st stackTrace) String() string {
	var buf strings.Builder
	for _, f := range st.StackTrace() {
		buf.WriteString(fmt.Sprintf("%+v\n", f))
		buf.WriteByte('\n')
	}
	return buf.String()
}
