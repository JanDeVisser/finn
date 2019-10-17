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

package tximport

import (
	"fmt"
	"github.com/JanDeVisser/finn/model"
	"github.com/JanDeVisser/grumble"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type CSVUploader struct {
	mgr     *grumble.EntityManager
	id      int
	account *model.Account
	w       http.ResponseWriter
	r       *http.Request
	ctx     map[string]interface{}
}

func MakeCSVUploader(w http.ResponseWriter, r *http.Request) (ret *CSVUploader) {
	var err error
	ret = new(CSVUploader)
	ret.w = w
	ret.r = r
	ret.ctx = make(map[string]interface{})

	ret.mgr, err = grumble.MakeEntityManager()
	if err != nil {
		ret.ctx["error"] = err
		return
	}

	idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	id64, err := strconv.ParseInt(idStr, 0, 0)
	if err != nil {
		ret.ctx["error"] = err
		return
	}
	ret.id = int(id64)
	ret.account, err = model.GetAccount(ret.mgr, ret.id)
	if err != nil {
		ret.ctx["error"] = err
		return
	}
	ret.ctx["Account"] = ret.account
	return
}

func (uploader *CSVUploader) ErrorRedirect() {
	http.Redirect(uploader.w, uploader.r,
		fmt.Sprintf("/account/error/%d", uploader.id), http.StatusSeeOther)
}

func (uploader *CSVUploader) SuccessRedirect() {
	http.Redirect(uploader.w, uploader.r,
		fmt.Sprintf("/account/%d", uploader.id), http.StatusSeeOther)
}

func (uploader *CSVUploader) Upload() {
	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	err := uploader.r.ParseMultipartForm(10 << 20)
	if err != nil {
		uploader.ctx["error"] = err
		return
	}

	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, header, err := uploader.r.FormFile("csv")
	if err != nil {
		uploader.ctx["error"] = err
		return
	}
	defer func() {
		_ = file.Close()
	}()
	uploader.ctx["header"] = header

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	tempFile, err := ioutil.TempFile("", "upload-*.csv")
	if err != nil {
		uploader.ctx["error"] = err
		return
	}
	defer func() {
		_ = os.Remove(tempFile.Name())
	}()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		uploader.ctx["error"] = err
		return
	}

	// write this byte array to our temporary file
	if written, err := tempFile.Write(fileBytes); err != nil || written != len(fileBytes) {
		uploader.ctx["error"] = err
		return
	}
	err = tempFile.Close()
	if err != nil {
		uploader.ctx["error"] = err
		return
	}

	txImport, err := MakeTXImport(uploader.account, tempFile.Name())
	if err != nil {
		uploader.ctx["error"] = err
		return
	}
	err = txImport.Read()
	if err != nil {
		uploader.ctx["error"] = err
		return
	}
}

func UploadCSV(w http.ResponseWriter, r *http.Request) {
	uploader := MakeCSVUploader(w, r)
	if _, ok := uploader.ctx["error"]; ok {
		uploader.ErrorRedirect()
		return
	}

	if r.Method == http.MethodGet {
		uploader.SuccessRedirect()
	} else {
		uploader.Upload()
		if _, ok := uploader.ctx["error"]; ok {
			uploader.ErrorRedirect()
		} else {
			uploader.SuccessRedirect()
		}
	}
}
