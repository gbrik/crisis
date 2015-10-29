package crisis

import (
	"gopkg.in/pg.v3"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type servlet func(http.ResponseWriter, *http.Request)

const (
	staticPath = "static/"
	htmlPath   = "webcontent/html/"
)

type pageInfo struct {
	CanEdit   bool
	ViewAs    int
	JSUrls    []string
	CSSUrl    string
	Types     []UnitType
	Factions  []Faction
	Divisions []Division
}

var mainPageTmpl *template.Template

func StartListening() {
	staticServer := http.FileServer(http.Dir(staticPath))
	http.Handle("/static/", http.StripPrefix("/static/", staticServer))

	ajaxHandler := GetAjaxHandlerInstance()
	http.HandleFunc("/ajax/", func(w http.ResponseWriter, r *http.Request) {
		ajaxHandler.HandleRequest(w, r)
	})

	imagePath := os.Getenv("CRISIS_IMAGE_DIR")
	http.HandleFunc("/uploadBG", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Path, r.Header)
		file, _, err := r.FormFile("background")
		maybePanic(err)

		out1, err := os.Create(imagePath + "1.png")
		maybePanic(err)
		out2, err := os.Create(staticPath + "bgs/1.png")
		maybePanic(err)

		_, err = io.Copy(out1, file)
		maybePanic(err)
		_, err = io.Copy(out2, file)
		maybePanic(err)
	})

	http.HandleFunc("/staff", mainPage)
	http.HandleFunc("/view", mainPage)

	go MoveDivisions()
}

func MoveDivisions() {
	for {
		time.Sleep(10 * time.Second)
		err := GetDatabaseInstance().db.RunInTransaction(func(tx *pg.Tx) error {
			return DoUnitMovement(tx)
		})
		if err != nil {
			log.Println(err)
		}
	}
}

func mainPage(res http.ResponseWriter, req *http.Request) {
	var err error

	if mainPageTmpl == nil {
		mainPageTmpl, err = template.ParseFiles(htmlPath + "mainpage.gohtml")
		maybePanic(err)
	}

	err = GetDatabaseInstance().db.RunInTransaction(func(tx *pg.Tx) error {
		authInfo, err := AuthInfoOf(tx, req)
		if err != nil {
			return err
		}

		types, err := GetUnitTypesByCrisisId(tx, authInfo.CrisisId)
		if err != nil {
			return err
		}

		facs, err := GetFactionsByCrisisId(tx, authInfo.CrisisId)
		if err != nil {
			return err
		}

		viewAs := -1
		if authInfo.ViewAs != nil {
			viewAs = *authInfo.ViewAs
		}

		return mainPageTmpl.Execute(res, pageInfo{
			JSUrls: []string{
				"static/jquery.mousewheel.js",
				"static/buckets.min.js",
				"static/compiled.js",
			},
			CSSUrl:    "static/main.css",
			Types:     types,
			Factions:  facs,
			CanEdit:   authInfo.CanEdit,
			ViewAs:    viewAs,
			ImagePath: os.Getenv("CRISIS_IMAGE_PATH"),
		})
	})
	maybePanic(err)
}
