package web

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kosmix.fr/streaming/engine/api/v1/admin"
	"kosmix.fr/streaming/engine/api/v1/auth"
	"kosmix.fr/streaming/engine/api/v1/browse"
	"kosmix.fr/streaming/engine/api/v1/dlrequest"
	"kosmix.fr/streaming/engine/api/v1/download"
	"kosmix.fr/streaming/engine/api/v1/iptv"
	"kosmix.fr/streaming/engine/api/v1/landing"
	"kosmix.fr/streaming/engine/api/v1/me"
	"kosmix.fr/streaming/engine/api/v1/metadata"
	"kosmix.fr/streaming/engine/api/v1/render"
	"kosmix.fr/streaming/engine/api/v1/trailer"
	engine "kosmix.fr/streaming/engine/app"

	"kosmix.fr/streaming/engine/api/v1/search"
	"kosmix.fr/streaming/engine/api/v1/share"
	"kosmix.fr/streaming/engine/api/v1/task"
	"kosmix.fr/streaming/engine/api/v1/torrents"
	"kosmix.fr/streaming/engine/api/v1/transcode"
	"kosmix.fr/streaming/engine/api/v1/upload"
	"kosmix.fr/streaming/engine/api/v1/watching"
	"kosmix.fr/streaming/engine/api/v1/watchlist"
)

func WebServer(db *gorm.DB, port string) {
	r := gin.Default()
	store := cookie.NewStore([]byte("anotherSecret"))
	r.Use(sessions.Sessions("mysession", store))
	r.MaxMultipartMemory = engine.Config.Limits.SeasonSize
	r.Use(func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Origin", engine.Config.Web.CrossOrigin)
		ctx.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, x-track, x-quality, x-current-time")
		ctx.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,")
		ctx.Header("Access-Control-Allow-Credentials", "true")
		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(204)
			return
		}
		session := sessions.Default(ctx)
		session.Options(sessions.Options{
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		})
	})
	r.Use(static.Serve("/", static.LocalFile("./build/", false)))
	r.NoRoute(func(ctx *gin.Context) { ctx.File("./build/index.html") })
	r.GET("/kill", func(ctx *gin.Context) { os.Exit(200) })
	r.POST("/api/metadata/update", func(ctx *gin.Context) { metadata.AssignFileToMedia(ctx, db) })
	r.GET("/api/metadata/clean", func(ctx *gin.Context) { metadata.ClearMoviesWithNoMediaAndNoTmdbId(ctx, db) })
	r.GET("/api/metadata/items", func(ctx *gin.Context) { metadata.GetUnAssignedMedias(ctx, db) })
	// to patch
	r.GET("/api/trailer", func(ctx *gin.Context) { trailer.HandleTrailerRequest(ctx, db) })
	r.GET("/api/download", func(ctx *gin.Context) { download.DownloadItem(ctx, db) })
	r.GET("/api/watchlist", func(ctx *gin.Context) { watchlist.WatchListEndpoint(ctx, db) })
	r.GET("/api/render", func(ctx *gin.Context) { render.RenderItem(ctx, db) })
	r.GET("/api/search", func(ctx *gin.Context) { search.MultiSearch(ctx, db) })
	r.GET("/api/browse", func(ctx *gin.Context) { browse.Browse(ctx, db) })
	r.GET("/api/home", func(ctx *gin.Context) { landing.Landing(db, ctx) })
	r.GET("/api/iptv", func(ctx *gin.Context) { iptv.ListIptv(ctx, db) })
	r.GET("/api/iptv/record/remove", func(ctx *gin.Context) { iptv.RemoveRecord(ctx, db) })
	r.POST("/api/iptv/record/add", func(ctx *gin.Context) { iptv.AddRecord(ctx, db) })
	r.GET("/api/iptv/ordered", func(ctx *gin.Context) { iptv.OrderedIptv(ctx, db) })
	r.GET("/api/iptv/logo", func(ctx *gin.Context) { iptv.Logo(ctx, db) })
	r.GET("/api/iptv/add", func(ctx *gin.Context) { iptv.AddIptv(ctx, db) })
	r.GET("/api/iptv/transcode", transcode.HeadersMiddleware(), func(ctx *gin.Context) { iptv.TranscodeIptv(ctx, db) })
	r.GET("/api/task", transcode.HeadersMiddleware(), func(ctx *gin.Context) { task.GetTask(db, ctx) })
	r.GET("/api/transcode", transcode.HeadersMiddleware(), func(ctx *gin.Context) { transcode.NewTranscoder(r, ctx, db) })
	r.GET("/api/transcode/:uuid/manifest", func(ctx *gin.Context) { transcode.TranscodeManifest(ctx, db) })
	r.GET("/api/transcode/stop/:uuid", func(ctx *gin.Context) { transcode.Stop(ctx, db) })
	r.GET("/api/transcode/segment/:uuid/:number", func(ctx *gin.Context) { transcode.TranscodeSegment(ctx, db) })
	r.GET("/api/transcode/:uuid/subtitle/:index", func(ctx *gin.Context) { transcode.TranscodeSubtitle(ctx, db) })
	r.POST("/api/transcode/convert", func(ctx *gin.Context) { transcode.Convert(db, ctx, r) })
	r.GET("/api/transcode/convert/action", func(ctx *gin.Context) { transcode.Action(db, ctx, r) })
	r.GET("/api/transcode/options", func(ctx *gin.Context) { transcode.ConvertOptions(db, ctx, r) })
	r.GET("/api/login", func(ctx *gin.Context) { auth.Login(ctx, db) })
	r.GET("/api/logout", func(ctx *gin.Context) { auth.Logout(ctx) })
	r.GET("/api/torrents/.torrent", func(ctx *gin.Context) { torrents.TorrentFile(ctx, db) })
	r.POST("/api/torrents/add", func(ctx *gin.Context) { torrents.TorrentAdd(ctx, db) })
	r.GET("/api/torrents/search", func(ctx *gin.Context) { torrents.SearchTorrents(ctx, db) })
	r.GET("/api/torrents/available", func(ctx *gin.Context) { torrents.AvailableTorrent(ctx, db) })
	r.GET("/api/torrents/zip", func(ctx *gin.Context) { torrents.TorrentZipDownload(ctx, db) })
	r.GET("/api/continue", func(ctx *gin.Context) { watching.DeleteFromWatchingList(ctx, db) })
	r.GET("/api/me", func(ctx *gin.Context) { me.HandleMe(db, ctx) })
	r.POST("/api/upload", func(ctx *gin.Context) { upload.UploadFile(ctx, db) })
	r.GET("/api/scan", func(ctx *gin.Context) { admin.Rescan(ctx, db) })
	r.GET("/api/share/add", func(ctx *gin.Context) { share.AddShare(ctx, db) })
	r.GET("/api/share/get", func(ctx *gin.Context) { share.GetShare(ctx, db) })
	r.GET("/api/share/remove", func(ctx *gin.Context) { share.DeleteShare(ctx, db) })
	r.POST("/api/request/new", func(ctx *gin.Context) { dlrequest.NewDownloadRequest(db, ctx) })
	r.GET("/api/request/remove", func(ctx *gin.Context) { dlrequest.DeleteRequest(db, ctx) })
	// r.GET("/api/pty", func(ctx *gin.Context) { admin.PtyConnector(ctx, db) })
	r.GET("/api/admin/ffprobe/:file_id", func(ctx *gin.Context) { admin.GetFFprobe(ctx, db, r) })
	r.GET("/api/torrents/file", func(ctx *gin.Context) { torrents.TorrentFileDownload(ctx, db) })
	r.GET("/api/torrents/action", func(ctx *gin.Context) { torrents.TorrentAction(ctx, db) })
	r.GET("/api/torrents/storage", func(ctx *gin.Context) { admin.GetAvailablePaths(ctx, db) })
	r.GET("/api/admin/info", func(ctx *gin.Context) { admin.AdminInfo(ctx, db) })
	r.GET("/test", func(ctx *gin.Context) { ctx.JSON(200, gin.H{"message": "ok"}) })
	fmt.Println("Starting server on port " + (port))
	if engine.IsSsl() {
		r.RunTLS(":"+port, engine.Config.Cert.Cert, engine.Config.Cert.Key)
	} else {
		r.Run(":" + port)
	}
}
