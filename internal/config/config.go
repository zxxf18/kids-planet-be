package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	PublicBasePath string
	Database       struct {
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
			Prefix           string
			UseSSL           bool
			URLExpiryMinutes int
			Buckets          struct {
				Audio  string
				Video  string
				Lyric  string
				Poster string
			}
		}
	}
}
