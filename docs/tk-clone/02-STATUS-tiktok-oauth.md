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

## 四、易忘坑(都已踩过)

- TikTok 后台对 pending App 锁配置 → 一切走沙箱
- 校验器发 HEAD;回调裸 GET 必须 200(不能 302)
- 网页登录风控:自动化环境必触发"下载App";手动+已登录App扫码可过
- login 二维码 1-2 分钟过期;App 内可"从相册"扫保存的码图
- TikTok 登录尝试次数超限锁 ~15-30 分钟
- config.yaml 的 tiktok 段是沙箱凭证(值不带引号,sed 注意)
