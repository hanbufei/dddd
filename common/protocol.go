package common

import (
	"github.com/hanbufei/dddd/structs"
	"github.com/hanbufei/dddd/utils"
	"fmt"
	"github.com/lcvvvv/gonmap"
	"github.com/projectdiscovery/gologger"
	"strconv"
	"strings"
	"sync"
	"time"
)

func GetProtocol(hostPorts []string, threads int) {
	if len(hostPorts) == 0 {
		return
	}

	hostPorts = utils.RemoveDuplicateElement(hostPorts)
	if len(hostPorts) < threads {
		threads = len(hostPorts)
	}

	workers := threads
	Addrs := make(chan string, len(hostPorts))
	defer close(Addrs)
	results := make(chan structs.ProtocolResult, len(hostPorts))
	defer close(results)
	var wg sync.WaitGroup

	//接收结果
	go func() {
		for found := range results {
			if found.Status == int(gonmap.Closed) {
				wg.Done()
				continue
			}
			if found.Status == gonmap.Open || found.Response == nil {
				wg.Done()
				continue
			}

			if found.Port == 23 && found.Response.FingerPrint.Service == "" {
				found.Response.FingerPrint.Service = "telnet"
			}
			hostPort := fmt.Sprintf("%s:%v", found.IP, found.Port)
			structs.GlobalIPPortMapLock.Lock()
			_, ok := structs.GlobalIPPortMap[hostPort]
			structs.GlobalIPPortMapLock.Unlock()
			if !ok {
				structs.GlobalBannerHMap.Set(hostPort, []byte(found.Response.Raw))
				structs.GlobalIPPortMapLock.Lock()
				structs.GlobalIPPortMap[hostPort] = found.Response.FingerPrint.Service
				structs.GlobalIPPortMapLock.Unlock()
			}
			if found.Response.FingerPrint.Service != "" {
				gologger.Silent().Msgf("[Nmap] %v://%v:%v", found.Response.FingerPrint.Service, found.IP, found.Port)
			}
			wg.Done()
		}
	}()

	//多线程扫描
	for i := 0; i < workers; i++ {
		go func() {
			scanner := gonmap.New()
			for addr := range Addrs {
				t := strings.Split(addr, ":")
				if len(t) < 2 {
					continue
				}
				ip := t[0]
				port, err := strconv.Atoi(t[1])
				if err != nil || port > 65535 {
					continue
				}
				status, response := scanner.ScanTimeout(ip, port, time.Second*10)
				var data structs.ProtocolResult
				data.IP = ip
				data.Port = port
				data.Status = int(status)
				data.Response = response
				results <- data
			}
		}()
	}

	//添加扫描目标
	for _, hostPort := range hostPorts {
		wg.Add(1)
		Addrs <- hostPort
	}
	wg.Wait()
}
