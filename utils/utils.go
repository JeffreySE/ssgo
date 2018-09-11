package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bndr/gotabulate"
	"github.com/daviddengcn/go-colortext"
	"github.com/go-ini/ini"
	"io/ioutil"
	"net"
	"os"
	"path/filepath" // cross platform for windows & linux
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ResultLogs struct {
	StartTime      string
	HostGroup      string
	SuccessHosts   []interface{}
	ErrorHosts     []interface{}
	EndTime        string
	CostTime       string
	TotalHostsInfo string
}

// 检测文件或文件夹是否存在
func IsPathExists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, err
	}
	return false, err
}

//获取当前路径
func GetCurrentDir() string {
	pwd, _ := os.Getwd()
	return pwd
}

// 根据输入路径获取该路径的物理路径，如果是相对路径，则附加在当前目录下，并判断附加后的路径是否存在
func GetRealPath(strPath string) (string, error) {
	if !filepath.IsAbs(strPath) {
		pwd := filepath.Join(GetCurrentDir(), strPath)
		absStrPath, _ := filepath.Abs(pwd)
		strPath = absStrPath
	}
	checkPath, err := IsPathExists(strPath)
	if !checkPath {
		return "", err
	}
	return strPath, nil
}

//检测当前目录下是否存在某一文件或路径
func IsPathExistInCurrentPath(path string) (bool, error) {
	pwd := GetCurrentDir()
	strPath := filepath.Join(pwd, path)
	rst, err := IsPathExists(strPath)
	if !rst {
		return false, err
	}
	return true, nil
}

//检测ssgo默认配置文件config.ini文件是否存在
func CheckDefaultINIFile(name string) (bool, error) {
	r, err := IsPathExistInCurrentPath(name)

	if !r {
		return false, err
	}
	return true, nil
}

// functions for parsing ip address
//检测IP地址，返回true or false
func CheckIp(strIPAddress string) bool {
	strIPAddress = strings.TrimSpace(strIPAddress)
	if net.ParseIP(strIPAddress) == nil {
		return false
	}
	return true
}

// 将IP地址的掩码转换为CIDR格式的掩码，比如，255.255.255.0 转换为 24
func IPMaskToCIDRMask(netmask string) (bool, string) {
	netMasks := strings.Split(netmask, ".")
	var ms []int
	for _, v := range netMasks {
		intV, err := strconv.Atoi(v)
		if err != nil {
			return false, "ERROR: '" + netmask + "' is not a valid subnet mask, subnet mask should be numbers,please check the subnet mask form!"
		}
		ms = append(ms, intV)
	}
	ipMask := net.IPv4Mask(byte(ms[0]), byte(ms[1]), byte(ms[2]), byte(ms[3]))
	ones, _ := ipMask.Size()
	if ones == 0 {
		return false, "ERROR: '" + netmask + "' is not a valid subnet mask,please check the subnet mask form!"
	}
	return true, strconv.Itoa(ones)
}

//获取可用IP地址 默认以逗号分隔符来根据IP地址表示形式来解析可用IP地址清单
//比如：192.168.100.200-192.168.100.204,#192.168.100.204,192.168.100.272
// 可以是单个IP地址、IP地址段、也可以用逗号分隔多个IP地址或IP地址段
func GetAvailableIP(strIPList string) ([]string, error) {
	var availableIPs []string
	strIPList = strings.TrimSpace(strIPList)
	if !strings.Contains(strIPList, ",") {
		ips, err := GetAvailableIPList(strIPList)
		if err != nil {
			return availableIPs, err
		}
		availableIPs = ips
		return availableIPs, nil
	}
	strIPs := strings.Split(strIPList, ",")
	for _, strIP := range strIPs {
		ips, err := GetAvailableIPList(strIP)
		if err != nil {
			continue
		}
		availableIPs = append(availableIPs, ips...)
	}
	if len(availableIPs) == 0 {
		return availableIPs, fmt.Errorf("ERROR: no valid IP Address found, please check your input")
	}

	return availableIPs, nil
}

// get file content
func GetFileContent(strFilePath string) ([]string, error) {
	var fileContent []string
	strFilePath = strings.TrimSpace(strFilePath)
	path, err := GetRealPath(strFilePath)
	if err != nil {
		return fileContent, err
	}
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return fileContent, err
	}
	strContent := string(buf)
	for _, lineStr := range strings.Split(strContent, "\n") {
		lineStr = strings.TrimSpace(lineStr)
		if lineStr == "" {
			continue
		}
		fileContent = append(fileContent, lineStr)
	}
	return fileContent, nil
}

// 从文件中获取可用IP地址清单
// ipAddresslist.example.txt 内容如下：
//192.168.100.200-192.168.100.204,#192.168.100.204,192.168.100.272,192.168.100.203,192.168.100.204,192.168.100.200"
//192.168.100.200-192.168.100.204"
//192.168.100.208"
//
func GetAvailableIPFromFile(strFilePath string) ([]string, error) {
	var availableIPs []string
	strContent, err := GetFileContent(strFilePath)
	if err != nil {
		return availableIPs, err
	}
	if len(strContent) == 0 {
		return availableIPs, errors.New("ERROR: Nothing found in '" + strFilePath + "' please check your input file content!")
	}
	for _, strIps := range strContent {
		ips, err := GetAvailableIP(strIps)
		if err != nil {
			continue
		}
		availableIPs = append(availableIPs, ips...)
	}
	if len(availableIPs) == 0 {
		return availableIPs, errors.New("ERROR: no valid IP Address found, please check your input file, '" + strFilePath + "'")
	}
	retIPs, err := DuplicateIPAddressCheck(availableIPs)
	if err != nil {
		return availableIPs, err
	}
	return retIPs, nil
}

// 检测解析后的IP地址清单中是否包含重复IP地址，并接收用户输入以确定是否移除这些重复IP地址
func DuplicateIPAddressCheck(ips []string) ([]string, error) {
	sort.Strings(ips)
	if len(ips) > len(Duplicate(ips)) {
	LabelConfirm:
		ok, err := Confirm("Duplicate IP Address Found, input 'yes or y' to remove duplicate IP Address, Input 'no or n' will keep duplicate IP Address,\nnothing will do by default! (y/n) ")
		if err != nil {
			goto LabelConfirm
		}
		if ok {
			// remove duplicate IP Address
			sort.Strings(ips)
			retAvailableIPs, err := DuplicateToStringSlice(ips)
			if err != nil {
				return retAvailableIPs, err
			}
			return retAvailableIPs, nil
		}
		// else did nothing keep duplicate IP Address, and return
		return ips, nil
	}
	return ips, nil
}

//从多行文本获取可用IP地址
//多行IP，通常从配置文件或IP地址清单文件中解析，比如：
//输入：
// `
//192.168.100.200-192.168.100.204,#192.168.100.204,192.168.100.272,192.168.100.203,192.168.100.204,192.168.100.200"
//192.168.100.200-192.168.100.204"
//192.168.100.208"
// `
func GetAvailableIPFromMultiLines(multiLines string) ([]string, error) {
	var availableIPs []string
	multiLines = strings.TrimSpace(multiLines)
	if len(multiLines) == 0 {
		return availableIPs, fmt.Errorf("ERROR: empty text, no valid IP Address found")
	}

	ipLists := strings.Split(multiLines, "\n")
	for _, strIps := range ipLists {
		ips, err := GetAvailableIP(strIps)
		if err != nil {
			continue
		}
		availableIPs = append(availableIPs, ips...)
	}
	if len(availableIPs) == 0 {
		return availableIPs, fmt.Errorf("ERROR: no valid IP Address found, please check your input")
	}
	return availableIPs, nil
}

// 支持从如下IP地址标示形式获取可用IP地址清单，比如：
// GetAvailableIPFromSingleIP 支持解析单个IP地址：192.168.100.100
// GetAvailableIPRangeWithDelimiter 支持解析包含分隔符范围的IP地址段：192.168.100.100-105,192, 192.168.100.106-192.168.100.108
// GetAvailableIPWithMask 支持解析包含子网掩码的IP地址段：192.168.100.100/28, 192.168.100.106/255.255.255.240
func GetAvailableIPList(strIP string) ([]string, error) {
	var availableIPs []string
	strIP = strings.TrimSpace(strIP)
	if !strings.HasPrefix(strIP, "#") {
		// 如果IP地址钱包含# 默认跳过该条目，代表注释
		if ips, err := GetAvailableIPFromSingleIP(strIP); err == nil {
			availableIPs = append(availableIPs, ips...)
		} else if ips, err := GetAvailableIPRangeWithDelimiter(strIP, "-"); err == nil {
			availableIPs = append(availableIPs, ips...)
		} else if ips, err := GetAvailableIPWithMask(strIP); err == nil {
			availableIPs = append(availableIPs, ips...)
		} else {
			return availableIPs, fmt.Errorf("ERROR: no valid IP Address found, please check")
		}
	}

	return availableIPs, nil
}

// 使用分隔符 获取可用IP地址范围，输出可用IP地址切片，
// 比如：192.168.1.100-192.168.1.103  返回[192.168.1.100 192.168.1.101 192.168.1.102 192.168.1.103]
// 比如：192.168.1.100-102  返回[192.168.1.100 192.168.1.101 192.168.1.102]
func GetAvailableIPRangeWithDelimiter(strIPRanges string, strDelimiter string) ([]string, error) {
	var availableIPs []string
	if strDelimiter == "." || strDelimiter == ":" || strDelimiter == "" {
		return availableIPs, errors.New("ERROR: strings like '.' or ':' or space  con't used for split a IP Address strings")
	}
	if !strings.Contains(strIPRanges, strDelimiter) {
		return availableIPs, errors.New("ERROR: can't find " + strDelimiter + "' in '" + strIPRanges + "' please check!")
	}
	strIPlist := strings.Split(strIPRanges, strDelimiter)
	startIP := strings.TrimSpace(strIPlist[0])
	endIP := strings.TrimSpace(strIPlist[1])
	if CheckIp(startIP) == false {
		return availableIPs, errors.New("ERROR: Start IP Address is not a valid IP Address range strings, e.g. 192.168.1.100-192.168.1.110")
	}
	_, startIPPrefix, startIPNo := GetIPAddressPrefixAndEndNo(startIP)
	var endIPPrefix []string
	var endIPNo int
	if CheckIp(endIP) == false {
		if v, ok := strconv.Atoi(endIP); ok != nil {
			return availableIPs, errors.New("ERROR: END IP Address is not a valid IP Address range strings, e.g. 192.168.1.100-192.168.1.110")
		} else {
			endIPNo = v
			endIPPrefix = startIPPrefix
		}
	} else {
		_, p, n := GetIPAddressPrefixAndEndNo(endIP)
		endIPPrefix = p
		endIPNo = n
	}
	// 检测 起始IP地址和终止IP地址的前三位是否相同
	if startIPPrefix[0] != endIPPrefix[0] || startIPPrefix[1] != endIPPrefix[1] || startIPPrefix[2] != endIPPrefix[2] {
		return availableIPs, errors.New("ERROR: the Start IP Address and END IP Address first three section are not same, Please confirm!, e.g. 192.168.1.100-192.168.1.110")
	}
	flag := endIPNo - startIPNo
	switch {
	case flag < 0:
		return availableIPs, errors.New("ERROR: the End IP Address must bigger than the Start IP Address, Please confirm!, e.g. 192.168.1.100-192.168.1.110")
	case flag == 0:
		availableIPs = append(availableIPs, startIP)
		return availableIPs, nil
	case flag > 0:
		for i := 0; i <= flag; i++ {
			ips := startIPPrefix
			ips = append(ips, strconv.Itoa(i+startIPNo))
			newIP := strings.Join(ips, ".")
			availableIPs = append(availableIPs, newIP)
		}
	}
	return availableIPs, nil
}

// 给定一个IP地址 返回该IP地址的前3位切片，并返回该IP地址的末尾
func GetIPAddressPrefixAndEndNo(strIP string) (bool, []string, int) {
	var strIPAddressPrefix []string
	strIP = strings.TrimSpace(strIP)
	if CheckIp(strIP) == false {
		return false, strIPAddressPrefix, 0
	}
	iplist := strings.Split(strIP, ".")
	strIPAddressPrefix = iplist[0 : len(iplist)-1]
	ipAddressEndNo, _ := strconv.Atoi(iplist[len(iplist)-1])
	return true, strIPAddressPrefix, ipAddressEndNo
}

// 判断一个字符串是否为IP地址，并返回可用IP地址切片
func GetAvailableIPFromSingleIP(ipAddress string) ([]string, error) {
	var availableIPs []string
	ipAddress = strings.TrimSpace(ipAddress)
	if !CheckIp(ipAddress) {
		return availableIPs, errors.New("ERROR: '" + ipAddress + "' is not a valid IP Address, please confirm!>>>")
	}
	availableIPs = append(availableIPs, ipAddress)
	return availableIPs, nil
}

// 引用函数
// 将ip/mask形式 统一转换为ip/cidrmask形式
// 比如：输入  192.168.1.100 输出 192.168.1.100/32
// 比如：输入  192.168.1.100/24 输出 192.168.1.100/24
// 比如：输入  192.168.1.100/255.255.255.0 输出 192.168.1.100/24
func IPAddressToCIDR(ipAddress string) (string, error) {
	ipAddress = strings.TrimSpace(ipAddress)
	if strings.Contains(ipAddress, "/") == true {
		ipAndMask := strings.Split(ipAddress, "/")
		ip := ipAndMask[0]
		if CheckIp(ip) == false {
			return "", errors.New("ERROR: '" + ip + "' is not a valid IP Address, please confirm!!!")
		}
		mask := ipAndMask[1]

		if strings.Contains(mask, ".") == true {
			ok, cidrMask := IPMaskToCIDRMask(mask)
			if !ok {
				return "", errors.New(cidrMask)
			}
			mask = cidrMask
		} else {
			intMask, err := strconv.Atoi(mask)
			if err != nil {
				return "", errors.New("ERROR: '" + mask + "' is not a valid network mask, please confirm!!!")
			}
			var cidrMaskNos = map[int]int{24: 24, 25: 25, 26: 26, 27: 27, 28: 28, 29: 29, 30: 30, 31: 31, 32: 32}
			if _, ok := cidrMaskNos[intMask]; !ok {
				return "", errors.New("ERROR: '" + mask + "' is not a valid network mask,for CIDR form masks, valid mask number should be one of [24,25,26,27,28,29,30,31,32] please confirm!")
			}
			mask = strconv.Itoa(intMask)
		}
		return ip + "/" + mask, nil
	} else {
		if net.ParseIP(ipAddress) == nil {
			return "", errors.New("ERROR: '" + ipAddress + "' is not a valid IP Address, please check!")
		}
		return fmt.Sprintf("%s/%d", ipAddress, 32), nil
	}
}

func GetAvailableIPWithMask(ipAndMask string) ([]string, error) {
	var availableIPs []string

	ipAndMask = strings.TrimSpace(ipAndMask)
	ipAndCIDRMask, err := IPAddressToCIDR(ipAndMask)
	if err != nil {
		return availableIPs, err
	}
	_, ipNet, _ := net.ParseCIDR(ipAndCIDRMask)

	firstIP, _ := networkRange(ipNet)
	ipNum := ipToInt(firstIP)
	size := networkSize(ipNet.Mask)
	pos := int32(1)
	max := size - 2 // -1 for the broadcast address, -1 for the gateway address

	var newNum int32
	for attempt := int32(0); attempt < max; attempt++ {
		newNum = ipNum + pos
		pos = pos%max + 1
		availableIPs = append(availableIPs, intToIP(newNum).String())
	}
	return availableIPs, nil
}

// Calculates the first and last IP addresses in an IPNet
func networkRange(network *net.IPNet) (net.IP, net.IP) {
	netIP := network.IP.To4()
	firstIP := netIP.Mask(network.Mask)
	lastIP := net.IPv4(0, 0, 0, 0).To4()
	for i := 0; i < len(lastIP); i++ {
		lastIP[i] = netIP[i] | ^network.Mask[i]
	}
	return firstIP, lastIP
}

// Given a netmask, calculates the number of available hosts
func networkSize(mask net.IPMask) int32 {
	m := net.IPv4Mask(0, 0, 0, 0)
	for i := 0; i < net.IPv4len; i++ {
		m[i] = ^mask[i]
	}
	return int32(binary.BigEndian.Uint32(m)) + 1
}

// Converts a 4 bytes IP into a 32 bit integer
func ipToInt(ip net.IP) int32 {
	return int32(binary.BigEndian.Uint32(ip.To4()))
}

// Converts 32 bit integer into a 4 bytes IP address
func intToIP(n int32) net.IP {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(n))
	return net.IP(b)
}

// go语言实现去重，可以接受任何类型
func Duplicate(a interface{}) (ret []interface{}) {
	va := reflect.ValueOf(a)
	for i := 0; i < va.Len(); i++ {
		if i > 0 && reflect.DeepEqual(va.Index(i-1).Interface(), va.Index(i).Interface()) {
			continue
		}
		ret = append(ret, va.Index(i).Interface())
	}
	return ret
}

//如果确定接收的是字符串类型的接口，去除字符串切片中的重复元素
func DuplicateToStringSlice(fromInterface interface{}) ([]string, error) {
	var str []string
	ret := Duplicate(fromInterface)

	for _, v := range ret {
		t := reflect.TypeOf(v)
		if t.String() == "string" {
			str = append(str, reflect.ValueOf(v).String())
		} else {
			return str, fmt.Errorf("ERROR: Interface Type convert to Strings Type Failed")
		}
	}
	return str, nil
}

// Get difference items between two slices
func DiffStringSlices(slice1 []string, slice2 []string) []string {
	var diffStr []string
	m := map[string]int{}
	for _, s1Val := range slice1 {
		m[s1Val] = 1
	}
	for _, s2Val := range slice2 {
		m[s2Val] = m[s2Val] + 1
	}
	for mKey, mVal := range m {
		if mVal == 1 {
			diffStr = append(diffStr, mKey)
		}
	}

	return diffStr
}

// 接受用户输入，确认是否继续下一步操作
func Confirm(str string) (bool, error) {
	var isTrue string
	fmt.Printf(str)
	fmt.Scanln(&isTrue)
	trueOrFalse, err := ParseBool(isTrue)
	if err != nil {
		return false, err
	}
	return trueOrFalse, nil
}

//从用户输入内容中解析 布尔值 true or false
func ParseBool(str string) (value bool, err error) {
	switch str {
	case "1", "t", "T", "true", "TRUE", "True", "YES", "yes", "Yes", "y", "ON", "on", "On":
		return true, nil
	case "0", "f", "F", "false", "FALSE", "False", "NO", "no", "No", "n", "OFF", "off", "Off":
		return false, nil
	}
	return false, fmt.Errorf("Parsing ERROR:  \"%s\"  can't convert to 'true' or 'false'", str)
}

// 控制台输出颜色控制，兼容Windows & Linux
func ColorPrint(logLevel string, textBefore interface{}, colorText string, textAfter ...interface{}) {
	color := ct.None
	switch logLevel {
	case "INFO":
		color = ct.Green
	case "WARNING":
		color = ct.Yellow
	case "ERROR":
		color = ct.Red
	}
	fmt.Printf("%s", textBefore)
	ct.Foreground(color, true)
	fmt.Printf("%s", colorText)
	ct.ResetColor()
	for _, v := range textAfter {
		fmt.Printf("%s", v)
	}
}

//  根据默认config.ini
func Cfg(iniFilePath string) (*ini.File, error) {
	var cf *ini.File
	if _, err := GetRealPath(iniFilePath); err != nil {
		return cf, fmt.Errorf("%s not exist! please check", iniFilePath)
	}
	isConfigINIExist, _ := CheckDefaultINIFile("config.ini")
	if !isConfigINIExist {
		fmt.Println("Default config.ini not exist")
		return cf, fmt.Errorf("default config.ini not exist, please confirm")
	}
	iniFile, _ := filepath.Abs(iniFilePath)
	cfg, err := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, iniFile)
	cfg.BlockMode = false

	if err != nil {
		return cf, fmt.Errorf("failed to read config file,please check:%v", err)
	}
	return cfg, nil
}

func PrintResultInTable(headers []string, data [][]string, maxTableCellWidth int) {
	tabulate := gotabulate.Create(data)
	tabulate.SetHeaders(headers)
	tabulate.SetAlign("center")
	tabulate.SetMaxCellSize(maxTableCellWidth)
	tabulate.SetWrapStrings(true)
	fmt.Println(tabulate.Render("grid"))
}

func PrintListHosts(hosts []string, maxTableCellWidth int, groupName ...string) {
	var headers []string
	var data [][]string
	headers = append(headers, "#", "Host")

	if len(groupName) != 0 {
		headers = append(headers, "Group Name")
	}
	todoHosts, err := DuplicateIPAddressCheck(hosts)
	if err != nil {
		fmt.Println(err)
		return
	}
	for i, v := range todoHosts {
		if len(groupName) != 0 {
			data = append(data, []string{strconv.Itoa(i + 1), v, groupName[0]})
		} else {
			data = append(data, []string{strconv.Itoa(i + 1), v})
		}
	}
	ColorPrint("INFO", "", ">>> Available Hosts", ":\n")
	PrintResultInTable(headers, data, maxTableCellWidth)
}

// format result with table style, supports output of the contents of the specified column
func FormatResultWithTableStyle(res []interface{}, maxTableCellWidth int, notIncludedFields []string) {
	var header = []string{"#"}
	var headerAll, headerNotInclude, headerInclude []string
	var data [][]string
	if len(res) > 0 {
		iRes := res[0]
		typeName := reflect.TypeOf(iRes)
		if len(notIncludedFields) > 0 {
			for i := 0; i < typeName.NumField(); i++ {
				for _, f := range notIncludedFields {
					if len(f) == 0 {
						continue
					}
					if f == typeName.Field(i).Name {
						headerNotInclude = append(headerNotInclude, typeName.Field(i).Name)
					}
				}
				headerAll = append(headerAll, typeName.Field(i).Name)
				headerInclude = DiffStringSlices(headerNotInclude, headerAll)
			}
			header = append(header, headerInclude...)
		} else {
			for i := 0; i < typeName.NumField(); i++ {
				header = append(header, typeName.Field(i).Name)
			}
		}
	}
	for i, v := range res {
		value := reflect.ValueOf(v)
		typeName := reflect.TypeOf(v)
		var row []string
		index := strconv.Itoa(i + 1)
		row = append(row, index)

		for i := 0; i < value.NumField(); i++ {
			if len(notIncludedFields) > 0 {
				var r []string
				for _, f := range headerInclude {
					if f == typeName.Field(i).Name {
						r = append(r, value.Field(i).String())
					} else {
						continue
					}
				}
				row = append(row, r...)
			} else {
				row = append(row, value.Field(i).String())
			}
		}
		data = append(data, row)
	}
	PrintResultInTable(header, data, maxTableCellWidth)
}

func FormatResultWithBasicStyle(i int, res SSHResult) {
	ColorPrint("INFO", "", ">>> ", fmt.Sprintf("No.%d, ", i+1))
	ColorPrint("INFO", "", "Host:", fmt.Sprintf("%s,", res.Host))
	if res.Status == "success" {
		ColorPrint("INFO", " Status:", fmt.Sprintf("%s", res.Status))
	} else {
		ColorPrint("ERROR", " Status:", fmt.Sprintf("%s", res.Status))
	}
	ColorPrint("INFO", ", Results:\n", "", fmt.Sprintf("%s\n\n", res.Result))
}

func LogSSHResultToFile(i int, res SSHResult, filePath string) {
	WriteAndAppendFile(filePath, fmt.Sprintf(">>> No.%d, Host: %s, Status: %s", i+1, res.Host, res.Status))
	WriteAndAppendFile(filePath, fmt.Sprintf("Result: %s", res.Result))
}

func SFTPFormatResultWithBasicStyle(i int, res SFTPResult) {
	ColorPrint("INFO", "", ">>> ", fmt.Sprintf("No.%d, ", i+1))
	ColorPrint("INFO", "", "Host:", fmt.Sprintf("%s,", res.Host))
	if res.Status == "success" {
		ColorPrint("INFO", " Status:", fmt.Sprintf("%s", res.Status))
	} else {
		ColorPrint("ERROR", " Status:", fmt.Sprintf("%s", res.Status))
	}
	ColorPrint("INFO", "", ", Source Path:", fmt.Sprintf("%s,", res.SourcePath))
	ColorPrint("INFO", "", " Destination Path:", fmt.Sprintf("%s,", res.DestinationPath))
	ColorPrint("INFO", " Results:\n", "", fmt.Sprintf("%s\n\n", res.Result))
}
func LogSFTPResultToFile(i int, res SFTPResult, filePath string) {
	WriteAndAppendFile(filePath, fmt.Sprintf(">>> No.%d, Host: %s, Status: %s", i+1, res.Host, res.Status))
	WriteAndAppendFile(filePath, fmt.Sprintf("Source Path: %s, Destination Path: %s", res.SourcePath, res.DestinationPath))
	WriteAndAppendFile(filePath, fmt.Sprintf("Result: %s", res.Result))
}

// =============================================================
// ResultLog format functions
func ResultLogInfo(resultLog ResultLogs, startTime time.Time, logToFile bool, logFilePath string) {
	endTime := time.Now()
	resultLog.StartTime = startTime.Format("2006-01-02 15:04:05")
	resultLog.EndTime = endTime.Format("2006-01-02 15:04:05")
	resultLog.CostTime = endTime.Sub(startTime).String()
	resultLog.TotalHostsInfo = fmt.Sprintf("%d(Success) + %d(Failed) = %d(Total)", len(resultLog.SuccessHosts), len(resultLog.ErrorHosts), len(resultLog.SuccessHosts)+len(resultLog.ErrorHosts))
	if logToFile {
		WriteAndAppendFile(logFilePath, fmt.Sprintf("Tips: process running done."))
		WriteAndAppendFile(logFilePath, fmt.Sprintf("\nStart Time: %s\nEnd Time: %s\nCost Time: %s\nTotal Hosts Running: %s\n", resultLog.StartTime, resultLog.EndTime, resultLog.CostTime, resultLog.TotalHostsInfo))
	} else {
		ColorPrint("INFO", "", "Tips: ", fmt.Sprintf("Process running done.\n"))
		fmt.Printf("Start Time: %s\nEnd Time: %s\nCost Time: %s\nTotal Hosts Running: %s\n", resultLog.StartTime, resultLog.EndTime, resultLog.CostTime, resultLog.TotalHostsInfo)
	}
}
func FormatResultLogWithSimpleStyle(resultLog ResultLogs, startTime time.Time, maxTableCellWidth int, notIncludeTableFields []string) {
	if len(resultLog.ErrorHosts) > 0 {
		ColorPrint("ERROR", "", "WARNING: ", "Failed hosts, please confirm!\n")
		FormatResultWithTableStyle(resultLog.ErrorHosts, maxTableCellWidth, notIncludeTableFields)
	}
	ResultLogInfo(resultLog, startTime, false, "")
}

// format result log with table style
func FormatResultLogWithTableStyle(chs []chan interface{}, resultLog ResultLogs, startTime time.Time, maxTableCellWidth int) {
	resultStatus := ""
	for _, resCh := range chs {
		result := <-resCh
		switch reflect.TypeOf(result).String() {
		case "utils.SSHResult":
			resultStatus = result.(SSHResult).Status
		case "utils.SFTPResult":
			resultStatus = result.(SFTPResult).Status
		}
		if resultStatus == "failed" {
			resultLog.ErrorHosts = append(resultLog.ErrorHosts, result)

		} else {
			resultLog.SuccessHosts = append(resultLog.SuccessHosts, result)
		}
	}

	if len(resultLog.SuccessHosts) > 0 {
		ColorPrint("INFO", "", "INFO: ", "Success hosts\n")
		FormatResultWithTableStyle(resultLog.SuccessHosts, maxTableCellWidth, []string{})
	}
	if len(resultLog.ErrorHosts) > 0 {
		ColorPrint("ERROR", "", "WARNING: ", "Failed hosts, please confirm!\n")
		FormatResultWithTableStyle(resultLog.ErrorHosts, maxTableCellWidth, []string{})
	}
	ResultLogInfo(resultLog, startTime, false, "")
}

func GetAllResultLog(chs []chan interface{}, resultLog ResultLogs, startTime time.Time) ResultLogs {
	resultStatus := ""
	for _, resCh := range chs {
		result := <-resCh
		switch reflect.TypeOf(result).String() {
		case "utils.SSHResult":
			resultStatus = result.(SSHResult).Status
		case "utils.SFTPResult":
			resultStatus = result.(SFTPResult).Status
		}
		if resultStatus == "failed" {
			resultLog.ErrorHosts = append(resultLog.ErrorHosts, result)

		} else {
			resultLog.SuccessHosts = append(resultLog.SuccessHosts, result)
		}
	}
	endTime := time.Now()
	resultLog.StartTime = startTime.Format("2006-01-02 15:04:05")
	resultLog.EndTime = endTime.Format("2006-01-02 15:04:05")
	resultLog.CostTime = endTime.Sub(startTime).String()
	resultLog.TotalHostsInfo = fmt.Sprintf("%d(Success) + %d(Failed) = %d(Total)", len(resultLog.SuccessHosts), len(resultLog.ErrorHosts), len(resultLog.SuccessHosts)+len(resultLog.ErrorHosts))

	return resultLog
}

// format result log with json style
func FormatResultToJson(logs []ResultLogs, isJsonRaw bool) {
	b, err := json.Marshal(logs)
	if err != nil {
		fmt.Println("json err:", err)
		return
	}
	if isJsonRaw != false {
		fmt.Println(string(b))
		return
	}
	var out bytes.Buffer
	err = json.Indent(&out, b, "", "    ")
	if err != nil {
		fmt.Println("Json Format ERROR:", err)
	}
	out.WriteTo(os.Stdout)
	return
}
func FormatResultLogWithJsonStyle(chs []chan interface{}, resultLog ResultLogs, startTime time.Time, isJsonRaw bool) {
	var allResultLog []ResultLogs
	resLog := GetAllResultLog(chs, resultLog, startTime)
	allResultLog = append(allResultLog, resLog)
	FormatResultToJson(allResultLog, isJsonRaw)
	return
}

func WriteAndAppendFile(filePath, strContent string) {
	strTime := GetCurrentDateNumbers
	if filePath == "log" {
		filePath = fmt.Sprintf("ssgo-%s.log", strTime)
	}
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("ERROR: write file failed:", err)
		return
	}
	appendTime := time.Now().Format("2006-01-02 15:04:05")
	fileContent := strings.Join([]string{appendTime, strContent, "\n"}, " ")
	buf := []byte(fileContent)
	f.Write(buf)
	f.Close()
}

func GetCurrentDateNumbers() (strTime string) {
	currTime := time.Now()
	strFormatTime := currTime.Format("2006-01-02")
	strTime = strings.Replace(strFormatTime, "-", "", -1)
	return
}
