// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package nas

import "testing"

func TestCatalystFailureMessage(t *testing.T) {
	t.Parallel()

	body := `{"status":"failed","message":"failed","responseCode":400,"data":"No NAS orders found for engagement 15770. Volume creation is not allowed."}`
	env, err := parseCatalystEnvelope([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	got := catalystFailureMessage(env)
	want := "No NAS orders found for engagement 15770. Volume creation is not allowed."
	if got != want {
		t.Fatalf("catalystFailureMessage() = %q, want %q", got, want)
	}
}
