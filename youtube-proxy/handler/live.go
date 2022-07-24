package handler

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/zjyl1994/livetv/global"
	"github.com/zjyl1994/livetv/service"
	"github.com/zjyl1994/livetv/util"
	"golang.org/x/net/proxy"
)

func M3UHandler(c *gin.Context) {
	content, err := service.M3UGenerate()
	if err != nil {
		log.Println(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "application/vnd.apple.mpegurl", []byte(content))
}
func checkAndSetProxy(httpTransport *http.Transport) error {
	//设置代理相关
	proxyStr := global.GetProxy()
	if proxyStr != "" {
		if strings.Contains(proxyStr, "socks") {
			dialer, err := proxy.SOCKS5("tcp", strings.ReplaceAll(proxyStr, "socks5://", ""), nil, proxy.Direct)
			if err != nil {
				return err
			}
			httpTransport.Dial = dialer.Dial
		} else {
			proxyStr := global.GetProxy()
			proxyURL, err := url.Parse(proxyStr)
			if err != nil {
				return err
			}
			httpTransport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	return nil
}

func LiveHandler(c *gin.Context) {
	var m3u8Body string
	channelCacheKey := c.Query("c")
	iBody, found := global.M3U8Cache.Get(channelCacheKey)
	if found {
		m3u8Body = iBody.(string)
	} else {
		channelNumber := util.String2Uint(c.Query("c"))
		if channelNumber == 0 {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		channelInfo, err := service.GetChannel(channelNumber)
		if err != nil {
			if gorm.IsRecordNotFoundError(err) {
				c.AbortWithStatus(http.StatusNotFound)
			} else {
				log.Println(err)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
			return
		}
		baseUrl, err := service.GetConfig("base_url")
		if err != nil {
			log.Println(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		liveM3U8, err := service.GetYoutubeLiveM3U8(channelInfo.URL)
		if err != nil {
			log.Println(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		httpTransport := &http.Transport{}

		err = checkAndSetProxy(httpTransport)
		if err != nil {
			log.Println(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		client := http.Client{Timeout: global.HttpClientTimeout, Transport: httpTransport}

		resp, err := client.Get(liveM3U8)
		if err != nil {
			log.Println(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		bodyString := string(bodyBytes)
		if channelInfo.Proxy {
			m3u8Body = service.M3U8Process(bodyString, baseUrl+"/jmsytb.ts?k=")
		} else {
			m3u8Body = bodyString
		}
		global.M3U8Cache.Set(channelCacheKey, m3u8Body, 3*time.Second)
	}
	c.Data(http.StatusOK, "application/vnd.apple.mpegurl", []byte(m3u8Body))
}

func TsProxyHandler(c *gin.Context) {
	zipedRemoteURL := c.Query("k")
	remoteURL, err := util.DecompressString(zipedRemoteURL)
	if err != nil {
		log.Println(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if remoteURL == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	httpTransport := &http.Transport{}
	err = checkAndSetProxy(httpTransport)

	client := http.Client{Timeout: global.HttpClientTimeout, Transport: httpTransport}
	if err != nil {
		log.Println(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	resp, err := client.Get(remoteURL)
	if err != nil {
		log.Println(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	c.DataFromReader(http.StatusOK, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}

func CacheHandler(c *gin.Context) {
	var sb strings.Builder
	global.URLCache.Range(func(k, v interface{}) bool {
		sb.WriteString(k.(string))
		sb.WriteString(" => ")
		sb.WriteString(v.(string))
		sb.WriteString("\n")
		return true
	})
	c.Data(http.StatusOK, "text/plain", []byte(sb.String()))
}
