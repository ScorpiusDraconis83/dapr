/*
Copyright 2025 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://wwb.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package raise

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dapr/dapr/tests/integration/framework"
	"github.com/dapr/dapr/tests/integration/framework/process/workflow"
	"github.com/dapr/dapr/tests/integration/suite"
	"github.com/dapr/durabletask-go/api"
	"github.com/dapr/durabletask-go/task"
)

func init() {
	suite.Register(new(active))
}

type active struct {
	workflow *workflow.Workflow
}

func (a *active) Setup(t *testing.T) []framework.Option {
	a.workflow = workflow.New(t)

	return []framework.Option{
		framework.WithProcesses(a.workflow),
	}
}

func (a *active) Run(t *testing.T, ctx context.Context) {
	a.workflow.WaitUntilRunning(t, ctx)

	a.workflow.Registry().AddOrchestratorN("active-event", func(ctx *task.OrchestrationContext) (any, error) {
		as1 := ctx.WaitForSingleEvent("abc1", time.Hour)
		as2 := ctx.CallActivity("bar")
		require.NoError(t, as2.Await(nil))
		require.NoError(t, as1.Await(nil))
		return nil, nil
	})
	a.workflow.Registry().AddActivityN("bar", func(ctx task.ActivityContext) (any, error) {
		time.Sleep(time.Second)
		return nil, nil
	})

	client := a.workflow.BackendClient(t, ctx)

	id, err := client.ScheduleNewOrchestration(ctx, "active-event", api.WithInstanceID("xyz"))
	require.NoError(t, err)
	time.Sleep(time.Second * 2)
	require.NoError(t, client.RaiseEvent(ctx, id, "abc1"))
	_, err = client.WaitForOrchestrationCompletion(ctx, id)
	require.NoError(t, err)

	newID, err := client.RerunWorkflowFromEvent(ctx, id, 0)
	time.Sleep(time.Second * 2)
	require.NoError(t, client.RaiseEvent(ctx, newID, "abc1"))
	require.NoError(t, err)
	_, err = client.WaitForOrchestrationCompletion(ctx, newID)
	require.NoError(t, err)
}
