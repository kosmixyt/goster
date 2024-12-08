# GOSTER

- Oriented towards French users
- Currently in testing
- patch are welcome
- Don't expose to internet !!!!! i'm a very bad dev
- You should host this on server with good CPU

## Features:

- Multiple File source protocol (local fs, sftp)
- Multiplatform support (tested on linux and windows)
- Fetch Torrent From Various providers (yggtorrents(with cloudflare), sharewoord)
- Transcoder on the fly : (subtitles supported (except pks), multi audio track, quality selector 360p-1080p)
- You can watch the movies/episode that are being downloaded, the access time to a film/episode that you do not have in your library is very generally <10sec (this depends on the number of seeders)
- Basic torrent action (pause, download as zip, download .torrent, delete (withfiles))
- Directory scanner (like plex and emby) and metadata search based on filenames
- File Converting (while downloading support)
- Multi-user
- Metadata manager
- Iptv Support
- Tmdb recommendating based on you watched movies/tv
- Movie/TV Request (for films/Tv season that have not yet been released on the providers)
- TMDB and iptv image caching
- SSl support

## REQUIRED:

- TMDB api_key: [GET](https://www.themoviedb.org/settings/api?language=fr)
- FFMPEG and FFPROBE installed on the machine (if without docker)

## Optional

- YGG torrent Account
- Sharewood account
- Omdb Api key [GET](https://www.omdbapi.com/apikey.aspx)

## Quick Setup

- Rename config.sample.json to config.json
- Provide you scan_paths (where your (movies/tv shows are located))
  2 types of scan_paths : - Sftp
  ```
  {
      "name": "homelab",
      "type": "sftp",
      "options": {
        "host": "host_ip",
        "port": "22",
        "user": "username",
        "pass": "password",
        "path": ["/DATA/DOWNLOADS/movie/", "/DATA/DOWNLOADS/serie/"]
      }
    }
  ```
  - Local
  ```
  {
      "name": "local example",
      "type": "local",
      "options": {
        "path": ["/path/to/media/movies/", "/path/to/media/serie"]
      }
  }
  ```
- Provide Your database Credentials
  - 2 types of database storage
    Mysql:
  ```
  "db":
  {
  "driver": "mysql",
  "host": "localhost",
  "port": "3306",
  "username": "root",
  "password": "",
  "database": "streaming",
  }
  ```
  Sqlite:
  ```
  "db":
  {
  "driver": "sqlite",
  "database": "streaming.db"
  }
  ```
- If you have ssl enter your ssl absolute path
- Provide ffmpeg and ffprobe path in .transcoder
- give the absolute path of the folder where you want goster to store temporary files (<10gb)
- give the absolute path (local!!!) where you want goster to store the downloaded torrents (a fast disk).
- Set your tmdb api key in metadata.tmdb
- Set your iso3166 language in metadata.tmdb_iso3166
- If you have omdb api key set it in metadata.omdb
- Set your short language id in metadata.tmdb_lang
- If you have an ygg account provide credentials in torrent_providers.ygg
- If you have an sharewood account provide credentials in torrent_providers.sharewood
- Run bin.exe|bin
- Wait 1/2min (check console) and go to http://localhost

### Default token : admin

# Docker Compose

you must configure config.json before, (you can let default value of cache_path)

```
services:
  app:
    image: goapp
    ports:
      - "80:80"
    volumes:
      - ./config.json:/app/config.json # config mount point
      - ./cache/:/app/cache/ # cache mount point
      #if you use sqlite uncomment
      # - ./streaming.db:/app/streaming.db
    networks:
      - private_network
networks:
  private_network:
    driver: bridge
```

# Todo :

- Trailer download patch
- Iptv record modal
- Add NextEpisode to transcode data, and under started tv shows
- Move torrent to other file storage (without / continue seeding)
- DDL provider
- User manager
