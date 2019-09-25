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
	"github.com/JanDeVisser/finn/model"
	"github.com/JanDeVisser/finn/render"
	"github.com/JanDeVisser/grumble"
	"log"
	"net/http"
	"strconv"
)

func institution(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/institution/"):]
	id64, err := strconv.ParseInt(idStr, 0, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id := int(id64)
	e, err := grumble.Get(&model.Institution{}, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	institution := e.(*model.Institution)
	err = institution.GetAccounts()
	render.RenderTemplate(w, "institution", institution)
}

func institutions(w http.ResponseWriter, r *http.Request) {
	institutions, err := model.GetInstitutions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render.RenderTemplate(w, "institutions", institutions)
}

func accounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := model.GetAccounts(nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render.RenderTemplate(w, "accounts", accounts)
}

func account(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/account/"):]
	id64, err := strconv.ParseInt(idStr, 0, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id := int(id64)
	account, err := model.GetAccount(id)
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
	ctx["Account"] = account
	ctx["Transactions"] = transactions
	render.RenderTemplate(w, "account", ctx)
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	institutions, err := model.GetInstitutions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := make(map[string]interface{})
	ctx["suggestUpload"] = len(institutions) == 0
	render.RenderTemplate(w, "index", ctx)
}

func main() {
	http.HandleFunc("/", mainPage)
	http.HandleFunc("/institution/", institution)
	http.HandleFunc("/institutions", institutions)
	http.HandleFunc("/accounts", accounts)
	http.HandleFunc("/account/", account)
	http.HandleFunc("/schema/upload", model.UploadSchema)
	log.Fatal(http.ListenAndServe(":8080", nil))
}