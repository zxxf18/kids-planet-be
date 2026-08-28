#!/bin/sh
set -eu

config_dir=/root/config/kids-planet
schema_file=${1:-$config_dir/schema.sql}
env_file=$config_dir/backend.env
mc_image=minio/mc:RELEASE.2021-04-22T17-40-00Z

test -f "$schema_file"
docker network inspect services-net >/dev/null
docker inspect mysql >/dev/null
docker inspect minio >/dev/null

install -d -m 700 "$config_dir"

# The schema is idempotent and is applied with the existing MySQL root account.
docker exec -i mysql sh -c 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD"' < "$schema_file"

db_password=$(openssl rand -hex 24)
db_sql=$(printf "%s" "CREATE USER IF NOT EXISTS 'kids_media'@'%' IDENTIFIED BY '$db_password'; ALTER USER 'kids_media'@'%' IDENTIFIED BY '$db_password'; GRANT SELECT, INSERT, UPDATE, DELETE ON kids_media.* TO 'kids_media'@'%'; FLUSH PRIVILEGES;")
printf '%s\n' "$db_sql" | docker exec -i mysql sh -c 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD"'

# Create a dedicated read-only MinIO user instead of exposing the administrator
# credentials to the application. The temporary client image is removed later.
minio_root_user=$(docker inspect minio --format '{{range .Config.Env}}{{println .}}{{end}}' | sed -n 's/^MINIO_ROOT_USER=//p')
minio_root_password=$(docker inspect minio --format '{{range .Config.Env}}{{println .}}{{end}}' | sed -n 's/^MINIO_ROOT_PASSWORD=//p')
test -n "$minio_root_user"
test -n "$minio_root_password"

app_access_key=kids-planet
app_secret_key=$(openssl rand -hex 32)
mc_host="http://$minio_root_user:$minio_root_password@minio:9000"

docker run --rm --network services-net \
  -e "MC_HOST_local=$mc_host" \
  -e "APP_ACCESS_KEY=$app_access_key" \
  -e "APP_SECRET_KEY=$app_secret_key" \
  --entrypoint /bin/sh "$mc_image" -c \
  'mc admin user add local "$APP_ACCESS_KEY" "$APP_SECRET_KEY" >/dev/null &&
   mc admin policy set local readonly user="$APP_ACCESS_KEY" >/dev/null'

umask 077
{
  printf 'KIDS_DB_DSN=kids_media:%s@tcp(mysql:3306)/kids_media?charset=utf8mb4&parseTime=true&loc=Local\n' "$db_password"
  printf 'KIDS_MINIO_ACCESS_KEY=%s\n' "$app_access_key"
  printf 'KIDS_MINIO_SECRET_KEY=%s\n' "$app_secret_key"
} > "$env_file"

docker rm -f kids-planet-be kids-planet-fe >/dev/null 2>&1 || true
docker run -d \
  --name kids-planet-be \
  --network services-net \
  --restart unless-stopped \
  --env-file "$env_file" \
  kids-planet-be:20260828-s3-kidstar >/dev/null
docker run -d \
  --name kids-planet-fe \
  --network services-net \
  --restart unless-stopped \
  kids-planet-fe:20260828-kidstar >/dev/null

docker image rm "$mc_image" >/dev/null 2>&1 || true
