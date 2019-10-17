/*
 * Copyright (c) 2019.
 *
 * This file is part of $project_name.
 *
 * $project_name  is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Foobar is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Foobar.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/JanDeVisser/finn/handler"
	"github.com/JanDeVisser/finn/model"
	"github.com/JanDeVisser/finn/render"
	"github.com/JanDeVisser/finn/tximport"
	"github.com/JanDeVisser/grumble"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func institution(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/institution/"):]
	id64, err := strconv.ParseInt(idStr, 0, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id := int(id64)
	mgr, err := grumble.MakeEntityManager()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	e, err := mgr.Get(model.Institution{}, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	institution := e.(*model.Institution)
	err = institution.GetAccounts()
	render.RenderTemplate(w, "institution", institution)
}

func institutions(w http.ResponseWriter, r *http.Request) {
	mgr, err := grumble.MakeEntityManager()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	institutions, err := model.GetInstitutions(mgr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render.RenderTemplate(w, "institutions", institutions)
}

func accounts(w http.ResponseWriter, r *http.Request) {
	mgr, err := grumble.MakeEntityManager()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	accounts, err := model.GetAccounts(mgr, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render.RenderTemplate(w, "accounts", accounts)
}

func account(w http.ResponseWriter, r *http.Request) {
	mgr, err := grumble.MakeEntityManager()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	idStr := r.URL.Path[len("/account/"):]
	id64, err := strconv.ParseInt(idStr, 0, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id := int(id64)
	account, err := model.GetAccount(mgr, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	transactions, err := account.GetTransactions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := make(map[string]interface{})
	ctx["key"] = idStr
	ctx["Account"] = account
	ctx["Transactions"] = transactions
	render.RenderTemplate(w, "account", ctx)
}

func serveFileHandler(w http.ResponseWriter, r *http.Request) {
	serveFile(w, r, r.URL.Path)
}

func serveFile(w http.ResponseWriter, r *http.Request, file string) {
	contentTypes := map[string]string{
		"css":  "text/css",
		"html": "text/html",
		"txt":  "text/plain",
		"js":   "text/javascript",
		"jpg":  "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
	}

	ext := file[strings.LastIndex(file, ".")+1:]
	ct, ok := contentTypes[ext]
	if !ok {
		ct = "text/plain"
	}
	w.Header().Add("Content-type", ct)
	fmt.Printf("%s (%s)\n", file, ct)
	if strings.Index(file, "/") == 0 {
		file = file[1:]
	}
	http.ServeFile(w, r, file)
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	serveFile(w, r, "html/index.html")
}

func oldMainPage(w http.ResponseWriter, r *http.Request) {
	mgr, err := grumble.MakeEntityManager()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	institutions, err := model.GetInstitutions(mgr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := make(map[string]interface{})
	ctx["suggestUpload"] = len(institutions) == 0
	render.RenderTemplate(w, "index", ctx)
}

func tools(w http.ResponseWriter, r *http.Request) {
	RedirectSuccess := func(msg string) {
		http.Redirect(w, r,
			fmt.Sprintf("/?message=%s", url.QueryEscape(msg)), http.StatusSeeOther)
	}
	RedirectError := func(err error) {
		http.Redirect(w, r, fmt.Sprintf("/?error=%s", url.QueryEscape(err.Error())), http.StatusSeeOther)
	}

	s := strings.Split(r.URL.Path[1:], "/")
	mgr, err := grumble.MakeEntityManager()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tool := s[1]
	switch tool {
	case "reset":
		err = mgr.ResetSchema()
		if err != nil {
			RedirectError(err)
			return
		}
		err = mgr.TX(func(db *sql.DB) error {
			for _, k := range grumble.Kinds() {
				if e := k.Reconcile(mgr.PostgreSQLAdapter); e != nil {
					return e
				}
			}
			return nil
		})
		if err != nil {
			RedirectError(err)
		} else {
			RedirectSuccess("Database reset")
		}
	default:
		RedirectError(errors.New(fmt.Sprintf("Unknown tool %q", tool)))
	}
}

func main() {
	http.HandleFunc("/", mainPage)
	http.HandleFunc("/css/", serveFileHandler)
	http.HandleFunc("/image/", serveFileHandler)
	http.HandleFunc("/javascript/", serveFileHandler)
	http.HandleFunc("/static/", serveFileHandler)
	http.HandleFunc("/json/", handler.JSON)
	http.HandleFunc("/institution/", institution)
	http.HandleFunc("/institutions", institutions)
	http.HandleFunc("/accounts", mainPage)
	http.HandleFunc("/account/upload/", tximport.UploadCSV)
	http.HandleFunc("/account/", mainPage)
	http.HandleFunc("/category/", mainPage)
	http.HandleFunc("/schema/upload", model.UploadSchema)
	http.HandleFunc("/tools/", tools)
	fmt.Println("Starting Listener")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
