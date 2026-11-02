import { computed, ref } from 'vue'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import en from 'element-plus/es/locale/lang/en'

const messages = {
  en: {
    navDashboard: 'Dashboard', navDramas: 'Dramas', navClips: 'Clips',
    navAccounts: 'Accounts', navNewTask: 'New Task', navTasks: 'Tasks', logout: 'Log out',

    signIn: 'Sign in', signUp: 'Create account', email: 'Email', password: 'Password',
    haveAccount: 'Already have an account? Sign in', noAccount: 'No account? Register',

    refresh: 'Refresh', del: 'Delete', cancel: 'Cancel', confirm: 'Confirm', copy: 'Copy',
    copied: 'Copied', status: 'Status', actions: 'Actions', createdAt: 'Created',
    save: 'Save', close: 'Close', submit: 'Submit', search: 'Search', all: 'All',

    dashHello: 'Creator workspace', dashAccounts: 'TikTok accounts', dashClips: 'Clips',
    dashActive: 'Active tasks', dashDelivered: 'Delivered (draft)', dashFailed: 'Failed',
    dashEmpty: 'Connect a TikTok account and create your first publish task.',

    dramasTitle: 'Drama library', episodes: 'Episodes', slice: 'Slice',
    segSec: 'Segment length (sec)', maxCount: 'Max clips', startSlice: 'Start slicing',
    sliceHint: 'Cuts are stream-copied (fast, keyframe aligned). Clips land in your library.',
    sliceStarted: 'Slice job queued — watch the Clips page.', duration: 'Duration',

    clipsTitle: 'Clip library', upload: 'Upload videos', uploaded: 'Uploaded',
    size: 'Size', sliceJobs: 'Slice jobs', clipCount: 'Clips made',
    jobQueued: 'queued', jobRunning: 'running', jobDone: 'done', jobFailed: 'failed',

    connect: 'Connect TikTok', connectTip: 'Opens TikTok authorization in a new tab.',
    displayName: 'Account', openID: 'Open ID', accOK: 'Connected', accDisconnected: 'Reconnect needed',
    authLinks: 'Authorization links (7 days)', createLink: 'Generate link', linkTitle: 'Link label',
    linkHint: 'Anyone who authorizes through this link adds their account to YOUR matrix.',
    expires: 'Expires in', useCount: 'Joins', usedBy: 'Last joined by',
    expired: 'expired', revoke: 'Revoke',

    stepAccounts: 'Accounts', stepClips: 'Clips', stepSettings: 'Settings & publish',
    pickAccounts: 'Select accounts', pickClips: 'Select clips',
    rule: 'Allocation rule', ruleOneToOne: 'One video per account',
    ruleAllVideos: 'Every video to every account', ruleRandomEven: 'Random even spread',
    loopWhenShort: 'Cycle clips when there are fewer clips than accounts',
    maxRepeat: 'Max repeats per video', caption: 'Caption', captionHint: '{clip} = clip title',
    privacy: 'Visibility', scheduleAt: 'Submit', scheduleNow: 'Immediately', scheduleLater: 'At time',
    disableComment: 'Disable comments', disableDuet: 'Disable duet', disableStitch: 'Disable stitch',
    previewTotal: 'Posts to create', createTask: 'Create task', next: 'Next', prev: 'Back',
    taskName: 'Task name', selected: 'selected',

    taskRule: 'Rule', taskTotal: 'Posts', taskStatus: 'Status', viewPosts: 'Details',
    cancelTask: 'Cancel task', cancelTaskConfirm: 'Cancel everything not yet uploaded?',
    check: 'Mark reviewed', checked: 'Reviewed',
    stScheduled: 'Scheduled', stQueued: 'Queued', stUploading: 'Uploading',
    stProcessing: 'Processing', stDelivered: 'Delivered (draft)', stPublished: 'Published',
    stFailed: 'Failed', stCanceled: 'Canceled', stPendingCheck: 'Review pending',
    tsScheduled: 'scheduled', tsRunning: 'running', tsDone: 'done', tsCanceled: 'canceled',
    account: 'Account', clip: 'Clip',

    loginFailed: 'Login failed', regFailed: 'Registration failed',
  },

  zh: {
    navDashboard: '首页', navDramas: '剧目库', navClips: '素材库',
    navAccounts: '账号管理', navNewTask: '新建任务', navTasks: '任务中心', logout: '退出登录',

    signIn: '登录', signUp: '注册账号', email: '邮箱', password: '密码',
    haveAccount: '已有账号?去登录', noAccount: '没有账号?去注册',

    refresh: '刷新', del: '删除', cancel: '取消', confirm: '确定', copy: '复制',
    copied: '已复制', status: '状态', actions: '操作', createdAt: '创建时间',
    save: '保存', close: '关闭', submit: '提交', search: '搜索', all: '全部',

    dashHello: '创作者工作台', dashAccounts: 'TikTok 账号', dashClips: '素材',
    dashActive: '进行中任务', dashDelivered: '已送达(草稿)', dashFailed: '失败',
    dashEmpty: '先连接 TikTok 账号,再创建第一个发布任务。',

    dramasTitle: '剧目库', episodes: '剧集', slice: '切片',
    segSec: '片段时长(秒)', maxCount: '最多切几段', startSlice: '开始切片',
    sliceHint: '流拷贝切片(快,按关键帧对齐)。切好的片段自动进素材库。', duration: '时长',

    clipsTitle: '素材库', upload: '上传视频', uploaded: '已上传',
    size: '大小', sliceJobs: '切片任务', clipCount: '产出片段',
    jobQueued: '排队中', jobRunning: '切片中', jobDone: '完成', jobFailed: '失败',

    connect: '连接 TikTok', connectTip: '在新标签页打开 TikTok 授权。',
    displayName: '账号', openID: 'Open ID', accOK: '已连接', accDisconnected: '需重新授权',
    authLinks: '账号授权链接(7 天有效)', createLink: '生成链接', linkTitle: '链接备注',
    linkHint: '谁通过这条链接完成授权,谁的号就进你的矩阵。',
    expires: '剩余有效', useCount: '已入号', usedBy: '最近入号',
    expired: '已过期', revoke: '撤销',

    stepAccounts: '选账号', stepClips: '选素材', stepSettings: '设置并发布',
    pickAccounts: '选择账号', pickClips: '选择素材',
    rule: '分配规则', ruleOneToOne: '每个账号一个视频',
    ruleAllVideos: '每个账号发全部视频', ruleRandomEven: '随机均匀分配',
    loopWhenShort: '视频不足账号数时循环使用',
    maxRepeat: '单视频最多重复', caption: '文案', captionHint: '{clip} 会替换为素材标题',
    privacy: '可见范围', scheduleAt: '提交时间', scheduleNow: '立即', scheduleLater: '定时',
    disableComment: '关闭评论', disableDuet: '关闭合拍', disableStitch: '关闭拼接',
    previewTotal: '将创建的发布数', createTask: '创建任务', next: '下一步', prev: '上一步',
    taskName: '任务名称', selected: '已选',

    taskRule: '规则', taskTotal: '发布数', taskStatus: '状态', viewPosts: '详情',
    cancelTask: '取消任务', cancelTaskConfirm: '取消所有尚未上传的发布?',
    check: '标记已核对', checked: '已核对',
    stScheduled: '待定时提交', stQueued: '排队中', stUploading: '上传中',
    stProcessing: '平台处理中', stDelivered: '已送达(草稿)', stPublished: '已发布',
    stFailed: '失败', stCanceled: '已取消', stPendingCheck: '待核对',
    tsScheduled: '定时中', tsRunning: '进行中', tsDone: '已完成', tsCanceled: '已取消',
    account: '账号', clip: '素材',

    loginFailed: '登录失败', regFailed: '注册失败',
  },
}

export const locale = ref(localStorage.getItem('viewly_studio_lang') ||
  (navigator.language && navigator.language.startsWith('zh') ? 'zh' : 'en'))

export function t(key, params) {
  const dict = messages[locale.value] || messages.en
  let s = dict[key] ?? messages.en[key] ?? key
  if (params) for (const [k, v] of Object.entries(params)) s = s.split('{' + k + '}').join(v)
  return s
}

export function setLocale(l) {
  locale.value = l
  localStorage.setItem('viewly_studio_lang', l)
}

export const epLocale = computed(() => (locale.value === 'zh' ? zhCn : en))
