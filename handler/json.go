/*
 * This file is part of Finn.
 *
 * Copyright (c) 2019 Jan de Visser <jan@finiandarcy.com>
 *
 * Finn is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Finn is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Finn.  If not, see <https://www.gnu.org/licenses/>.
 */

package handler

import (
	"encoding/json"
	"fmt"
	"github.com/JanDeVisser/grumble"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type JSONRequest struct {
	mgr    *grumble.EntityManager
	Method string
	Kind   *grumble.Kind
	Id     int
	w      http.ResponseWriter
	r      *http.Request
}

func (req *JSONRequest) Execute() {
	v := reflect.ValueOf(req)
	m := v.MethodByName(req.Method)
	if m.IsValid() {
		m.Call([]reflect.Value{})
	} else {
		panic(fmt.Sprintf("Cannot serve method %q for JSON requests", req.Method))
	}
}

func (req *JSONRequest) WriteJSON(obj interface{}) {
	jsonText, err := json.Marshal(obj)
	if err != nil {
		http.Error(req.w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.w.Header().Add("Content-type", "text/json")
	_, err = req.w.Write(jsonText)
	if err != nil {
		http.Error(req.w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = req.w.Write([]byte("\n"))
}

func (req *JSONRequest) GET() {
	var obj interface{}
	var err error
	if req.Id > 0 {
		log.Printf("JSON.GET %s.%d", req.Kind.Kind, req.Id)
		obj, err = req.mgr.Get(req.Kind, req.Id)
	} else {
		log.Printf("JSON.GET q=%q", req.r.URL.Query().Encode())
		obj, err = req.mgr.Query(req.Kind, req.r.URL.Query())
	}
	if err != nil {
		http.Error(req.w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.WriteJSON(obj)
}

func JSON(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.RawQuery)
	mgr, err := grumble.MakeEntityManager()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s := strings.Split(r.URL.Path[1:], "/")
	kind := grumble.GetKind(s[1])
	if kind == nil {
		http.Error(w, fmt.Sprintf("Unknown kind '%s'", s[1]), http.StatusInternalServerError)
		return
	}
	var id int64 = 0
	switch {
	case len(s) == 3:
		id, err = strconv.ParseInt(s[2], 0, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case r.Form.Get("id") != "":
		id, err = strconv.ParseInt(r.Form.Get("id"), 0, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	req := &JSONRequest{mgr: mgr, Method: r.Method, Kind: kind, Id: int(id), w: w, r: r}
	req.Execute()
}
