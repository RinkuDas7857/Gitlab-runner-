package steps

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/gitlab-runner/common"
)

func Test_Step2Step(t *testing.T) {
	variables := common.JobVariables{
		common.JobVariable{
			Key:   "FOO",
			Value: "foo",
		},
		common.JobVariable{
			Key:   "BAR",
			Value: "bar",
		},
		common.JobVariable{
			Key:   "BAZ",
			Value: "${FOO}-${BAR}",
		},
		common.JobVariable{
			Key:   "/logs/${BAZ}",
			Value: "the content of the file",
			File:  true,
		},
		common.JobVariable{
			Key:   "RUNNER_TEMP_PROJECT_DIR",
			Value: "/tmp/1234",
		},
	}

	step := common.Step{
		Name: "some-test-script",
		Script: common.StepScript{
			"first-command",
			"second-command",
		},
	}

	got := ExpandStep(step, variables.Expand())

	assert.Len(t, got, 2)

	for i, cmd := range []string{"first-command", "second-command"} {
		assert.Equal(t, "some-test-script", got[i].Name)
		assert.Equal(t, cmd, got[i].Inputs["script"].GetStringValue())
		assert.Equal(t, "foo", got[i].Env["FOO"])
		assert.Equal(t, "bar", got[i].Env["BAR"])
		assert.Equal(t, "foo-bar", got[i].Env["BAZ"])
		// ...not convinced this is right. should maybe be /tmp/1234/logs/foo-bar
		assert.Equal(t, "/tmp/1234/logs/${BAZ}", got[i].Env["/logs/${BAZ}"])
	}
}
