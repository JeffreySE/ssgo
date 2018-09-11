package main

import (
	"fmt"
	"github.com/JeffreySE/ssgo/utils"
	"github.com/go-ini/ini"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"strings"
	"time"
)

var (
	app       = kingpin.New("ssgo", "A SSH-based command line tool for operating remote hosts.")
	_         = app.HelpFlag.Short('h')
	example   = app.Flag("example", "Show examples of ssgo's command.").Short('e').Default("false").Bool()
	inventory = app.Flag("inventory", "For advanced use case, you can specify a host warehouse .ini file (Default is 'config.ini' file in current directory.)").Short('i').ExistingFile()
	group     = app.Flag("group", "Remote host group name in the inventory file, which must be used with '-i' or '--inventory' argument!").Short('g').String()
	hostFile  = app.Flag("host-file", "A file contains remote host or host range IP Address.(e.g. 'hosts.example.txt' in current directory.)").ExistingFile()
	hostList  = app.Flag("host-list", "Remote host or host range IP Address. e.g. 192.168.10.100,192.168.10.101-192.168.10.103,192.168.20.100/28,192.168.30.11-15").String()
	password  = app.Flag("pass", "The SSH login password for remote hosts.").Short('p').String()
	user      = app.Flag("user", "The SSH login user for remote hosts. default is 'root'").Short('u').Default("root").String()
	port      = app.Flag("port", "The SSH login port for remote hosts. default is '22'").Short('P').Default("22").Int()
	//timeout           = app.Flag("timeout", "Set ssh connection timeout.").Short('t').Default("10s").Duration()
	maxExecuteNum     = app.Flag("maxExecuteNum", "Set Maximum concurrent count of hosts.").Short('n').Default("20").Int()
	output            = app.Flag("output", "Output result'log to a file.(Be default if your input is \"log\",ssgo will output logs like \"ssgo-%s.log\")").Short('o').String()
	formatMode        = app.Flag("format", "For pretty look in terminal,you can format the result with table,simple,json or other style.(Default is simple)").Short('F').Default("simple").String()
	jsonRaw           = app.Flag("json-raw", "By default, the json data will be formatted and output by the console. You can specify the --json-raw parameter to output raw json data.(Default is false)").Default("false").Bool()
	maxTableCellWidth = app.Flag("maxTableCellWidth", "For pretty look,you can set the printed table's max cell width in terminal.(Default is 40)").Short('w').Default("40").Int()

	list = app.Command("list", "List available remote hosts from your input. ")

	run        = app.Command("run", "Run commands on remote hosts.")
	scriptFile = run.Flag("script", "Want execute script on remote hosts ? Just specify the path of your script.").PlaceHolder("shell-script.sh").Short('s').ExistingFile()
	scriptArgs = run.Flag("args", "Shell script arguments,use this flag with --script if you need.").Short('a').Default("").String()
	cmdArgs    = run.Flag("cmd", "Specify the commands or command file you want execute on remote hosts. By default will run 'echo pong' command if nothing is specified!").Short('c').Default("").String()

	sshCopy         = app.Command("copy", "Transfer files between local machine and remote hosts.")
	copyAction      = sshCopy.Flag("action", "ssgo's copy command do upload or download operations(only accept \"upload\" or \"download\" action)").Required().Short('a').String()
	sourcePath      = sshCopy.Flag("src", "Source file or directory path on the local machine or remote hosts").Short('s').Required().String()
	destinationPath = sshCopy.Flag("dst", "Destination file or directory path on the remote host or local machine.").Short('d').Default("").String()
)

var (
	allResultLogs []utils.ResultLogs
)

func main() {
	app.Version("1.0.3")
	app.VersionFlag.Short('v')
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
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
				return
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
				isFinished := false
				for index, s := range cfg.Sections() {
					if s.Name() == "DEFAULT" {
						continue
					}

					if s.HasKey("hosts") {
						h, _ := s.GetKey("hosts")
						resHosts, err := utils.GetAvailableIPFromMultiLines(h.String())
						if err != nil {
							utils.ColorPrint("ERROR", ">>>", "ERROR: ", err, "\n")
							return
						}
						hosts = resHosts
						userName := s.Key("user").String()
						password := s.Key("pass").String()
						port := s.Key("port").MustInt()

						cmds, err := checkCommandArgs()
						if err != nil {
							utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
							return
						}
						if *formatMode != "json" {
							utils.ColorPrint("INFO", ">>> Group Name: ", "["+s.Name()+"]\n")
						}
						if index == len(cfg.Sections())-1 {
							isFinished = true
						}
						if *scriptFile != "" {
							doSSHCommands(userName, password, fmt.Sprintf("from hostgroup %s@%s file", s.Name(), *inventory), "", port, hosts, []string{}, *scriptFile, *scriptArgs, "script", isFinished)
						}
						if *cmdArgs != "" {
							doSSHCommands(userName, password, fmt.Sprintf("from hostgroup %s@%s file", s.Name(), *inventory), "", port, hosts, cmds, "", "", "cmd", isFinished)
						}

					}
				}
				return
			} else {
				s, _ := cfg.GetSection(*group)
				if s.HasKey("hosts") {
					h, _ := s.GetKey("hosts")
					resHosts, err := utils.GetAvailableIPFromMultiLines(h.String())
					if err != nil {
						utils.ColorPrint("ERROR", ">>>", "ERROR: ", err, "\n")
						return
					}
					hosts = resHosts
					userName := s.Key("user").String()
					password := s.Key("pass").String()
					port := s.Key("port").MustInt()
					cmds, err := checkCommandArgs()
					if err != nil {
						utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
						return
					}
					if *formatMode != "json" {
						utils.ColorPrint("INFO", ">>> Group Name: ", "["+s.Name()+"]\n")
					}
					if *scriptFile != "" {
						doSSHCommands(userName, password, fmt.Sprintf("from hostgroup %s@%s file", s.Name(), *inventory), "", port, hosts, []string{}, *scriptFile, *scriptArgs, "script", true)
					}
					if *cmdArgs != "" {
						doSSHCommands(userName, password, fmt.Sprintf("from hostgroup %s@%s file", s.Name(), *inventory), "", port, hosts, cmds, "", "", "cmd", true)
					}
				}
			}
			return
		} else if *hostFile != "" {
			hosts, err := utils.GetAvailableIPFromFile(*hostFile)
			if err != nil {
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
				return
			}
			cmds, err := checkCommandArgs()
			if err != nil {
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
				return
			}
			if *scriptFile != "" {
				doSSHCommands(*user, *password, fmt.Sprintf("from file (%s)", *hostFile), "", *port, hosts, []string{}, *scriptFile, *scriptArgs, "script", true)
				return
			}
			if *cmdArgs != "" {
				doSSHCommands(*user, *password, fmt.Sprintf("from file (%s)", *hostFile), "", *port, hosts, cmds, "", "", "cmd", true)
			}
			return
		} else if *hostList != "" {
			hosts, err := utils.GetAvailableIP(*hostList)
			if err != nil {
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
				return
			}
			cmds, err := checkCommandArgs()
			if err != nil {
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
				return
			}
			if *scriptFile != "" {
				doSSHCommands(*user, *password, fmt.Sprintf("from list (%s)", *hostList), "", *port, hosts, []string{}, *scriptFile, *scriptArgs, "script", true)
				return
			}
			if *cmdArgs != "" {
				doSSHCommands(*user, *password, fmt.Sprintf("from list (%s)", *hostList), "", *port, hosts, cmds, "", "", "cmd", true)
			}
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
				return
			}
			if *group == "all" {
				//get all hosts in config.ini file
				isFinished := false
				for index, s := range cfg.Sections() {
					if s.Name() == "DEFAULT" {
						continue
					}
					if s.HasKey("hosts") {
						h, _ := s.GetKey("hosts")
						resHosts, err := utils.GetAvailableIPFromMultiLines(h.String())
						if err != nil {
							utils.ColorPrint("ERROR", ">>>", "ERROR: ", err, "\n")
						}
						hosts = resHosts
						userName := s.Key("user").String()
						password := s.Key("pass").String()
						port := s.Key("port").MustInt()
						utils.ColorPrint("INFO", ">>> Group Name: ", "["+s.Name()+"]\n")
						if index == len(cfg.Sections())-1 {
							isFinished = true
						}
						if *copyAction == "upload" {
							doSFTPFileTransfer(userName, password, s.Name(), "", port, hosts, *sourcePath, *destinationPath, "upload", isFinished)
						} else if *copyAction == "download" {
							doSFTPFileTransfer(userName, password, s.Name(), "", port, hosts, *sourcePath, *destinationPath, "download", isFinished)
						} else {
							utils.ShowFileTransferUsage()
						}
					}
				}
			} else {
				s, _ := cfg.GetSection(*group)
				if s.HasKey("hosts") {
					h, _ := s.GetKey("hosts")
					resHosts, err := utils.GetAvailableIPFromMultiLines(h.String())
					if err != nil {
						utils.ColorPrint("ERROR", ">>>", "ERROR: ", err, "\n")
						return
					}
					hosts = resHosts
					userName := s.Key("user").String()
					password := s.Key("pass").String()
					port := s.Key("port").MustInt()
					if *copyAction == "upload" {
						doSFTPFileTransfer(userName, password, s.Name(), "", port, hosts, *sourcePath, *destinationPath, "upload", true)
					} else if *copyAction == "download" {
						doSFTPFileTransfer(userName, password, s.Name(), "", port, hosts, *sourcePath, *destinationPath, "download", true)
					} else {
						utils.ShowFileTransferUsage()
					}
				}
			}
			return
		} else if *hostFile != "" {
			hosts, err := utils.GetAvailableIPFromFile(*hostFile)
			if err != nil {
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
				return
			}
			if *copyAction == "upload" {
				doSFTPFileTransfer(*user, *password, "from-file", "", *port, hosts, *sourcePath, *destinationPath, "upload", true)
			} else if *copyAction == "download" {
				doSFTPFileTransfer(*user, *password, "from-file", "", *port, hosts, *sourcePath, *destinationPath, "download", true)
			} else {
				utils.ShowFileTransferUsage()
			}
			return
		} else if *hostList != "" {
			hosts, err := utils.GetAvailableIP(*hostList)
			if err != nil {
				utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
				return
			}
			if *copyAction == "upload" {
				doSFTPFileTransfer(*user, *password, "from-list", "", *port, hosts, *sourcePath, *destinationPath, "upload", true)
			} else if *copyAction == "download" {
				doSFTPFileTransfer(*user, *password, "from-list", "", *port, hosts, *sourcePath, *destinationPath, "download", true)
			} else {
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
		hosts, err := utils.GetAvailableIPFromMultiLines(h.String())
		if err != nil {
			utils.ColorPrint("ERROR", "", "ERROR:", err, "\n")
			return

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

func doSSHCommands(user, password, hostGroupName, key string, port int, todoHosts, cmds []string, scriptFilePath, scriptArgs, action string, isFinished bool) {
	var resultLog utils.ResultLogs
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
	resultLog.StartTime = startTime.Format("2006-01-02 15:04:05")
	resultLog.HostGroup = hostGroupName
	switch *formatMode {
	case "simple", "table":
		utils.ColorPrint("INFO", "", "Tips:", fmt.Sprintf("Process running start: %s\n", resultLog.StartTime))
	}
	if *output != "" {
		utils.WriteAndAppendFile(*output, fmt.Sprintf("Tips: process running start: %s", resultLog.StartTime))
	}
	chres := make([]chan interface{}, len(todoHosts))
	for i, host := range todoHosts {
		chres[i] = make(chan interface{}, 1)
		go func(h string, a string, chr chan interface{}) {
			pool.AddOne()
			switch a {
			case "script":
				utils.SSHRunShellScript(user, password, h, key, scriptFilePath, scriptArgs, port, chr)
			case "cmd":
				utils.DoSSHRunFast(user, password, h, key, cmds, port, chr)
			}
			pool.DelOne()
		}(host, action, chres[i])
		if *formatMode == "simple" || *output != "" {
			res := <-chres[i]
			if res.(utils.SSHResult).Status == "failed" {
				resultLog.ErrorHosts = append(resultLog.ErrorHosts, res)
			} else {
				resultLog.SuccessHosts = append(resultLog.SuccessHosts, res)
			}
			utils.FormatResultWithBasicStyle(i, res.(utils.SSHResult))
			if *output != "" {
				utils.LogSSHResultToFile(i, res.(utils.SSHResult), *output)
			}
		}
	}
	switch *formatMode {
	case "simple":
		utils.FormatResultLogWithSimpleStyle(resultLog, startTime, *maxTableCellWidth, []string{"Result"})
	case "table":
		utils.FormatResultLogWithTableStyle(chres, resultLog, startTime, *maxTableCellWidth)
	case "json":
		if *inventory != "" && *group == "all" {
			log := utils.GetAllResultLog(chres, resultLog, startTime)
			allResultLogs = append(allResultLogs, log)
			if isFinished {
				utils.FormatResultToJson(allResultLogs, *jsonRaw)
			}
		} else {
			utils.FormatResultLogWithJsonStyle(chres, resultLog, startTime, *jsonRaw)
		}
	}
	if *output != "" {
		utils.ResultLogInfo(resultLog, startTime, true, *output)
	}
	pool.Wg.Wait()
}

func doSFTPFileTransfer(user, password, hostGroupName, key string, port int, todoHosts []string, sourcePath, destinationPath, action string, isFinished bool) {
	var resultLog utils.ResultLogs
	todoHosts, err := utils.DuplicateIPAddressCheck(todoHosts)
	if err != nil {
		fmt.Println(err)
		return
	}
	pool := utils.NewPool(*maxExecuteNum, len(todoHosts))
	startTime := time.Now()
	resultLog.StartTime = startTime.Format("2006-01-02 15:04:05")
	resultLog.HostGroup = hostGroupName
	if *formatMode == "simple" || *formatMode == "table" {
		utils.ColorPrint("INFO", "", "Tips:", fmt.Sprintf("Process running start: %s\n", resultLog.StartTime))
	}
	if *output != "" {
		utils.WriteAndAppendFile(*output, fmt.Sprintf("Tips: process running start: %s", resultLog.StartTime))
	}
	chres := make([]chan interface{}, len(todoHosts))
	for i, host := range todoHosts {
		chres[i] = make(chan interface{}, 1)
		go func(h string, a, s, d string, chr chan interface{}) {
			pool.AddOne()
			switch a {
			case "upload":
				utils.SFTPUpload(user, password, h, key, port, s, d, chr)
			case "download":
				utils.SFTPDownload(user, password, h, key, port, s, d, chr)
			}
			pool.DelOne()
		}(host, action, sourcePath, destinationPath, chres[i])
		if *formatMode == "simple" || *output != "" {
			res := <-chres[i]
			if res.(utils.SFTPResult).Status == "failed" {
				resultLog.ErrorHosts = append(resultLog.ErrorHosts, res)
			} else {
				resultLog.SuccessHosts = append(resultLog.SuccessHosts, res)
			}
			utils.SFTPFormatResultWithBasicStyle(i, res.(utils.SFTPResult))
			if *output != "" {
				utils.LogSFTPResultToFile(i, res.(utils.SFTPResult), *output)
			}
		}
	}
	switch *formatMode {
	case "simple":
		utils.FormatResultLogWithSimpleStyle(resultLog, startTime, *maxTableCellWidth, []string{})
	case "table":
		utils.FormatResultLogWithTableStyle(chres, resultLog, startTime, *maxTableCellWidth)
	case "json":
		if *inventory != "" && *group == "all" {
			log := utils.GetAllResultLog(chres, resultLog, startTime)
			allResultLogs = append(allResultLogs, log)
			if isFinished {
				utils.FormatResultToJson(allResultLogs, *jsonRaw)
			}
		} else {
			utils.FormatResultLogWithJsonStyle(chres, resultLog, startTime, *jsonRaw)
		}
	}
	if *output != "" {
		utils.ResultLogInfo(resultLog, startTime, true, *output)
	}
	pool.Wg.Wait()
}
