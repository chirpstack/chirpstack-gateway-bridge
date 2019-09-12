package commands

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/lora-gateway-bridge/internal/config"
	"github.com/brocaar/lora-gateway-bridge/internal/integration"
	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
)

type command struct {
	Command              string
	MaxExecutionDuration time.Duration
}

var (
	mux sync.RWMutex

	commands map[string]command
)

// Setup configures the gateway commands.
func Setup(conf config.Config) error {
	mux.Lock()
	defer mux.Unlock()

	commands = make(map[string]command)

	for k, v := range conf.Commands.Commands {
		commands[k] = command{
			Command:              v.Command,
			MaxExecutionDuration: v.MaxExecutionDuration,
		}

		log.WithFields(log.Fields{
			"command":                k,
			"command_exec":           v.Command,
			"max_execution_duration": v.MaxExecutionDuration,
		}).Info("commands: configuring command")
	}

	go executeLoop()

	return nil
}

func executeLoop() {
	for cmd := range integration.GetIntegration().GetGatewayCommandExecRequestChan() {
		go func(cmd gw.GatewayCommandExecRequest) {
			executeCommand(cmd)
		}(cmd)
	}
}

func executeCommand(cmd gw.GatewayCommandExecRequest) {
	var gatewayID lorawan.EUI64
	copy(gatewayID[:], cmd.GatewayId)

	stdout, stderr, err := execute(cmd.Command, cmd.Stdin, cmd.Environment)
	resp := gw.GatewayCommandExecResponse{
		GatewayId: cmd.GatewayId,
		ExecId:    cmd.ExecId,
		Stdout:    stdout,
		Stderr:    stderr,
	}
	if err != nil {
		resp.Error = err.Error()
	}

	var id uuid.UUID

	if err := integration.GetIntegration().PublishEvent(gatewayID, "exec", id, &resp); err != nil {
		log.WithError(err).Error("commands: publish command execution event error")
	}
}

func execute(command string, stdin []byte, environment map[string]string) ([]byte, []byte, error) {
	mux.RLock()
	defer mux.RUnlock()

	cmdConfig, ok := commands[command]
	if !ok {
		return nil, nil, errors.New("command does not exist")
	}

	cmdArgs, err := ParseCommandLine(cmdConfig.Command)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parse command error")
	}
	if len(cmdArgs) == 0 {
		return nil, nil, errors.New("no command is given")
	}

	log.WithFields(log.Fields{
		"command":                command,
		"exec":                   cmdArgs[0],
		"args":                   cmdArgs[1:],
		"max_execution_duration": cmdConfig.MaxExecutionDuration,
	}).Info("commands: executing command")

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	for k, v := range environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, errors.Wrap(err, "get stdin pipe error")
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, errors.Wrap(err, "get stdout pipe error")
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, errors.Wrap(err, "get stderr pipe error")
	}

	go func() {
		defer stdinPipe.Close()
		if _, err := stdinPipe.Write(stdin); err != nil {
			log.WithError(err).Error("commands: write to stdin error")
		}
	}()

	if err := cmd.Start(); err != nil {
		return nil, nil, errors.Wrap(err, "starting command error")
	}

	time.AfterFunc(cmdConfig.MaxExecutionDuration, func() {
		pgid, err := syscall.Getpgid(cmd.Process.Pid)
		if err == nil {
			if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
				panic(err)
			}
		}
	})

	stdoutB, _ := ioutil.ReadAll(stdoutPipe)
	stderrB, _ := ioutil.ReadAll(stderrPipe)

	if err := cmd.Wait(); err != nil {
		return nil, nil, errors.Wrap(err, "waiting for command to finish error")
	}

	return stdoutB, stderrB, nil
}

// source: https://stackoverflow.com/questions/34118732/parse-a-command-line-string-into-flags-and-arguments-in-golang
func ParseCommandLine(command string) ([]string, error) {
	var args []string
	state := "start"
	current := ""
	quote := "\""
	escapeNext := true
	for i := 0; i < len(command); i++ {
		c := command[i]

		if state == "quotes" {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = "start"
			}
			continue
		}

		if escapeNext {
			current += string(c)
			escapeNext = false
			continue
		}

		if c == '\\' {
			escapeNext = true
			continue
		}

		if c == '"' || c == '\'' {
			state = "quotes"
			quote = string(c)
			continue
		}

		if state == "arg" {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = "start"
			} else {
				current += string(c)
			}
			continue
		}

		if c != ' ' && c != '\t' {
			state = "arg"
			current += string(c)
		}
	}

	if state == "quotes" {
		return []string{}, errors.New(fmt.Sprintf("Unclosed quote in command line: %s", command))
	}

	if current != "" {
		args = append(args, current)
	}

	return args, nil
}
