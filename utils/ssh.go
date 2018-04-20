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
		Timeout: 30 * time.Second,
		Config:  config,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// connet to ssh
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

func SSHRunShellScript(user, password, host, key, scriptFilePath, scriptAgrs string, port int, ch chan SSHResult) {
	var sshResult SSHResult
	var cmds []string
	sshResult.Host = host
	session, err := connect(user, password, host, key, port)
	if err != nil {
		sshResult.Status = "failed"
		sshResult.Result = fmt.Sprintf("ERROR: while connecting host %s, an error occured,error message: %s", sshResult.Host, err)
		ch <- sshResult
		return
	}
	defer session.Close()
	var outbt, errbt bytes.Buffer
	session.Stdout = &outbt
	session.Stderr = &errbt

	resSftpResult := SFTPSimpleUpload(user, password, host, key, port, scriptFilePath, "")
	if resSftpResult.Status == "false" {
		sshResult.Status = "failed"
		sshResult.Result = fmt.Sprintf("ERROR: copy local Shell script %s to host %s failed, error message: %s", scriptFilePath, sshResult.Host, err.Error())
		ch <- sshResult
		return
	}

	scriptFileRemotePath := resSftpResult.DestinationPath + "/" + filepath.Base(scriptFilePath)
	executeScriptCmd := fmt.Sprintf("%s %s %s", "/bin/sh", scriptFileRemotePath, scriptAgrs)
	removeScriptBeforeExitCmd := fmt.Sprintf("rm -rf %s", scriptFileRemotePath)
	cmds = append(cmds, executeScriptCmd, removeScriptBeforeExitCmd, "exit")
	cmd := strings.Join(cmds, " && ")
	err = session.Run(cmd)
	if err != nil {
		sshResult.Status = "failed"
		sshResult.Result = fmt.Sprintf("ERROR: while running command (%s) on host %s, an error occured %s", cmd, sshResult.Host, err.Error())
		ch <- sshResult
		return
	}
	if errbt.String() != "" {
		sshResult.Status = "failed"
		sshResult.Result = errbt.String()
		ch <- sshResult
	} else {
		sshResult.Status = "success"
		sshResult.Result = outbt.String()
		ch <- sshResult
	}
	ch <- sshResult
	return
}
func DoSSHRunFast(user, password, host, key string, cmdList []string, port int, ch chan SSHResult) {
	var sshResult SSHResult
	sshResult.Host = host
	session, err := connect(user, password, host, key, port)
	if err != nil {
		sshResult.Status = "failed"
		sshResult.Result = fmt.Sprintf("ERROR: while connecting host %s, an error occured %s", sshResult.Host, err)
		ch <- sshResult
		return
	}
	defer session.Close()

	var outbt, errbt bytes.Buffer
	session.Stdout = &outbt
	session.Stderr = &errbt

	newCmd := strings.Join(cmdList, " && ")
	err = session.Run(newCmd)
	if err != nil {
		sshResult.Status = "failed"
		res := outbt.String()
		res = strings.TrimSpace(res)

		sshResult.Result = fmt.Sprintf("%s\nERROR: while running one or more command failed on host %s, an error occured %s", res,sshResult.Host, err.Error())
		ch <- sshResult
		return
	}
	if errbt.String() != "" {
		sshResult.Status = "failed"
		res := errbt.String()
		res = strings.TrimSpace(res)
		sshResult.Result = res
		ch <- sshResult
	} else {
		sshResult.Status = "success"
		res := outbt.String()
		res = strings.TrimSpace(res)
		sshResult.Result = res
		ch <- sshResult
	}
	return
}
