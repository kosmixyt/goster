# GOSTER

- Oriented towards French users
- Currently in testing
- patch are welcome
- Don't expose to internet !!!!! i'm a very bad dev
- You should host this on server with good CPU
-

# Screenshots:

#### Landing page

![Convert](https://i.ibb.co/mHDyvNj/reallanding.png)

#### Movie Details

![Convert](https://i.ibb.co/c8C6f8r/render-Movie.png)

#### Tv Show Details

![Convert](https://i.ibb.co/W30K6fz/renderTv.png)

#### Mosaique

![Convert](https://i.ibb.co/vs2GthB/landing.png)

#### /me

![Convert](https://i.ibb.co/QJtvnLr/myaccount.png)

#### Metadata updater

![Convert](https://i.ibb.co/Syf33CN/metadata.png)

#### Player

![Convert](https://i.ibb.co/pRmVhD0/player.png)

#### Recommendations

![Convert](https://i.ibb.co/jv64TWx/recomm.png)

## Features:

- Multiple File source protocol (local fs, sftp)
- Multiplatform support (tested on linux and windows)
- Fetch Torrent From Various providers (yggtorrents(with cloudflare), sharewoord)
- Transcoder on the fly : (subtitles supported (except pks), multi audio track, quality selector 360p-1080p)
- You can watch the movies/episode that are being downloaded, the access time to a film/episode that you do not have in your library is very generally <10sec (this depends on the number of seeders)
- Basic torrent action (pause, download as zip, download .torrent, delete (withfiles))
- Directory scanner (like plex and emby) and metadata search based on filenames
- File Converting (while downloading support)
- Multi-user (currently only via db)
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

## config.json Setup

- Rename config.sample.json to config.json
- Edit config.json as you wont, 2driver type : mysq,sqlite

```
{
  "scan_paths": [
    {
      "name": "homelab",
      "type": "sftp",
      "options": {
        "host": "host_ip",
        "port": "22",
        "user": "username",
        "pass": "password",
        "path": [
          "/DATA/DOWNLOADS/movie/",
          "/DATA/DOWNLOADS/serie/"
        ]
      }
    },
    {
      "name": "local storage",
      "type": "local",
      "options": {
        "path": [
          "/path/to/movie",
          "/path/to/serie"
        ]
      }
    }
  ],
  "db": {
    "driver": "sqlite",
    "host": "",
    "port": "",
    "username": "",
    "password": "",
    "database": "./db/streaming.db"
  },
  "cert": {
    "key": "path to ssl key or leave empty",
    "cert": "path to ssl cert or leave empty"
  },
  "limits": {
    "check_interval": 24,
    "movie_size": 5368709120,
    "season_size": 21474836480
  },
  "transcoder": {
    "ffmpeg": "/usr/bin/ffmpeg",
    "ffprobe": "/usr/bin/ffprobe"
  },
  "torrents": {
    "download_path": "/app/dl/"
  },
  "web": {
    "public_url": "/api",
    "public_port": "80",
    "cross_origin": "http://localhost:3000"
  },
  "cache_path": "/app/cache/",
  "metadata": {
    "tmdb": "tmdb api_key",
    "tmdb_iso3166": "FR",
    "tmdb_movie_watch_providers": {
      "Netflix": 8,
      "Disney Plus": 337,
      "Amazon Prime Video": 119,
      "Apple Tv Plus": 350,
      "Canal +": 381,
      "Google Play Movies": 3,
      "Filmo TV": 138,
      "Amazon Video": 10
    },
    "tmdb_tv_watch_providers": {},
    "omdb": "omdb api key",
    "tmdb_lang": "fr",
    "tmdb_lang_imgs": [
      "fr",
      "en",
      "null"
    ]
  },
  "cloudflare": {
    "challenge_resolver": "puppeteer",
    "flaresolverr_url": "leave empty",
    "capsolverr_proxy_url": "leave empty",
    "capsolverr_api_key": "leave empty"
  },
  "torrent_providers": {
    "sharewood": {
      "key": "",
      "username": "",
      "password": ""
    },
    "ygg": {
      "username": "ygg username",
      "password": "ygg password"
    }
  }
}

```

### Default token : admin

# Docker Compose

you must configure config.json before (you can let default value of cache_path)
and the config.json must be in the same directory of docker-compose.yaml

```
services:
  app:
    image: kosmix/goapp
    ports:
      - "80:80"
    volumes:
      - ./config.json:/app/config.json # config mount point
      - ./cache/:/app/cache/ # cache mount point
      - ./dl/:/app/dl/ # torrent mount point
      #if you use sqlite uncomment
      # - ./db/streaming.db:/app/db/streaming.db
    networks:
      - private_network
networks:
  private_network:
    driver: bridge
```

- Wait 1/2min (check console) and go to http://localhost

# Todo :

- Trailer download patch
- Iptv record modal
- Add NextEpisode to transcode data, and under started tv shows
- Move torrent to other file storage (without / continue seeding)
- DDL provider
- User manager

## Contact :

Discord : "KOSMIX.FR"
