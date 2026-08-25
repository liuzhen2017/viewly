# Viewly — Short Drama Platform

自研海外短剧付费平台（对标 LingChuan / ReelShort 模式），前后端完全独立，无第三方源码依赖。

## 架构

```
Viewly/
├── backend/          Go 后端（Gin + GORM + MySQL + JWT）
│   ├── cmd/server    API 服务 (:8080)
│   ├── cmd/seed      种子数据
│   ├── migrations/   建表 SQL
│   └── static/       海报等静态资源
├── frontend/         用户端 H5（Vue 3 + Vite）(:5173)
└── admin/            管理后台（Vue 3 + Element Plus）(:5174)
```

数据库：MySQL 8（`cascade-app` 库，20 张表）。所有时间以 UTC 存储，业务日界（签到/任务重置）按 `config.yaml → app.timezone`（默认 Asia/Kolkata）计算。

## 多租户（共表模式）

平台支持多运营商站点（SaaS 矩阵），同一套库表通过 `tenant_id` 隔离：

- **租户解析**：前端从域名首段取 slug（生产 `demo.viewly.com` → demo；本地 `demo.localhost:5173`），随 `X-Tenant-Slug` 请求头发送；后端中间件解析（进程内缓存，免每请求查库），未知/停用租户直接 404/403。无请求头时落到默认租户 `main`
- **数据隔离**：dramas/episodes/categories/banners/coin_packages/vip_plans/orders/users/admin_users 全部带 `tenant_id`，所有查询按租户过滤；邮箱、设备号、管理员用户名的唯一性均为「租户内唯一」（同一邮箱可在不同租户站点各自注册）
- **令牌防跨租户**：用户/管理员 JWT 加载后校验所属租户与请求租户一致，跨站携带 token 返回 401
- **角色**：`role=super`（admin_users.tenant_id 为 NULL）为平台超管，可在任意租户上下文操作，并独占租户开通 API（`GET/POST /api/admin/tenants`）；租户管理员只能管理本租户内容、用户与订单
- **前端隔离**：H5 与 Admin 的 token 按租户分 key 存储；Admin 侧栏显示当前租户，超管可在「Tenants」页开新站/切换管理目标
- 本地体验：访问 `http://demo.localhost:5173/` 即演示租户；或在控制台 `localStorage.setItem('viewly_tenant','demo')` 后刷新

预置账号：平台超管 `admin / admin123`（任意租户域名可登录）；演示租户 `demo / demo123`（`demo.localhost:5174`）。

## 广告变现（Google AdSense / AdMob）

每租户独立配置自己的广告账号（收入归各自），平台只做路由与验证：

- **AdSense（H5 展示广告）**：后台「广告设置」填 Publisher ID 并启用后，H5 自动注入 AdSense 脚本；`GET /ads.txt` 按访问域名（Host 首段）动态输出该租户的授权行。审核通过前广告位空白属正常
- **激励视频（看广告得金币）**：两种发币模式
  - `client`（H5）：前端播完上报 `POST /api/rewards/watch-ad/complete`，服务端强制每日上限（默认 5 次/天 × 50 币），防刷有限但损失可控
  - `ssv`（App 打包后推荐）：AdMob 服务端验证回调 `GET /api/webhooks/admob/ssv`，用 Google 公钥（googleapis.com/admob/v1/publicKeys，24h 缓存）做 RSA-SHA256 验签，验签通过 + transaction_id 去重后才发币。AdMob 后台配置 callback URL，custom_data 传 `uid=<用户ID>&tenant=<slug>`
- **审核要求**：H5 已内置 `/privacy`、`/terms` 页面（AdSense 审核硬性要求），Profile 页底部有入口

## 快速启动

```bash
# 1. 后端（依赖 Go 1.24+）
cd backend
go run ./cmd/seed -config config.yaml     # 首次：建种子数据（admin/admin123 + 10部剧118集）
go run ./cmd/server -config config.yaml   # 启动 API :8080

# 2. 用户端 H5
cd frontend && npm install && npm run dev   # http://localhost:5173

# 3. 管理后台
cd admin && npm install && npm run dev      # http://localhost:5174  (admin/admin123)
```

生产部署：`go build -o server ./cmd/server`（单二进制），前端 `npm run build` 后由 Nginx 托管 `dist/`，API 反代到 :8080。**注意**：当前数据库在孟买（AWS），本机到库单查询 RTT ~500ms；生产环境务必与库同区域部署。

## 商业模型

- **金币解锁**：每部剧前 3 集免费，之后每集 40 金币（价格在后台按集可调），解锁永久有效
- **VIP 订阅**：周/月/季卡，有效期内所有剧集免费看
- **充值**：$1.99–$19.99 金币档位（含赠送币），订单落库 + 支付回调结算
- **增长飞轮**：游客零门槛进入 → 7 天签到（50/天，第 7 天 99）→ 每日任务（看 5/10/20 集、分享/点赞/收藏）→ 社媒关注任务（一次性 +100×4）→ 金币消耗 → 充值

## 关键 API

用户端（`/api`，Bearer JWT）：
- `POST /auth/guest` 设备游客登录（自动建号+60金币）· `POST /auth/register|login|bind` 邮箱体系
- `GET /home` 首页聚合（横幅/精选/新品/频道，6 路并发查询）· `GET /dramas/:id` 详情+逐集解锁态
- `GET /episodes/:id/play` 播放鉴权（免费/VIP/已解锁，否则 402+价格）· `POST /episodes/:id/unlock` 金币解锁
- `GET /rewards` · `POST /rewards/checkin` · `POST /rewards/tasks/:key/claim` 签到与任务
- `POST /orders` 下单 · `POST /orders/:no/mock-pay` 开发模拟支付 · `GET /wallet/transactions` 流水
- `POST /favorites|likes|follows/:dramaID` 收藏/点赞/追剧（点赞、收藏同时推进任务进度）

管理端（`/api/admin`）：仪表盘统计、剧目/剧集/分类/横幅/档位 CRUD、用户查询与调币、订单列表与手工核销。

## 资金安全设计

- 金币变动全部走 `service.Credit/Debit`：事务内 `SELECT ... FOR UPDATE` 锁用户行 + 不可变流水表（`coin_transactions` 记录变动后余额），并发下余额不可能为负、账实一致
- 订单结算 `settleOrder` 以 `UPDATE ... WHERE status='pending'` 的行数做幂等，重复回调不会重复发币
- 解锁幂等：`episode_unlocks` 上 `(user_id, episode_id)` 唯一键

## 上线前必做

1. **接入真实支付**：Stripe Checkout（网页）为主。实现 `POST /api/webhooks/stripe`（验签 → 匹配 `client_reference_id`=order_no → 复用 `settleOrder`），关闭 `server.mock_pay`。iOS App 内购需走 Apple IAP（现有字段已兼容 kind=vip）
2. **配置修改**：`jwt_secret` 换强随机值；数据库账密入环境变量；`app.timezone` 按目标市场调整
3. **视频与防盗链**：种子数据用的是公共演示视频。生产应接入 OSS+CDN+HLS 私有签名 URL（`episodes.video_url` 存源地址，播放时换签名临时地址）
4. **风控**：任务进度目前信任客户端上报的分享行为，上线需加设备指纹/频控；游客号绑定邮箱前的刷号行为需监控
5. **合规**：版权内容授权、目标市场内容分级、隐私政策/User Agreement 页面
