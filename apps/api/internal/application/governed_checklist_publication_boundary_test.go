package application

import (
	"reflect"
	"testing"
)

func TestNormalApplicationServiceDoesNotExposeDirectChecklistPublication(t *testing.T) {
	// Production break: adding an exported direct-publication method would let
	// an Admin principal call it and must fail this normal-profile boundary.
	serviceType := reflect.TypeOf((*Service)(nil))
	if _, exists := serviceType.MethodByName("CreateChecklistTemplateVersion"); exists {
		t.Fatal("normal application service exposes direct checklist publication")
	}
}
