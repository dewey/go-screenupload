# go-screenupload

Simple script to automatically upload screenshots to a remote host, archive screenshots locally and copy URL to clipboard.

# Usage

Configure with environment variables and run the binary


`USER` - Username used on the remote server

`HOST` - Hostname of the remote server

`PORT` - Port used for SSH on remote server (Default: `22`)

`RPATH` - Remote Path where files should be moved on the remote server

`RURL` - URL where the image will be hosted (public_www directory)

`LPATH` - Local Path where we are going to watch for new additions

`ARCHIVE` - Path to directory where files will be archived

`FILTER` - Regex to filter out files that should be automatically uploaded (Default: `^Screen.Shot.[0-9-]*.\w*.[0-9.]*.png` for Mac OS screen shots)
