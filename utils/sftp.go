package utils

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"os"
	"path"
	"time"
)

type SFTPResult struct {
	Host            string
	Status          string
	SourcePath      string
	DestinationPath string
	Result          string
}

// coped from https://github.com/shanghai-edu/multissh (thank you very much)
func sftpConnect(user, password, host, key string, port int) (*sftp.Client, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		config       ssh.Config
		sftpClient   *sftp.Client
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

	// connect to ssh
	addr = fmt.Sprintf("%s:%d", host, port)

	if sshClient, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	// create sftp client
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		return nil, err
	}

	return sftpClient, nil
}

func SFTPSimpleUpload(user, password, host, key string, port int, sourcePath, destinationPath string) SFTPResult {
	var (
		err        error
		sftpClient *sftp.Client
		sftpResult SFTPResult
	)
	sftpResult.Host = host
	sftpResult.SourcePath = sourcePath
	sftpResult.DestinationPath = destinationPath
	sftpClient, err = sftpConnect(user, password, host, key, port)
	if err != nil {
		sftpResult.Status = "failed"
		sftpResult.Result = fmt.Sprintf("ERROR: sftp connect to %s failed, error message:%s", sftpResult.Host, err.Error())
		return sftpResult
	}
	defer sftpClient.Close()

	srcFile, err := os.Open(sourcePath)
	if err != nil {
		sftpResult.Status = "failed"
		sftpResult.Result = fmt.Sprintf("ERROR: os open file \"%s\" failed, error message:%s", sftpResult.Host, err.Error())
		return sftpResult
	}
	defer srcFile.Close()

	var remoteFileName = path.Base(sourcePath)
	dstFile, err := sftpClient.Create(path.Join(destinationPath, remoteFileName))
	if err != nil {
		sftpResult.Status = "failed"
		sftpResult.Result = fmt.Sprintf("ERROR: while upload file \"%s\" to remote path \"%s\" ,error message:%s ", sftpResult.SourcePath, sftpResult.DestinationPath, err.Error())
		return sftpResult
	}
	defer dstFile.Close()

	buf := make([]byte, 1024)
	for {
		n, _ := srcFile.Read(buf)
		if n == 0 {
			break
		}
		dstFile.Write(buf)
	}
	if destinationPath == "" {
		currWorkDir, _ := sftpClient.Getwd()
		sftpResult.DestinationPath = currWorkDir
	}
	sftpResult.Status = "success"
	sftpResult.Result = fmt.Sprintf("Upload finished!:)")
	return sftpResult
}

func SFTPUpload(user, password, host, key string, port int, sourcePath, destinationPath string, chr chan interface{}) {
	var (
		err        error
		sftpClient *sftp.Client
		sftpResult SFTPResult
	)
	sftpResult.Host = host
	sftpResult.SourcePath = sourcePath
	sftpResult.DestinationPath = destinationPath
	sftpClient, err = sftpConnect(user, password, host, key, port)
	if err != nil {
		sftpResult.Status = "failed"
		sftpResult.Result = fmt.Sprintf("ERROR: sftp connect to %s failed, error message:%s", sftpResult.Host, err.Error())
		chr <- sftpResult
		return
	}
	defer sftpClient.Close()
	if destinationPath == "" {
		currWorkDir, _ := sftpClient.Getwd()
		sftpResult.DestinationPath = currWorkDir
	}
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		sftpResult.Status = "failed"
		sftpResult.Result = fmt.Sprintf("ERROR: os open file \"%s\" failed, error message:%s", sftpResult.Host, err.Error())
		chr <- sftpResult
		return
	}
	defer srcFile.Close()
	srcFileInfo, _ := srcFile.Stat()
	remoteFileName := srcFileInfo.Name()
	dstFile, err := sftpClient.Create(path.Join(destinationPath, remoteFileName))
	if err != nil {
		sftpResult.Status = "failed"
		sftpResult.Result = fmt.Sprintf("ERROR: while upload file \"%s\" to remote path \"%s\" ,error message:%s ", sftpResult.SourcePath, sftpResult.DestinationPath, err.Error())
		chr <- sftpResult
		return
	}
	defer dstFile.Close()

	buf := make([]byte, 1024)
	for {
		n, _ := srcFile.Read(buf)
		if n == 0 {
			break
		}
		dstFile.Write(buf)
	}

	sftpResult.Status = "success"
	sftpResult.Result = fmt.Sprintf("Upload finished!:)")
	chr <- sftpResult
	return
}

func SFTPDownload(user, password, host, key string, port int, sourcePath, destinationPath string, chr chan interface{}) {
	var (
		err        error
		sftpClient *sftp.Client
		sftpResult SFTPResult
	)
	sftpResult.Host = host
	sftpResult.SourcePath = sourcePath
	sftpResult.DestinationPath = destinationPath
	sftpClient, err = sftpConnect(user, password, host, key, port)
	if err != nil {
		sftpResult.Status = "failed"
		sftpResult.Result = fmt.Sprintf("ERROR: sftp connect to %s failed, error message:%s", sftpResult.Host, err.Error())
		chr <- sftpResult
		return
	}
	defer sftpClient.Close()

	srcFile, err := sftpClient.Open(sourcePath)
	if err != nil {
		sftpResult.Status = "failed"
		sftpResult.Result = fmt.Sprintf("ERROR: sftp open file failed %s, error message:%s", sftpResult.SourcePath, err.Error())
		chr <- sftpResult
		return
	}
	defer srcFile.Close()
	if destinationPath == "" {
		currWorkDir, _ := os.Getwd()
		sftpResult.DestinationPath = currWorkDir
	}
	fileInfo, _ := srcFile.Stat()
	var localFileName = fmt.Sprintf("%s_%s", sftpResult.Host, fileInfo.Name())
	dstFile, err := os.Create(path.Join(destinationPath, localFileName))
	if err != nil {
		sftpResult.Status = "failed"
		sftpResult.Result = fmt.Sprintf("ERROR: while download file \"%s\" to local path \"%s\" ,error message: %s", sftpResult.SourcePath, sftpResult.DestinationPath, err.Error())
		chr <- sftpResult
		return
	}
	defer dstFile.Close()

	if _, err := srcFile.WriteTo(dstFile); err != nil {
		sftpResult.Status = "failed"
		sftpResult.Result = fmt.Sprintf("ERROR: while download file \"%s\" to local path \"%s\" ,error message:%s ", sftpResult.SourcePath, sftpResult.DestinationPath, err.Error())
		chr <- sftpResult
		return
	}

	sftpResult.Status = "success"
	sftpResult.Result = fmt.Sprintf("Download finished!:)")
	chr <- sftpResult
	return
}
