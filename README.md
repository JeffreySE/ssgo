### 1. ssgo
### 2. 中文介绍
ssgo是一个基于SSH协议开发的小工具，面向系统管理员，主要用于在远程主机上执行命令、脚本或传输文件
### 3. 小特性
* 默认并发执行
* 支持单条、多条命令、脚本执行（直接在远程主机执行本地脚本，可以接受脚本参数）
* 对于复杂场景，可以像Ansible那样指定一个仓库主机清单文件，包含主机组，对应主机组的登录用户、密码、端口
* 支持指定主机清单文件（一个包含主机IP地址的文件）
* 支持格式化输出结果，目前支持简单样式（默认）和表格格式、json格式
* 可以在某一个命令后指定`--example`获取该命令的使用案例
* 输出结果包含颜色，兼容Linux & Windows平台
* 可用IP地址支持以下形式：
    * 单个IP地址：`192.168.100.2`
    * IP地址段：`192.168.100.2-5` 或 `192.168.100.2-192.168.100.5` 都是可以的
    * 包含子网掩码的IP地址范围：`192.168.100.0/28`或`192.168.100.0/255.255.255.240` 都是可以的
    * 组合形式：`192.168.100.2,192.168.100.3-192.168.100.5,192.168.100.0/29` 只要以英文逗号分隔开即可
* 重复主机检测：当ssgo执行操作时，会从主机清单中检测重复IP地址的存在，防止在主机上进行重复操作
* 支持输出命令执行结果到日志文件

### 4. 一点背景
作为运维人员经常需要和远程主机（好吧 有时候是一大堆）打交道，执行脚本、执行命令、传输文件是三个最基本的诉求了，前期有接触过Ansible 使用已经很方便了，但是不够轻量；也使用Python基于paramiko库编写过小轮子，无奈有些虚机、系统上竟然没有预装paramiko组件，更可恨的是还要解决一堆依赖关系，对系统进行修改。。。    
为了解决这些痛点，重新使用Go语言造了一个轮子 基本满足这些需求，初学Go语言，开发过程中学到了很多，当然啦，代码比较烂，请见谅   
灵感来源：[https://github.com/shanghai-edu/multissh](https://github.com/shanghai-edu/multissh)非常感谢

### 5. 使用帮助
``` bash
λ ssgo.exe -h
usage: ssgo [<flags>] <command> [<args> ...]

A SSH-based command line tool for operating remote hosts.

Flags:
  -h, --help                  Show context-sensitive help (also try --help-long
                              and --help-man).
  -e, --example               Show examples of ssgo's command.
  -i, --inventory=INVENTORY   For advanced use case, you can specify a host
                              warehouse .ini file (Default is 'config.ini' file
                              in current directory.)
  -g, --group=GROUP           Remote host group name in the inventory file,
                              which must be used with '-i' or '--inventory'
                              argument!
      --host-file=HOST-FILE   A file contains remote host or host range IP
                              Address.(e.g. 'hosts.example.txt' in current
                              directory.)
      --host-list=HOST-LIST   Remote host or host range IP Address. e.g.
                              192.168.10.100,192.168.10.101-192.168.10.103,192.168.20.100/28,192.168.30.11-15
  -p, --pass=PASS             The SSH login password for remote hosts.
  -u, --user="root"           The SSH login user for remote hosts. default is
                              'root'
  -P, --port=22               The SSH login port for remote hosts. default is
                              '22'
  -n, --maxExecuteNum=20      Set Maximum concurrent count of hosts.
  -o, --output=OUTPUT         Output result'log to a file.(Be default if your
                              input is "log",ssgo will output logs like
                              "ssgo-%s.log")
  -F, --format="simple"       For pretty look in terminal,you can format the
                              result with table,simple,json or other
                              style.(Default is simple)
      --json-raw              By default, the json data will be formatted and
                              output by the console. You can specify the
                              --json-raw parameter to output raw json
                              data.(Default is false)
  -w, --maxTableCellWidth=40  For pretty look,you can set the printed table's
                              max cell width in terminal.(Default is 40)
  -v, --version               Show application version.

Commands:
  help [<command>...]
    Show help.

  list
    List available remote hosts from your input.

  run [<flags>]
    Run commands on remote hosts.

  copy --action=ACTION --src=SRC [<flags>]
    Transfer files between local machine and remote hosts.
```
### 6. 核心组件
* **ssgo list**   
   用于从用户输入的参数中获取可用IP地址清单（好吧，这个功能比较鸡肋）
* **ssgo run**   
   用于在远程主机上执行命令或脚本
* **ssgo copy**   
   用于在本地和远程主机之间传输文件
   

### 7. 文件示例
**config.ini文件**
``` bash
# ssgo tools use ini config file for advanced usage
# Important Tips： you can't use 'all' as a host group name, cause 'all' will be identified as all host in your config.ini file.

[dc]
user = root
pass = root
port = 22
hosts = 192.168.100.1

[web]
user = root
pass = root
port = 22
hosts = 192.168.100.2,192.168.100.3-192.168.100.4,192.168.100.8

[db]
user = root
pass = root
port = 22
hosts = 192.168.100.5-6

[docker]
user = root
pass = root
port = 22
hosts = """
192.168.100.7
192.168.100.9
192.168.100.10
192.168.100.1-192.168.100.3
"""
```
**host-file.example.txt文件**   
**备注**：如果某一个IP地址开头包含了“#”ssgo默认会忽略它

``` text
192.168.100.1,192.168.100.2-192.168.100.4
#192.168.100.5  #host-list strings with "#" prefix will be ignored!
192.168.100.6
192.168.100.7-10
```
**cmd-file.example.txt文件**   
**备注**：如果该文件内某一行命令执行失败，在该主机上后续命令不会继续被执行

``` text
echo -e "***************************************"
hostname
ssh -V
df -Th
cd /opt/
ls -l
cat /etc/rsyslog.conf  | grep "#kern.*"
cat /etc/passwd | awk -F ':' '{print $1}'
echo -e "***************************************"
```
**bash.sh文件**   
**备注**：可以在远程主机直接执行本地Shell脚本，也可以接受脚本参数
``` bash
#!/bin/bash

echo "HostName:$(hostname)"
echo "I am a test Shell script running on the remote server!"
echo "Script Args \$1: $1"
echo "Script Args \$2: $2"
echo "What happens if an exception occurs during script execution?"
ls ThisFileIsNotExist
```

### 8. 使用示例
#### 8.1. ssgo run 执行命令
##### 8.1.1. `--host-list` 参数相关
**`ssgo run --host-list`单个命令**
``` bash
➜ ./ssgo run --host-list 192.168.100.1 -u root -p root -c "hostname"

Tips:Process running start: 2018-04-20 17:13:25
>>> No.1, Host:192.168.100.1, Status:success, Results:
42f432e85ab6

Tips: Process running done.
End Time: 2018-04-20 17:13:25
Cost Time: 128.218018ms
Total Hosts Running: 1(Success) + 0(Failed) = 1(Total)
```

**`ssgo run --host-list` 多个主机 执行单个命令，以表格样式输出结果**   
**备注** ： -F, --format 格式化命令执行后的输出结果
``` bash
➜ ./ssgo run --host-list 192.168.100.1,192.168.100.2-4 -u root -p root -c "hostname" -F table

Tips:Process running start: 2018-04-20 17:15:11
INFO: Success hosts
+---+---------------+---------+--------------+
| # |      Host     |  Status |    Result    |
+---+---------------+---------+--------------+
| 1 | 192.168.100.1 | success | 42f432e85ab6 |
+---+---------------+---------+--------------+
| 2 | 192.168.100.2 | success | 39beb225f669 |
+---+---------------+---------+--------------+
| 3 | 192.168.100.3 | success | dfc9aed2f3ce |
+---+---------------+---------+--------------+
| 4 | 192.168.100.4 | success | 8080c8c88026 |
+---+---------------+---------+--------------+

Tips: Process running done.
End Time: 2018-04-20 17:15:11
Cost Time: 129.434302ms
Total Hosts Running: 4(Success) + 0(Failed) = 4(Total)
```

**`ssgo run --host-list` 多个主机 执行多个命令**   
**备注**：执行多个命令时，就不建议使用表格样式输出结果了，那样会不好看的，你懂的
``` bash
➜ ./ssgo run --host-list 192.168.100.1,192.168.100.2-4 -u root -p root -c "hostname;pwd;date"

Tips:Process running start: 2018-04-20 17:17:33
>>> No.1, Host:192.168.100.1, Status:success, Results:
42f432e85ab6
/root
Fri Apr 20 09:17:34 UTC 2018

>>> No.2, Host:192.168.100.2, Status:success, Results:
39beb225f669
/root
Fri Apr 20 09:17:34 UTC 2018

>>> No.3, Host:192.168.100.3, Status:success, Results:
dfc9aed2f3ce
/root
Fri Apr 20 09:17:34 UTC 2018

>>> No.4, Host:192.168.100.4, Status:success, Results:
8080c8c88026
/root
Fri Apr 20 09:17:34 UTC 2018

Tips: Process running done.
End Time: 2018-04-20 17:17:34
Cost Time: 301.24909ms
Total Hosts Running: 4(Success) + 0(Failed) = 4(Total)
```

##### 8.1.2. ` --host-file` 参数相关
**备注**：`--host-file` 参数只需指定一个主机清单文件即可，表格样式会比较容易比对输出结果，对吧

``` bash
➜ ./ssgo run --host-file host-file.example.txt  -u root -p root -c "date" -F table
Tips:Process running start: 2018-04-20 17:21:15
INFO: Success hosts
+---+----------------+---------+------------------------------+
| # |      Host      |  Status |            Result            |
+---+----------------+---------+------------------------------+
| 1 |  192.168.100.1 | success | Fri Apr 20 09:21:15 UTC 2018 |
+---+----------------+---------+------------------------------+
| 2 | 192.168.100.10 | success | Fri Apr 20 09:21:15 UTC 2018 |
+---+----------------+---------+------------------------------+
| 3 |  192.168.100.2 | success | Fri Apr 20 09:21:15 UTC 2018 |
+---+----------------+---------+------------------------------+
| 4 |  192.168.100.3 | success | Fri Apr 20 09:21:15 UTC 2018 |
+---+----------------+---------+------------------------------+
| 5 |  192.168.100.4 | success | Fri Apr 20 09:21:15 UTC 2018 |
+---+----------------+---------+------------------------------+
| 6 |  192.168.100.6 | success | Fri Apr 20 09:21:15 UTC 2018 |
+---+----------------+---------+------------------------------+
| 7 |  192.168.100.7 | success | Fri Apr 20 09:21:15 UTC 2018 |
+---+----------------+---------+------------------------------+
| 8 |  192.168.100.8 | success | Fri Apr 20 09:21:15 UTC 2018 |
+---+----------------+---------+------------------------------+
| 9 |  192.168.100.9 | success | Fri Apr 20 09:21:15 UTC 2018 |
+---+----------------+---------+------------------------------+

Tips: Process running done.
End Time: 2018-04-20 17:21:15
Cost Time: 172.372373ms
Total Hosts Running: 9(Success) + 0(Failed) = 9(Total)
```

##### 8.1.3. `-i, --inventory`和 `-g, --group`参数相关
**备注**：`-i, --inventory`和 `-g, --group`参数需要组合使用，`-i`指定config.ini主机仓库文件，`-g`指定主机组名称，如果是`-g`的参数为`all`,则该主机仓库中所有的主机会被识别，用来执行操作
``` bash
➜ ./ssgo run -i config.ini -g docker -c "date" -F table


Tips:Process running start: 2018-04-20 17:25:21
INFO: Success hosts
+---+----------------+---------+------------------------------+
| # |      Host      |  Status |            Result            |
+---+----------------+---------+------------------------------+
| 1 |  192.168.100.1 | success | Fri Apr 20 09:25:21 UTC 2018 |
+---+----------------+---------+------------------------------+
| 2 | 192.168.100.10 | success | Fri Apr 20 09:25:21 UTC 2018 |
+---+----------------+---------+------------------------------+
| 3 |  192.168.100.2 | success | Fri Apr 20 09:25:21 UTC 2018 |
+---+----------------+---------+------------------------------+
| 4 |  192.168.100.3 | success | Fri Apr 20 09:25:21 UTC 2018 |
+---+----------------+---------+------------------------------+
| 5 |  192.168.100.7 | success | Fri Apr 20 09:25:21 UTC 2018 |
+---+----------------+---------+------------------------------+
| 6 |  192.168.100.9 | success | Fri Apr 20 09:25:21 UTC 2018 |
+---+----------------+---------+------------------------------+

Tips: Process running done.
End Time: 2018-04-20 17:25:21
Cost Time: 134.046557ms
Total Hosts Running: 6(Success) + 0(Failed) = 6(Total)
```

#### 8.2. 使用ssgo run命令在远程主机执行本地脚本
**备注**：`ssgo run`的`-s,--script`命令用于指定要执行的本地脚本路径，`-a,--args`参数可选，用于指定该脚本的参数，建议执行脚本时，不再指定-F table以表格输出结果，因输出内容包含多余换行标识会导致输出表格内容错乱

``` bash
➜ ./ssgo run -i config.ini -g web -s demo.sh -a "tiger rabbit"
>>> Group Name: [web]
Tips:Process running start: 2018-04-27 14:27:56
>>> No.1, Host:192.168.100.2, Status:failed, Results:
HostName:39beb225f669
I am a test Shell script running on the remote server!
Script Args $1: tiger
Script Args $2: rabbit
What happens if an exception occurs during script execution?
ls: cannot access 'ThisFileIsNotExist': No such file or directory
ERROR: while running script (demo.sh) on host 192.168.100.2, an error occured Process exited with status 2

>>> No.2, Host:192.168.100.3, Status:failed, Results:
HostName:dfc9aed2f3ce
I am a test Shell script running on the remote server!
Script Args $1: tiger
Script Args $2: rabbit
What happens if an exception occurs during script execution?
ls: cannot access 'ThisFileIsNotExist': No such file or directory
ERROR: while running script (demo.sh) on host 192.168.100.3, an error occured Process exited with status 2

>>> No.3, Host:192.168.100.4, Status:failed, Results:
HostName:8080c8c88026
I am a test Shell script running on the remote server!
Script Args $1: tiger
Script Args $2: rabbit
What happens if an exception occurs during script execution?
ls: cannot access 'ThisFileIsNotExist': No such file or directory
ERROR: while running script (demo.sh) on host 192.168.100.4, an error occured Process exited with status 2

>>> No.4, Host:192.168.100.8, Status:failed, Results:
HostName:b1bda3a80a08
I am a test Shell script running on the remote server!
Script Args $1: tiger
Script Args $2: rabbit
What happens if an exception occurs during script execution?
ls: cannot access 'ThisFileIsNotExist': No such file or directory
ERROR: while running script (demo.sh) on host 192.168.100.8, an error occured Process exited with status 2

WARNING: Failed hosts, please confirm!
+---+---------------+--------+
| # |      Host     | Status |
+---+---------------+--------+
| 1 | 192.168.100.2 | failed |
+---+---------------+--------+
| 2 | 192.168.100.3 | failed |
+---+---------------+--------+
| 3 | 192.168.100.4 | failed |
+---+---------------+--------+
| 4 | 192.168.100.8 | failed |
+---+---------------+--------+

Tips: Process running done.
Start Time: 2018-04-27 14:27:56
End Time: 2018-04-27 14:27:56
Cost Time: 701.917888ms
Total Hosts Running: 0(Success) + 4(Failed) = 4(Total)

```

#### 8.3. `ssgo copy`上传文件
**备注**:   

* `ssgo copy`命令下载文件需要制定`-a` 或`--action` 参数为`upload`
* 当进行上传或下载操作时，当`-d, --dst`的参数为`""`空白时，默认文件将会被上传或下载至本地或远程主机的当前工作目录
* 示例：向远程主机192.168.100.1，192.168.100.2，192.168.100.3，192.168.100.4上上传本地demo.sh文件，并已表格返回命令执行结果

``` bash
➜ ./ssgo copy -a upload --host-list 192.168.100.1-4 -u root -p root -s demo.sh -d "/root/" -F table
Tips:Process running start: 2018-04-27 15:51:09
INFO: Success hosts
+---+---------------+---------+------------+-----------------+--------------------+
| # |      Host     |  Status | SourcePath | DestinationPath |       Result       |
+---+---------------+---------+------------+-----------------+--------------------+
| 1 | 192.168.100.1 | success |   demo.sh  |      /root/     | Upload finished!:) |
+---+---------------+---------+------------+-----------------+--------------------+
| 2 | 192.168.100.2 | success |   demo.sh  |      /root/     | Upload finished!:) |
+---+---------------+---------+------------+-----------------+--------------------+
| 3 | 192.168.100.3 | success |   demo.sh  |      /root/     | Upload finished!:) |
+---+---------------+---------+------------+-----------------+--------------------+
| 4 | 192.168.100.4 | success |   demo.sh  |      /root/     | Upload finished!:) |
+---+---------------+---------+------------+-----------------+--------------------+

Tips: Process running done.
Start Time: 2018-04-27 15:51:09
End Time: 2018-04-27 15:51:09
Cost Time: 133.040933ms
Total Hosts Running: 4(Success) + 0(Failed) = 4(Total)

```

#### 8.4. `ssgo copy`下载文件

**备注**:   

* `ssgo copy`命令下载文件需要制定`-a` 或`--action` 参数为`download`
* 当进行上传或下载操作时，当`-d, --dst`的参数为`""`空白时，默认文件将会被上传或下载至本地或远程主机的当前工作目录
* **注意：**ssgo默认所有从远程主机下载的文件下载到本地目录后会在原文件名上添加对应文件所在主机IP地址前缀

``` bash
➜ ./ssgo copy -a download -i config.ini -g docker -s "demo.sh" -d /tmp/temp/ -F table

Tips:Process running start: 2018-04-23 10:08:27
INFO: Success hosts
+---+----------------+---------+-------------+------------------+----------------------+
| # |      Host      |  Status | Source Path | Destination Path |        Result        |
+---+----------------+---------+-------------+------------------+----------------------+
| 1 |  192.168.100.1 | success |   demo.sh   |    /tmp/temp/    | Download finished!:) |
+---+----------------+---------+-------------+------------------+----------------------+
| 2 | 192.168.100.10 | success |   demo.sh   |    /tmp/temp/    | Download finished!:) |
+---+----------------+---------+-------------+------------------+----------------------+
| 3 |  192.168.100.2 | success |   demo.sh   |    /tmp/temp/    | Download finished!:) |
+---+----------------+---------+-------------+------------------+----------------------+
| 4 |  192.168.100.3 | success |   demo.sh   |    /tmp/temp/    | Download finished!:) |
+---+----------------+---------+-------------+------------------+----------------------+
| 5 |  192.168.100.7 | success |   demo.sh   |    /tmp/temp/    | Download finished!:) |
+---+----------------+---------+-------------+------------------+----------------------+
| 6 |  192.168.100.9 | success |   demo.sh   |    /tmp/temp/    | Download finished!:) |
+---+----------------+---------+-------------+------------------+----------------------+

Tips: Process running done.
End Time: 2018-04-23 10:08:27
Cost Time: 142.740437ms
Total Hosts Running: 6(Success) + 0(Failed) = 6(Total)

```
**下载后的文件**

``` bash
➜ ll /tmp/temp/
总用量 24
-rw-r--r-- 1 root root 1024 4月  23 10:08 192.168.100.10_demo.sh
-rw-r--r-- 1 root root 1024 4月  23 10:08 192.168.100.1_demo.sh
-rw-r--r-- 1 root root 1024 4月  23 10:08 192.168.100.2_demo.sh
-rw-r--r-- 1 root root 1024 4月  23 10:08 192.168.100.3_demo.sh
-rw-r--r-- 1 root root 1024 4月  23 10:08 192.168.100.7_demo.sh
-rw-r--r-- 1 root root 1024 4月  23 10:08 192.168.100.9_demo.sh
```


#### 8.5. `ssgo list`获取可用IP地址清单
* 从`--host-list`获取可用IP地址清单（友好提示是否过滤重复主机）
``` bash
➜ ./ssgo list --host-list 192.168.100.1,192.168.100.3-5,192.168.100.5,192.168.100.1 
Duplicate IP Address Found, input 'yes or y' to remove duplicate IP Address, Input 'no or n' will keep duplicate IP Address,
nothing will do by default! (y/n) y
>>> Available Hosts:
+---+---------------+
| # |      Host     |
+---+---------------+
| 1 | 192.168.100.1 |
+---+---------------+
| 2 | 192.168.100.3 |
+---+---------------+
| 3 | 192.168.100.4 |
+---+---------------+
| 4 | 192.168.100.5 |
+---+---------------+

```

* 从`--host-file`获取可用IP地址清单   
``` bash
➜ ./ssgo list --host-file host-file.example.txt
```
* 从config.ini文件获取可用IP地址清单
**示例：** 获取config.ini主机仓库文件中，主机群组为docker的主机清单
``` bash
➜ ./ssgo list -i config.ini -g docker

>>> Group Name: [docker]
>>> Hosts From: 
192.168.100.7
192.168.100.9
192.168.100.10
192.168.100.1-192.168.100.3

>>> Available Hosts:
+---+----------------+------------+
| # |      Host      | Group Name |
+---+----------------+------------+
| 1 |  192.168.100.1 |   docker   |
+---+----------------+------------+
| 2 | 192.168.100.10 |   docker   |
+---+----------------+------------+
| 3 |  192.168.100.2 |   docker   |
+---+----------------+------------+
| 4 |  192.168.100.3 |   docker   |
+---+----------------+------------+
| 5 |  192.168.100.7 |   docker   |
+---+----------------+------------+
| 6 |  192.168.100.9 |   docker   |
+---+----------------+------------+

```

### 9. 其他
#### 9.1. 使用json格式化命令执行结果
**示例：** 将本地demo.sh文件上传至config.ini主机仓库文件主机群组为docker的主机`/tmp/`目录下，并将结果格式化为Json格式
``` bash
➜ ./ssgo copy -a upload -i config.ini -g docker -s demo.sh -d "/tmp/"  -F json
[
    {
        "StartTime": "2018-04-27 14:33:34",
        "HostGroup": "docker",
        "SuccessHosts": [
            {
                "Host": "192.168.100.1",
                "Status": "success",
                "SourcePath": "demo.sh",
                "DestinationPath": "/tmp/",
                "Result": "Upload finished!:)"
            },
            {
                "Host": "192.168.100.10",
                "Status": "success",
                "SourcePath": "demo.sh",
                "DestinationPath": "/tmp/",
                "Result": "Upload finished!:)"
            },
            {
                "Host": "192.168.100.2",
                "Status": "success",
                "SourcePath": "demo.sh",
                "DestinationPath": "/tmp/",
                "Result": "Upload finished!:)"
            },
            {
                "Host": "192.168.100.3",
                "Status": "success",
                "SourcePath": "demo.sh",
                "DestinationPath": "/tmp/",
                "Result": "Upload finished!:)"
            },
            {
                "Host": "192.168.100.7",
                "Status": "success",
                "SourcePath": "demo.sh",
                "DestinationPath": "/tmp/",
                "Result": "Upload finished!:)"
            },
            {
                "Host": "192.168.100.9",
                "Status": "success",
                "SourcePath": "demo.sh",
                "DestinationPath": "/tmp/",
                "Result": "Upload finished!:)"
            }
        ],
        "ErrorHosts": null,
        "EndTime": "2018-04-27 14:33:34",
        "CostTime": "129.689291ms",
        "TotalHostsInfo": "6(Success) + 0(Failed) = 6(Total)"
    }
]                                   
```

##### 9.1.1. 输出命令执行结果为原始json数据
**示例：** 查询192.168.100.1，192.168.100.2，192.168.100.3，192.168.100.4主机的时间，并将原始json结果数据输出
``` bash
➜ ./ssgo run --host-list 192.168.100.1-4 -u root -p root -c date -F json --json-raw
[{"StartTime":"2018-04-27 14:44:36","HostGroup":"from list (192.168.100.1-4)","SuccessHosts":[{"Host":"192.168.100.1","Status":"success","Result":"Fri Apr 27 06:44:36 UTC 2018"},{"Host":"192.168.100.2","Status":"success","Result":"Fri Apr 27 06:44:36 UTC 2018"},{"Host":"192.168.100.3","Status":"success","Result":"Fri Apr 27 06:44:36 UTC 2018"},{"Host":"192.168.100.4","Status":"success","Result":"Fri Apr 27 06:44:36 UTC 2018"}],"ErrorHosts":null,"EndTime":"2018-04-27 14:44:36","CostTime":"98.982031ms","TotalHostsInfo":"4(Success) + 0(Failed) = 4(Total)"}]
```


#### 9.2. 输出日志文件
#### 9.3. 关于`-n, --maxExecuteNum`参数
考虑到并发控制，当主机比较多的情况下，可以适当提高并发数，默认为20   
#### 9.4. 关于`-w, --maxTableCellWidth`参数
`-w, --maxTableCellWidth` 用于美化表格输出列的最大宽度，当发现表格输出列的最大宽度不足以完美存放单行文字时，可以适当调整该数值，默认大小为40

### 10. License
Apache License 2.0
