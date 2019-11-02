package metadata

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/commands"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
)

var (
	mux sync.RWMutex

	static map[string]string
	cmnds  map[string]string
	cached map[string]string

	interval     time.Duration
	maxExecution time.Duration
)

// Setup configures the metadata package.
func Setup(conf config.Config) error {
	mux.Lock()
	defer mux.Unlock()

	static = conf.MetaData.Static
	cmnds = conf.MetaData.Dynamic.Commands

	interval = conf.MetaData.Dynamic.ExecutionInterval
	maxExecution = conf.MetaData.Dynamic.MaxExecutionDuration

	go func() {
		for {
			runCommands()
			time.Sleep(interval)
		}
	}()

	return nil
}

// Get returns the (cached) metadata.
func Get() map[string]string {
	mux.RLock()
	defer mux.RUnlock()

	return cached
}

func runCommands() {
	newKV := make(map[string]string)
	for k, v := range static {
		newKV[k] = v
	}

	for k, cmd := range cmnds {
		out, err := runCommand(cmd)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"key": k,
				"cmd": cmd,
			}).Error("metadata: execute command error")
			continue
		}

		newKV[k] = out
	}

	mux.Lock()
	defer mux.Unlock()
	cached = newKV
}

func runCommand(cmdStr string) (string, error) {
	cmdArgs, err := commands.ParseCommandLine(cmdStr)
	if err != nil {
		return "", errors.Wrap(err, "parse command error")
	}
	if len(cmdArgs) == 0 {
		return "", errors.New("no command is given")
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(maxExecution))
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "execution error")
	}

	if !utf8.Valid(out) {
		return "", errors.New("command did not return valid utf8 string")
	}

	return strings.TrimRight(string(out), "\n\r"), nil
}
