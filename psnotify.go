package main

import (
    "fmt"
    "regexp"
    
    "github.com/mindreframer/golang-stuff/github.com/jondot/gosigar/psnotify"
    netutil "github.com/shirou/gopsutil/net"
    ps "github.com/shirou/gopsutil/process"

)

type Host struct {
    localPort  uint32
    remoteIP   string
    remotePort uint32
}

type Tunnels struct {
    index int
    list []Host
    pids map[int32]int
}

var tunnels Tunnels

func AddConnection(ev * psnotify.ProcEventFork) {
    ppid := int32(ev.ParentPid)
    cpid := int32(ev.ChildPid)
    proc, _ := ps.NewProcess(ppid)
    cmdline, _ := proc.CmdlineSlice()

    if len(cmdline) > 0 {
        matched, _ := regexp.MatchString("sshd: ci@notty*", cmdline[0])
        if matched {
            fmt.Println(cmdline)
            conns, _ := netutil.ConnectionsPid("inet", ppid)
            for conn := range conns {
                if conns[conn].Family == 2 && conns[conn].Status == "LISTEN" {
                    tunnels.list = append(tunnels.list, Host{conns[conn].Laddr.Port,
                                              conns[conn].Raddr.IP, 
                                              conns[conn].Raddr.Port})
                    tunnels.pids[conns[conn].Pid] = len(tunnels.list) - 1
                    fmt.Println(tunnels.list[len(tunnels.list) - 1])                    
                }
            }
            
            watcher, err := psnotify.NewWatcher()
            if err != nil {
                fmt.Println(err)
            }

            err = watcher.Watch(ev.ChildPid, psnotify.PROC_EVENT_EXEC)
            if err != nil {
                fmt.Println(err)
            }

            execev := <-watcher.Exec    
            if (int32(execev.Pid) == cpid) {
                childproc, _ := ps.NewProcess(cpid)
                childcmdline, _ := childproc.CmdlineSlice()
                fmt.Println(childcmdline)                            
            }
        }
    }
}

func remove(slice []Host, s int) []Host {
    return append(slice[:s], slice[s+1:]...)
}

func RemoveConnection(ev * psnotify.ProcEventExit) {
    idx, ok := tunnels.pids[int32(ev.Pid)]
    if ok {
        fmt.Println("Deleting", tunnels.list[idx])
        tunnels.list = remove(tunnels.list, idx)
    }
}

func watchSSH(pid int) {
    watcher, err := psnotify.NewWatcher()
    if err != nil {
        fmt.Println(err)
    }

    // Process events
    go func() {
        for {
            select {
            case ev := <-watcher.Fork:
                AddConnection(ev)
            case <-watcher.Exec:
            case ev := <-watcher.Exit:
                RemoveConnection(ev)

            case <-watcher.Error:
            }
        }
    }()

    err = watcher.Watch(pid, psnotify.PROC_EVENT_ALL)
    if err != nil {
        fmt.Println(err)
    }
}

func main() {

    tunnels.list = make([]Host, 1)
    tunnels.index = -1
    tunnels.pids = make(map[int32]int)
    
    pids, _ := ps.Pids()
    for pid := range pids  {
        proc, _ := ps.NewProcess(pids[pid])
        cmdline, _ := proc.CmdlineSlice()
        if len(cmdline) > 0 {
            matched, _ := regexp.MatchString(`/usr/sbin/sshd`, cmdline[0])
            if matched {    
                fmt.Printf("Watching %d\n", pids[pid])
                watchSSH(int(pids[pid]))
            }            
        }
    }
    
    var num int
    /* ... do stuff ... */
    fmt.Scanf("%d", &num)
}