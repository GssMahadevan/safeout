### Can multiple process route same *safeout* daemon
Yes. You can configure safeout.yaml with multiple process having different fifo for each process and restart safeout server.
BT, Default sample safeout.yaml is routed for two servers.

### Can one process route multiple entries in yaml configuration
Yes. For example: You can add two fifos for one process -- one for *stdout* and one for *stderr*. Please ensure that you have different disk files for stdout/stderr
Assuming that you have one process (named *p1*) and you configured the yaml with fifo for stdout (named */tmp/p1_out.pipe* and */tmp/p1_err.pipe*). Then you need to route your stderr to stdout suing following coommand:
```
p1 >/tmp/p1_out.pipe 2>/tmp/p1_err.pipe
```
### Can one process stdout/stderr routed to same fifo
Yes. Assuming that you have one process (named *p1*) and you configured the yaml with fifo (named */tmp/p1.pipe*). Then you need to route your stderr to stdout suing following coommand:
```
p1 >/tmp/p1.pipe 2>&1
```

### Can safeout program run after servers
No. You have to run the **safeout** server before user processes. 

### Can I change safeout.yaml after *safeout* server started and expect the configuration is reread
No. At present *safeout* program needs to be restarted for any configuration changes. Also keep in mind that, if safeout programs stops, user process might have issues as they are routing their stdout/stderr to *safeout* program.

