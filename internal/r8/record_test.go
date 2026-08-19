package r8

import (
	"testing"

	"github.com/example/route-analysis-service/internal/domain/analytics"
	"github.com/example/route-analysis-service/internal/domain/model"
)

func TestDatasetStateMachineGatesValidation(t *testing.T) {
	if !model.DatasetTransition(model.DatasetImporting, model.DatasetReady) {
		t.Fatal("importing dataset should be able to move to ready")
	}
	if !model.DatasetTransition(model.DatasetDraft, model.DatasetImporting) {
		t.Fatal("draft dataset should be able to move to importing")
	}
	invalid := model.Dataset{Name: "x", Version: 1, Status: model.DatasetStatus("bogus")}
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected invalid dataset status to be rejected")
	}
	if err := analytics.ValidateDatasetState(model.DatasetDraft); err == nil {
		t.Fatal("draft dataset should not pass validation")
	}
	if !analytics.DatasetValidationEligible(model.DatasetReady) {
		t.Fatal("ready dataset should be eligible for validation")
	}
	if analytics.DatasetValidationEligible(model.DatasetArchived) {
		t.Fatal("archived dataset should not be eligible for validation")
	}
}
