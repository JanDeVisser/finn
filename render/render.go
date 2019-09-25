package render

import (
	"html/template"
	"net/http"
)

var templates = template.Must(template.ParseFiles(
	"html/index.html",
	"html/accounts.html",
	"html/account.html",
	"html/institutions.html",
	"html/institution.html",
	"html/schemaupload.html",
))

func RenderTemplate(w http.ResponseWriter, tmpl string, contents interface{}) {
	err := templates.ExecuteTemplate(w, tmpl + ".html", contents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
