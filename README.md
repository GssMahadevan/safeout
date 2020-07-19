# safeout
Allow unix like os programs/daemons to safely route theier stdout/stderr without filling up disk

## Purpose
 - Safely route **stdout/stderr** from a program/daemon to disk or partition without any risk of filling up disk in case of 
   - rougue modules/sub-systems writing repeatedly to logs to stdout/stderr
   - uncontrolled third party errors writing repeatedly to logs to stdout/stderr
   - unhandled error/stacktrace messages writing repeatedly to logs to stdout/stderr
   - give control to user on maximum file size of log file that ios going to be created by stdout/stderr messages
   - give one backup of overwritten messages / per process
   - allow multiple user programs using this single **safeout** program to route safely to multiple disk locations
   
## Design and Concept
This program is written using go-language. This **safeout** daemon/program creates on fifo(ie., named pipe) per user process and goes on listening for data from the fifo. Once data from fifo is read, the daemon will route the messages to  user configured log-file (with configured maximum size checks on this log file). Once the log-file of specific user process is reached maximum size, this daemon renames the log-file to  backup-copy for that specific process and restart log-file the user process from start. With this check, user-process logs never crosses 2 * max-size-confgured for a specifc process ( i.e., including backup copy size ), there by saving uncontrolled disk-full in case of direct stdout/stderr messages routing to disk

## Start server
- With default yaml configuration file named **safeout.yaml** in  PWD
  * safeout
- With yaml  yaml configuration file located at  **/tmp/safeout.yaml** 
  * safeout --cfg /tmp/safeout.yaml
  
## Stop server
Two ways one can stop this server
 - Simply issue Ctrl-C
 - Send signal SIGTERM

## Configuration
Sample Configuration is provided in git repo

## Features
 - [x] Safe routing of stdout/stderr to file
 - [x] Allow multiple processes route theier stdout/stderr to single server via fifo/named-pipes in unix
 - [x] Ensure file size check for each process
 - [x] Have one backup file per each process/pipe combination
 - [ ] Compress backup copies
 - [ ] Support for Windows/Mac
 - [ ] Support for multiple backup copies
 - [ ] Detect dynamic congiration changes to add new processes for this server
 
## Issues
Please raise issues/suggestion for improvement/pacthes at github/fork
Please keep in mind that I do this development on my free time. So my responses might be late
 
