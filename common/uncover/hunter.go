package uncover

import (
	"github.com/hanbufei/dddd/structs"
	"github.com/hanbufei/dddd/utils"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/retryablehttp-go"
	"github.com/projectdiscovery/subfinder/v2/pkg/passive"
	"gopkg.in/yaml.v3"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

type HunterResp struct {
	Code    int        `json:"code"`
	Data    hunterData `json:"data"`
	Message string     `json:"message"`
}

type infoArr struct {
	URL      string `json:"url"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Domain   string `json:"domain"`
	Protocol string `json:"protocol"`
	IsWeb    string `json:"is_web"`
	City     string `json:"city"`
	Company  string `json:"company"`
	Code     int    `json:"status_code"`
	Title    string `json:"web_title"`
	Country  string `json:"country"`
}

type hunterData struct {
	InfoArr   []infoArr `json:"arr"`
	Total     int       `json:"total"`
	RestQuota string    `json:"rest_quota"`
}

func getHunterKeys() []string {
	var apiKeys []string
	f, err := os.Open("config/subfinder-config.yaml")
	if err != nil {
		gologger.Fatal().Msg("打开API Key配置文件config/subfinder-config.yaml失败")
		return []string{}
	}
	defer f.Close()

	sourceApiKeysMap := map[string][]string{}
	err = yaml.NewDecoder(f).Decode(sourceApiKeysMap)
	for _, source := range passive.AllSources {
		sourceName := strings.ToLower(source.Name())
		if sourceName == "hunter" {
			apiKeys = sourceApiKeysMap[sourceName]
			break
		}
	}
	if len(apiKeys) == 0 {
		gologger.Fatal().Msg("未获取到Hunter API Key")
		return []string{}
	}

	return apiKeys
}

// SearchHunter 从Hunter中搜索目标
func SearchHunterCore(keyword string, pageSize int, maxQueryPage int) []string {
	opts := retryablehttp.DefaultOptionsSpraying
	client := retryablehttp.NewClient(opts)

	url := "https://hunter.qianxin.com/openApi/search"
	keys := getHunterKeys()
	randKey := keys[rand.Intn(len(keys))]

	page := 1
	currentQueryCount := 0

	var results []string
	for page <= maxQueryPage {
		req, err := retryablehttp.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			gologger.Fatal().Msgf("Hunter API请求构建失败。")
		}
		unc := keyword
		search := base64.URLEncoding.EncodeToString([]byte(unc))
		q := req.URL.Query()
		q.Add("search", search)
		q.Add("api-key", randKey)
		q.Add("page", fmt.Sprintf("%d", page))
		q.Add("page_size", fmt.Sprintf("%d", pageSize))
		q.Add("is_web", "3")
		req.URL.RawQuery = q.Encode()

		resp, errDo := client.Do(req)
		if errDo != nil {
			gologger.Error().Msgf("[Hunter] %s 资产查询失败！请检查网络状态。Error:%s", keyword, errDo.Error())
			time.Sleep(time.Second * 3)
			continue
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			gologger.Error().Msgf("获取Hunter 响应Body失败: %v", err.Error())
			time.Sleep(time.Second * 3)
			continue
		}

		var responseJson HunterResp
		if err = json.Unmarshal(data, &responseJson); err != nil {
			gologger.Error().Msgf("[Hunter] 返回数据Json解析失败! Error:%s", err.Error())
			time.Sleep(time.Second * 3)
			continue
		}

		if responseJson.Code != 200 {
			gologger.Error().Msgf("[Hunter] %s 搜索失败！Error:%s", keyword, responseJson.Message)

			if strings.Contains(responseJson.Message, "今日免费积分已用") &&
				strings.Contains(responseJson.Message, "导出数据将扣除权益积分") {
				time.Sleep(time.Second * 3)
				continue
			}

			if responseJson.Message == "请求太多啦，稍后再试试" {
				time.Sleep(time.Second * 3)
				continue
			}
			return results
		}

		if responseJson.Data.Total == 0 {
			gologger.Error().Msgf("[Hunter] %s 无结果。", keyword)
			return results
		}

		for _, v := range responseJson.Data.InfoArr {
			if v.IsWeb == "是" {
				gologger.Silent().Msgf("[Hunter] [%d] %s [%s] [%s] [%s]", v.Code, v.URL, v.Title, v.City, v.Company)
				results = append(results, v.URL)
			} else {
				gologger.Silent().Msgf("[Hunter] %s://%s:%d", v.Protocol, v.IP, v.Port)
				results = append(results, fmt.Sprintf("%s:%v", v.IP, v.Port))
			}
		}

		currentQueryCount += len(responseJson.Data.InfoArr)
		gologger.Info().Msgf("[Hunter] [%s] 当前第 [%d] 页 查询进度: %d/%d %v", keyword, page, currentQueryCount,
			responseJson.Data.Total, responseJson.Data.RestQuota)

		if currentQueryCount >= responseJson.Data.Total {
			return results
		}

		page += 1

		// 避免请求过于频繁
		time.Sleep(time.Second * 3)

	}

	return results
}

func HunterSearch(keywords []string) []string {
	var results []string
	for _, keyword := range keywords {
		result := SearchHunterCore(keyword,
			structs.GlobalConfig.HunterPageSize,
			structs.GlobalConfig.HunterMaxPageCount)
		results = append(results, result...)
	}
	return utils.RemoveDuplicateElement(results)
}
