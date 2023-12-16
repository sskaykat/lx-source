/**
 * @name Lx-Custom-Source
 * @description Client
 * version 1.0.1
 * @author Zxwy
 * @homepage https://github.com/ZxwyWebSite/lx-source
 */

// 脚本配置
const version = '1.0.2' // 脚本版本
const apiaddr = 'http://127.0.0.1:1011/' // 服务端地址，末尾加斜杠
const apipass = '' // 验证密钥，由服务端自动生成 '${apipass}'
const devmode = true // 调试模式

// 常量 & 默认值
const { EVENT_NAMES, request, on, send } = window.lx ?? globalThis.lx
const defaults = {
  type: 'music', // 目前固定为 music
  actions: ['musicUrl'], // 目前固定为 ['musicUrl']
  qualitys: ['128k', '320k', 'flac', 'flac24bit'], // 当前脚本的该源所支持获取的Url音质，有效的值有：['128k', '320k', 'flac', 'flac24bit']
}
const defheaders = {
  'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.84 Safari/537.36 HBPC/12.1.2.300',
  'Accept': 'application/json, text/plain, */*',
  'X-LxM-Auth': apipass,
}
const conf = {
  api: {
    addr: apiaddr, // 服务端地址，末尾加斜杠
    pass: apipass, // 验证密钥，由服务端自动生成 '${apipass}'
    glbv: 'v1'     // 大版本号
  },
  info: {
    version: version, // 脚本版本
    devmode: devmode, // 调试模式
  },
}

const httpRequest = (url, options) => new Promise((resolve, reject) => {
  options.headers = { ...defheaders, ...options.headers } // 添加默认请求头
  request(url, options, (err, resp) => {
    if (err) return reject(err)
    resolve(resp.body)
  })
})

const musicUrl = async (source, info, quality) => {
  const id = info.hash ?? info.copyrightId ?? info.songmid // 音乐id kg源为hash, mg源为copyrightId
  const query = `${source}/${id}/${quality}`; console.log('创建任务: %s, 音乐信息: %O', query, info)
  const body = await httpRequest(`${apiaddr}link/${query}`, { method: 'get' }); console.log('返回数据: %O', body)
  return body.data != '' ? body.data : Promise.reject(body.msg) // 没有获取到链接则将msg作为错误抛出
}

// 注册应用API请求事件
// source 音乐源，可能的值取决于初始化时传入的sources对象的源key值
// info 请求附加信息，内容根据action变化
// action 请求操作类型，目前只有musicUrl，即获取音乐URL链接，
//    当action为musicUrl时info的结构：{type, musicInfo}，
//        info.type：音乐质量，可能的值有128k / 320k / flac / flac24bit（取决于初始化时对应源传入的qualitys值中的一个），
//        info.musicInfo：音乐信息对象，里面有音乐ID、名字等信息
on(EVENT_NAMES.request, ({ source, action, info }) => {
  // 回调必须返回 Promise 对象
  switch (action) {
    // action 为 musicUrl 时需要在 Promise 返回歌曲 url
    case 'musicUrl':
      return musicUrl(source, info.musicInfo, info.type).catch(err => {
        console.log('发生错误: %o', err)
        return Promise.reject(err)
      })
  }
})

// 脚本初始化 (目前只有检查更新)
const init = () => {
  'use strict';
  console.log('初始化脚本, 版本: %s, 服务端地址: %s', version, apiaddr)
  var stat = false; var msg = ''; var updUrl = ''; var sourcess = {}
  httpRequest(apiaddr, { method: 'get' })
    .catch((err) => { msg = '初始化失败: ' + err ?? '连接服务端超时'; console.log(msg) })
    .then((body) => {
      if (!body) { msg = '初始化失败：' + '无返回数据'; return }
      console.log('获取服务端数据成功: %o', body)
      // 检查Api大版本
      if (body.msg != `Hello~::^-^::~${conf.api.glbv}~`) {
        msg = 'Api大版本不匹配，请检查服务端与脚本是否兼容！'; return
      }
      // 检查脚本更新
      const script = body.script // 定位到Script部分
      const lv = version.split('.'); const rv = script.ver.split('.') // 分别对主次小版本检查更新
      for (var i = 0; i < 3; i++) {
        if (lv[i] < rv[i]) {
          console.log('发现更新, 版本: %s, 信息: %s, 地址: %s, 强制推送: %o', script.ver, script.log, script.url, script.force)
          msg = `${script.force ? '强制' : '发现'}更新：` + script.log; updUrl = script.url; if (script.force) return; break
        }
      }
      // 激活可用源
      const source = body.source // 定位到Source部分
      // const defs = { type: 'music', actions: ['musicUrl'] }
      Object.keys(source).forEach(v => {
        if (source[v] == true) {
          sourcess[v] = {
            name: v,
            ...defaults,
            // ...defs, qualitys: source[v].qualitys, // 支持返回音质时启用 使用后端音质表
          }
        }
      })
      // 完成初始化
      stat = true
    })
    .finally(() => {
      // 脚本初始化完成后需要发送inited事件告知应用
      send(EVENT_NAMES.inited, {
        status: stat, // 初始化成功 or 失败 (初始化失败不打开控制台, 使用更新提示接口返回信息)
        openDevTools: stat ? devmode : false, // 是否打开开发者工具，方便用于调试脚本 'devmode' or 'stat ? devmode : false'
        sources: sourcess, // 使用服务端源列表
        // sources: { // 当前脚本支持的源
        //   wy: { name: '网易音乐', ...defaults, },
        //   mg: { name: '咪咕音乐', ...defaults, },
        //   kw: { name: '酷我音乐', ...defaults, },
        //   // kg: { name: '酷狗音乐', ...defaults, }, // 暂不支持，仅供换源
        // },
      })
      // 发送更新提示
      if (msg) send(EVENT_NAMES.updateAlert, { log: '提示：' + msg, updateUrl: updUrl ? apiaddr + updUrl : '' })
    })
}

console.log('\n     __      __  __      ______  ______  __  __  ____    ______  ______\n    / /     / / / /     / ____/ / __  / / / / / / __ \\  / ____/ / ____/\n   / /     / /_/ / __  / /___  / / / / / / / / / /_/ / / /     / /___\n  / /      \\_\\ \\  /_/ /___  / / / / / / / / / /  ___/ / /     / ____/\n / /___  / / / /     ____/ / / /_/ / / /_/ / / / \\   / /___  / /___\n/_____/ /_/ /_/     /_____/ /_____/ /_____/ /_/ \\_\\ /_____/ /_____/\n=======================================================================\n')
init() // 启动!!!