package engine

import (
	"fmt"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"kosmix.fr/streaming/kosmixutil"
)

// var ScanPaths []string

var db *gorm.DB = nil
var SystemInfo *kosmixutil.SystemInfoOut = &kosmixutil.SystemInfoOut{}

func GetDbConn() *gorm.DB {
	var connector gorm.Dialector
	switch Config.DB.Driver {
	case "sqlite":
		connector = sqlite.Open(Config.DB.Database)
	case "mysql":
		url := Config.DB.Username + ":" + Config.DB.Password + "@tcp(" + Config.DB.Host + ":" + Config.DB.Port + ")/" + Config.DB.Database + "?parseTime=True&charset=utf8mb4&loc=Local"
		connector = mysql.Open(url)
	default:
		panic("Unknown database driver")
	}
	dbClient, err := gorm.Open(connector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic(err)
	}
	return dbClient
}

func Init() *gorm.DB {
	dbClient := GetDbConn()
	go kosmixutil.GetDynamicData()
	if ss, err := kosmixutil.GetSystemInfo(); err == nil {
		SystemInfo = ss
	} else {
		panic(err)
	}
	dbClient.AutoMigrate(
		&StoragePathElement{},
		&StorageDbElement{},
		&MediaQuality{},
		&MediaQualityProfile{},
		&Torrent_File{},
		&Upload{},
		&DownloadRequest{},
		&Share{},
		&Task{},
		&IptvItem{},
		&FILE{},
		&User{},
		&Torrent{},
		&PROVIDER{},
		&MOVIE{},
		&EPISODE{},
		&TV{},
		&SEASON{},
		&GENRE{},
		&WATCHING{},
		&GENERATED_TOKEN{},
		&Record{},
	)
	admin := &User{
		NAME:                  "admin",
		EMAIL:                 "",
		TOKEN:                 "admin",
		ADMIN:                 true,
		CAN_DOWNLOAD:          true,
		CAN_CONVERT:           true,
		CAN_TRANSCODE:         true,
		CAN_ADD_FILES:         true,
		SHARES:                []Share{},
		CAN_UPLOAD:            true,
		CAN_DELETE:            true,
		CAN_EDIT:              true,
		MAX_TRANSCODING:       100000,
		TRANSCODING:           0,
		ALLOWED_UPLOAD_NUMBER: 20,
		CURRENT_UPLOAD_NUMBER: 0,
		CURRENT_UPLOAD_SIZE:   0,
		ALLOWED_UPLOAD_SIZE:   1000_000_000_0,
		REAL_UPLOAD_SIZE:      0,
	}

	dbClient.Create(admin)
	db = dbClient
	ProviderInit()
	CreateClient()
	InitIptv(dbClient)
	go InitRecords(dbClient)
	go InitIntervals(dbClient)
	os.RemoveAll(HLS_OUTPUT_PATH)
	os.MkdirAll(HLS_OUTPUT_PATH, os.ModePerm)
	fmt.Println("Scanning locations")
	InitStoragesConnection(Config.Locations)
	// Scan(dbClient)
	InitTorrents(dbClient)

	// VerifyDB(dbClient)
	return dbClient
}
