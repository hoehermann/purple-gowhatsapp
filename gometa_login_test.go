//go:build gometa

package main

import (
	"strings"
	"testing"

	"go.mau.fi/util/l10n"
	"maunium.net/go/mautrix/bridgev2"
)

// gmMFAStep is messagix's 2FA method picker: one select field whose options carry the method as
// their value and a label that may differ from it.
func gmMFAStep(values ...string) *bridgev2.LoginStep {
	options := make([]bridgev2.LoginInputOption, len(values))
	for i, v := range values {
		options[i] = bridgev2.LoginInputOption{Value: v, Label: l10n.Unlocalized("Use " + strings.ToLower(v))}
	}
	return &bridgev2.LoginStep{
		Type: bridgev2.LoginStepTypeUserInput,
		UserInputParams: &bridgev2.LoginUserInputParams{Fields: []bridgev2.LoginInputDataField{{
			ID: "mfatype", Type: bridgev2.LoginInputFieldTypeSelect, Options: options,
		}}},
	}
}

func TestGometaLoginPicksTheCodeMethodByValue(t *testing.T) {
	h := &gometaHandler{}
	credsSent := true
	out, err := h.fillLoginStep(gmMFAStep("Notification on another device", "Text message", "Authentication app"), "", "", &credsSent)
	if err != nil {
		t.Fatalf("fillLoginStep: %v", err)
	}
	if got := out["mfatype"]; got != "Authentication app" {
		t.Fatalf("mfatype = %q, want the authentication app's value", got)
	}
}

func TestGometaLoginRefusesWithoutACodeMethod(t *testing.T) {
	h := &gometaHandler{}
	credsSent := true
	_, err := h.fillLoginStep(gmMFAStep("Notification on another device"), "", "", &credsSent)
	if err == nil {
		t.Fatal("fillLoginStep accepted a picker with no code method")
	}
	if !strings.Contains(err.Error(), "([Notification on another device])") {
		t.Fatalf("error %q does not name the methods offered", err)
	}
}
