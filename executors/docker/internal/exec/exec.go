package exec

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/executors/docker/internal/wait"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/docker"
	steps_api "gitlab.com/gitlab-org/step-runner/pkg/service"
	"gitlab.com/gitlab-org/step-runner/proto"
)

// conn is an interface wrapper used to generate mocks that are next used for tests
// nolint:deadcode
//
//go:generate mockery --name=conn --inpackage
type conn interface {
	net.Conn
}

// reader is an interface wrapper used to generate mocks that are next used for tests
// nolint:deadcode
//
//go:generate mockery --name=reader --inpackage
type reader interface {
	io.Reader
}

type IOStreams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

//go:generate mockery --name=Docker --inpackage
type Docker interface {
	Exec(ctx context.Context, containerID string, streams IOStreams, gracefulExitFunc wait.GracefulExitFunc) error
}

// NewDocker returns a client for starting a new container and running a
// command inside of it.
//
// The context passed is used to wait for any created container to stop. This
// is likely an executor's context. This means that waits to stop are only ever
// canceled should the job be aborted (either manually, or by exceeding the
// build time).
func NewDocker(ctx context.Context, c docker.Client, waiter wait.KillWaiter, logger logrus.FieldLogger) Docker {
	return &defaultDocker{
		ctx:    ctx,
		c:      c,
		waiter: waiter,
		logger: logger,
	}
}

type defaultDocker struct {
	ctx    context.Context
	c      docker.Client
	waiter wait.KillWaiter
	logger logrus.FieldLogger
}

//nolint:funlen
func (d *defaultDocker) Exec(ctx context.Context, containerID string, streams IOStreams, gracefulExitFunc wait.GracefulExitFunc) error {
	d.logger.Debugln("Attaching to container", containerID, "...")

	hijacked, err := d.c.ContainerAttach(ctx, containerID, attachOptions())
	if err != nil {
		return err
	}
	defer hijacked.Close()

	d.logger.Debugln("Starting container", containerID, "...")
	err = d.c.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	// stdout/stdin error channels, buffered intentionally so that if select{}
	// below exits, the go routines don't block forever upon container exit.
	stdoutErrCh := make(chan error, 1)
	stdinErrCh := make(chan error, 1)

	// Copy any output to the build trace
	go func() {
		_, errCopy := stdcopy.StdCopy(streams.Stdout, streams.Stderr, hijacked.Reader)

		// this goroutine can continue even whilst StopKillWait is in flight,
		// allowing a graceful stop. If reading stdout returns, we must close
		// attached connection, otherwise kills can be interfered with and
		// block indefinitely.
		hijacked.Close()

		stdoutErrCh <- errCopy
	}()

	// Write the input to the container and close its STDIN to get it to finish
	go func() {
		_, errCopy := io.Copy(hijacked.Conn, streams.Stdin)
		_ = hijacked.CloseWrite()
		if errCopy != nil {
			stdinErrCh <- errCopy
		}
	}()

	// Wait until either:
	// - the job is aborted/cancelled/deadline exceeded
	// - stdin has an error
	// - stdout returns an error or nil, indicating the stream has ended and
	//   the container has exited
	select {
	case <-ctx.Done():
		err = errors.New("aborted")
	case err = <-stdinErrCh:
	case err = <-stdoutErrCh:
	}

	if err != nil {
		d.logger.Debugln("Container", containerID, "finished with", err)
	}

	// Try to gracefully stop, then kill and wait for the exit.
	// Containers are stopped so that they can be reused by the job.
	//
	// It's very likely that at this point, the context passed to Exec has
	// been cancelled, so is unable to be used. Instead, we use the context
	// passed to NewDocker.
	return d.waiter.StopKillWait(d.ctx, containerID, nil, gracefulExitFunc)
}

func attachOptions() types.ContainerAttachOptions {
	return types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	}
}

type stepsDocker struct {
	ctx    context.Context
	client docker.Client
	waiter wait.KillWaiter
	logger logrus.FieldLogger
	jobID  string
	steps  []*proto.Step
}

func NewStepsDocker(ctx context.Context, client docker.Client, waiter wait.KillWaiter, logger logrus.FieldLogger,
	jobID int64, steps []*proto.Step,
) Docker {
	return &stepsDocker{
		ctx:    ctx,
		client: client,
		waiter: waiter,
		logger: logger,
		steps:  steps,
		jobID:  strconv.FormatInt(jobID, 10),
	}
}

func (sd *stepsDocker) Exec(ctx context.Context, containerID string, streams IOStreams, gracefulExitFunc wait.GracefulExitFunc) error {
	sd.logger.Debugln("Executing steps on container", containerID, "...")

	// This is only necessary if we want to capture logs generated in the build container by the step-runner, and then
	// we need to adjust the ContainerAttachOptions.
	// hijacked, err := sd.client.ContainerAttach(ctx, containerID, types.ContainerAttachOptions{})
	// if err != nil {
	// 	return err
	// }
	// defer hijacked.Close()

	sd.logger.Debugln("Starting container", containerID, "...")
	err := sd.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	sd.sendSteps(ctx, streams)

	return sd.waiter.StopKillWait(sd.ctx, containerID, nil, gracefulExitFunc)
}

const serverAddr = "localhost:8765"

func (sd *stepsDocker) sendSteps(ctx context.Context, streams IOStreams) error {
	client, err := steps_api.NewClient(serverAddr)
	if err != nil {
		return err
	}
	defer client.Close()

	err = client.RunStep(ctx, sd.jobID, sd.steps)
	if err != nil {
		return err
	}
	defer client.Cancel(ctx, sd.jobID)

	stream, err := client.Follow(ctx, sd.jobID)
	if err != nil {
		return err
	}

	var result []*proto.StepResult
	defer func() {
		if result == nil {
			return
		}
		// save results file here, and configure it to be an artifact.
	}()

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		result = append(result, res.GetResult())
		streams.Stdout.Write(res.GetOutput())
	}
}
