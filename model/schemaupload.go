package model

import (
	"database/sql"
	"fmt"
	"github.com/JanDeVisser/finn/render"
	"github.com/JanDeVisser/grumble"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

func uploadPage(ctx map[string]interface{}, w http.ResponseWriter, r *http.Request) {
	render.RenderTemplate(w, "schemaupload", ctx)
}

func RedirectSuccess(w http.ResponseWriter, r *http.Request, msg string) {
	http.Redirect(w, r,
		fmt.Sprintf("/?message=%s", url.QueryEscape(msg)), http.StatusSeeOther)
}

func RedirectError(w http.ResponseWriter, r *http.Request, err error) {
	http.Redirect(w, r, fmt.Sprintf("/?error=%s", url.QueryEscape(err.Error())), http.StatusSeeOther)
}

func uploadSchema(w http.ResponseWriter, r *http.Request) {
	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, _, err := r.FormFile("schema")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	tempFile, err := ioutil.TempFile("", "upload-*.json")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// write this byte array to our temporary file
	_, err = tempFile.Write(fileBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mgr, err := grumble.MakeEntityManager()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = mgr.TX(func(conn *sql.DB) error {
		return ImportSchema(mgr, tempFile.Name())
	})
	if err != nil {
		RedirectError(w, r, err)
	} else {
		RedirectSuccess(w, r, "Schema successfully uploaded")
	}
}

func UploadSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		uploadPage(make(map[string]interface{}), w, r)
	} else {
		uploadSchema(w, r)
	}
}
