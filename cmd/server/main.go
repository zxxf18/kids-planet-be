package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zxxf18/kids-planet-be/internal/config"
	"github.com/zxxf18/kids-planet-be/internal/httpapi"
	"github.com/zxxf18/kids-planet-be/internal/media"
	"github.com/zxxf18/kids-planet-be/internal/storage"
	"github.com/zxxf18/kids-planet-be/internal/store"
)

var configFile = flag.String("f", "etc/backend.local.yaml", "config file")

func main() {
	flag.Parse()
	var cfg config.Config
	conf.MustLoad(*configFile, &cfg, conf.UseEnv())
	database, err := store.NewMySQL(cfg.Database.DSN)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	prober, err := media.NewProber(cfg.Media.ResourceRoot, cfg.Media.GeneratedRoot, cfg.Media.FFprobePath, cfg.Media.FFmpegPath)
	if err != nil {
		log.Fatal(err)
	}
	assets, err := storage.NewAssets(cfg, prober)
	if err != nil {
		log.Fatal(err)
	}
	scanner := media.NewScanner(assets, database)
	server := rest.MustNewServer(cfg.RestConf)
	defer server.Stop()
	httpapi.New(database, prober, scanner, assets, cfg.PublicBasePath).Register(server)
	fmt.Printf("kids media API listening on %s:%d\n", cfg.Host, cfg.Port)
	server.Start()
}
