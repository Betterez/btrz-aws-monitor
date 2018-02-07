package sshconnector

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"os"
)

const (
	// UseKey - use key file name as parameter
	UseKey = 1
	// UsePassword - use password as ssh parameter
	UsePassword = 2
)

// PublicKeyFile - return public key info
func PublicKeyFile(file string) (ssh.AuthMethod, error) {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(key), nil
}

//CreateSSHSession - return ssh session
func CreateSSHSession(serverAddress, username, authenticationParam string, serverPort, mode int16) (*ssh.Session, error) {
	sshConfig := &ssh.ClientConfig{
		User: username,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	var auth []ssh.AuthMethod
	if mode == UseKey {
		if _, err := os.Stat(authenticationParam); os.IsNotExist(err) {
			return nil, err
		}
		publicKey, err := PublicKeyFile(authenticationParam)
		if err != nil {
			return nil, err
		}
		auth = []ssh.AuthMethod{publicKey}
	} else if mode == UsePassword {
		auth = []ssh.AuthMethod{ssh.Password(authenticationParam)}
	}
	sshConfig.Auth = auth
	connectionString := fmt.Sprintf("%s:%d", serverAddress, serverPort)
	connection, err := ssh.Dial("tcp", connectionString, sshConfig)
	if err != nil {
		return nil, err
	}
	sshSession, err := connection.NewSession()
	if err != nil {
		return nil, err
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	err = sshSession.RequestPty("xterm", 80, 40, modes)
	if err != nil {
		sshSession.Close()
		return nil, err
	}
	return sshSession, nil
}
