# go-screenupload

Simple script to automatically upload screenshots to a remote host, archive screenshots locally and copy URL to clipboard.

# Usage

Configure with environment variables like:

```
USER=dewey HOST=example.com PORT=22 RPATH=/home/example/public_www/img.example.com/ LPATH=/Users/dewey/Desktop/ ARCHIVE=/Users/dewey/.archive FILTER=^Screen.Shot.[0-9-]*.\w*.[0-9.]*.png go run upload.go
```
