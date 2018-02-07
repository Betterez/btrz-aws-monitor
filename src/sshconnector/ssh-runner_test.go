package sshconnector

import (
	"bytes"
	"testing"
)

const (
	sshPort = 22
)

func TestConnection(t *testing.T) {
	t.SkipNow()
	_, err := CreateSSHSession("192.168.0.61", "tal", "123", 22, UsePassword)
	if err != nil {
		t.Fatal("err", err, "Connecting to host")
	}
}

func TestCommands(t *testing.T) {
	t.SkipNow()
	serverAddress := "192.168.100.100"
	serverKey := "../../secrets/sample-key.pem"
	session, err := CreateSSHSession(serverAddress, "ubuntu", serverKey, 22, UseKey)
	if err != nil {
		t.Fatal("err", err, "Connecting to host")
	}
	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Run("ls -shla")
	t.Log(stdoutBuf.String())
	t.Log("ssh completed")
	defer session.Close()
}

func TestAgentRegistration(t *testing.T) {
	t.SkipNow()
	agentAddress := "192.168.100.100"
	session, err := CreateSSHSession(agentAddress, "tal", "123", sshPort, UsePassword)
	if err != nil {
		t.Fatal(err)
	}
	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Run("echo yo, what >> mtx.txt")
	session.Run("ls")
	t.Log(stdoutBuf.String())
	session.Close()
}
