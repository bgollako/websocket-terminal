package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"time"
)

type SshClient struct {
	username string
	password string
	address  string
	port     int
	client   *ssh.Client
	session  *ssh.Session
}

func NewSshClient(username, password, address string, port int) *SshClient {
	return &SshClient{
		username: username,
		password: password,
		address:  address,
		port:     port,
	}
}

func (s *SshClient) Connect(opWriter io.Writer, errWriter io.Writer) (io.WriteCloser, error) {
	var err error
	config := &ssh.ClientConfig{
		Config: ssh.Config{},
		User:   s.username,
		Auth: []ssh.AuthMethod{
			ssh.PasswordCallback(func() (secret string, err error) {
				return s.password, nil
			}),
		},
		HostKeyCallback:   ssh.InsecureIgnoreHostKey(),
		BannerCallback:    ssh.BannerDisplayStderr(),
		ClientVersion:     "",
		HostKeyAlgorithms: nil,
		Timeout:           time.Second * 10,
	}

	s.client, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", s.address, s.port), config)
	if err != nil {
		return nil, err
	}

	s.session, err = s.client.NewSession()
	if err != nil {
		return nil, err
	}

	modes := ssh.TerminalModes{
		//ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// Change to vt100 later on
	err = s.session.RequestPty("xterm", 80, 40, modes)
	if err != nil {
		return nil, err
	}
	s.session.Stdout = opWriter
	s.session.Stderr = errWriter
	ipWriter, err := s.session.StdinPipe()
	if err != nil {
		return nil, err
	}
	err = s.session.Shell()
	if err != nil {
		return nil, err
	}
	return ipWriter, nil
}

func (s *SshClient) Close() {
	if err := s.session.Close(); err != nil {
		log.Println(err)
	}
	if err := s.client.Close(); err != nil {
		log.Println(err)
	}
}
