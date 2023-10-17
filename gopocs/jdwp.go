package gopocs

import (
	"github.com/hanbufei/dddd/common"
	"github.com/hanbufei/dddd/structs"
	"fmt"
	"github.com/projectdiscovery/gologger"
	"strings"
	"time"
)

func JDWPScan(info *structs.HostInfo) (err error) {
	realhost := fmt.Sprintf("%s:%v", info.Host, info.Ports)
	client, err := common.WrapperTcpWithTimeout("tcp", realhost, time.Duration(6)*time.Second)
	defer func() {
		if client != nil {
			client.Close()
		}
	}()
	if err != nil {
		return err
	}

	err = client.SetDeadline(time.Now().Add(time.Duration(6) * time.Second))
	if err != nil {
		return err
	}

	_, err = client.Write([]byte("JDWP-Handshake"))
	if err != nil {
		return err
	}

	rev := make([]byte, 1024)
	n, errRead := client.Read(rev)
	if errRead != nil {
		return errRead
	}

	if !strings.Contains(string(rev[:n]), "JDWP-Handshake") {
		// 不是JDWP
		return err
	}

	_, err = client.Write([]byte("\x00\x00\x00\x0b\x00\x00\x00\x01\x00\x01\x07"))
	if err != nil {
		return err
	}

	rev = make([]byte, 1024)
	n, errRead = client.Read(rev)
	if errRead != nil {
		return errRead
	}

	if n == 0 {
		return err
	}

	_, err = client.Write([]byte("\x00\x00\x00\x0b\x00\x00\x00\x03\x00\x01\x01"))
	if err != nil {
		return err
	}

	rev = make([]byte, 1024)
	n, errRead = client.Read(rev)
	if errRead != nil {
		return errRead
	}

	data := string(rev[:n])
	if !strings.Contains(data, "Java Debug Wire Protocol") {
		return err
	}

	javaInfo := data[15:]
	result := fmt.Sprintf("[GoPoc] JDWP://%s Unauthorized", realhost)
	gologger.Silent().Msg(result)

	GoPocWriteResult(structs.GoPocsResultType{
		PocName:     "JDWP-Unauthorized",
		Security:    "CRITICAL",
		Target:      realhost,
		InfoLeft:    javaInfo,
		Description: "JDWP未授权访问,可尝试RCE",
	})

	return err
}
