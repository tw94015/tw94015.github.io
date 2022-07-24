package global

import (
	_ "github.com/joho/godotenv/autoload"
	"os"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

var defaultConfigValue = map[string]string{
	"ytdl_cmd":  "youtube-dl",
	"ytdl_args": "--no-warnings -f best -g {url}",
	"base_url":  "http://127.0.0.1:9000",
	"password":  "password",
}

var (
	HttpClientTimeout = 60 * time.Second
	ConfigCache       sync.Map
	URLCache          sync.Map
	M3U8Cache         = cache.New(3*time.Second, 10*time.Second)
)

func GetProxy() string {
	return os.Getenv("PROXY")
}
