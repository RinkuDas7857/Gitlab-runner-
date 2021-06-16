package buildtest

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/trace"
)

func RunBuildWithMasking(t *testing.T, config *common.RunnerConfig, setup BuildSetupFn) {
	const (
		maskedKey      = "MASKED_KEY"
		clearTextKey   = "CLEARTEXT_KEY"
		maskedKeyOther = "MASKED_KEY_OTHER"
	)

	resp, err := common.GetRemoteSuccessfulBuildPrintVars(config.Shell, maskedKey, clearTextKey, maskedKeyOther)
	require.NoError(t, err)

	build := &common.Build{
		JobResponse: resp,
		Runner:      config,
	}

	build.Variables = append(
		build.Variables,
		common.JobVariable{Key: maskedKey, Value: "MASKED_VALUE", Masked: true},
		common.JobVariable{Key: clearTextKey, Value: "CLEARTEXT_VALUE", Masked: false},
		common.JobVariable{Key: maskedKeyOther, Value: "MASKED_VALUE_OTHER", Masked: true},
	)

	if setup != nil {
		setup(build)
	}

	buf, err := trace.New()
	require.NoError(t, err)
	defer buf.Close()

	err = build.Run(&common.Config{}, &common.Trace{Writer: buf})
	assert.NoError(t, err)

	buf.Finish()

	contents, err := buf.Bytes(0, math.MaxInt64)
	assert.NoError(t, err)

	assert.NotContains(t, string(contents), "MASKED_KEY=MASKED_VALUE")
	assert.Contains(t, string(contents), "MASKED_KEY=[MASKED]")

	assert.NotContains(t, string(contents), "MASKED_KEY_OTHER=MASKED_VALUE_OTHER")
	assert.NotContains(t, string(contents), "MASKED_KEY_OTHER=[MASKED]_OTHER")
	assert.Contains(t, string(contents), "MASKED_KEY_OTHER=[MASKED]")

	assert.NotContains(t, string(contents), "CLEARTEXT_KEY=[MASKED]")
	assert.Contains(t, string(contents), "CLEARTEXT_KEY=CLEARTEXT_VALUE")
}
