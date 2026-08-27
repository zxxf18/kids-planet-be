package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	Database struct {
		DSN string
	}
	Media struct {
		ResourceRoot  string
		GeneratedRoot string
		FFprobePath   string
		FFmpegPath    string
	}
	Storage struct {
		Mode  string
		MinIO struct {
			Endpoint         string
			AccessKey        string
			SecretKey        string
			Bucket           string
			Prefix           string
			UseSSL           bool
			URLExpiryMinutes int
		}
	}
}
