# Viewly / FoxDrama 项目接管指南

> 更新：2026-09-11 · 最新提交 `ea7997a` · **⏸️ 项目已暂停（2026-09-11）：实例已停机、弹性 IP 已释放、旧机已删除；磁盘+S3 保留（~¥25/月）。恢复方法见文末 §10**

## 1. 项目组成

```
用户手机浏览器
   │
   ▼
Cloudflare (DNS+CDN+SSL, 免费版)
   ├── www.likeviewly.com   → H5 前端（Vue3，静态文件）
   ├── admin.likeviewly.com → 管理后台（Vue3 + Element Plus，静态文件）
   ├── api.likeviewly.com   → Go 后端（Gin，systemd 服务）
   ├── cdn.likeviewly.com   → S3 桶同名直连（视频/图片，CF 缓存规则：视频后缀 边缘1天/浏览器1小时）
   └── upload.likeviewly.com(DNS-only) → 阿里云香港中转(8.218.36.231, nginx 反代) → EC2
                                        （大陆上传加速专用，为什么走它：CF 免费版 100MB 请求体上限 + 大陆直连慢）

AWS EC2 (i-0725beea79ab9f1ac, 香港ap-east-1, EIP 95.41.49.139, t4g.small ARM 2GB — 2026-09-11 由 t3.small i-0fc3dcf22007a2776 迁移而来，旧机已停用待删除)
   ├── Go 二进制 /opt/viewly/server（systemd 服务名 viewly）
   ├── 配置     /opt/viewly/config.yaml（真实密钥，不入库）
   ├── MySQL    本机 127.0.0.1（库名 cascade-app，用户 viewly）
   ├── nginx    /etc/nginx/sites-enabled/{viewly, upload}
   └── 静态站   /var/www/h5, /var/www/admin

S3 桶 cdn.likeviewly.com（桶名=域名 → 虚拟主机寻址，免 CF 付费 Origin Rule）
   视频上传链路：后台浏览器 → 阿里云中转 → EC2(Go) → 16MB分块 → S3 Multipart
   上传完成自动 faststart 转封装（moov 前置，否则网页播放卡死）
```

## 2. 代码与仓库

- **GitHub**: https://github.com/liuzhen2017/viewly （main 分支即生产版本）
- `backend/`  Go 服务：cmd/server 启动；migrations/ 是 SQL 迁移（001-005 **已全部应用到生产库**）
- `frontend/` H5；`admin/` 管理后台；两边都是纯静态构建产物部署
- `backend/config.example.yaml` 配置模板（真实 config.yaml 被忽略，内容见 §5）

```bash
git clone https://github.com/liuzhen2017/viewly.git
```

## 3. 怎么连服务器（三选一）

| 方式 | 步骤 | 适合 |
|---|---|---|
| **A. AWS 网页终端（最快，零配置）** | AWS 控制台 → EC2 → 选中 i-0fc3dcf22007a2776 → 「连接」→ EC2 Instance Connect → 浏览器里直接开终端 | 临时运维 |
| B. 原有 SSH 密钥 | 原 Mac 上 `~/.ssh/viewly_hk` 这个私钥文件拷到你电脑的 `~/.ssh/`，然后 `ssh -i ~/.ssh/viewly_hk ubuntu@95.41.49.139`（密钥对名 viewly-hk；ARM 机的二进制要 GOARCH=arm64 编译）| 长期使用 |
| C. 你自己的新密钥（推荐） | 你电脑上 `ssh-keygen -t ed25519 -f ~/.ssh/viewly_hk` 生成，然后把 `cat ~/.ssh/viewly_hk.pub` 的内容追加到 EC2 的 `/home/ubuntu/.ssh/authorized_keys`（用方式 A 进去改） | 长期使用，不依赖旧机器 |

阿里云中转机：`ssh root@8.218.36.231`（密码问项目交接人；配置在 `/etc/nginx/sites-available/upload-relay`，备份同名 .bak）

## 4. 本地开发环境

```bash
# 需要：Go 1.21+、Node 18+、MySQL 8（或改 config 指向远程测试库）
cd backend
cp config.example.yaml config.yaml   # 填本地 MySQL；tiktok 段见 §5
go run ./cmd/server                  # :8080

cd ../frontend && npm i && npm run dev   # H5，vite 代理 /api → :8080
cd ../admin && npm i && npm run dev      # 后台，同样代理
```

本地登录后台：admin / admin123（**生产同密码，上线前必须改**，见 §6）

## 5. 生产 config.yaml 关键段（在 EC2 /opt/viewly/config.yaml）

```yaml
mysql:
  host: 127.0.0.1
  port: 3306
  user: viewly
  password: "（在服务器上查看：sudo cat /opt/viewly/config.yaml）"
  database: cascade-app
aws:
  region: ap-east-1
  bucket: cdn.likeviewly.com
  cdn_base: https://cdn.likeviewly.com
tiktok:
  client_key: ""      # ← 待填：TikTok 门户 FoxDrama Publisher 应用凭证
  client_secret: ""   # ← 待填
  redirect_url: https://api.likeviewly.com/api/tiktok/callback
```

数据库迁移：`mysql -u viewly -p -h 127.0.0.1 cascade-app < migrations/00X_xxx.sql`（新环境按序执行 001→005）

## 6. 部署流程（在任一台有代码的电脑）

```bash
# 后端（改了 Go 代码）
cd backend && GOOS=linux GOARCH=amd64 go build -o /tmp/server ./cmd/server
scp -i ~/.ssh/viewly_hk /tmp/server ubuntu@95.41.49.139:/tmp/server_new
ssh -i ~/.ssh/viewly_hk ubuntu@95.41.49.139 \
  "sudo mv /opt/viewly/server /opt/viewly/server.bak && sudo mv /tmp/server_new /opt/viewly/server \
   && sudo chmod +x /opt/viewly/server && sudo systemctl restart viewly && sudo systemctl is-active viewly"
# ⚠️ 重启前确认没有正在进行的视频上传（journalctl -u viewly | grep mp/chunk）

# H5 / 后台（改了前端）
cd frontend && npm run build     # 或 cd admin && npm run build
rsync -az --delete -e "ssh -i ~/.ssh/viewly_hk" dist/ ubuntu@95.41.49.139:/tmp/h5_new
ssh -i ~/.ssh/viewly_hk ubuntu@95.41.49.139 \
  "sudo rm -rf /var/www/h5_old && sudo mv /var/www/h5 /var/www/h5_old && sudo mv /tmp/h5_new /var/www/h5 && sudo chmod -R o+rX /var/www/h5"
# admin 同理，路径 /var/www/admin

# 验证部署是否生效：curl 页面看新的 chunk 哈希文件名，或 grep 新代码里的字符串字面量
# （minify 会改函数名，认字符串/路由 chunk 文件最可靠）
```

## 7. 安全待办（密钥都在聊天里贴过，尽快轮换）

- [ ] **admin/admin123 后台密码**（有真实流量前必须改）
- [ ] 阿里云 root 密码（8.218.36.231）
- [ ] Cloudflare API Token、GitHub PAT
- [ ] TikTok 开发者账号密码（d443y2515@outlook.com）

## 8. 当前状态与进行中事项

**已上线**：多租户短剧平台全功能（金币/广告解锁/批量上传/Watch流自动播放+选集/批量改价/轮播图跳转）、ads.txt（AdSense pub-7287113634373702 待 Google 审核）、上传中转链路（~7MB/s，断流自动重试不冻死）。

**进行中 — TikTok 托管发布**（后台「🎵 TikTok」页）：
- 平台侧已就绪：账号托管(OAuth)、素材库、一键发布队列、发布记录
- 待办（TikTok 门户 developers.tiktok.com，应用 FoxDrama Publisher / 7682083528905115655）：
  1. 凭证页点眼睛图标，复制 Client key / Client secret → 填入 config.yaml 的 tiktok 段 → 重启 viewly
  2. Products → Add products → 勾 Content Posting API → Save
  3. URL properties → 加回调 `https://api.likeviewly.com/api/tiktok/callback`
  4. （要公开发布才需要）补齐图标/描述/演示视频 → Submit for review；未过审只能发 SELF_ONLY（草稿）

**已知小坑**：S3 里有 ~8MB 的 e2e 测试残留文件可清；有 1 个早期视频 remux 失败（moov 仍后置）遇播不动时先怀疑它。

## 9. 常用排查命令

```bash
sudo journalctl -u viewly -f                          # 后端实时日志
sudo journalctl -u viewly --since '1 hour ago' | grep -E '\| [45][0-9]{2} \|'   # 错误请求
sudo journalctl -u viewly -b -1                       # 上次重启前的日志
# 上传卡死排查：看阿里云中转 access log 是否有 chunk 流入
# 播放卡死排查：先 ffprobe 该视频 moov 位置（moov 在尾部 = 需 remux）
```

## 10. 暂停与恢复（2026-09-11 起）

**当前状态**：服务器 i-0725beea79ab9f1ac 已停机（数据全保留在 30GB 磁盘）；弹性 IP 95.41.49.139 已释放（恢复时会分配新 IP）；旧实例已删除。仍在计费：磁盘 $2.4/月 + S3 ~$1.1/月。

**恢复步骤**（约 10 分钟）：
1. CloudShell：`aws ec2 start-instances --region ap-east-1 --instance-ids i-0725beea79ab9f1ac`
2. 分配并绑定新弹性 IP：`aws ec2 allocate-address` → `aws ec2 associate-address --instance-id i-0725beea79ab9f1ac --allocation-id <新id>`，记下新 IP
3. Cloudflare 后台把 www / api / admin / 根域 / upload 五条 A 记录的 95.41.49.139 改成新 IP（cdn 那条是 CNAME 不用动）
4. 阿里云中转（8.218.36.231）：改 `/etc/nginx/sites-available/upload-relay` 里的 proxy_pass IP 为新 IP，`nginx -s reload`
5. 验证：www/api/admin 打开、ads.txt 正常、后台传一个测试文件走通上传链路
6. 若 HTTPS 证书过期（超 90 天没运行）：SSH 上去跑 `sudo certbot renew --force-renewal`（需要 DNS 已指好）
