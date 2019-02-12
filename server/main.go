package main

import (
	"flag"
	"github.com/ccontavalli/goutils/misc"
	"github.com/ccontavalli/sbexr/server/db"
	"log"
	"net/http"
	"net/http/fcgi"
	"runtime"
	"time"
)

import _ "net/http/pprof"

const kMaxResults = 30

var listen = flag.String("listen", "", "Address to listen on - if unspecified, use fcgi. Example: 0.0.0.0:9000.")
var indexdir = flag.String("index-dir", "", "Directory with indexes to load.")
var webroot = flag.String("web-root", "", "Directory with pages to serve.")
var indexfiles = misc.MultiString("web-index-files", []string{
	"NEWS", "README", "README.md",
	"00-INDEX", "CHANGES", "Changes",
	"ChangeLog", "changelog", "Kconfig",
}, "Files to display when rendering a directory.")

type Index struct {
	Tree      db.TagSet
	BinSymbol db.TagSet

	Sources *SourceServer
}

func NewIndex(indexroot, sourcesroot string) Index {
	return Index{
		Tree:      db.NewTagSet("tree", db.NewSingleDirTagSetHandler(indexroot, ".files.json", db.LoadJsonTree)),
		BinSymbol: db.NewTagSet("symbol", db.NewSingleDirTagSetHandler(indexroot, ".symbols.json", db.LoadSymbols)),
		Sources:   NewSourceServer(sourcesroot),
	}
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU() - 1)
}

func Updater(index *Index) {
	for {
		index.Tree.Update()
		index.BinSymbol.Update()

		index.Tree.AddHandlers()
		index.BinSymbol.AddHandlers()

		index.Sources.Update()

		runtime.GC()
		time.Sleep(10 * time.Second)
	}
}

func main() {
	flag.Parse()
	if len(*indexdir) <= 0 {
		log.Fatal("Must supply flag --index-dir - to match the one used with sbexr")
	}
	if len(*webroot) <= 0 {
		log.Fatal("Must supply flag --webroot - to match the one used with sbexr")
	}

	index := NewIndex(*indexdir, *webroot)
	go Updater(&index)

	if *webroot != "" {
		http.Handle("/", index.Sources)
	}

	var err error
	if *listen != "" {
		err = http.ListenAndServe(*listen, nil) // set listen port
	} else {
		err = fcgi.Serve(nil, nil)
	}
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
