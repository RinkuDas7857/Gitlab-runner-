package steps

import (
	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/step-runner/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// convert a common.Step to a proto.Step, to be run in a legacy `bash` step.
func ExpandStep(step common.Step, variables common.JobVariables) []*proto.Step {
	env := map[string]string{}
	for _, jv := range variables {
		env[jv.Key] = variables.Get(jv.Key)
	}

	var result []*proto.Step
	for _, line := range step.Script {
		step := proto.Step{
			Name: string(step.Name),
			Step: "https+git://gitlab.com/avonbertoldi/bash-step",
			Inputs: map[string]*structpb.Value{
				"script": structpb.NewStringValue(line),
			},
			Env: env,
		}
		result = append(result, &step)
	}

	return result
}

func ExpandSteps(steps []common.Step, variables common.JobVariables) []*proto.Step {
	result := []*proto.Step{}
	for _, step := range steps {
		result = append(result, ExpandStep(step, variables)...)
	}
	return result
}
