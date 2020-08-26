package lib

import (
	"encoding/json"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/kyokomi/emoji"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const Version = "v0.0.2"

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
		Version: Version,
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

func Main() {
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
	var packages []Package
	if db != nil {
		data, err := db.Get([]byte(pkg), nil)
		if err == nil {
			json.Unmarshal(data, &packages)
		}
	}
	var err error
	if len(packages) == 0 || noCache {
		// 判断是否是一个完整的包名.
		if isPackageName(pkg) {
			return install(pkg)
		}
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

func selectAndInstall(packages []Package) error {
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
	install(packages[a.Index].Path)
	emoji.Println(":100: done")
	return nil
}

func install(packageName string) error {
	var args []string
	args = append(args, "get")
	if len(goFlags) != 0 {
		arr := strings.Split(goFlags, " ")
		for _, a := range arr {
			args = append(args, a)
		}
	}
	args = append(args, packageName)
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	emoji.Println(fmt.Sprintf(":face_with_tongue: go %s", strings.Join(args, " ")))
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func isPackageName(pkg string) bool {
	arr := strings.Split(pkg, "/")
	if len(arr) > 2 {
		return true
	}
	return false
}
