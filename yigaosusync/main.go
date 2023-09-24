package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caiguanhao/yigaosu"
	"github.com/go-git/go-git/v5"
	"github.com/gopsql/goconf"
	"golang.org/x/crypto/ssh"
)

const (
	cardsJsonpPrefix = "__yigaosuCards"
	billsJsonpPrefix = "__yigaosuBills"
)

type (
	configs struct {
		YigaosuPhone             string `Yigaosu login phone`
		YigaosuEncryptedPassword string `Yigaosu login encrypted password`

		GitRemoteName  string `Name for the git remote repository`
		GitRemoteUrl   string `URL for the git remote repository`
		GitBranchName  string `Name of a branch`
		LocalDirectory string `Path to local directory of the repository`
		SshKey         string `SSH private key location`
		UserName       string `Author name to use when creating a new commit`
		UserEmail      string `Author email to use when creating a new commit`
	}

	card struct {
		CardNo string
	}

	bill struct {
		Amount  string
		BeginAt time.Time
		EndAt   time.Time
		From    string
		To      string
	}
)

var (
	conf  configs
	debug bool
)

func newCard(b yigaosu.ETCCard) card {
	return card{
		CardNo: b.CardNo,
	}
}

func newBill(b yigaosu.ETCCardBill) bill {
	return bill{
		Amount:  b.TotalAmount,
		BeginAt: time.Unix(b.StartTime/1000, 0).UTC(),
		EndAt:   time.Unix(b.EndTime/1000, 0).UTC(),
		From:    strings.TrimSuffix(b.StartStation, "驶入"),
		To:      strings.TrimSuffix(b.EndStation, "驶出"),
	}
}

func main() {
	defaultConfigFile := ".yigaosusync.conf"
	if home, _ := os.UserHomeDir(); home != "" {
		defaultConfigFile = filepath.Join(home, defaultConfigFile)
	}
	configFile := flag.String("c", defaultConfigFile, "location of the config file")
	createConfig := flag.Bool("C", false, "create (update if exists) config file and exit")
	gitForcePush := flag.Bool("force-push", false, "git force push only")
	flag.BoolVar(&debug, "debug", false, "debug")
	flag.Parse()

	content, err := ioutil.ReadFile(*configFile)
	if err == nil {
		err = goconf.Unmarshal(content, &conf)
	}
	if err != nil {
		conf = configs{}
	}
	if *createConfig {
		if conf.SshKey == "" {
			privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
			keyPem := pem.EncodeToMemory(&pem.Block{
				Type:    "RSA PRIVATE KEY",
				Headers: nil,
				Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
			})
			conf.SshKey = string(keyPem)
			pub, _ := ssh.NewPublicKey(&privateKey.PublicKey)
			log.Println("new public key:", strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub))))
		}
		content, err := goconf.Marshal(conf)
		if err != nil {
			log.Fatal(err)
		}
		content = append([]byte("// vim: set syntax=go :\n"), content...)
		if err := ioutil.WriteFile(*configFile, content, 0600); err != nil {
			log.Fatal(err)
		} else {
			log.Println("Config file written:", *configFile)
		}
		return
	}
	conf.LocalDirectory = expandTilde(conf.LocalDirectory)

	client := &gitClient{
		Remote:    conf.GitRemoteName,
		RemoteUrl: conf.GitRemoteUrl,
		Branch:    conf.GitBranchName,
		LocalDir:  conf.LocalDirectory,
		SshKey:    conf.SshKey,
	}

	if *gitForcePush {
		err = client.ForcePush()
		if err != nil {
			log.Println(err)
		}
		return
	}

	err = client.AddFiles(addFilesOpts{
		UserName:  conf.UserName,
		UserEmail: conf.UserEmail,
		AddFiles: func(w *git.Worktree) (commitMsg string, err error) {
			var files []string
			files, err = write()
			if err != nil {
				return
			}
			for _, file := range files {
				if _, err = w.Add(file); err != nil {
					return
				}
			}
			status, _ := w.Status()
			var changed []string
			for file := range status {
				changed = append(changed, file)
			}
			if len(changed) > 0 {
				commitMsg = "update " + strings.Join(changed, ", ")
			} else {
				commitMsg = "no update"
			}
			return
		},
	})
	if err != nil {
		log.Println(err)
	}
}

func write() (filenames []string, err error) {
	ctx := context.Background()
	if debug {
		ctx = context.WithValue(ctx, "DEBUG", true)
	}

	var client *yigaosu.Client
	client, err = yigaosu.Login(ctx, conf.YigaosuPhone, conf.YigaosuEncryptedPassword)
	if err != nil {
		err = fmt.Errorf("Login: %w", err)
		return
	}

	var cards []yigaosu.ETCCard
	cards, err = client.GetETCCards(ctx)
	if err != nil {
		err = fmt.Errorf("GetETCCards: %w", err)
		return
	}

	buffers := map[string]*bytes.Buffer{}
	buffers["cards.js"] = writeJS(cardsJsonpPrefix, len(cards), func(i int) interface{} {
		return newCard(cards[i])
	})

	for _, card := range cards {
		var allBills []yigaosu.ETCCardBill
		page := 1
		for {
			var bills []yigaosu.ETCCardBill
			bills, err = client.GetETCCardBillsPage(ctx, card, 100, page)
			if err != nil {
				err = fmt.Errorf("GetETCCardBillsPage: %w", err)
				return
			}
			allBills = append(allBills, bills...)
			if len(bills) == 0 {
				break
			}
			page += 1
		}
		buffers[card.CardNo+".js"] = writeJS(billsJsonpPrefix, len(allBills), func(i int) interface{} {
			return newBill(allBills[i])
		})
	}

	for name, b := range buffers {
		path := filepath.Join(conf.LocalDirectory, name)
		log.Println("writing", path)
		err = os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			return
		}
		err = ioutil.WriteFile(path, b.Bytes(), 0644)
		if err != nil {
			return
		}
		filenames = append(filenames, name)
	}

	return
}

func writeJS(prefix string, length int, process func(int) interface{}) *bytes.Buffer {
	var b bytes.Buffer
	fmt.Fprint(&b, prefix)
	fmt.Fprintln(&b, "([")
	for i := 0; i < length; i++ {
		j, _ := json.Marshal(process(i))
		fmt.Fprint(&b, string(j), ",")
		fmt.Fprintln(&b)
	}
	fmt.Fprintln(&b, "null")
	fmt.Fprintln(&b, "])")
	return &b
}

func expandTilde(path string) string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return path
	}
	path = strings.TrimSpace(path)
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return strings.Replace(path, "~/", home+"/", 1)
	}
	return path
}
