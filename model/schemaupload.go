package model

import (
	"github.com/JanDeVisser/finn/render"
	"io/ioutil"
	"net/http"
)

func uploadPage(ctx map[string]interface{}, w http.ResponseWriter, r *http.Request) {
	render.RenderTemplate(w, "schemaupload", ctx)
}

func uploadSchema(w http.ResponseWriter, r *http.Request) {
	ctx := make(map[string]interface{})

	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		ctx["error"] = err.Error()
		uploadPage(ctx, w, r)
		return
	}

	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, header, err := r.FormFile("schema")
	if err != nil {
		ctx["error"] = err.Error()
		uploadPage(ctx, w, r)
		return
	}
	defer file.Close()
	ctx["header"] = header

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	tempFile, err := ioutil.TempFile("data/tmp", "upload-*.json")
	if err != nil {
		ctx["error"] = err.Error()
		uploadPage(ctx, w, r)
		return
	}

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		ctx["error"] = err.Error()
		uploadPage(ctx, w, r)
		return
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)
	err = tempFile.Close()
	if err != nil {
		ctx["error"] = err.Error()
		uploadPage(ctx, w, r)
		return
	}

	err = ImportSchema(tempFile.Name())
	if err != nil {
		ctx["error"] = err.Error()
	} else {
		ctx["success"] = true
	}
	uploadPage(ctx, w, r)
}

func UploadSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		uploadPage(make(map[string]interface{}), w, r)
	} else {
		uploadSchema(w, r)
	}
}

