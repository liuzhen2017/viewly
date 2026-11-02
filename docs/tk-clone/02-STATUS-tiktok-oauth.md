# TikTok 接入进度 & 未完成任务(2026-10-29 晚)

> 给"在家继续"的快速恢复指南。凭证不放本文件(见服务器 /opt/viewly/config.yaml)。

## 一、已完成(全部验证过)

1. **TikTok 开发者账号(用户本人)**:294442437@qq.com,组织 FoxDrama(7701952612089857032,Owner)
2. **生产 App**:FoxDrama Publisher(7701956461361924114),client_key aw9bottugn5fj83u
   - Basic info 已配:图标(桌面 foxdrama-app-icon.png)/名称/Entertainment/描述/ToS/Privacy/Platforms=Web/Web URL
   - 域名验证通过:likeviewly.com DNS TXT(Cloudflare,记录 b03cf8ac287ed460133eedab0419e7cd)
   - 法务页:https://www.likeviewly.com/terms + /privacy(nginx alias /var/www/legal/)
   - **卡点**:审核表单要求"说明文+demo video"才能保存/提交;未提交前 URL properties 无 Add 按钮(新版门户锁死)
3. **旧 App(旧账号 d443y2515@outlook.com)**:7682083528905115655 / awlnhpf0ze6us41p,同样 pending,已被沙箱路线取代(凭证已弃用)
4. **⭐ 沙箱 App(当前激活路线)**:在旧 App 下建了沙箱 **dev-test**(7702011456522684434)
   - client_key **sbawe091fimle74vvb**(sb 开头),secret 在服务器 config.yaml(备份 /tmp/config.yaml.bak.sandbox)
   - 沙箱内已加产品并保存:Login Kit + Content Posting API + Webhooks(0 错误,Category=Entertainment)
   - Login Kit Redirect URI:https://api.likeviewly.com/api/tiktok/callback
   - Webhooks Callback:https://api.likeviewly.com/api/tiktok/webhook(Test event 已过,回 200)
   - Direct Post 开关:未开(默认 draft 模式,够用)
5. **后端代码(全部已部署 EC2)**:
   - callback 裸 GET 200 落地页 + HEAD 路由(TikTok 校验器发 HEAD)
   - webhook 端点(url_verification challenge 回显)
   - config 已切沙箱凭证,connect-url 生成正常
6. **服务器三道网络关全绿**:DNS 域名验证 / callback GET+HEAD 200 / webhook 200

## 二、未完成(按顺序)

1. **沙箱 Target User 授权**(就差这一步!):
   - 打开授权链接(用 admin 接口 /api/admin/tiktok/connect-url 生成,或直接用:
     `https://www.tiktok.com/v2/auth/authorize/?client_key=sbawe091fimle74vvb&redirect_uri=https%3A%2F%2Fapi.likeviewly.com%2Fapi%2Ftiktok%2Fcallback&response_type=code&scope=user.info.basic%2Cvideo.publish%2Cvideo.upload&state=t1.xxx`)
   - 需要真人登录 TikTok(测试号 foxdrama5 / @Bao1234,密码对但网页登录触发风控)
   - **可行路径 A**:手机 TikTok App 扫网页二维码(App 内:搜索页→扫码图标→相册可选码图)
   - **可行路径 B**:等风控冷却(几小时)后网页邮箱登录
   - 登录后自动到授权确认页 → Authorize → 回调绑号(admin 后台 🎵 TikTok 页可见)
2. **发布草稿测试**:绑号成功后,admin 选一集视频 → 发布(SELF_ONLY 草稿)→ TikTok App 里看到
3. **录 demo video**:上述全程录屏(审核要求沙箱环境演示,域名要和表单里一致)
4. **提交审核**:App review 表单(说明文已拟好,在聊天记录里;贴入 1000 字框)+ 传视频 → Submit
5. 审核通过后:生产 App 补 URL properties + 换回生产凭证(config.yaml)

## 三、TKPostAI 克隆 MVP(方案见 01-reference-teardown.md)

排期:MVP = 账号授权链接 + 批量发布任务(分配算法)+ 任务状态机 + ffmpeg 切片机;
不做 AI 剪辑(外接)、AI 润色、Facebook(后补)。

### 实施进度(2026-11-01,代码已完成待验证)

- `backend/migrations/011_tk_clone.sql`:user 归属列(accounts/clips/posts)+ tk_auth_states / tk_tasks / tk_slice_jobs 三新表(**未应用到生产库**)
- 分配引擎 `tk_allocate.go` + 单测(ONE_TO_ONE / ALL_VIDEOS / RANDOM_EVEN + 循环补足)
- OAuth:state 落库(直连 30min 单次 / 授权链接 7 天多人),链接授权的号挂**生成者**名下;旧 admin `t<tid>.xxx` 路径兼容不受影响
- 用户侧 API `tk_clone.go`(账号/链接/素材/切片/任务/核对);上传端点同款挂到用户组
- worker:启动时自动恢复队列、到期 scheduled→queued、SELF_ONLY 完成态记 delivered(待核对/已核对)、**tiktok.mock 模式**(OAuth 未通前全链路可演示)
- 切片机 `slicer.go`:CDN 下载→ffmpeg 流拷贝切片→S3→素材库,错误入库不静默;config 加 ffmpeg.path/tmp_dir
- 前端 `studio/`(新应用,Vue3+EP):登录注册/剧目库(切片入口)/素材库/账号管理(授权链接)/三步建任务向导/任务中心(状态机+核对);部署位 studio.likeviewly.com
- 部署:server-install.sh 加 studio server 块 + ffmpeg 安装;三个前端 RESERVED 子域加 studio
- **本地验证已过**(2026-11-01):`go build ./...` ✅ / `go vet ./internal/...` ✅ / 分配引擎 9 单测 ✅ / `studio npm run build` ✅(本机无 MySQL/Docker,带库的 e2e 挪到服务器做;本机 Go 1.17 太老,便携版 1.24.13 在 `G:\temp\temp\go-toolchain\`,用前 export GOROOT/PATH)
- **待办**:服务器部署(011 迁移 + arm64 构建 + DNS/nginx/certbot + ffmpeg)→ mock 模式 e2e → OAuth 恢复后关 mock 真发验证

## 三.五、⏸️ 暂停点(2026-11-01,授权卡在 redirect_uri)

- 扫码+登录成功(foxdrama5 网页登录态建立),但授权页报 "We couldn't log in with TikTok … redirect_uri"
- Login Kit(sandbox dev-test)里 URI 与请求逐字一致(`https://api.likeviewly.com/api/tiktok/callback`)仍报 → 首次实测,此前从未跑通过授权这步
- **高嫌疑结论**:域名验证是在新账号(294442437@qq.com)做的;沙箱挂旧账号(d443y2515@outlook.com),TikTok 运行时按 App 所属账号校验 redirect 域名。DNS 现状:根域 TXT `tiktok-developers-site-verification=LTOG6aOFvZh3e7IpSG7v1qZ3lB53JH1x`(新账号码,注意与本文档上文 b03cf8… 不同,中途换过)
- **Plan A(优先)**:旧账号后台(sandbox 或父 App 的 Basic info)找 Verify domain → 拿新 TXT 码 → Cloudflare 加第二条 TXT(Type TXT / Name @ / 内容原样;与现有条并存不冲突)→ portal 点 Verify → 重测授权链接(client_key sbawe091fimle74vvb,state=t1.xxx 格式任意,tenant 解析只看 t 后数字遇非数字截断)
- **Plan B**:找不到验证入口就走新账号(域名已验证)建沙箱 → config.yaml 换新 sb 凭证 → 重生成链接
- 重测注意:若原浏览器 tab 没关,foxdrama5 登录态还在,重开链接直达授权确认页免扫码;授权成功=302 到 admin.likeviewly.com/#/tiktok?connected=1
- 已重存过一次 Login Kit URI(手敲防空字符),保存后仍报,可排除"没存上/隐形字符"

## 四、易忘坑(都已踩过)

- TikTok 后台对 pending App 锁配置 → 一切走沙箱
- 校验器发 HEAD;回调裸 GET 必须 200(不能 302)
- 网页登录风控:自动化环境必触发"下载App";手动+已登录App扫码可过
- login 二维码 1-2 分钟过期;App 内可"从相册"扫保存的码图
- TikTok 登录尝试次数超限锁 ~15-30 分钟
- config.yaml 的 tiktok 段是沙箱凭证(值不带引号,sed 注意)
