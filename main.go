package main

import (
	"fmt"
	"os"
	"strings"
	"time"
	"github.com/JeffreySE/ssgo/utils"
	"github.com/go-ini/ini"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	help              = app.HelpFlag.Short('h')
	app               = kingpin.New("ssgo", "A SSH-based command line tool for operating remote hosts.")
	example           = app.Flag("example", "Show examples of ssgo command.").Short('e').Default("false").Bool()
	inventory         = app.Flag("inventory", "For advanced use case, you can define a host warehouse .ini file (Default is 'config.ini' file in current directory.)").Short('i').ExistingFile()
	group             = app.Flag("group", "Remote host group name in the inventory file, which must be used with '-i' or '--inventory' argument!").Short('g').String()
	hostFile          = app.Flag("host-file", "A file contains remote host or host range IP Address.(e.g. 'hosts.example.txt' in current directory.)").ExistingFile()
	hostList          = app.Flag("host-list", "Remote host or host range IP Address. e.g. 192.168.10.100,192.168.10.101-192.168.10.103,192.168.20.100/28,192.168.30.11-15").String()
	password          = app.Flag("pass", "The SSH login password for remote hosts.").Short('p').String()
	user              = app.Flag("user", "The SSH login user for remote hosts. default is 'root'").Short('u').Default("root").String()
	port              = app.Flag("port", "The SSH login port for remote hosts. default is '22'").Short('P').Default("22").Int()
	//timeout           = app.Flag("timeout", "Set ssh connection timeout.").Short('t').Default("10s").Duration()
	maxExecuteNum     = app.Flag("maxExecuteNum", "Set Maximum number of hosts concurrent amount.").Short('n').Default("20").Int()
	formatResult      = app.Flag("format", "For pretty look in terminal,you can format the result with table,simple or other style.(Default is simple)").Short('F').Default("simple").String()
	maxTableCellWidth = app.Flag("maxTableCellWidth", "For pretty look,you can set the printed table's max cell width in terminal.(Default is 40)").Short('w').Default("40").Int()

	list = app.Command("list", "List available remote hosts from your input. ")

	run       = app.Command("run", "Run commands on remote hosts.")
	scriptFile = run.Flag("script", "Want execute script on remote hosts ? Just define the path of your script.").ExistingFile()
	scriptArgs = run.Flag("args", "Shell script arguments,use this flag with --script if you need.").Default("").String()
	cmdArgs    = run.Flag("cmds", "Define the commands or command file you want execute on remote hosts. By default will run 'echo pong' command if nothing is defined!").Short('c').Default("").String()

	sshCopy        = app.Command("copy", "Transfer files between local machine and remote hosts.")
	copyAction = sshCopy.Flag("action", "ssgo copy command do upload or download operations(only accept \"upload\" or \"download\" action)").Required().Short('a').String()
	sourcePath = sshCopy.Flag("src", "Source file or directory path on the local machine or remote hosts").Short('s').Required().String()
	destinationPath = sshCopy.Flag("dst", "Destination file or directory path on the remote host or local machine.").Short('d').Default("").String()

)

func main() {
	kingpin.Version("1.0.1")
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case list.FullCommand():
		if *example != false {
			utils.ShowListCommandUsage()
		} else if *inventory != "" && *group != "" {
			cfg, err := utils.Cfg(*inventory)
			if err != nil {
				utils.ColorPrint("ERROR", ">>>", "ERROR: ", err, "\n")
			}
			if *group == "all" {
				//get all hosts in config.ini file
				for _, s := range cfg.Sections() {
					if s.Name() == "DEFAULT" {
						continue
					}
					listCommandAction(s)
				}
			} else {
				s, _ := cfg.GetSection(*group)
				listCommandAction(s)
			}
			return
		} else if *hostFile != "" {
			hosts, err := utils.GetAvailableIPFromFile(*hostFile)
			if err != nil {
				fmt.Errorf("ERROR: %s", err)
			}
			utils.PrintListHosts(hosts, *maxTableCellWidth)
			return
		} else if *hostList != "" {
			hosts, err := utils.GetAvailableIP(*hostList)
			if err != nil {
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
			}
			utils.PrintListHosts(hosts, *maxTableCellWidth)
			return
		} else {
			utils.ShowListCommandUsage()
		}
	case run.FullCommand():
		if *example != false {
			utils.ShowRunCommandUsage()
		} else if *inventory != "" && *group != "" {
			var hosts []string
			cfg, err := utils.Cfg(*inventory)
			if err != nil {
				utils.ColorPrint("ERROR", ">>>", "ERROR: ", err, "\n")
			}
			if *group == "all" {
				//get all hosts in config.ini file
				for _, s := range cfg.Sections() {
					if s.Name() == "DEFAULT" {
						continue
					}
					if s.HasKey("hosts") {
						h, _ := s.GetKey("hosts")
						resHosts, err := utils.GetAvailableIPFromMultilines(h.String())
						if err != nil {
							utils.ColorPrint("ERROR", ">>>", "ERROR: ", err, "\n")
						}
						hosts = resHosts
						userName := s.Key("user").String()
						password := s.Key("pass").String()
						port := s.Key("port").MustInt()
						cmds, err := checkCommandArgs()
						if err != nil {
							utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
						}
						utils.ColorPrint("INFO", ">>> Group Name: ", "["+s.Name()+"]\n")
						if *scriptFile != "" {
							doSSHCommands(userName, password, "", port, hosts, []string{}, *scriptFile, *scriptArgs, "script")
							return
						}
						doSSHCommands(userName, password, "", port, hosts, cmds, "", "", "cmds")
					}
				}
			} else {
				s, _ := cfg.GetSection(*group)
				if s.HasKey("hosts") {
					h, _ := s.GetKey("hosts")
					resHosts, err := utils.GetAvailableIPFromMultilines(h.String())
					if err != nil {
						utils.ColorPrint("ERROR", ">>>", "ERROR: ", err, "\n")
					}
					hosts = resHosts
					userName := s.Key("user").String()
					password := s.Key("pass").String()
					port := s.Key("port").MustInt()
					cmds, err := checkCommandArgs()
					if err != nil {
						utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
					}
					if *scriptFile != "" {
						doSSHCommands(userName, password, "", port, hosts, []string{}, *scriptFile, *scriptArgs, "script")
						return
					}
					doSSHCommands(userName, password, "", port, hosts, cmds, "", "", "cmds")
				}
			}
			return
		} else if *hostFile != "" {
			hosts, err := utils.GetAvailableIPFromFile(*hostFile)
			if err != nil {
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
			}
			cmds, err := checkCommandArgs()
			if err != nil {
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
			}
			if *scriptFile != "" {
				doSSHCommands(*user, *password, "", *port, hosts, []string{}, *scriptFile, *scriptArgs, "script")
				return
			}
			doSSHCommands(*user, *password, "", *port, hosts, cmds, "", "", "cmds")
			return
		} else if *hostList != "" {
			hosts, err := utils.GetAvailableIP(*hostList)
			if err != nil {
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
			}
			cmds, err := checkCommandArgs()
			if err != nil {
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
			}
			if *scriptFile != "" {
				doSSHCommands(*user, *password, "", *port, hosts, []string{}, *scriptFile, *scriptArgs, "script")
				return
			}
			doSSHCommands(*user, *password, "", *port, hosts, cmds, "", "", "cmds")
			return
		} else {
			utils.ShowRunCommandUsage()
		}
	case sshCopy.FullCommand():
		if *example != false {
			utils.ShowFileTransferUsage()
		} else if *inventory != "" && *group != "" {
			var hosts []string
			cfg, err := utils.Cfg(*inventory)
			if err != nil {
				utils.ColorPrint("ERROR", ">>>", "ERROR: ", err, "\n")
			}
			if *group == "all" {
				//get all hosts in config.ini file
				for _, s := range cfg.Sections() {
					if s.Name() == "DEFAULT" {
						continue
					}
					if s.HasKey("hosts") {
						h, _ := s.GetKey("hosts")
						resHosts, err := utils.GetAvailableIPFromMultilines(h.String())
						if err != nil {
							utils.ColorPrint("ERROR", ">>>", "ERROR: ", err, "\n")
						}
						hosts = resHosts
						userName := s.Key("user").String()
						password := s.Key("pass").String()
						port := s.Key("port").MustInt()
						utils.ColorPrint("INFO", ">>> Group Name: ", "["+s.Name()+"]\n")
						if *copyAction == "upload" {
							doSFTPFileTransfer(userName, password, "", port, hosts, *sourcePath, *destinationPath, "upload")
						}else if *copyAction == "download"{
							doSFTPFileTransfer(userName, password, "", port, hosts, *sourcePath, *destinationPath, "download")
						}else {
							utils.ShowFileTransferUsage()
						}
					}
				}
			} else {
				s, _ := cfg.GetSection(*group)
				if s.HasKey("hosts") {
					h, _ := s.GetKey("hosts")
					resHosts, err := utils.GetAvailableIPFromMultilines(h.String())
					if err != nil {
						utils.ColorPrint("ERROR", ">>>", "ERROR: ", err, "\n")
					}
					hosts = resHosts
					userName := s.Key("user").String()
					password := s.Key("pass").String()
					port := s.Key("port").MustInt()
					if *copyAction == "upload" {
						doSFTPFileTransfer(userName, password, "", port, hosts, *sourcePath, *destinationPath, "upload")
					}else if *copyAction == "download"{
						doSFTPFileTransfer(userName, password, "", port, hosts, *sourcePath, *destinationPath, "download")
					}else {
						utils.ShowFileTransferUsage()
					}
				}
			}
			return
		} else if *hostFile != "" {
			hosts, err := utils.GetAvailableIPFromFile(*hostFile)
			if err != nil {
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
			}
			if *copyAction == "upload" {
				doSFTPFileTransfer(*user, *password, "", *port, hosts, *sourcePath, *destinationPath, "upload")
			}else if *copyAction == "download"{
				doSFTPFileTransfer(*user, *password, "", *port, hosts, *sourcePath, *destinationPath, "download")
			}else {
				utils.ShowFileTransferUsage()
			}
			return
		} else if *hostList != "" {
			hosts, err := utils.GetAvailableIP(*hostList)
			if err != nil {
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
			}
			if *copyAction == "upload" {
				doSFTPFileTransfer(*user, *password, "", *port, hosts, *sourcePath, *destinationPath, "upload")
			}else if *copyAction == "download"{
				doSFTPFileTransfer(*user, *password, "", *port, hosts, *sourcePath, *destinationPath, "download")
			}else {
				utils.ShowFileTransferUsage()
			}
			return
		} else {
			utils.ShowFileTransferUsage()
		}
	}
}

func listCommandAction(sec *ini.Section) {
	utils.ColorPrint("INFO", "", ">>> Group Name: ", "["+sec.Name()+"]\n")
	if sec.HasKey("hosts") {
		h, _ := sec.GetKey("hosts")
		utils.ColorPrint("INFO", "", ">>> Hosts From: ", h.String(), "\n")
		hosts, err := utils.GetAvailableIPFromMultilines(h.String())
		if err != nil {
			fmt.Errorf("ERROR: %s", err)
		}
		utils.PrintListHosts(hosts, *maxTableCellWidth, sec.Name())
	}
}

func checkCommandArgs() ([]string, error) {
	var cmds []string
	if *cmdArgs != "" {
		strPath, err := utils.GetRealPath(*cmdArgs)
		if err != nil {
			// not real path
			if !strings.Contains(*cmdArgs, ";") {
				cmds = append(cmds, *cmdArgs)
			}
			cmds = strings.Split(*cmdArgs, ";")
		} else {
			// cmd file path, just get the commands content
			cmdFileContent, err := utils.GetFileContent(strPath)
			cmds = cmdFileContent
			if err != nil {
				fmt.Println("ERROR:", err)
				return cmds, err
			}
		}
	}
	return cmds, nil
}

func doSSHCommands(user, password, key string, port int, todoHosts, cmds []string, scriptFilePath, scriptArgs, action string) {
	var errHosts, successHosts []utils.SSHResult
	if len(cmds) == 0 {
		cmds = append(cmds, "echo pong")
	}
	todoHosts, err := utils.DuplicateIPAddressCheck(todoHosts)
	if err != nil {
		fmt.Println(err)
		return
	}
	pool := utils.NewPool(*maxExecuteNum, len(todoHosts))
	startTime := time.Now()
	utils.ColorPrint("INFO", "", "Tips:", fmt.Sprintf("Process running start: %s\n", startTime.Format("2006-01-02 15:04:05")))
	chs := make([]chan utils.SSHResult, len(todoHosts))
	for i, host := range todoHosts {
		chs[i] = make(chan utils.SSHResult, 1)
		go func(h string, ch chan utils.SSHResult, a string) {
			pool.AddOne()
			switch a {
			case "script":
				utils.SSHRunShellScript(user, password, h, key, scriptFilePath, scriptArgs, port, ch)
			case "cmds":
				utils.DoSSHRunFast(user, password, h, key, cmds, port, ch)
			}
			pool.DelOne()
		}(host, chs[i], action)
		if *formatResult == "simple" {
			res := <-chs[i]
			if res.Status == "failed" {
				errHosts = append(errHosts, res)
			} else {
				successHosts = append(successHosts, res)
			}
			utils.FormatResultWithBasicStyle(i, res)
		}
	}
	if *formatResult == "table" {
		//formatResultWithTableStyle(chs)
		for _, resCh := range chs {
			result := <-resCh
			if result.Status == "failed" {
				errHosts = append(errHosts, result)
			} else {
				successHosts = append(successHosts, result)
			}
		}
		if len(successHosts) > 0 {
			utils.ColorPrint("INFO", "", "INFO: ", "Success hosts\n")
			utils.FormatResultWithTableStyle(successHosts, *maxTableCellWidth)
		}
	}
	pool.Wg.Wait()
	if len(errHosts) > 0 {
		utils.ColorPrint("ERROR", "", "WARNING: ", "Failed hosts, please confirm!\n")
		utils.FormatErorCommandsResultWithTableStyle(errHosts, *maxTableCellWidth)
	}
	endTime := time.Now()
	utils.ColorPrint("INFO", "", "Tips: ", fmt.Sprintf("Process running done.\n"))
	fmt.Printf("End Time: %s\nCost Time: %s\nTotal Hosts Running: %s\n", endTime.Format("2006-01-02 15:04:05"), endTime.Sub(startTime), fmt.Sprintf("%d(Success) + %d(Failed) = %d(Total)\n", len(successHosts), len(errHosts), len(successHosts)+len(errHosts)))
}

func doSFTPFileTransfer(user, password, key string, port int, todoHosts []string, sourcePath, destinationPath, action string) {
	var errHosts, successHosts []utils.SFTPResult
	todoHosts, err := utils.DuplicateIPAddressCheck(todoHosts)
	if err != nil {
		fmt.Println(err)
		return
	}
	pool := utils.NewPool(*maxExecuteNum, len(todoHosts))
	startTime := time.Now()
	utils.ColorPrint("INFO", "", "Tips:", fmt.Sprintf("Process running start: %s\n", startTime.Format("2006-01-02 15:04:05")))
	chs := make([]chan utils.SFTPResult, len(todoHosts))
	for i, host := range todoHosts {
		chs[i] = make(chan utils.SFTPResult, 1)
		go func(h string, ch chan utils.SFTPResult, a, s, d string) {
			pool.AddOne()
			switch a {
			case "upload":
				utils.SFTPUpload(user, password, h, key, port, s, d, ch)
			case "download":
				utils.SFTPDownload(user, password, h, key, port, s, d, ch)
			}
			pool.DelOne()
		}(host, chs[i], action, sourcePath, destinationPath)
		if *formatResult == "simple" {
			res := <-chs[i]
			if res.Status == "failed" {
				errHosts = append(errHosts, res)
			} else {
				successHosts = append(successHosts, res)
			}
			utils.SFTPFormatResultWithBasicStyle(i, res)
		}
	}
	if *formatResult == "table" {
		//formatResultWithTableStyle(chs)
		for _, resCh := range chs {
			result := <-resCh
			if result.Status == "failed" {
				errHosts = append(errHosts, result)
			} else {
				successHosts = append(successHosts, result)
			}
		}
		if len(successHosts) > 0 {
			utils.ColorPrint("INFO", "", "INFO: ", "Success hosts\n")
			utils.SFTPFormatResultWithTableStyle(successHosts, *maxTableCellWidth)
		}
	}
	pool.Wg.Wait()
	if len(errHosts) > 0 {
		utils.ColorPrint("ERROR", "", "WARNING: ", "Failed hosts, please confirm!\n")
		utils.SFTPFormatResultWithTableStyle(errHosts, *maxTableCellWidth)
	}
	endTime := time.Now()
	utils.ColorPrint("INFO", "", "Tips: ", fmt.Sprintf("Process running done.\n"))
	fmt.Printf("End Time: %s\nCost Time: %s\nTotal Hosts Running: %s\n", endTime.Format("2006-01-02 15:04:05"), endTime.Sub(startTime), fmt.Sprintf("%d(Success) + %d(Failed) = %d(Total)\n", len(successHosts), len(errHosts), len(successHosts)+len(errHosts)))
}
