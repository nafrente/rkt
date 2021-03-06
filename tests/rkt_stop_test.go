// Copyright 2016 The rkt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/coreos/rkt/tests/testutils"
)

func TestRktStop(t *testing.T) {
	image := patchTestACI("rkt-stop-test.aci", "--name=rkt-stop-test", "--exec=/inspect --read-stdin --silent-sigterm")
	defer os.Remove(image)

	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	// Define tests
	tests := []struct {
		cmd        string
		expectKill bool
	}{
		// Test regular stop
		{
			"stop",
			false,
		},
		// Test forced stop
		{
			"stop --force",
			true,
		},
	}

	// Run tests
	for i, tt := range tests {
		// Prepare image
		cmd := fmt.Sprintf("%s --insecure-options=image prepare %s", ctx.Cmd(), image)
		podUUID := runRktAndGetUUID(t, cmd)

		// Run image
		cmd = fmt.Sprintf("%s --insecure-options=image run-prepared --interactive %s", ctx.Cmd(), podUUID)
		child := spawnOrFail(t, cmd)

		// Wait for prompt to make sure the pod is started
		if err := expectTimeoutWithOutput(child, "Enter text:", time.Minute); err != nil {
			t.Fatalf("Can't start pod")
		}

		runCmd := fmt.Sprintf("%s %s %s", ctx.Cmd(), tt.cmd, podUUID)
		t.Logf("Running test #%d, %s", i, runCmd)
		spawnOrFail(t, runCmd)

		// Make sure the pod is stopped
		var podInfo *podInfo
		exitedSuccessfully := false
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			podInfo = getPodInfo(t, ctx, podUUID)
			if podInfo.state == "exited" {
				exitedSuccessfully = true
				break
			}
		}
		if !exitedSuccessfully {
			t.Fatalf("Expected pod %q to be exited, but it is %q", podUUID, podInfo.state)
		}

		if tt.expectKill {
			child.Wait()
		} else {
			waitOrFail(t, child, 0)
		}
	}
}
