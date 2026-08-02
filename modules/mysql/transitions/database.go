package transitions

import (
	"net/http"

	"github.com/kuetix/engine/engine/domain"
	"github.com/kuetix/engine/engine/domain/interfaces"
	"github.com/kuetix/engine/engine/workflow"
)

type databaseTransitions struct {
	workflow.BaseServiceTransition
}

func NewDatabaseTransitions() interfaces.ServiceTransitions {
	return &databaseTransitions{}
}

// Ping opens (or reuses) a connection for dsn and reports whether it is reachable.
func (t *databaseTransitions) Ping(dsn string) (r domain.FlowStepResult) {
	_, resolvedDSN, err := resolveDB(dsn)
	if err != nil {
		r.Error = err
		r.StatusCode = http.StatusServiceUnavailable
		return
	}

	r.Success = true
	r.StatusCode = http.StatusOK
	r.Response = map[string]interface{}{
		"target":    redactDSN(resolvedDSN),
		"connected": true,
	}
	return
}
