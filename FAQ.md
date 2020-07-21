### Can multiple process redirect same *safeout* daemon
Yes. You can configure safeout.yaml with multiple process having different fifo for each process and restart safeout server.
BT, Default sample safeout.yaml is redirected for two servers.

### Can one process redirect multiple entries in yaml configuration
Yes. For example: You can add two fifos for one process -- one for **stdout** and one for **stderr**. Please ensure that you have different disk files for stdout/stderr
Assuming that you have one process (named **p1**) and you configured the yaml with fifo for stdout (named **/tmp/p1_out.pipe** and **/tmp/p1_err.pipe**). Then you need to redirect your stderr to stdout using following coommand:
```
p1 >/tmp/p1_out.pipe 2>/tmp/p1_err.pipe
```
### Can one process stdout/stderr redirected to same fifo
Yes. Assuming that you have one process (named **p1**) and you configured the yaml with fifo (named **/tmp/p1.pipe**). Then you need to redirect your stderr to stdout using following coommand:
```
p1 >/tmp/p1.pipe 2>&1
```

### Can safeout program run after servers
No. You have to run the **safeout** server before user processes. If you donot start  **safeout** daemon before user processes, then user process will block till **safeout** daemon is starts running (and listens to corresponding pipe that is reading respective user processes)

### Can I change safeout.yaml after *safeout* server started and expect the configuration is reread
No. At present **safeout** program needs to be restarted for any configuration changes. Also keep in mind that, if safeout programs stops, user process might have issues as they are routing their stdout/stderr to *safeout* program.


### Are there any prebuilt binary
Yes. Please find them in **prebuilt-binary** directory. BTW these prebuilt binaries are stripped and compressed binaries.
