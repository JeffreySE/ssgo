package utils

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
	"time"
)

type SSHResult struct {
	Host   string
	Status string
	Result string
}

// coped from https://github.com/shanghai-edu/multissh (thank you very much)
func connect(user, password, host, key string, port int) (*ssh.Session, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		config       ssh.Config
		session      *ssh.Session
		err          error
	)
	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	if key == "" {
		auth = append(auth, ssh.Password(password))
	} else {
		pemBytes, err := ioutil.ReadFile(key)
		if err != nil {
			return nil, err
		}

		var signer ssh.Signer
		if password == "" {
			signer, err = ssh.ParsePrivateKey(pemBytes)
		} else {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(password))
		}
		if err != nil {
			return nil, err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}

	clientConfig = &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: 5 * time.Second,
		Config:  config,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// connect to ssh
	addr = fmt.Sprintf("%s:%d", host, port)

	if client, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	// create session
	if session, err = client.NewSession(); err != nil {
		return nil, err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
		return nil, err
	}

	return session, nil
}

func SSHRunShellScript(user, password, host, key, scriptFilePath, scriptArgs string, port int, chr chan interface{}) {
	var sshResult SSHResult
	var cmds []string
	sshResult.Host = host
	session, err := connect(user, password, host, key, port)
	if err != nil {
		sshResult.Status = "failed"
		sshResult.Result = fmt.Sprintf("ERROR: while connecting host %s, an error occured,error message: %s", sshResult.Host, err)
		chr <- sshResult
		return
	}
	defer session.Close()
	var outBuffer, errBuffer bytes.Buffer
	session.Stdout = &outBuffer
	session.Stderr = &errBuffer

	resSftpResult := SFTPSimpleUpload(user, password, host, key, port, scriptFilePath, "")
	if resSftpResult.Status == "false" {
		sshResult.Status = "failed"
		sshResult.Result = fmt.Sprintf("ERROR: copy local Shell script %s to host %s failed, error message: %s", scriptFilePath, sshResult.Host, err.Error())
		chr <- sshResult
		return
	}

	scriptFileRemotePath := resSftpResult.DestinationPath + "/" + filepath.Base(scriptFilePath)
	executeScriptCmd := fmt.Sprintf("%s %s %s", "/bin/sh", scriptFileRemotePath, scriptArgs)
	//removeScriptBeforeExitCmd := fmt.Sprintf("ls %s", scriptFileRemotePath)
	removeScriptBeforeExitCmd := fmt.Sprintf("rm -rf %s", scriptFileRemotePath)
	cmds = append(cmds, executeScriptCmd, removeScriptBeforeExitCmd, "exit")
	cmd := strings.Join(cmds, " && ")
	err = session.Run(cmd)
	if err != nil {
		sshResult.Status = "failed"
		res := outBuffer.String()
		res = strings.TrimSpace(res)
		sshResult.Result = fmt.Sprintf("%s\nERROR: while running script (%s) on host %s, an error occured %s", res, scriptFilePath, sshResult.Host, err.Error())
		chr <- sshResult
		return
	}
	if errBuffer.String() != "" {
		sshResult.Status = "failed"
		sshResult.Result = errBuffer.String()
		chr <- sshResult
	} else {
		sshResult.Status = "success"
		sshResult.Result = outBuffer.String()
		chr <- sshResult
	}
	chr <- sshResult
	return
}
func DoSSHRunFast(user, password, host, key string, cmdList []string, port int, chr chan interface{}) {
	var sshResult SSHResult
	sshResult.Host = host
	session, err := connect(user, password, host, key, port)
	if err != nil {
		sshResult.Status = "failed"
		sshResult.Result = fmt.Sprintf("ERROR: while connecting host %s, an error occured %s", sshResult.Host, err)
		chr <- sshResult
		return
	}
	defer session.Close()

	var outBuffer, errBuffer bytes.Buffer
	session.Stdout = &outBuffer
	session.Stderr = &errBuffer

	newCmd := strings.Join(cmdList, " && ")
	err = session.Run(newCmd)
	if err != nil {
		sshResult.Status = "failed"
		res := outBuffer.String()
		res = strings.TrimSpace(res)

		sshResult.Result = fmt.Sprintf("%s\nERROR: while running one or more command failed on host %s, an error occured %s", res, sshResult.Host, err.Error())
		chr <- sshResult
		return
	}
	if errBuffer.String() != "" {
		sshResult.Status = "failed"
		res := errBuffer.String()
		res = strings.TrimSpace(res)
		sshResult.Result = res
		chr <- sshResult
	} else {
		sshResult.Status = "success"
		res := outBuffer.String()
		res = strings.TrimSpace(res)
		sshResult.Result = res
		chr <- sshResult
	}
	return
}
