package gopocs

import (
	"github.com/hanbufei/dddd/common"
	"github.com/hanbufei/dddd/structs"
	"fmt"
	"github.com/projectdiscovery/gologger"
	"strings"
	"time"
)

func MemcachedScan(info *structs.HostInfo) (err error) {
	realhost := fmt.Sprintf("%s:%v", info.Host, info.Ports)
	client, err := common.WrapperTcpWithTimeout("tcp", realhost, time.Duration(6)*time.Second)
	defer func() {
		if client != nil {
			client.Close()
		}
	}()
	if err == nil {
		err = client.SetDeadline(time.Now().Add(time.Duration(6) * time.Second))
		if err == nil {
			_, err = client.Write([]byte("stats\n")) //Set the key randomly to prevent the key on the server from being overwritten
			if err == nil {
				rev := make([]byte, 1024)
				n, errRead := client.Read(rev)
				if errRead == nil {
					if strings.Contains(string(rev[:n]), "STAT") {
						result := fmt.Sprintf("[GoPoc] Memcached://%s Unauthorized", realhost)
						gologger.Silent().Msg(result)

						GoPocWriteResult(structs.GoPocsResultType{
							PocName:     "Memcached-Unauthorized",
							Security:    "HIGH",
							Target:      realhost,
							InfoLeft:    string(rev[:n]),
							Description: "Memcached未授权访问",
						})

					}
				}
			}
		}
	}
	return err
}
