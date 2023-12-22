package router

import (
	"lx-source/src/caches"
	"lx-source/src/env"
	"lx-source/src/middleware/auth"
	"lx-source/src/middleware/dynlink"
	"lx-source/src/middleware/loadpublic"
	"lx-source/src/middleware/resp"
	"lx-source/src/middleware/util"
	"lx-source/src/sources"
	"net/http"

	"github.com/ZxwyWebSite/ztool"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// 载入路由
func InitRouter() *gin.Engine {
	r := gin.Default()
	// Gzip压缩
	if env.Config.Main.Gzip {
		r.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/file/"})))
	}
	// 源信息
	r.GET(`/`, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			`version`:   env.Version,         // 服务端程序版本
			`name`:      `lx-music-source`,   // 名称
			`msg`:       `Hello~::^-^::~v1~`, // Api大版本
			`developer`: []string{`Zxwy`},    // 开发者列表, 可在保留原作者的基础上添加你自己的名字?
			// 仓库地址
			`github`: `https://github.com/ZxwyWebSite/lx-source`,
			// 可用平台
			`source`: gin.H{
				`mg`: true,
				`wy`: true,
				`kg`: []string{`128k`, `320k`}, // 测试结构2, 启用时返回音质列表, 禁用为false
				`tx`: gin.H{ // "测试结构 不代表最终方式"
					`enable`:   false,
					`qualitys`: []string{`128k`, `320k`, `flac`, `flac24bit`},
				},
				`kw`: true,
			},
			// 自定义源脚本更新
			`script`: env.Config.Script,
		})
	})
	// 静态文件
	loadpublic.LoadPublic(r)
	// r.StaticFile(`/favicon.ico`, `public/icon.ico`)
	// r.StaticFile(`/lx-custom-source.js`, `public/lx-custom-source.js`)
	// 解析接口
	r.GET(`/link/:s/:id/:q`, auth.InitHandler(linkHandler)...)
	dynlink.LoadHandler(r)
	// r.GET(`/file/:t/:x/:f`, dynlink.FileHandler())
	// if cache, ok := caches.UseCache.(*localcache.Cache); ok {
	// 	r.Static(`/file`, cache.Path)
	// }
	// if env.Config.Cache.Mode == `local` {
	// 	r.Static(`/file`, env.Config.Cache.Local_Path)
	// }
	// 数据接口
	// r.GET(`/file/:t/:hq/:n`, func(c *gin.Context) {
	// 	c.String(http.StatusOK, time.Now().Format(`20060102150405`))
	// })
	// 暂不对文件接口进行验证 脚本返回链接无法附加请求头 只可在Get添加Query
	// g := r.Group(``)
	// {
	// 	g.Use(authHandler)
	// 	g.GET(`/link/:s/:id/:q`, linkHandler)
	// 	g.Static(`/file`, LocalCachePath)
	// }
	return r
}

// 数据返回格式
const (
	CacheHIT  = `Cache HIT`   // 缓存已命中
	CacheMISS = `Cache MISS`  // 缓存未命中
	CacheSet  = `Cache Seted` // 缓存已设置
)

// 外链解析
func linkHandler(c *gin.Context) {
	resp.Wrap(c, func() *resp.Resp {
		// 获取传入参数 检查合法性
		parms := util.ParaMap(c)
		// getParam := func(p string) string { return strings.TrimSuffix(strings.TrimPrefix(c.Param(p), `/`), `/`) } //strings.Trim(c.Param(p), `/`)
		s := parms[`s`]   //c.Param(`s`)   //getParam(`s`)   // source 平台 wy, mg, kw
		id := parms[`id`] //c.Param(`id`) //getParam(`id`) // sid 音乐ID wy: songmid, mg: copyrightId
		q := parms[`q`]   //c.Param(`q`)   //getParam(`q`)   // quality 音质 128k / 320k / flac / flac24bit
		env.Loger.NewGroup(`LinkQuery`).Debug(`s: %v, id: %v, q: %v`, s, id, q)
		if ztool.Chk_IsNilStr(s, q, id) {
			return &resp.Resp{Code: 6, Msg: `参数不全`} // http.StatusBadRequest
		}
		cquery := caches.NewQuery(s, id, q)
		// fmt.Printf("%+v\n", cquery)
		defer cquery.Free()
		// _, ok := sources.UseSource.Verify(cquery) // 获取请求音质 同时检测是否支持(如kw源没有flac24bit) qualitys[q][s]rquery
		// if !ok {
		// 	return &resp.Resp{Code: 6, Msg: `不支持的平台或音质`}
		// }

		// 查询内存
		clink, ok := env.Cache.Get(cquery.Query())
		if ok {
			if str, ok := clink.(string); ok {
				env.Loger.NewGroup(`MemCache`).Debug(`MemHIT [%q]=>[%q]`, cquery.Query(), str)
				if str == `` {
					return &resp.Resp{Code: 2, Msg: `MemCache Reject`} // 拒绝请求，当前一段时间内解析出错
				}
				return &resp.Resp{Msg: `MemCache HIT`, Data: str}
			}
		}
		// 查询缓存
		var cstat bool
		if caches.UseCache != nil {
			cstat = caches.UseCache.Stat()
		}
		sc := env.Loger.NewGroup(`StatCache`)
		if cstat {
			sc.Debug(`Method: Get, Query: %v`, cquery.Query())
			if link := caches.UseCache.Get(cquery); link != `` {
				env.Cache.Set(cquery.Query(), link, 3600)
				return &resp.Resp{Msg: CacheHIT, Data: link}
			}
		} else {
			sc.Debug(`Disabled`)
		}
		// 解析歌曲外链
		outlink, emsg := sources.UseSource.GetLink(cquery)
		if emsg != `` {
			if emsg == sources.Err_Verify { // Verify Failed: 不支持的平台或音质
				return &resp.Resp{Code: 6, Msg: ztool.Str_FastConcat(emsg, `: 不支持的平台或音质`)}
			}
			env.Cache.Set(cquery.Query(), ``, 600) // 发生错误的10分钟内禁止再次查询
			return &resp.Resp{Code: 2, Msg: emsg}
		}
		// 缓存并获取直链
		if outlink != `` && cstat {
			sc.Debug(`Method: Set, Link: %v`, outlink)
			if link := caches.UseCache.Set(cquery, outlink); link != `` {
				env.Cache.Set(cquery.Query(), link, 3600)
				return &resp.Resp{Msg: CacheSet, Data: link}
			}
		}
		// 无法获取直链 直接返回原链接
		env.Cache.Set(cquery.Query(), outlink, 1200)
		return &resp.Resp{Msg: CacheMISS, Data: outlink}
	})
}
