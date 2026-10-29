# TKPostAI 拆解笔记(自建参考)

> 目的:自建一个面向 TikTok 渠道的短剧分销/发布平台,以 TKPostAI 为功能参考。
> 原则:只对标功能,不使用对方的名称、文案、素材、接口实现。

## 一、TKPostAI 是什么(我们实际接触到的)

面向短剧出海的分销/推广工具:创作者(或小 B 工作室)登录后选剧、拿推广链接、
进剪辑、发布到 TikTok 获客变现。它自己不持有剧,剧由平台方(我们这种)提供。

### 产品模块(按我们观察到的面)

1. **账号体系**
   - username/password 注册(用户名 ≥ 8 字符)
   - 登录发 Bearer token,2 小时有效
   - 我们作为集成方给用户自动开号(fox%07dt%d 规则),说明它支持纯 API 开号
2. **推广链接(drama links)**
   - 集成方拿 token 调接口,传 dramaid/title,得到一条链接
   - 链接直达已登录的创作工作台(免手动登录)
   - 链接有使用次数概念(remaining_uses)
   - 有防重放(同链接短时间重复访问会 REPLAY_DETECTED,15 分钟内可复用缓存)
3. **创作者工作台**
   - 可推广剧目列表
   - 从工作台进"AI 视频剪辑"——跳转到 DramaCut(带 tenantId/externalUserId 凭证)
4. **对外集成 API(我们有完整文档)**
   - POST /api/integrations/auth/register {username, password}
   - POST /api/integrations/auth/login → bearer token(2h)
   - POST /api/integrations/drama/links {appid, token, dramaid, title}
   - 旧版 HMAC 签名接口已废弃 → 说明他们接口也在迭代,核心就是"开号+发链"
5. **商业模式(推断)**
   - 向平台方/分销商收订阅或按量费
   - 剪辑能力是外挂增值(DramaCut),不是自研

### 对我们最有用的启示

- 它把"平台方"和"创作者"分成两类账号,中间用**链接**衔接——这个结构值得照抄
- token 2h + 链接直达,是低摩擦设计,对转化很重要
- 剪辑可以外接,不必自研

## 二、我们已有的底子(Viewly 代码库)

| 需要的能力 | 现状 | 位置 |
|---|---|---|
| TikTok OAuth + 账号托管 | 80% 已建,休眠中 | backend tiktok 表/handler、admin TikTok 页 |
| 发布队列(串行 worker) | 已建雏形 | backend publish worker |
| 剧库 + 上传 + admin | 生产可用 | 现有平台 |
| 多租户 | 生产可用 | 现有平台 |
| 内容分发/镜像 | 生产可用(OSS 管线) | /tmp/oss_mirror.py + 集成接口 |
| 集成方 API 模式(Bearer + externalUserId) | 生产可用 | integration.go(我们自己就实现了同款) |
| 用户工作台前端 | 需要新写 | — |
| 计费/订阅 | 需要新写(第一版可免) | — |

TikTok 那套差的手工步骤(需要 TikTok 开发者账号操作,一次性):
1. 开发者后台给 App(FoxDrama Publisher)添加产品:Content Posting API
2. URL properties 里配 redirect:https://api.likeviewly.com/api/tiktok/callback

## 三、MVP 建议(先做最小闭环)

第一版只做一条链路:**注册 → 选剧 → 连 TikTok → 排队发布 → 看结果**

1. 分销账号注册/登录(直接复用现有用户体系+JWT)
2. 工作台页面:剧目列表 → 选剧 → 生成发布任务
3. TikTok 连接页(唤醒现有 OAuth 代码)
4. 发布队列跑通 + 发布列表展示状态
5. 数据:先只记"发布成功/失败 + TikTok 播放数(接口有)"

第一版不做:剪辑入口(以后外接)、订阅计费(内测免费)、多级分销、数据大屏。

## 四、法律与安全注意

1. 只对标功能,不用对方的品牌、截图、文案
2. TikTok 发布必须走官方 Content Posting API(我们已在做),不要碰非官方接口
3. 剪辑如外接第三方,凭证一律服务端存储,不进前端
