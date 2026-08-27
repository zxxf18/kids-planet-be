# kids-planet-be

童声星球后端服务，基于 Go、go-zero、MySQL、MinIO 和 ffprobe/ffmpeg，提供媒体扫描、检索、播放地址与诊断接口。

## 本地运行

1. 使用 `deploy/sql/schema.sql` 初始化 MySQL。
2. 复制 `etc/backend.example.yaml` 为 `etc/backend.local.yaml` 并填写本地数据库及 MinIO 配置。
3. 将素材放入以下目录；仓库只保留空目录占位文件，不包含任何音频、视频、歌词或封面：

   ```text
   resources/
     song/      *.mp3
     video/     *.mp4
     lyrics/    *.jpg
     poster/    *.jpg
   ```

4. 启动服务：

   ```bash
   go run ./cmd/server -f etc/backend.local.yaml
   ```

5. 扫描素材：

   ```bash
   curl -X POST -H 'Content-Type: application/json' -d '{}' \
     http://127.0.0.1:8888/api/v1/admin/scan
   ```

## 验证

```bash
go test ./...
```

## 镜像

```bash
docker build -t kids-planet-be .
cp etc/backend.docker.yaml etc/backend.docker.local.yaml
# 编辑 backend.docker.local.yaml，填写容器网络中的 MySQL/MinIO 地址和凭据
docker run --rm -p 8888:8888 \
  -v "$PWD/resources:/data/resources:ro" \
  -v "$PWD/etc/backend.docker.local.yaml:/app/etc/backend.yaml:ro" \
  kids-planet-be
```

生产环境应通过外部配置注入数据库与 MinIO 凭据，不要把真实凭据写入镜像或仓库。管理接口 `/api/v1/admin/*` 应限制在可信内网。

## License

GNU Affero General Public License v3.0，详见 `LICENSE`。
