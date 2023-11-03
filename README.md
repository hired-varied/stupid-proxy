# A stupid proxy

A proxy that's stupid enough to allow you audiut it with a cup of coffee.

这是一个笨拙的代理程序，代码没几行，所以你可以很容易审计所有代码，目前服务端已经完成。

这个仓库的代码只有核心的代理逻辑，并且设计之初就打算使用 Treafik 作为接入边界。
核心的代理逻辑就是一个非常简单的 HTTP/HTTPS 代理，支持最朴素的用户名密码验证，代理验证默认不会显式返回给客户端（这样可以避免检测）。
但是考虑到如果直接作为 Chrome 的代理使用，还是需要一个触发器，所以这里留下了一个 URL，当客户端访问这个 URL 的时候会触发代理请求（后话）。
目前这种代理的方式是有 TLS over TLS 特征的（可以被检测），这个项目本身倒是可以支持 HTTP2 连接复用同一个 TCP 连接，可能某种程度上可能可以解决。

# 部署指南

0. 你需要有一台 Linux 服务器，一个域名，并把域名指向你的服务器，开放 `80` 和 `443` 端口。

1. 安装 docker：

```shell
sudo apt update
sudo apt install docker.io docker-compose -y
```

2. 前往任意目录，当然最好新开一个目录，保存以下内容到 `docker-compose.yaml`：

```yaml
version: "3"

services:
  traefik:
    image: traefik:latest
    restart: always
    command:
      - "--global.sendAnonymousUsage=false"
      - "--providers.docker"
      - "--providers.docker.exposedByDefault=false"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.web.http.redirections.entrypoint.to=websecure"
      - "--entrypoints.web.http.redirections.entrypoint.scheme=https"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.tmhttpchallenge.acme.httpchallenge=true"
      - "--certificatesresolvers.tmhttpchallenge.acme.httpchallenge.entrypoint=web"
      - "--certificatesresolvers.tmhttpchallenge.acme.email=${LETSENCRYPT_EMAIL}"
      - "--certificatesresolvers.tmhttpchallenge.acme.storage=/etc/acme/acme.json"
    ports:
      - 80:80
      - 443:443
    volumes:
      - ./acme/:/etc/acme/
      - /var/run/docker.sock:/var/run/docker.sock:ro

  stupid-proxy:
    image: ghcr.io/hired-varied/stupid-proxy:latest
    restart: always
    volumes:
      - ./config.yaml:/config.yaml
    expose:
      - "3000"
    labels:
      - "traefik.enable=true"
      - "traefik.tcp.services.stupid-proxy.loadbalancer.proxyprotocol.version=2"
      - "traefik.tcp.routers.stupid-proxy.rule=HostSNI(`${PROXY_FQDN}`)"
      - "traefik.tcp.routers.stupid-proxy.entrypoints=websecure"
      - "traefik.tcp.routers.stupid-proxy.tls"
      - "traefik.tcp.routers.stupid-proxy.tls.certresolver=tmhttpchallenge"
    cap_drop:
      - all
```

3. 保存以下内容到 `.env`（请根据实际情况需要修改）：

```env
LETSENCRYPT_EMAIL=your-email@domain.com
PROXY_FQDN=your.domain.com
```

4. 保存以下内容到 `config.yaml` （注意修改用户名密码，触发 URL 到不为人知的值）：

```yaml
upstream_addr: "http://example.com" # 想伪装的目标
listen_addr: 0.0.0.0:3000
auth_trigger_path: /trigger-is-a-secret-path
auth:
  username1: password1
  username2: password2
```

5. 启动：

```shell
sodo docker-compose up -d
```

检查日志：

```shell
sodo docker-compose logs -f
```

备份：备份这个目录下的所有文件即可。

6. 使用支持的客户端进行连接，例如 iOS 的 Shadowrocket，代理类型选 HTTPS / HTTP2 都可以；Chrome / FireFox 也是可以的，但是需要写一个插件出发代理认证请求（这里还未提供）。

