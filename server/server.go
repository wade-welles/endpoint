/* Copyright 2017, Ashish Thakwani. 
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.LICENSE file.
 */
package server

import (
    "log"
    "encoding/json"
    "os"
    "os/exec"
    "strconv"
    "bytes"
    "io"
    "net"
    "fmt"
    
    "../utils"
)

/*
 *  Map for holding active connections.
 */
var conns map[int]*utils.Host

/*
 *  Event callback for connection add & remove.
 */
type ConnAddedEvent   func(p int, h *utils.Host)
type ConnRemovedEvent func(p int, h *utils.Host)
 

/*
 *  Add connection to map and invoke callback.
 */
func addConnection(h *utils.Host, a ConnAddedEvent) {
    p := h.Pid
    
    conns[p] = h

    //Send AddedEvent Callback.
    a(p, h)
}

/*
 *  Remove connection from map and invoke callback.
 */
func removeConnection(h *utils.Host, r ConnRemovedEvent) {
    p := h.Pid
    
    h, ok := conns[p]
    if ok {
        delete(conns, p)
        
        //Send RemoveEvent Callback
        r(p, h)
    }
}


/*
 *  Wait for lock file to be released.
 */
func waitForClose(p int) bool {
    // Add user
	cmd := "flock"
    args := []string{utils.RUNPATH + strconv.Itoa(p), "-c", "echo done"}
    
    _, err := exec.Command(cmd, args...).Output()
    if err != nil {
        return false
    }
    
    return true
}

/*
 *  Handle Socket connection
 */
func handleClient(c net.Conn, a ConnAddedEvent, r ConnRemovedEvent) {
    var h utils.Host
    var b bytes.Buffer
    
    // Copy socket data to buffer
    io.Copy(&b, c)
    c.Close()
    
    log.Printf("Payload: %s\n", b.String())

    err := json.Unmarshal(b.Bytes(), &h) 
    utils.Check(err)
    
    addConnection(&h, a)
    
    if waitForClose(int(h.Pid)) == true {
        removeConnection(&h, r)
    }
}

/*
 *  Monitor incoming connections and invoke callback
 *  when client is added or removed.
 */
func Monitor(a ConnAddedEvent, r ConnRemovedEvent) {
    // Initialize connections map to store active connections.
    conns = make(map[int]*utils.Host)
    
    // Get process pid and open unix socket.
    p := os.Getpid()
    f := utils.RUNPATH + strconv.Itoa(p) + ".sock"
    
    l, _ := net.Listen("unix", f)
    fmt.Printf("Waiting for Connections\n")

    for {
        // Handle incoming conection.
        fd, err := l.Accept()
        utils.Check(err)

        go handleClient(fd, a, r)
    }
    

}