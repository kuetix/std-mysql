package transitions

import (
	"net/http"

	"github.com/kuetix/engine/engine/domain"
	"github.com/kuetix/engine/engine/domain/interfaces"
	"github.com/kuetix/engine/engine/workflow"
)

type execTransitions struct {
	workflow.BaseServiceTransition
}

func NewExecTransitions() interfaces.ServiceTransitions {
	return &execTransitions{}
}

// Exec runs a parameterized INSERT/UPDATE/DELETE (or DDL) statement.
func (t *execTransitions) Exec(dsn, sqlText string, args []interface{}) (r domain.FlowStepResult) {
	db, _, err := resolveDB(dsn)
	if err != nil {
		r.Error = err
		r.StatusCode = http.StatusServiceUnavailable
		return
	}

	result, err := db.Exec(sqlText, args...)
	if err != nil {
		r.Error = err
		r.StatusCode = http.StatusBadRequest
		return
	}

	lastInsertID, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()

	r.Success = true
	r.StatusCode = http.StatusOK
	r.Response = map[string]interface{}{
		"lastInsertId": lastInsertID,
		"rowsAffected": rowsAffected,
	}
	return
}
