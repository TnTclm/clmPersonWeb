// 统一请求封装：很多业务不用每次都写，只写一次在这就行

async function request(method, url, params, isForm = false) {
  // headers = 请求的“信封信息”
  // 注意：JSON 请求才声明 Content-Type；
  // 文件上传（FormData）绝对不能手动写 Content-Type——
  // 里面要带浏览器自动生成的“分界线(boundary)”，写死就传不上去
  const headers = {}
  if (!isForm) {
    headers['Content-Type'] = 'application/json'
  }
  // localStorage：浏览器自带的"小仓库"，存的东西关浏览器也不丢
  // 登录成功后存的 JWT 门禁卡就在这里，key 叫 'token'
  const token = localStorage.getItem('token')
  if (token) {
    headers.Authorization = 'Bearer ' + token
  }
  const res = await fetch(url, {
    method: method,
    headers: headers,
    // JSON 请求要转成字符串；FormData 直接给原信封
    body: params ? (isForm ? params : JSON.stringify(params)) : null,
  })
  // 401 = 未登录/令牌过期
  if (res.status === 401) {
    // 门禁卡作废：把令牌从本地小仓库里扔掉
    localStorage.removeItem('token')
    // 共享登录态也要一起清空——不清的话 header 右上角还挂着你的头像昵称，
    // 人已经被踢到登录页了，界面却显示"已登录"，前后矛盾
    // 同样用动态 import（me.js 里 import 了 api.js，静态引入会绕成环）
    const { me } = await import('@/stores/me')
    me.value = null
    // 动态 import 引入路由对象再跳转（避免和 router 循环依赖）
    const { default: router } = await import('@/router')
    router.push('/login')
    throw new Error('请先登录')
  }
  const data = await res.json()
  if (data.code !== 200) {
    throw new Error(data.msg || '请求失败')
  }
  return data
}

// 对外暴露的方法，语义一目了然
export const api = {
  get: (url) => request('GET', url),
  post: (url, body) => request('POST', url, body),
  put: (url, body) => request('PUT', url, body),
  del: (url) => request('DELETE', url),
  upload: (url, formData) => request('POST', url, formData, true), // 文件上传专用
}
