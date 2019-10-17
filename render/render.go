package render

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

var Master *template.Template
var templates *template.Template

type RenderConfig struct {
	Preload bool
}

var Default = RenderConfig{Preload: true}
var Config *RenderConfig

func PrintDate(d time.Time) string {
	return d.Format("02-Jan-2006")
}

func Odd(i int) bool {
	return i%2 != 0
}

func Even(i int) bool {
	return !Odd(i)
}

var funcs = template.FuncMap{
	"date": PrintDate,
	"odd":  Odd,
	"even": Even,
}

func init() {
	var err error

	Config = &Default
	var jsonText []byte
	if jsonText, err = ioutil.ReadFile("conf/render.conf"); err == nil {
		Config = new(RenderConfig)
		err = json.Unmarshal(jsonText, Config)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Could not JSON decode render config: %s", err)
			Config = &Default
		}
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "Could not read render config: %s", err)
	}

	if Config.Preload {
		Master = template.New("master").Funcs(funcs)
		Master = template.Must(Master.ParseFiles("render/master.html"))
		templates = template.Must(template.Must(Master.Clone()).ParseGlob("html/*.html"))
	}
}

func RenderTemplate(w http.ResponseWriter, tmpl string, context interface{}) {
	var err error
	if Config.Preload {
		err = templates.ExecuteTemplate(w, tmpl+".html", context)
	} else {
		t := template.New("master").Funcs(funcs)
		t = template.Must(t.ParseFiles("render/master.html", fmt.Sprintf("html/%s.html", tmpl)))
		err = t.ExecuteTemplate(w, "master.html", context)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
