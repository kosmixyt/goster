package engine

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"kosmix.fr/streaming/engine/storage"
	"kosmix.fr/streaming/kosmixutil"
)

type MemoryStorage struct {
	Conn      storage.Storage
	DbElement *StorageDbElement
}

var Storages []*MemoryStorage = make([]*MemoryStorage, 0)

func GetStorageConFromId(id uint) *MemoryStorage {
	for _, storage := range Storages {
		if storage.DbElement.ID == id {
			return storage
		}
	}
	return nil
}
func InitStoragesConnection(locations []StorageElement) error {
	toNodeDeleteStorage := make([]uint, 0)
	toNotDeletePath := make([]uint, 0)
	localDeclared := false
	for _, location := range locations {
		if localDeclared && location.TYPE == "local" {
			return fmt.Errorf("Local storage declared")
		}

		if location.TYPE == "local" {
			localDeclared = true
		}
		storage, err := DispatchStorage(location.TYPE)
		if err != nil {
			panic(err)
		}
		fmt.Println("Initiating storage: ", location.Name)
		channel := make(chan error)
		go storage.Init(location.Name, channel, location.Options, location.Paths)
		err = <-channel
		if err != nil {
			for _, _ = range Storages {
				// stor.Conn.
			}
			return err
		}
		fmt.Println("Storage initiated: ", location.Name, len(storage.Paths()))
		var ExistantStorage StorageDbElement
		if tx := db.Where("name = ?", storage.Name()).First(&ExistantStorage); tx.Error != nil {
			StorageDbElement := StorageDbElement{
				Name:  storage.Name(),
				Paths: []*StoragePathElement{},
			}
			if tx := db.Create(&StorageDbElement); tx.Error != nil {
				panic(tx.Error)
			}
			for _, path := range storage.Paths() {
				PathElement := StoragePathElement{
					Path:      path.Path,
					StorageId: StorageDbElement.ID,
					Size:      path.Size,
					Files:     []*FILE{},
					Records:   []*Record{},
				}
				if tx := db.Create(&PathElement); tx.Error != nil {
					panic(tx.Error)
				}
				StorageDbElement.Paths = append(StorageDbElement.Paths, &PathElement)
				toNotDeletePath = append(toNotDeletePath, PathElement.ID)
			}
			toNodeDeleteStorage = append(toNodeDeleteStorage, StorageDbElement.ID)
			ExistantStorage = StorageDbElement
		} else {
			for _, path := range storage.Paths() {
				var ExistantPath StoragePathElement
				if tx := db.Where("path = ? AND storage_id = ?", path.Path, ExistantStorage.ID).First(&ExistantPath); tx.Error != nil {
					ExistantPath = StoragePathElement{
						Path:      path.Path,
						StorageId: ExistantStorage.ID,
						Files:     []*FILE{},
						Records:   []*Record{},
					}
					if tx := db.Create(&ExistantPath); tx.Error != nil {
						panic(tx.Error)
					}
				}
				ExistantStorage.Paths = append(ExistantStorage.Paths, &ExistantPath)
				toNotDeletePath = append(toNotDeletePath, ExistantPath.ID)
			}
			toNodeDeleteStorage = append(toNodeDeleteStorage, ExistantStorage.ID)
		}
		Storages = append(Storages, &MemoryStorage{
			Conn:      storage,
			DbElement: &ExistantStorage,
		})
	}
	db.Where("id NOT IN ?", toNodeDeleteStorage).Delete(&StorageDbElement{})
	db.Where("id NOT IN ?", toNotDeletePath).Delete(&StoragePathElement{})
	return nil
}

var scan_mtx = sync.Mutex{}

func Scan(db *gorm.DB) error {
	if !scan_mtx.TryLock() {
		return fmt.Errorf("Scan already running")
	}
	defer scan_mtx.Unlock()
	var files_ar []*storage.FileData
	for _, storage := range Storages {
		storage.DbElement.LoadPaths()
		for _, path := range storage.DbElement.Paths {
			fmt.Println("Scanning path: ", path, "of", storage.DbElement)
			files, err := storage.Conn.RecursiveScan(path.toStorage())
			if err != nil {
				fmt.Println(path.Path)
				panic(err)
			}
			for _, f := range files {
				f.ROOT_PATH = path
				f.Path = strings.TrimSuffix(strings.TrimPrefix(f.Path, path.Path), f.FileName)
				f.StorerDbId = storage.DbElement.ID
				files_ar = append(files_ar, &f)
			}

		}
	}
	tonotDelete := make([]uint, 0)
	for i, file := range files_ar {
		if strings.Contains(file.FileName, "$") {
			panic("File contains $, skipping it")
		}
		var fileInDb FILE
		isVideoFile := kosmixutil.IsVideoFile(file.FileName)
		if err := db.Preload("StoragePathElement").Where("filename = ?  AND sub_path = ? AND is_media = ?", file.FileName, file.Path, isVideoFile).First(&fileInDb).Error; err != nil {
			fileInDb = FILE{
				FILENAME:           file.FileName,
				SUB_PATH:           strings.TrimPrefix(file.Path, file.ROOT_PATH.(*StoragePathElement).Path),
				SIZE:               file.Size,
				IS_MEDIA:           isVideoFile,
				StoragePathElement: file.ROOT_PATH.(*StoragePathElement),
			}
			Year := kosmixutil.GetYear(file.FileName)
			if isVideoFile {
				if !fileInDb.IsEpisode() {
					tmdbMovies, err := kosmixutil.SearchForMovie(fileInDb.GetTitle(), Year)
					if err != nil {
						panic(err)
					}
					var movieInDb *MOVIE
					if len(tmdbMovies.Results) == 0 {
						movieInDb = emptyMovie(fileInDb.GetTitle(), Year)
						if tx := db.Create(&movieInDb); tx.Error != nil {
							panic(tx.Error)
						}
					} else {
						tmdbMovie := tmdbMovies.Results[0]
						if tx := db.Where("tmdb_id = ?", tmdbMovie.ID).First(movieInDb); tx.Error != nil {
							movieInDb, err = InsertMovieInDb(db, tmdbMovie.ID, int64(Year), true, func() *gorm.DB { return db })
							if err != nil {
								panic(err)
							}
							fmt.Println("Movie not found in database, inserting it")
						}
					}
					fileInDb.MOVIE_ID = movieInDb.ID
				} else {
					var Name = kosmixutil.ReturnGood(fileInDb.GetTitle())
					var TvInDb *TV
					if len(strings.TrimSpace(Name)) == 0 {
						panic("No serie found" + Name)
					}
					if tx := db.Where("name = ?", Name).Preload("SEASON").Preload("SEASON.EPISODES").First(&TvInDb); tx.Error != nil {
						tmdbTv, err := kosmixutil.SearchForSerie(Name, strconv.Itoa(Year))
						if err != nil {
							panic(err)
						}
						if len(tmdbTv.Results) == 0 {
							fmt.Println("Not found in database and not found in tmdb: ", Name, file.FileName, " adding as unassigned file")
							fileInDb.TV_ID = 0
							fileInDb.SEASON_ID = 0
							fileInDb.EPISODE_ID = 0
							fileInDb.MOVIE_ID = 0
							db.Save(&fileInDb)
							tonotDelete = append(tonotDelete, fileInDb.ID)
							continue
						} else {
							tmdbSerie := tmdbTv.Results[0]
							tempTvDb, err := GetSerieDb(db, tmdbSerie.ID, strconv.Itoa(Year), true, func() *gorm.DB {
								return db.Preload("SEASON").Preload("SEASON.EPISODES")
							})
							if err != nil {
								panic(err)
							}
							TvInDb = tempTvDb
						}
						season := TvInDb.GetSeason(fileInDb.SeasonNumber(), true, db)
						episode := season.GetEpisode(fileInDb.EpisodeNumber(), true, db)
						if season.ID == 0 || episode.ID == 0 {
							panic("Episode not loaded from database")
						}
						fileInDb.SEASON_ID = season.ID
						fileInDb.EPISODE_ID = episode.ID
						fileInDb.TV_ID = TvInDb.ID
					} else {
						season := TvInDb.GetSeason(fileInDb.SeasonNumber(), true, db)
						episode := season.GetEpisode(fileInDb.EpisodeNumber(), true, db)
						if season.ID != 0 && episode.ID == 0 {
							fmt.Println(season.ID, episode.ID)
							panic("Episode not found in database")
						}
						fileInDb.EPISODE_ID = episode.ID
						fileInDb.SEASON_ID = season.ID
						fileInDb.TV_ID = TvInDb.ID
					}
					db.Save(&TvInDb)
				}
			}
			db.Save(&fileInDb)
			tonotDelete = append(tonotDelete, fileInDb.ID)
		} else {
			fmt.Println("File found in database, skipping it", i)
			fmt.Println("File: ", fileInDb.FILENAME, fileInDb.SUB_PATH, "Size: ", fileInDb.SIZE, "Is media: ", fileInDb.IS_MEDIA, "ID: ", fileInDb.ID)
			tonotDelete = append(tonotDelete, fileInDb.ID)
		}
	}
	queryTime := time.Now()
	fmt.Println("Query time: Insert", time.Since(queryTime))
	DeleteFilesInDb(tonotDelete, db)
	return nil
}

func DeleteFilesInDb(ids []uint, db *gorm.DB) {
	var filesToDelete []FILE
	tx := db.Where("id NOT IN ? AND torrent_id IS NULL", ids).Delete(&FILE{})
	if tx.Error != nil {
		fmt.Println("Error while deleting files: ", tx.Error)
	} else {
		fmt.Println("File affetcted by the delete: ", tx.RowsAffected, len(ids))
	}
	db.Where("file_id IN ?", filesToDelete).Delete(&WATCHING{})
	db.Where("id IN ?", filesToDelete).Delete(&FILE{})
}
func VerifyDB(db *gorm.DB) {
	var files []FILE
	if tx := db.Find(&files); tx.Error != nil {
		panic(tx.Error)
	}
	for _, file := range files {
		if _, err := file.stats(); os.IsNotExist(err) && !file.IsTorrentFile() {
			fmt.Println("!!!!! File not found in filesystem: ", file.GetPath(true))
		}
	}
	var anonymousFiles []FILE
	if tx := db.Where("movie_id IS NOT NULL AND (episode_id IS NOT NULL OR season_id IS NOT NULL OR tv_id IS NOT NULL)").Find(&anonymousFiles); tx.Error != nil {
		panic(tx.Error)
	}
	var unmatchedFiles []FILE
	if tx := db.Where("(episode_id IS NOT NULL AND (season_id IS NULL OR tv_id IS NULL))").Find(&unmatchedFiles); tx.Error != nil {
		panic(tx.Error)
	}
	if len(unmatchedFiles) > 0 {
		panic("Unmatched files found")
	}
	fmt.Println("Anonymous files: ", len(anonymousFiles), "Total files: ", len(files), "Unmatched files: ", len(unmatchedFiles))
}
