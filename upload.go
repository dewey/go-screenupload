package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/atotto/clipboard"
	"github.com/deckarep/gosx-notifier"
	"github.com/fsnotify/fsnotify"
	"github.com/tmc/scp"
)

// Config contains all the configuration options
type Config struct {
	UserName string // Username used on the remote server
	HostName string // Hostname of the remote server
	Port     string // Port used for SSH on remote server
	RPath    string // Remote Path where files should be moved on the remote server
	RUrl     string // URL where the image will be accessible on the remote server
	LPath    string // Local Path where we are going to watch for new additions
	Archive  string // Path to directory where files will be archived
	Filter   string // Regex to filter out files that should be automatically uploaded
}

// File contains all the information about a file
type File struct {
	Path      string
	Extension string
	Name      string
	URL       string
}

var cfg Config

func init() {
	cfg = Config{
		UserName: os.Getenv("USER"),
		HostName: os.Getenv("HOST"),
		Port:     os.Getenv("PORT"),
		RPath:    os.Getenv("RPATH"),
		LPath:    os.Getenv("LPATH"),
		Archive:  os.Getenv("ARCHIVE"),
		Filter:   os.Getenv("FILTER"),
	}

	// set default values
	if os.Getenv("PORT") == "" {
		cfg.Port = "22"
	}
	if os.Getenv("FILTER") == "" {
		cfg.Filter = `^Screen.Shot.[0-9-]*.\w*.[0-9.]*.png`
	}
}

func main() {
	var reFilename = regexp.MustCompile(cfg.Filter)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op == fsnotify.Create {
					if event.Op == fsnotify.Create && reFilename.MatchString(filepath.Base(event.Name)) {
						err := upload(cfg, File{
							Path:      event.Name,
							Extension: filepath.Ext(event.Name),
							Name:      filepath.Base(event.Name),
						})
						if err != nil {
							log.Fatal(err)
						}
					}
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(cfg.LPath)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

// upload is uploading a given file to a remote server via SCP
func upload(cfg Config, f File) error {
	agent, err := getAgent()
	if err != nil {
		log.Fatalln("failed to connect to SSH_AUTH_SOCK:", err)
	}

	// use existing public keys
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", cfg.HostName, cfg.Port), &ssh.ClientConfig{
		User: cfg.UserName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(agent.Signers),
		},
	})

	if err != nil {
		log.Fatalln("failed to dial:", err)
	}

	session, err := client.NewSession()
	if err != nil {
		log.Fatalln("failed to create session: " + err.Error())
	}

	// rename or rename and archive if enabled
	fn, err := rename(cfg, f)
	if err != nil {
		return err
	}

	err = scp.CopyPath(fn.Path, cfg.RPath, session)
	if err != nil {
		return err
	}

	// remove renamed file after upload
	if cfg.Archive == "" {
		err := trash(cfg, fn)
		if err != nil {
			return err
		}
	}

	// send notification using OS default notifier
	fn.URL = fmt.Sprintf("%s/%s", cfg.RUrl, fn.Name)

	// add url to clipboard
	clipboard.WriteAll(fn.URL)

	err = notify(fn)
	if err != nil {
		return err
	}
	return nil
}

// getAgent will use the system ssh agent
func getAgent() (agent.Agent, error) {
	agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	return agent.NewClient(agentConn), err
}

// generateHash will return a sha1 hash for a given filename
func generateHash(str string) (hash string, err error) {
	if str != "" {
		h := sha1.New()
		h.Write([]byte(str))
		s := hex.EncodeToString(h.Sum(nil))
		return s, nil
	}
	return "", errors.New("error generating hash")
}

func notify(f File) error {
	//At a minimum specifiy a message to display to end-user.
	n := gosxnotifier.NewNotification("The URL is now in your clipboard.")
	n.Title = "Screen Upload"
	n.Subtitle = "Upload finished"
	n.Sender = "com.apple.Terminal"
	n.Link = f.URL
	err := n.Push()

	if err != nil {
		return err
	}
	return nil
}

// Rename will rename and/or remove a file
func rename(cfg Config, f File) (file File, err error) {
	hash, err := generateHash(fmt.Sprintf("%s:%d", f.Name, int32(time.Now().Unix())))
	if err != nil {
		return File{}, errors.New("error generating filename")
	}
	fn := File{
		Extension: f.Extension,
		Name:      fmt.Sprintf("%s%s", hash, f.Extension),
	}

	// if we are not archiving a file just rename it without moving
	if cfg.Archive == "" {
		fn.Path = fmt.Sprintf("%s%s", filepath.Join(cfg.LPath, hash), f.Extension)
		err = os.Rename(f.Path, fn.Path)
		if err != nil {
			return File{}, err
		}
	} else {
		fn.Path = fmt.Sprintf("%s%s", filepath.Join(cfg.Archive, hash), f.Extension)
		err = os.Rename(f.Path, fn.Path)
		if err != nil {
			return File{}, err
		}
	}
	return fn, nil
}

// Trash removes a given file
func trash(cfg Config, f File) error {
	err := os.Remove(f.Path)
	if err != nil {
		return err
	}
	return nil
}
