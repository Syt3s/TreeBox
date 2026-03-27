package config

var (
	BuildTime   string
	BuildCommit = "dev"
)

var (
	App struct {
		Production      bool   `ini:"production"`
		ExternalURL     string `ini:"external_url"`
		UptraceDSN      string `ini:"uptrace_dsn"`
		MaintenanceMode bool   `ini:"maintenance_mode"`
	}

	Server struct {
		Port    int    `ini:"port"`
		Salt    string `ini:"salt"`
		XSRFKey string `ini:"xsrf_key"`
	}

	Database struct {
		DSN string

		Type     string `ini:"type"`
		User     string `ini:"user"`
		Password string `ini:"password"`
		Host     string `ini:"host"`
		Port     uint   `ini:"port"`
		Name     string `ini:"name"`
		Schema   string `ini:"schema"` // for postgres
	}

	Redis struct {
		Addr     string `ini:"addr"`
		Password string `ini:"password"`
	}

	Recaptcha struct {
		Domain         string `ini:"domain"`
		SiteKey        string `ini:"site_key"`
		ServerKey      string `ini:"server_key"`
		TurnstileStyle bool   `ini:"turnstile_style"`
	}

	Mail struct {
		Account  string `ini:"account"`
		Password string `ini:"password"`
		Port     int    `ini:"port"`
		SMTP     string `ini:"smtp"`
	}

	Pixel struct {
		Host string `ini:"host"`
	}

	Upload struct {
		DefaultAvatar     string `ini:"default_avatar"`
		DefaultBackground string `ini:"default_background"`

		ImageEndpoint      string `ini:"image_endpoint"`
		ImageAccessID      string `ini:"image_access_id"`
		ImageAccessSecret  string `ini:"image_access_secret"`
		ImageBucket        string `ini:"image_bucket"`
		ImageBucketCDNHost string `ini:"image_bucket_cdn_host"`

		AliyunEndpoint      string `ini:"aliyun_endpoint"`
		AliyunAccessID      string `ini:"aliyun_access_id"`
		AliyunAccessSecret  string `ini:"aliyun_access_secret"`
		AliyunBucket        string `ini:"aliyun_bucket"`
		AliyunBucketCDNHost string `ini:"aliyun_bucket_cdn_host"`
	}

	Service struct {
		Backends []struct {
			Prefix     string `ini:"prefix"`
			ForwardURL string `ini:"forward_url"`
		}
	}
)
