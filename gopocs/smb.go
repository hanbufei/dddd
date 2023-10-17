package gopocs

import (
	"context"
	"github.com/hanbufei/dddd/structs"
	_ "embed"
	"fmt"
	"github.com/hirochachacha/go-smb2"
	"github.com/projectdiscovery/gologger"
	"net"
	"time"
)

var smbUserPasswdDict string

func SmbScan(info *structs.HostInfo) (tmperr error) {
	starttime := time.Now().Unix()

	userPasswdList := sortUserPassword(info, smbUserPasswdDict, []string{})

	for _, userPass := range userPasswdList {
		flag, err := doWithTimeOut(info, userPass.UserName, userPass.Password)
		if flag == true && err == nil {
			return err
		} else {
			tmperr = err
			if CheckErrs(err) {
				return err
			}
			if time.Now().Unix()-starttime > (int64(len(userPasswdList)) * 6) {
				return err
			}
		}
	}

	return tmperr
}

func SmblConn(info *structs.HostInfo, user string, pass string) (flag bool, err error) {
	flag = false

	conn, err := net.Dial("tcp", info.Host+":"+info.Ports)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     user,
			Password: pass,
		},
	}

	s, err := d.Dial(conn)
	if err != nil {
		return false, err
	}
	defer s.Logoff()
	flag = true

	showShare := ""
	names, err := s.ListSharenames()
	if err == nil {
		for _, name := range names {
			showShare += name + "\n"
		}
	}

	var result string
	result = fmt.Sprintf("[GoPoc] SMB://%v:%v:%v %v", info.Host, info.Ports, user, pass)
	gologger.Silent().Msg(result)
	showData := fmt.Sprintf("Host: %v:%v\nUsername: %v\nPassword: %v\n", info.Host, info.Ports, user, pass)

	GoPocWriteResult(structs.GoPocsResultType{
		PocName:     "SMB-Login",
		Security:    "CRITICAL",
		Target:      info.Host + ":" + info.Ports,
		InfoLeft:    showData,
		InfoRight:   showShare,
		Description: "SMB弱口令",
	})

	return true, nil
}

type resType struct {
	err  error
	flag bool
}

func doWithTimeOut(info *structs.HostInfo, user string, pass string) (flag bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(7)*time.Second)
	defer cancel()

	c := make(chan resType, 1)

	go func() {

		flag, err = SmblConn(info, user, pass)
		c <- resType{
			err:  err,
			flag: flag,
		}
	}()
	select {
	case <-ctx.Done():
		res := <-c
		return false, res.err
	case res := <-c:
		return res.flag, res.err
	}
}
