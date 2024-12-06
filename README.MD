**SETUP : RENAME config.sample.json TO config.json**
EDIT config.json

1.  "scan_paths" : Where you store your media ([]string)
2.  "db" : Database information
3.  "cert" : ssl path (leave empty key and cert if no ssl)
4.  sharewood.tv logins and PassKey
5.  Limits : Max movie Size and Season size
6.  Transcoder: ffmpeg and ffprobe path (leave ffmpeg and ffprobe for commands)
7.  web : Public url (final url of instance), port, and cross origin for header (leave empty)
8.  metadata :
9.  tmdb api key ([https://www.themoviedb.org/settings/api?language=fr-FR](https://www.themoviedb.org/settings/api?language=fr-FR))
10. omdb api key ([https://www.omdbapi.com/apikey.aspx](https://www.omdbapi.com/apikey.aspx))
11. tmdb_lang : metadata language "en"
12. tmdb_lang_imgs : metadata image language ordered by preference for us/en (en,null) for de (de,en,null)

Current Indexer supported :
Yggtorrent (with and without cloudflare)
sharewood.tv (any time)

Wrapper must in the same directory as the executable.
