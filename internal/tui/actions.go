package tui

import "os/exec"

type tuiCommandFactory interface {
	Start(tasksDir, exerciseName string) *exec.Cmd
	Check(tasksDir, exerciseName string) *exec.Cmd
	Env(tasksDir, exerciseName string) *exec.Cmd
	Kubeconfig(tasksDir, exerciseName string) *exec.Cmd
	SSH(tasksDir, nodeRef, exerciseName string) *exec.Cmd
	Reset(tasksDir, exerciseName string) *exec.Cmd
	Workstation(exerciseName string) *exec.Cmd
}

type defaultTUICommandFactory struct{}

func (defaultTUICommandFactory) Start(tasksDir, exerciseName string) *exec.Cmd {
	return newGymctlStartCmd(tasksDir, exerciseName)
}

func (defaultTUICommandFactory) Check(tasksDir, exerciseName string) *exec.Cmd {
	return newGymctlCheckCmd(tasksDir, exerciseName)
}

func (defaultTUICommandFactory) Env(tasksDir, exerciseName string) *exec.Cmd {
	return newGymctlEnvCmd(tasksDir, exerciseName)
}

func (defaultTUICommandFactory) Kubeconfig(tasksDir, exerciseName string) *exec.Cmd {
	return newGymctlKubeconfigCmd(tasksDir, exerciseName)
}

func (defaultTUICommandFactory) SSH(tasksDir, nodeRef, exerciseName string) *exec.Cmd {
	return newGymctlSSHCmd(tasksDir, nodeRef, exerciseName)
}

func (defaultTUICommandFactory) Reset(tasksDir, exerciseName string) *exec.Cmd {
	return newGymctlResetCmd(tasksDir, exerciseName)
}

func (defaultTUICommandFactory) Workstation(exerciseName string) *exec.Cmd {
	return newWorkstationExecCmd(exerciseName)
}

func (m Model) actionCommands() tuiCommandFactory {
	if m.commands != nil {
		return m.commands
	}
	return defaultTUICommandFactory{}
}
