package main

import (
	"encoding/json"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/golang/gddo/database"
	"github.com/kyokomi/emoji"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const apiURL = "https://api.godoc.org/search?q="
const Version = "v0.0.1"

var (
	db      *leveldb.DB
	noCache bool
	goFlags string
	dbPath  string
	one     sync.Once
)

var (
	rootCmd = cobra.Command{
		Use:     "gosearch",
		Aliases: nil,
		Short:   "search golang packages and install it",
		Long:    "search pop golang packages then install it, you can provide a part of package name or full package name",
		Example: "gosearch cobra",
		Run:     search,
	}
	versionCmd = &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "get gosearch's version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
			return
		},
	}
	cleanCache = &cobra.Command{
		Use:   "clean",
		Short: "clean package caches",
		Long:  "we store search histories in a configure directory,this command will help to clean the directory",
		Run: func(cmd *cobra.Command, args []string) {
			closeDB()
			os.RemoveAll(dbPath)
			emoji.Println(":stuck_out_tongue_winking_eye: clean success")
		},
	}
)

func init() {
	dir, _ := os.UserConfigDir()
	dbPath = filepath.Join(dir, "gosearch/cache")
	db, _ = leveldb.OpenFile(dbPath, nil)
	rootCmd.Args = cobra.ArbitraryArgs
	rootCmd.Flags().BoolVar(&noCache, "no-cache", false, "search from server directly")
	rootCmd.Flags().StringVar(&goFlags, "goflag", "", "setting go get flags,default is empty")
	rootCmd.LocalNonPersistentFlags()

	rootCmd.AddCommand(
		versionCmd,
		cleanCache,
	)
}

func closeDB() {
	one.Do(func() {
		if db != nil {
			db.Close()
		}
	})
}

func main() {
	defer closeDB()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func search(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Usage()
		return
	}
	for _, a := range args {
		if err := searchPackage(a); err != nil {
			return
		}
	}
}

func searchPackage(pkg string) error {
	var packages []database.Package
	if db != nil {
		data, err := db.Get([]byte(pkg), nil)
		if err == nil {
			json.Unmarshal(data, &packages)
		}
	}
	var err error
	if len(packages) == 0 || noCache {
		packages, err = doSearch(pkg)
		defer func() {
			if err == nil {
				data, _ := json.Marshal(packages)
				db.Put([]byte(pkg), data, nil)
			}
		}()
	}
	if err != nil {
		return err
	}
	if err := selectAndInstall(packages); err != nil {
		return err
	}
	return nil
}

func selectAndInstall(packages []database.Package) error {
	var items []string
	for _, p := range packages {
		items = append(items, fmt.Sprintf("%s\t%s\t%s", p.Name, p.Path, p.Synopsis))
	}
	var a = survey.OptionAnswer{}
	prompt := &survey.Select{
		Message: "select a package",
		Options: items,
	}
	if err := survey.AskOne(prompt, &a); err != nil {
		return err
	}
	var args []string
	args = append(args, "get")
	if len(goFlags) != 0 {
		arr := strings.Split(goFlags, " ")
		for _, a := range arr {
			args = append(args, a)
		}
	}
	args = append(args, packages[a.Index].Path)
	emoji.Println(fmt.Sprintf(":face_with_tongue: go %s", strings.Join(args, " ")))

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	emoji.Println(":100: done")

	return nil
}

type Response struct {
	Results []database.Package `json:"results"`
}

func doSearch(pkg string) ([]database.Package, error) {
	url := fmt.Sprintf("%s%s", apiURL, pkg)
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(ioutil.Discard, resp.Body)
		return nil, fmt.Errorf("failed to search package, server return code=%s", resp.Status)
	}
	var res Response
	err = json.NewDecoder(resp.Body).Decode(&res)
	return res.Results, err
}
