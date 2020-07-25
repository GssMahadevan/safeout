# safeout
Allow unix like os programs/daemons to safely redirect theier stdout/stderr without filling up disk

## Why
While I was doing home IoT stuff on my edge-router which was running on Raspberry-Pi, I had an issue with third party library that spews messages when my WiFi was shutdown in night. As the stdout/stderr are redirect to a file Raspberry-Pi, the log size crossed to 6GB in 12 hours. We do typically see these scenarios in real life as well. Only way out is to stop the main process , truncate  the log-file and restart the process. Usage scenario in details blog'ed at **[my blog](http://mahadevangorti.blogspot.com/2020/07/how-i-saved-my-raspberry-iot-edge-host.html)**

Few approaches  came  into mind:
 * Write simple linux kernel module to handle a specially tagged file to handle max size case
 * Write simple custom use-file system (using libfuse )
 
 * Write simple go-lang program that handles the case
   * Finally go-approach is chosen as:
     * Program Can be run in multiple OSes(Windows/MAc) and multiple linux distros and kernel versions without recompile for each kernel
     * Higher productvity
     * Long term maintainance
    
## Purpose
 - Safely redirect **stdout/stderr** from a program/daemon to disk or partition without any risk of filling up disk in case of 
   - rougue modules/sub-systems writing repeatedly to logs to stdout/stderr
   - uncontrolled third party errors writing repeatedly to logs to stdout/stderr
   - unhandled error/stacktrace messages writing repeatedly to logs to stdout/stderr
   - give control to user on maximum file size of log file that ios going to be created by stdout/stderr messages
   - give one backup of overwritten messages / per process
   - allow multiple user programs using this single **safeout** program to route safely to multiple disk locations
   
## Design and Concept
This program is written using go-language. This **safeout** daemon/program creates one fifo(ie., named pipe) per user process (user can create one fifo for stdout and one fifo for stderr also)  and goes on listening for data from the fifo(s). Once data from fifo is read, the daemon will redirect the messages to  user configured log-file (with configured maximum size checks on this log file) for a given fifo. Once the log-file of specific user process is reached maximum size, this daemon renames the log-file to  backup-copy for that specific process and restart log-file the user process from start. With this check, user-process logs never crosses 2 * max-size-confgured for a specifc process ( i.e., including backup copy size ), there by saving uncontrolled disk-full in case of direct stdout/stderr messages redirected to disk.

For simpler traditional unix stdin redirect cases (like multilog,s6,etc), **safeout** also supports similiar behaviour. But in this case, one needs to **run multiple instances** of **safeout** instances for each program. 

## Simple traditional unix pipe redirection (1 safeout instance per one user program redirection)
 - To ensure that **user_program** output.log never crosses 500000 bytes (and have one backup named 'output.log.backup')
```bash
user_program | safeout --outfile output.log --outmax 500000 
```

- To ensure that **user_program** output.log never crosses 500000 bytes (and have one backup named 'output.log.backup') for both stdout/stderr
```bash
user_program  2>&1 | safeout --outfile output.log --outmax 500000 
```
## Start server to handle multiple user programs using single instance of safeout
- With default yaml configuration file named **safeout.yaml** in  PWD
```bash
 safeout
 ```
- With yaml  yaml configuration file located at  **/tmp/safeout.yaml** 
```bash
safeout --cfg /tmp/safeout.yaml
```
  
## Stop server
Two ways one can stop this server
 - Simply issue Ctrl-C
 - Send signal SIGTERM

## Configuration
Sample Configuration is provided in git repo

## How to Build
#### Prerequisitives
 - Install latest go-lang
#### Build
 - Clone this repository to your desk using (here we are assuming that you are using X86_64 processor)
```bash
git clone https://github.com/GssMahadevan/safeout
cd safeout
go build
```
 - Build for Raspberry-Pi (in your desktop using cross compilation)
```bash
GOARCH=arm GOARM=7 go build -o safeout.a7 safeout.go 
```
 - In case you want smaller binary
```bash
# for host processor binary
go build -ldflags="-s -w"  -o safeout safeout.go 
# for cross compiled for arm.v7 binary
GOARCH=arm GOARM=7 go build -ldflags="-s -w"  -o safeout.a7 safeout.go 
```

## Examples
### Example where stdout/stderr are redirectedto same file (with disk full protection)
 - Suppose you have process named **gvl_server** that wants to create logs at **/data/logs/gvl.log** . So to configure this **safeout** daemon to handle this situation, you need to write a yaml-configuration like following:
```yaml
description: This program ensures disk does not become full when stdout/stderr are routed to filesystem
version: 1.0
common:
 c1: &c1
  maxfiles : 1
  maxsize  : 10_000_000
perms:
 p1: &p1
  permFifo: "0644"
  permFile: "0644" 

safeouts:
  gvl_server:
    com : *c1
    perms : *p1
    fifoName: /data/logs/gvl.pipe
    fileName: /data/logs/d/gvl.log
    compress: False  
```
 - With above configuration  saved as safeout.yaml in PWD, start the **safeout** deamon PWD using command:
```bash
./safeout
```
 - Now after successful runing of **safeout** daemon, run your program **gvl_server** usinng command (here we are redirecting stderr/stdout to same fifo as mentioned in yaml file)):
 ```bash
 gvl_server >/data/logs/gvl.pipe 2>&1 
 ```
### Example where stdout/stderr are redirected to to different files (with disk full protection)
 - Suppose you have process named **gvl_server** that wants to create logs  for stdout at **/data/logs/gvl_stdout.log**  and stderr at **/data/logs/gvl_stderr.log**. So to configure this **safeout** daemon to handle this situation, you need to write a yaml-configuration like following:
```yaml
description: This program ensures disk does not become full when stdout/stderr are routed to filesystem
version: 1.0
common:
 c1: &c1
  maxfiles : 1
  maxsize  : 10_000_000
perms:
 p1: &p1
  permFifo: "0644"
  permFile: "0644" 

safeouts:
  gvl_server_stdout:
    com : *c1
    perms : *p1
    fifoName: /data/logs/gvl_stdout.pipe
    fileName: /data/logs/d/gvl_stdout.log
    compress: False  

  gvl_server_stderr:
    com : *c1
    perms : *p1
    fifoName: /data/logs/gvl_stderr.pipe
    fileName: /data/logs/d/gvl_stderr.log
    compress: False  
    
```
 - With above configuration  saved as safeout.yaml in PWD, start the **safeout** deamon PWD using command:
```bash
./safeout
```
 - Now after successful runing of **safeout** daemon, run your program **gvl_server** usinng command (here we are redirecting stderr/stdout to different fifos as mentioned in yaml file):
 ```bash
 gvl_server >/data/logs/gvl_stdout.pipe 2> /data/logs/gvl_stderr.pipe
 ```
 
## Features
 - [x] Safe redirect of stdout/stderr to file
 - [x] Allow multiple processes redirect their stdout/stderr to single server via fifo/named-pipes in unix
 - [x] Ensure file size check for each process
 - [x] Have one backup file per each process/pipe combination
 - [ ] Compress backup copies
 - [ ] Support for Windows/Mac
 - [ ] Support for multiple backup copies
 - [ ] Detect dynamic congiration changes to add new processes for this server
 
## Similiar Tools
There are some similir tools in Unix world named multilog,tinylog,cyclog,s6-log,svlogd,etc. You can see more on at http://zpz.github.io/blog/log-rotation-of-stdout/ . 

How this program differs from those tools:
 * Ability to handle stdout/stderr of a process with disk fill protection **in realtime at low cost**
 * User application simple redirect the stderr/stdout to a fifo that is configured in yaml file
 * Ability of **single safeout** program  **to handle multiple processes**(and their stdout/stderr combindly/seperately)
  * Traditional unix damons tools like multilog,s6, etc always have 1:1 mapping (i.e., 1 user process redirection needs one multilog process)
 * Ability to handle stdout/stderr in seperately/combindely  w.r.t. multilog/s6,etc. 
 * Application command without any pipe redirections (as needed for s6, multilog, etc). Such redirection might not work in some instances (like ```os.system(cmd)``` )and one needs to wrap the application in shell-script to handle such scenarios. Anyhow this issue might not be major one.
## Caveat
Even  though this program is running successfully on my  raspberry-pi for 3 days(as on 20-Jul-2020), please keep in mind that this program is in ***alpha*** version  and there could be surprizes and bugs. So please do thorough testing it before you deploy this in real life production systems. I can't guarantee anything and any bad consequences of your stdout/stderr  logs are not redirected/saved/missed/etc are not my responsibility :)
## Issues
Please raise issues/suggestion for improvement/pacthes at github. You may also fork and customize the program for your needs.
Please keep in mind that I do this development on my free time. So my responses might be late.
 
