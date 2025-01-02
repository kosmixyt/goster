package engine

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"time"

	"io"
	"path/filepath"
	"sync"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	"gorm.io/gorm"
	"kosmix.fr/streaming/kosmixutil"
)

var client *torrent.Client

func CreateClient() {
	var err error
	defaultConfig := torrent.NewDefaultClientConfig()
	defaultConfig.DisableIPv6 = true
	// because it crashes
	defaultConfig.EstablishedConnsPerTorrent = 75

	defaultConfig.Seed = true
	client, err = torrent.NewClient(defaultConfig)

	if err != nil {
		panic(err)
	}
}

var TORRENT_ITEMS = []*GlTorrentItem{}

func GetTorrent(uuid uint) *GlTorrentItem {
	for _, item := range TORRENT_ITEMS {
		if item.DB_ITEM.ID == uuid {
			return item
		}
	}
	return nil
}
func DeleteTorrent(uuid uint) {
	for i, item := range TORRENT_ITEMS {
		if item.DB_ITEM.ID == uuid {
			TORRENT_ITEMS = append(TORRENT_ITEMS[:i], TORRENT_ITEMS[i+1:]...)
			return
		}
	}
}
func SearchAndDownloadTorrent(db *gorm.DB, user *User, movie *MOVIE, episode *EPISODE, season *SEASON, serie *TV, preferedTorrent_file_id string, reader chan *FILE, errorChan chan error, task *Task, progress func(string)) {
	if movie == nil && (serie == nil || episode == nil || season == nil) {
		reader <- nil
		errorChan <- errors.New("not enoth args")
		return
	}
	if !user.HaveOneUploadCredit() {
		reader <- nil
		errorChan <- errors.New("user reached upload limit")
		return
	}
	var preferedOrderedProvider []*Torrent_File
	if preferedTorrent_file_id != "" && preferedTorrent_file_id != "-1" {
		var preferedTorrent Torrent_File
		if err := db.Where("uuid = ?", preferedTorrent_file_id).First(&preferedTorrent); err.Error != nil {
			reader <- nil
			errorChan <- err.Error
			return
		}
		preferedOrderedProvider = append(preferedOrderedProvider, &preferedTorrent)
	} else {
		progress("Searching for torrent")
		tempTorrents, err := FindBestTorrentFor(serie, movie, season, episode, 1, GetMaxSize(GetType(serie, movie)))
		if err != nil {
			reader <- nil
			errorChan <- err
			return
		}
		preferedOrderedProvider = tempTorrents
	}

	/// tempppp
	// reader <- nil
	// errorChan <- errors.New("download are currently disabled")
	// return
	progress("Found " + strconv.Itoa(len(preferedOrderedProvider)) + " torrents")
	task.AddLog("Found", strconv.Itoa(len(preferedOrderedProvider)), "torrents")
	for i := 0; i < len(preferedOrderedProvider); i++ {
		progress("Downloading torrent " + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(preferedOrderedProvider)))
		item, err := AddTorrent(user, preferedOrderedProvider[i], GetMediaId(movie, serie), GetType(serie, movie), GetMaxSize(GetType(serie, movie)), progress)
		if err != nil {
			if err.UserLimit {
				reader <- nil
				errorChan <- err.Err
				return
			}
			task.AddLog("Error adding torrent", err.Err.Error())
			fmt.Println("Error adding torrent", err)
			continue
		}
		// user
		renderers := RegisterFilesInTorrent(item, movie, season)
		for i := 0; i < len(renderers); i++ {
			if movie != nil {
				if renderers[i].MOVIE_ID == movie.ID {
					task.AddLog("Found movie file", renderers[i].FILENAME)
					reader <- renderers[i]
					errorChan <- nil
					return
				}
			}
			if serie != nil && episode != nil && season != nil {
				if renderers[i].TV_ID == serie.ID && renderers[i].SEASON_ID == season.ID && renderers[i].EPISODE_ID == episode.ID {
					task.AddLog("Found episode file", renderers[i].FILENAME)
					reader <- renderers[i]
					errorChan <- nil
					return
				}
			}

		}
		panic("validate torrent must be wrong")
	}
	// panic("No torrent found")
	// reader <- &FileRender{ERR: errors.New("no torrent found")}
	reader <- nil
	errorChan <- errors.New("no torrent found")
}
func GetMediaId(movie *MOVIE, serie *TV) uint {
	if movie != nil {
		return movie.ID
	}
	return serie.ID
}
func AssignTorrentToMedia(db *gorm.DB, user *User, movie *MOVIE, season *SEASON, serie *TV, torrentItem *Torrent_File, task *Task) error {
	wg := sync.WaitGroup{}
	channel := make(chan map[string]*Torrent_File, 1)
	wg.Add(1)
	torrentItem.ValidateTorrent(&wg, channel, season, nil, movie, 0)
	wg.Wait()
	if len(channel) == 0 {
		return errors.New("no torrent found")
	}
	for success := range <-channel {
		if success != "" {
			return errors.New(success)
		}
	}
	if movie != nil {
		if movie.HasFile(nil) {
			return errors.New("movie already has file")
		}
	}
	if serie != nil {
		if season.HasFile() {
			return errors.New("season already has file")
		}
	}
	item, err := AddTorrent(user, torrentItem, GetMediaId(movie, serie), GetType(serie, movie), GetMaxSize(GetType(serie, movie)), func(s string) {})
	if err != nil {
		return err.Err
	}
	RegisterFilesInTorrent(item, movie, season)
	return nil
} // for tv shows db season.episodes must be preloaded
func RegisterFilesInTorrent(item *GlTorrentItem, movie *MOVIE, season *SEASON) []*FILE {
	files := item.Torrent.Files()
	renders := make([]*FILE, len(files))
	var episodes_numbers []int = make([]int, len(files))
	for i := 0; i < len(files); i++ {
		fileObject := files[i]
		fileObject.SetPriority(torrent.PiecePriorityNone)
		file := FILE{
			TORRENT_ID: item.DB_ITEM.ID,
			// PATH:               filepath.Dir(Config.Torrents.DownloadPath + fileObject.Path()),
			SUB_PATH:           filepath.Dir(fileObject.Path()),
			StoragePathElement: nil,
			IS_MEDIA:           false,
			FILENAME:           filepath.Base(fileObject.Path()),
			SIZE:               fileObject.Length(),
		}
		if !kosmixutil.IsVideoFile(fileObject.Path()) {
			db.Create(&file)
			renders[i] = &file
			continue
		}
		if movie != nil {
			file.IS_MEDIA = true
			file.MOVIE_ID = movie.ID
			renders[i] = &file
		}
		if season != nil {
			file.IS_MEDIA = true
			episode_number, _ := kosmixutil.GetEpisode(fileObject.Path())
			if episode_number == 0 {
				panic("Episode number is " + strconv.Itoa(episode_number))
			}
			if slices.Contains(episodes_numbers, episode_number) {
				panic("Episode number already found")
			}
			episodes_numbers = append(episodes_numbers, episode_number)
			file.SEASON_ID = season.ID
			file.TV_ID = season.TV_ID
			file.EPISODE_ID = season.GetEpisode(episode_number, true, db).ID
			renders[i] = &file
		}
		db.Save(&file)
	}
	return renders
}

type AddTorrentError struct {
	Err       error
	UserLimit bool
}

func AddTorrent(user *User, torrent *Torrent_File, mediaId uint, mediaType string, max_size int64, progress func(string)) (*GlTorrentItem, *AddTorrentError) {
	if !user.HaveUploadRight() {
		return nil, &AddTorrentError{errors.New("u:user reached upload limit"), true}
	}
	if !user.HaveOneUploadCredit() {
		return nil, &AddTorrentError{errors.New("u:user reached upload limit"), true}
	}
	bufferedTorrent, err := torrent.Load()
	if err != nil {
		return nil, &AddTorrentError{err, false}
	}
	metadata, err := metainfo.Load(bytes.NewReader(bufferedTorrent))
	if err != nil {
		fmt.Println("Error loading torrent file", err.Error())
		return nil, &AddTorrentError{err, false}
	}
	progress("Adding torrent to client")
	torrentElem, new, err := client.AddTorrentSpec(TorrentSpecFromMetaInfo(metadata, storage.NewFile(Config.Torrents.DownloadPath)))
	if err != nil || !new {
		return nil, &AddTorrentError{err, false}
	}
	fmt.Println("size of torrent is ", torrentElem.Length())
	progress("Waiting for torrent info")
	torrent_size := torrentElem.Length()
	if !user.CanUpload(torrent_size) {
		torrentElem.Drop()
		return nil, &AddTorrentError{errors.New("u:user reached upload limit"), true}
	}
	if torrent_size > max_size {
		torrentElem.Drop()
		return nil, &AddTorrentError{errors.New("u:torrent size is too big"), false}
	}
	user.Add_upload(torrent_size)
	torrentItemDb := Torrent{
		DL_PATH:      Config.Torrents.DownloadPath,
		PATH:         torrent.GetFileName(),
		Name:         torrentElem.Name(),
		InfoHash:     torrentElem.InfoHash().String(),
		FINISHED:     false,
		Progress:     0,
		PROVIER_NAME: torrent.PROVIDER,
		DOWNLOAD:     0,
		UPLOAD:       0,
		Size:         torrentElem.Length(),
		Paused:       false,
		USER_ID:      user.ID,
	}
	if tx := db.Create(&torrentItemDb); tx.Error != nil {
		torrentElem.Drop()
		user.RemoveDeleteCredit(torrent_size)
		return nil, &AddTorrentError{tx.Error, false}
	}
	<-torrentElem.GotInfo()
	progress("Torrent info received")
	item_tos := GlTorrentItem{
		OWNER_ID:       user.ID,
		Torrent:        torrentElem,
		DB_ITEM:        &torrentItemDb,
		MEDIA_UUID:     mediaId,
		MEDIA_TYPE:     mediaType,
		START_DOWNLOAD: 0,
		START_UPLOAD:   0,
		START:          time.Now(),
	}
	TORRENT_ITEMS = append(TORRENT_ITEMS, &item_tos)
	progress("Registering handlers")
	//go RegisterHandlers(item_tos.Torrent, &torrentItemDb, db, user)
	progress("Handlers registered")
	return &item_tos, nil
}
func FindBestTorrentFor(serie *TV, movie *MOVIE, season *SEASON, episode *EPISODE, min_seed int, max_size int64) ([]*Torrent_File, error) {
	start := time.Now()
	// cache
	if movie != nil {
		var torrent_files []*Torrent_File
		db.Where("movie_id = ?", movie.ID).Find(&torrent_files)
		if len(torrent_files) > 0 {
			return torrent_files, nil
		}
	}
	if season != nil {
		var torrent_files []*Torrent_File
		db.Where("season_id = ?", season.ID).Find(&torrent_files)
		if len(torrent_files) > 0 {
			return torrent_files, nil
		}
	}
	providers := SearchTorrent(serie, movie, season)
	uniqueProviders := make(map[string]*Torrent_File)
	for _, provider := range providers {
		if provider.SEED < min_seed {
			continue
		}
		uniqueProviders[provider.LINK] = provider
	}
	providers = make([]*Torrent_File, 0, len(uniqueProviders))
	for _, provider := range uniqueProviders {
		providers = append(providers, provider)
	}
	for _, provider := range providers {
		if movie != nil {
			provider.MOVIE_ID = &movie.ID
		}
		if season != nil {
			provider.SEASON_ID = &season.ID
			provider.TV_ID = &season.TV_ID
		}
	}
	if len(providers) > 15 {
		providers = providers[0:15]
	}
	var WaitGroup sync.WaitGroup
	channel := make(chan map[string]*Torrent_File, len(providers))
	for i, prov := range providers {
		WaitGroup.Add(1)
		go prov.ValidateTorrent(&WaitGroup, channel, season, episode, movie, i)
	}
	WaitGroup.Wait()
	close(channel)
	l := len(providers)
	secondFiltered := []*Torrent_File{}
	for i := 0; i < l; i++ {
		for success, dlItem := range <-channel {
			fmt.Println("Success", success == "", dlItem.LINK)
			if success == "" && dlItem.LINK != "" {
				secondFiltered = append(secondFiltered, dlItem)
				db.Save(&dlItem)
			}
		}
	}
	if len(secondFiltered) == 0 {
		return nil, errors.New("no torrent found")
	}
	sort.SliceStable(secondFiltered, func(i, j int) bool {
		return secondFiltered[i].SEED > secondFiltered[j].SEED
	})
	fmt.Println("Search time", time.Since(start), len(providers), len(secondFiltered))
	// le nombre de torrent valid peux varier d'une requete à l'autre, c'est due au faite que les torrents sont coupé à 15
	// et que les torrents sont validé après (5 torrents valid peuvent etre parmis les 15pris dans une requete, mais pas forcement dans une autre)
	return secondFiltered, nil
}
func TorrentSpecFromMetaInfoErr(mi *metainfo.MetaInfo, storage storage.ClientImpl) (*torrent.TorrentSpec, error) {
	info, err := mi.UnmarshalInfo()
	if err != nil {
		err = fmt.Errorf("unmarshalling info: %w", err)
	}
	return &torrent.TorrentSpec{
		Storage:                  storage,
		Trackers:                 mi.UpvertedAnnounceList(),
		InfoHash:                 mi.HashInfoBytes(),
		InfoBytes:                mi.InfoBytes,
		DisplayName:              info.Name,
		Webseeds:                 mi.UrlList,
		DisableInitialPieceCheck: true,
		DisallowDataUpload:       false,
		DisallowDataDownload:     false,
		DhtNodes: func() (ret []string) {
			ret = make([]string, 0, len(mi.Nodes))
			for _, node := range mi.Nodes {
				ret = append(ret, string(node))
			}
			return
		}(),
	}, err
}
func TorrentSpecFromMetaInfo(mi *metainfo.MetaInfo, storage storage.ClientImpl) *torrent.TorrentSpec {
	ts, err := TorrentSpecFromMetaInfoErr(mi, storage)
	if err != nil {
		panic(err)
	}
	return ts
}
func GetType(tv *TV, movie *MOVIE) string {
	if movie != nil {
		return Movie
	}
	if tv != nil {
		return Tv
	}
	panic("both null")
}
func SearchTorrent(tv *TV, movie *MOVIE, season *SEASON) []*Torrent_File {
	Type := GetType(tv, movie)
	var names []string
	if movie == nil {
		names = season.GetSearchName(tv)
	} else {
		names = movie.GetSearchName()
		if len(names) == 0 {
			return []*Torrent_File{}
		}
	}
	var wg sync.WaitGroup
	channel := make(chan []*Torrent_File, len(names))
	for _, query := range names {
		wg.Add(1)
		go Search(Type, query, true, channel, &wg)
	}
	wg.Wait()
	close(channel)
	var results []*Torrent_File
	for i := 0; i < len(names); i++ {
		item := <-channel
		for _, citem := range item {
			if season != nil {
				citem.SEASON_ID = &season.ID
				citem.TV_ID = &season.TV_ID
				citem.MOVIE = nil
				citem.MOVIE_ID = nil
			}
			if movie != nil {
				citem.MOVIE = movie
				citem.TV = nil
				citem.TV_ID = nil
				citem.SEASON = nil
				citem.SEASON_ID = nil
				citem.MOVIE_ID = &movie.ID
			}
		}
		results = append(results, item...)
	}
	return results
}
func InitTorrents(db *gorm.DB) {
	var torrents []*Torrent
	db.Preload("USER").Preload("FILES").Find(&torrents)
	channels := make(chan error, len(torrents))
	for _, item := range torrents {
		go func(torrentItem *Torrent) {
			file, err := os.Open(Joins(FILES_TORRENT_PATH, torrentItem.PATH))
			if err != nil {
				panic("Error opening file" + err.Error())
			}
			defer file.Close()
			metadata, err := metainfo.Load(file)
			if err != nil {
				panic(err)
			}
			torrentElem, new, err := client.AddTorrentSpec(TorrentSpecFromMetaInfo(metadata, storage.NewFile(torrentItem.DL_PATH)))
			if err != nil {
				panic(err)
			}
			if !new {
				fmt.Println("Torrent already exists", torrentItem.ID)
				panic("Torrent already exists")
			}
			if torrentItem.Paused {
				torrentElem.DisallowDataDownload()
				torrentElem.DisallowDataUpload()
			}
			for _, file := range torrentElem.Files() {
				file.SetPriority(torrent.PiecePriorityNone)
			}
			mtype, muuid := "", uint(0)
			for _, file := range torrentItem.FILES {
				if file.IS_MEDIA {
					mtype = file.GetMediaType()
					if mtype == Movie {
						muuid = file.MOVIE_ID
					}
					if mtype == Tv {
						muuid = file.TV_ID
					}

					break
				}
			}
			TORRENT_ITEMS = append(TORRENT_ITEMS, &GlTorrentItem{
				OWNER_ID:       torrentItem.USER_ID,
				Torrent:        torrentElem,
				MEDIA_UUID:     muuid,
				MEDIA_TYPE:     mtype,
				DB_ITEM:        torrentItem,
				START_DOWNLOAD: torrentItem.DOWNLOAD,
				START_UPLOAD:   torrentItem.UPLOAD,
				START:          torrentItem.CreatedAt,
			})
			channels <- nil
		}(item)
	}
	for i := 0; i < len(torrents); i++ {
		if err := <-channels; err != nil {
			panic(err)
		}
	}
	go handlers_push_db()
}
func handlers_push_db() {
	fmt.Println("Starting handlers")
	for {
		for _, item := range TORRENT_ITEMS {
			if tx := db.Where("id = ?", item.DB_ITEM.ID).Find(&item.DB_ITEM); tx.Error != nil {
				panic(tx.Error)
			}
			if item.START_DOWNLOAD == 0 && item.DB_ITEM.TIME_TO_1_PERCENT == 0 && float64(item.Torrent.BytesCompleted())/float64(item.Torrent.Length()) > 0.01 {
				item.DB_ITEM.TIME_TO_1_PERCENT = time.Since(item.START).Seconds()
			}
			total := item.Torrent.Length()
			stats := item.Torrent.Stats()
			update := map[string]interface{}{
				"DOWNLOAD": stats.BytesReadUsefulData.Int64() + item.START_DOWNLOAD,
				"UPLOAD":   stats.BytesWrittenData.Int64() + item.START_UPLOAD,
				"PROGRESS": (float64(item.Torrent.BytesCompleted()) / float64(total)),
				"FINISHED": kosmixutil.BoolInt(item.Torrent.BytesCompleted() == total),
			}

			if tx := db.Model(&item.DB_ITEM).
				Omit("FILES").
				Updates(update).Error; tx != nil {
				panic(tx)
			}
		}
		time.Sleep(5 * time.Second)
	}
}

var movingIds = []uint{}

func MoveTargetStorage(torrentItem *torrent.Torrent, dbElement *Torrent, db *gorm.DB, targetPath string, progress *func(int64, int64), success chan error) {
	torrentItem.DisallowDataDownload()
	torrentItem.DisallowDataUpload()
	var dbFiles []*FILE
	if err := db.Preload("TORRENT").Find(&dbFiles, "torrent_id = ?", dbElement.ID).Error; err != nil {
		panic(err)
	}
	f := torrentItem.Files()
	totalSize := 0
	for _, file := range f {
		totalSize += int(file.BytesCompleted())
	}
	if dbElement.DL_PATH == targetPath {
		success <- errors.New("already in target path")
		return
	}
	if torrentItem.BytesCompleted() < torrentItem.Length() {
		success <- errors.New("torrent not finished")
		return
	}
	if slices.Contains(movingIds, dbElement.ID) {
		success <- errors.New("torrent already moving")
		return
	}
	movedSize := int64(0)
	toDeletePath := []string{}
	movingIds = append(movingIds, dbElement.ID)
	for _, file := range dbFiles {
		var torrentFile *torrent.File
		for _, fileItem := range f {
			if filepath.Base(fileItem.Path()) == file.FILENAME && file.SIZE == fileItem.Length() {
				torrentFile = fileItem
			}
		}
		if torrentFile == nil {
			panic("File not found")
		}
		dirInTorrent := filepath.Dir(torrentFile.Path())
		destinationPath := path.Join(targetPath, dirInTorrent)
		if err := os.MkdirAll(destinationPath, os.ModePerm); err != nil {
			panic(err)
		}
		if torrentFile.BytesCompleted() > 0 {
			source := torrentFile.NewReader()
			destination, err := os.Create(path.Join(targetPath, torrentFile.Path()))
			if err != nil {
				panic(err)
			}
			latestPercent := 0
			for {
				buffCopy := make([]byte, 1024*100)
				n, err := source.Read(buffCopy)
				if err != nil {
					if err != io.EOF {
						panic(err)
					}
					break
				}
				_, err = destination.Write(buffCopy[:n])
				if err != nil {
					panic(err)
				}
				movedSize += int64(n)
				if int(float64(movedSize)/float64(totalSize)*100) > latestPercent {
					latestPercent = int(float64(movedSize) / float64(totalSize) * 100)
					(*progress)(movedSize, int64(totalSize))
				}
			}
			source.Close()
			destination.Close()
			toDeletePath = append(toDeletePath, path.Join(dbElement.DL_PATH, torrentFile.Path()))
		}
		db.Model(file).Update("PATH", path.Join(targetPath, dirInTorrent))
	}
	item := GetTorrent(dbElement.ID)
	if item.Torrent == nil {
		panic("Torrent not found")
	}
	file, err := os.Open(Joins(FILES_TORRENT_PATH, dbElement.PATH))
	if err != nil {
		panic(err)
	}
	defer file.Close()
	metadata, err := metainfo.Load(file)
	if err != nil {
		panic(err)
	}
	torrentItem.Drop()
	fmt.Println("Torrent dropped")
	torrentElem, new, err := client.AddTorrentSpec(TorrentSpecFromMetaInfo(metadata, storage.NewFile(targetPath)))
	fmt.Println("Wait for data")
	<-torrentElem.GotInfo()
	fmt.Println("Data received")
	if dbElement.Paused {
		torrentElem.DisallowDataDownload()
		torrentElem.DisallowDataUpload()
	} else {
		torrentElem.AllowDataDownload()
		torrentElem.AllowDataUpload()
	}
	fmt.Println("Torrent download all")
	torrentElem.DownloadAll()
	if tx := db.Model(dbElement).Updates(Torrent{DL_PATH: targetPath}); tx.Error != nil {
		panic(tx.Error)
	}
	if err != nil {
		panic(err)
	}
	if !new {
		panic("Torrent already exists")
	}
	for _, file := range toDeletePath {
		if err := os.Remove(file); err != nil {
			panic(err)
		}
	}
	item.Torrent = torrentElem // update torrent item
	//go RegisterHandlers(torrentElem, dbElement, db, &dbElement.USER)
	fmt.Println("Torrent moved to: ", targetPath)
	movingIds = slices.DeleteFunc(movingIds, func(i uint) bool { return i == dbElement.ID })
	success <- nil
}
func CleanDeleteTorrent(withFiles bool, torrent *GlTorrentItem, db *gorm.DB) error {
	if torrent.Torrent == nil {
		panic("Torrent not found")
	}
	var files []*FILE
	if err := db.Preload("TORRENT").Find(&files, "torrent_id = ?", torrent.DB_ITEM.ID).Error; err != nil {
		panic(err)
	}
	// if !withFiles && files[0].TORRENT.DL_PATH == Config.Torrents.DownloadPath {
	// return errors.New("can't delete default download path must move to normal storage before")
	// }
	if len(files) == 0 {
		panic("No files found")
	}
	for _, tr := range Transcoders {
		v, ok := tr.Source.(*FILE)
		if !ok {
			continue
		}
		if v.TORRENT_ID == torrent.DB_ITEM.ID {
			tr.Destroy("Torrent deleted")
		}
	}
	for _, convert := range Converts {
		if convert.SOURCE_FILE.TORRENT_ID == torrent.DB_ITEM.ID {
			if err := convert.Command.Process.Kill(); err != nil {
				return err
			}
			convert.Task.SetAsError(errors.New("Torrent deleted"))
			convert.Task.SetAsFinished()
		}
	}
	DeleteTorrent(torrent.DB_ITEM.ID)
	var dbtorrent *Torrent
	torrent.Torrent.Drop()
	for _, file := range files {
		if withFiles {
			fileWrapper, err := file.LoadStorage()
			if err != nil {
				fmt.Println("file is in torrent_path")
				os.Remove(file.GetPath(true))
			} else {
				if _, err := file.stats(); err == nil {
					if err := fileWrapper.toConn().Remove(file.GetPath(true)); err != nil {
						panic(err)
					}
				}
			}
			db.Delete(file)
		} else {
			db.Model(&FILE{}).Where("torrent_id = ?", torrent.DB_ITEM.ID).Update("torrent_id", nil)
		}
	}
	dbtorrent = files[0].TORRENT
	if dbtorrent == nil {
		panic("Torrent not found")
	}
	db.Delete(dbtorrent)
	return nil
}
