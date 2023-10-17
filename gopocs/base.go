package gopocs

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"github.com/hanbufei/dddd/common/report"
	"github.com/hanbufei/dddd/structs"
	"github.com/hanbufei/dddd/utils"
	"encoding/base64"
	"net"
	"os"
	"strings"
	"sync"
)

var PluginList = map[string]interface{}{
	"NetBios-GetHostInfo": NetBIOS,
	"RPC-GetHostInfo":     Findnet,
	"SSH-Crack":           SshScan,
	"FTP-Crack":           FtpScan,
	"Mysql-Crack":         MysqlScan,
	"Mssql-Crack":         MssqlScan,
	"Oracle-Crack":        OracleScan,
	"MongoDB-Crack":       MongodbScan,
	"RDP-Crack":           RdpScan,
	"Redis-Crack":         RedisScan,
	"SMB-MS17-010":        MS17010,
	"PostgreSQL-Crack":    PostgresScan,
	"SMB-Crack":           SmbScan,
	"Telnet-Crack":        TelnetScan,
	"Memcache-Crack":      MemcachedScan,
	"JDWP-Scan":           JDWPScan,
	"Shiro-Key-Crack":     ShiroKeyCheck,
}

var WriteResultLock sync.Mutex

func ReadBytes(conn net.Conn) (result []byte, err error) {
	size := 4096
	buf := make([]byte, size)
	for {
		count, err := conn.Read(buf)
		if err != nil {
			break
		}
		result = append(result, buf[0:count]...)
		if count < size {
			break
		}
	}
	if len(result) > 0 {
		err = nil
	}
	return result, err
}

var key = "0123456789abcdef"

func AesEncrypt(orig string, key string) string {
	// 转成字节数组
	origData := []byte(orig)
	k := []byte(key)
	// 分组秘钥
	// NewCipher该函数限制了输入k的长度必须为16, 24或者32
	block, _ := aes.NewCipher(k)
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 补全码
	origData = PKCS7Padding(origData, blockSize)
	// 加密模式
	blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])
	// 创建数组
	cryted := make([]byte, len(origData))
	// 加密
	blockMode.CryptBlocks(cryted, origData)
	return base64.StdEncoding.EncodeToString(cryted)
}
func AesDecrypt(cryted string, key string) string {
	// 转成字节数组
	crytedByte, _ := base64.StdEncoding.DecodeString(cryted)
	k := []byte(key)
	// 分组秘钥
	block, _ := aes.NewCipher(k)
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 加密模式
	blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])
	// 创建数组
	orig := make([]byte, len(crytedByte))
	// 解密
	blockMode.CryptBlocks(orig, crytedByte)
	// 去补全码
	orig = PKCS7UnPadding(orig)
	return string(orig)
}

// 补码
// AES加密数据块分组长度必须为128bit(byte[16])，密钥长度可以是128bit(byte[16])、192bit(byte[24])、256bit(byte[32])中的任意一个。
func PKCS7Padding(ciphertext []byte, blocksize int) []byte {
	padding := blocksize - len(ciphertext)%blocksize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// 去码
func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func generateKeys(key string) []string {
	var results []string
	results = append(results, strings.ToLower(key))
	results = append(results, strings.ToUpper(key))
	results = append(results, strings.ToUpper(strings.ToUpper(key[:1])+key[1:]))
	return results
}

func splitUserPass(userPasswd string) (user string, oriPass string) {
	sp := strings.Split(userPasswd, " : ")
	user = ""
	oriPass = ""
	if len(sp) != 2 {
		user = sp[0]
		oriPass = ""
	} else {
		user = sp[0]
		oriPass = sp[1]
	}
	user = strings.TrimSuffix(user, "\r")
	oriPass = strings.TrimSuffix(oriPass, "\r")
	return user, oriPass
}

func sortUserPassword(info *structs.HostInfo, UserPasswdDict string, DefaultKeys []string) []structs.UserPasswd {
	var userPasswdList []structs.UserPasswd

	upList := info.UserPass

	// 兼容Windows
	UserPasswdDict = strings.ReplaceAll(UserPasswdDict, "\r\n", "\n")
	for _, v := range strings.Split(UserPasswdDict, "\n") {
		upList = append(upList, v)
	}
	upList = utils.RemoveDuplicateElement(upList)

	// 统计变形后的字典
	for _, userPasswd := range upList {
		user, oriPass := splitUserPass(userPasswd)

		if strings.Contains(oriPass, "{{key}}") {
			for _, sKey := range info.InfoStr {
				newKeys := generateKeys(sKey)
				for _, nKey := range newKeys {
					newPass := strings.Replace(oriPass, "{{key}}", nKey, -1)
					userPasswdList = append(userPasswdList, structs.UserPasswd{UserName: user, Password: newPass})
				}

			}
			for _, dk := range DefaultKeys {
				newKeys := generateKeys(dk)
				for _, nKey := range newKeys {
					newPass := strings.Replace(oriPass, "{{key}}", nKey, -1)
					userPasswdList = append(userPasswdList, structs.UserPasswd{UserName: user, Password: newPass})
				}
			}

		} else {
			userPasswdList = append(userPasswdList, structs.UserPasswd{UserName: user, Password: oriPass})
		}
	}
	return RemoveDuplicateUserPass(userPasswdList)
}

func RemoveDuplicateUserPass(input []structs.UserPasswd) []structs.UserPasswd {
	temp := map[structs.UserPasswd]struct{}{}
	var result []structs.UserPasswd
	for _, item := range input {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func CheckErrs(err error) bool {
	if err == nil {
		return false
	}
	errs := []string{
		"closed by the remote host", "too many connections",
		"i/o timeout", "EOF", "A connection attempt failed",
		"established connection failed", "connection attempt failed",
		"Unable to read", "is not allowed to connect to this",
		"no pg_hba.conf entry",
		"No connection could be made",
		"invalid packet size",
		"bad connection",
	}
	for _, key := range errs {
		if strings.Contains(strings.ToLower(err.Error()), strings.ToLower(key)) {
			return true
		}
	}
	return false
}

func GoPocWriteResult(result structs.GoPocsResultType) {
	WriteResultLock.Lock()
	report.AddResultByGoPocResult(result)
	WriteResultLock.Unlock()
}

func readDict(name string) string {
	bt, err := os.ReadFile(name)
	if err != nil {
		return ""
	}
	return string(bt)
}

// initDic 初始化用于爆破的字典
func initDic() {
	basePath := "config/dict/"
	ftpUserPasswdDict = readDict(basePath + "ftp.txt")
	mssqlUserPasswdDict = readDict(basePath + "mssql.txt")
	mysqlUserPasswdDict = readDict(basePath + "mysql.txt")
	oracleUserPasswdDict = readDict(basePath + "oracle.txt")
	postgreSQLUserPasswdDict = readDict(basePath + "postgresql.txt")
	rdpUserPasswdDict = readDict(basePath + "rdp.txt")
	redisUserPasswdDict = readDict(basePath + "redis.txt")
	smbUserPasswdDict = readDict(basePath + "smb.txt")
	sshUserPasswdDict = readDict(basePath + "ssh.txt")
	telnetUserPasswdDict = readDict(basePath + "telnet.txt")
}
