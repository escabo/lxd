package drivers

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/canonical/lxd/shared/logger"
)

// ErrExecDisconnected is returned when the guest disconnects the exec session.
var ErrExecDisconnected = errors.New("Disconnected")

// Cmd represents a running command for an Qemu VM.
type qemuCmd struct {
	attachedChildPid int
	cmd              lxd.Operation
	dataDone         chan bool
	controlSendCh    chan api.InstanceExecControl
	controlResCh     chan error
	cleanupFunc      func()
}

// PID returns the attached child's process ID.
func (c *qemuCmd) PID() int {
	return c.attachedChildPid
}

// Signal sends a signal to the command.
func (c *qemuCmd) Signal(sig unix.Signal) error {
	command := api.InstanceExecControl{
		Command: "signal",
		Signal:  int(sig),
	}

	// Check handler hasn't finished.
	select {
	case <-c.dataDone:
		return errors.New("no such process") // Aligns with error retured from unix.Kill in lxc's Signal().
	default:
	}

	c.controlSendCh <- command
	err := <-c.controlResCh
	if err != nil {
		return err
	}

	logger.Debugf(`Forwarded signal "%d" to lxd-agent`, sig)
	return nil
}

// Wait for the command to end and returns its exit code and any error.
func (c *qemuCmd) Wait() (int, error) {
	err := c.cmd.Wait()

	exitStatus := -1
	opAPI := c.cmd.Get()
	if opAPI.Metadata != nil {
		exitStatusRaw, ok := opAPI.Metadata["return"].(float64)
		if ok {
			exitStatus = int(exitStatusRaw)

			// Convert special exit statuses into errors.
			switch exitStatus {
			case 127:
				err = ErrExecCommandNotFound
			case 126:
				err = ErrExecCommandNotExecutable
			}
		}
	}

	if err != nil {
		// Error of type EOF indicates the session ended unexpectedly,
		// so we inform the client of the disconnection with a more
		// descriptive message.
		// The error can be different depending on why the VM disconnected
		// so we handle these cases similarly.
		if errors.Is(err, io.EOF) || strings.Contains(err.Error(), io.ErrUnexpectedEOF.Error()) || strings.Contains(err.Error(), syscall.ECONNRESET.Error()) {
			return exitStatus, ErrExecDisconnected
		}

		return exitStatus, err
	}

	<-c.dataDone

	if c.cleanupFunc != nil {
		defer c.cleanupFunc()
	}

	return exitStatus, nil
}

// WindowResize resizes the running command's window.
func (c *qemuCmd) WindowResize(fd, winchWidth, winchHeight int) error {
	command := api.InstanceExecControl{
		Command: "window-resize",
		Args: map[string]string{
			"width":  strconv.Itoa(winchWidth),
			"height": strconv.Itoa(winchHeight),
		},
	}

	// Check handler hasn't finished.
	select {
	case <-c.dataDone:
		return errors.New("no such process") // Aligns with error retured from unix.Kill in lxc's Signal().
	default:
	}

	c.controlSendCh <- command
	err := <-c.controlResCh
	if err != nil {
		return err
	}

	logger.Debugf(`Forwarded window resize "%dx%d" to lxd-agent`, winchWidth, winchHeight)
	return nil
}
