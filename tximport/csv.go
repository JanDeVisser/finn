package tximport

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JanDeVisser/finn/model"
	"github.com/JanDeVisser/grumble"
	"io"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Importer interface {
	Process(*TXImport) error
}

type ImporterFactory func(*model.Account) (Importer, error)

func setValueInObject(obj interface{}, name string, value interface{}) {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	fld := v.FieldByNameFunc(func(s string) bool {
		return strings.ToLower(s) == strings.ToLower(name)
	})
	if fld.IsValid() {
		fld.Set(reflect.ValueOf(value))
	}
}

func setValuesInObject(obj interface{}, values map[string]interface{}, fallback map[string]interface{}) {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	for name, value := range values {
		fld := v.FieldByNameFunc(func(s string) bool {
			return strings.ToLower(s) == strings.ToLower(name)
		})
		if fld.IsValid() {
			fld.Set(reflect.ValueOf(value))
		} else if fallback != nil {
			fallback[name] = value
		}
	}
}

type ImportStatus string

const (
	Initial     ImportStatus = "Initial"
	Read                     = "Read"
	InProgress               = "InProgress"
	Completed                = "Completed"
	ImportError              = "Error"
	Partial                  = "Partial"
)

type TXImport struct {
	grumble.Key
	Timestamp time.Time
	FileName  string
	Status    ImportStatus
	Data      string
	Total     int
	Good      int
	Bad       int
	Errors    string
	importer  Importer
}

func (imp *TXImport) FindOrCreate(kind interface{}, field string, value string) (e grumble.Persistable, err error) {
	e, err = imp.Manager().By(grumble.GetKind(kind), field, value)
	if err != nil {
		err = errors.New(fmt.Sprintf("By(%q = %q): %s", field, value, err))
		return
	}
	if e == nil {
		e, err = imp.Manager().Make(grumble.GetKind(kind), nil, 0)
		if err != nil {
			return nil, err
		}
		setValueInObject(e, field, value)
		err = imp.Manager().Put(e)
	}
	return
}

func (imp *TXImport) SetReference(e grumble.Persistable, field string, referenceKind interface{}, value string) (err error) {
	if value != "" {
		var ref grumble.Persistable
		ref, err = imp.FindOrCreate(referenceKind, "Name", value)
		if err != nil {
			return
		}
		setValueInObject(e, field, ref)
	}
	return
}

func (imp *TXImport) AddError(err error) {
	if err != nil {
		if imp.Errors == "" {
			imp.Errors = err.Error()
		} else {
			imp.Errors = imp.Errors + "\n" + err.Error()
		}
	}
}

func (imp *TXImport) Update(status ImportStatus, err error) {
	if err != nil {
		imp.AddError(err)
	}
	imp.Status = status
	if e := imp.Manager().Put(imp); e != nil {
		panic(fmt.Sprintf("Error updating import record: %s", e.Error()))
	}
}

func (imp *TXImport) Read() (err error) {
	imp.Update(InProgress, nil)
	err = imp.Manager().TX(func(db *sql.DB) (err error) {
		err = imp.importer.Process(imp)
		return
	})
	switch {
	case err != nil:
		imp.Update(ImportError, err)
	case imp.Bad > 0 && imp.Good == 0:
		imp.Update(ImportError, nil)
	case imp.Bad > 0 && imp.Good > 0:
		imp.Update(Partial, nil)
	default:
		imp.Update(Completed, nil)
	}
	return
}

func MakeTXImport(account *model.Account, fileName string) (imp *TXImport, err error) {
	imp = &TXImport{
		FileName: fileName, Status: Initial,
		Timestamp: time.Now(),
	}
	imp.Initialize(account, 0)
	var data []byte
	var status ImportStatus
	//if data, err = ioutil.ReadFile(filepath.Join("data", account.AccName, fileName)); err != nil {
	if data, err = ioutil.ReadFile(fileName); err != nil {
		status = ImportError
	} else {
		imp.Data = string(data)
		status = Read
	}
	imp.Update(status, err)
	if imp.importer, err = GetImporter(account); err != nil {
		imp.Update(ImportError, err)
		return
	}
	return
}

type ImportField struct {
	Name    string
	Num     int
	Type    string
	Options map[string]interface{}
}

func (fld *ImportField) Convert(str string) (ret interface{}, err error) {
	switch fld.Type {
	case "str":
		ret = str
	case "int":
		var i64 int64
		i64, err = strconv.ParseInt(str, 0, 0)
		if err != nil {
			return
		}
		ret = int(i64)
	case "float":
		ret, err = strconv.ParseFloat(str, 0)
	case "bool":
		ret, err = strconv.ParseBool(str)
	case "date":
		layout := "01/02/2006"
		l, ok := fld.Options["format"]
		if ok {
			if layout, ok = l.(string); !ok {
				layout = "01/02/2006"
			}
		}
		ret, err = time.Parse(layout, str)
	}
	return
}

type Template struct {
	Template string
	MatchOn  string
	Type     string
	Contact  string
	Category string
	Project  string
	re       *regexp.Regexp
}

func MakeTemplate(def map[string]interface{}) (ret Template, err error) {
	ret = Template{MatchOn: "description"}
	setValuesInObject(&ret, def, nil)
	ret.re, err = regexp.Compile(ret.Template)
	return
}

type CSVImporter struct {
	Account    *model.Account
	Mappings   []*ImportField
	Templates  []Template
	Config     map[string]interface{}
	HeaderLine bool
}

func (imp *CSVImporter) parseTemplate() (err error) {
	fileName := filepath.Join("data", imp.Account.AccName+".json")
	var jsonText []byte
	if jsonText, err = ioutil.ReadFile(fileName); err != nil {
		return
	}
	var jsonData interface{}
	err = json.Unmarshal(jsonText, &jsonData)
	if err != nil {
		return
	}
	data := jsonData.(map[string]interface{})

	mappings := make([]*ImportField, 0)
	m, ok := data["mapping"]
	if ok {
		mapping := m.([]interface{})
		for ix, f := range mapping {
			importField := &ImportField{Num: ix, Options: make(map[string]interface{})}
			switch col := f.(type) {
			case nil:
				importField.Name = ""
			case string:
				importField.Name = strings.ToLower(col)
				importField.Type = "str"
			case map[string]interface{}:
				setValuesInObject(importField, col, importField.Options)
				importField.Name = strings.ToLower(importField.Name)
			}
			mappings = append(mappings, importField)
		}
	}
	imp.Mappings = make([]*ImportField, len(mappings))
	for _, mapping := range mappings {
		imp.Mappings[mapping.Num] = mapping
	}

	imp.HeaderLine = false
	imp.Config = make(map[string]interface{})
	c, ok := data["config"]
	if ok {
		config := c.(map[string]interface{})
		setValuesInObject(imp, config, imp.Config)
	}

	imp.Templates = make([]Template, 0)
	t, ok := data["templates"]
	if ok {
		templates := t.([]interface{})
		for _, t := range templates {
			tpl := t.(map[string]interface{})
			var template Template
			template, err = MakeTemplate(tpl)
			if err != nil {
				return
			}
			imp.Templates = append(imp.Templates, template)
		}
	}
	return
}

func (imp *CSVImporter) Process(txImport *TXImport) (err error) {
	rdr := csv.NewReader(strings.NewReader(txImport.Data))
	if imp.HeaderLine {
		if _, err = rdr.Read(); err != nil {
			return
		}
	}
	var e error
	txImport.Good = 0
	txImport.Bad = 0
	txImport.Total = 0
	for record, e := rdr.Read(); e == nil; record, e = rdr.Read() {
		txImport.Total++
		if err = imp.ProcessLine(record, txImport); err != nil {
			txImport.Bad++
			txImport.AddError(err)
		} else {
			txImport.Good++
		}
	}
	if e != io.EOF {
		err = e
	}
	return
}

func (imp *CSVImporter) ProcessLine(line []string, txImport *TXImport) (err error) {
	fields := make(map[string]string)
	for ix, f := range line {
		if ix >= len(imp.Mappings) {
			break
		}
		if imp.Mappings[ix].Name == "" {
			continue
		}
		fields[imp.Mappings[ix].Name] = f
	}
	imp.ApplyTemplates(fields)
	err = imp.SaveTransaction(txImport, fields)
	return
}

func (imp *CSVImporter) ApplyTemplates(fields map[string]string) {
	for _, tmpl := range imp.Templates {
		v, ok := fields[tmpl.MatchOn]
		if !ok {
			continue
		}
		if tmpl.re.MatchString(v) {
			if tmpl.Type != "" {
				fields["type"] = tmpl.Type
			}
			if tmpl.Contact != "" {
				fields["contact"] = tmpl.Contact
			}
			if tmpl.Category != "" {
				fields["category"] = tmpl.Category
			}
			if tmpl.Project != "" {
				fields["project"] = tmpl.Project
			}
		}
	}
}

func (imp *CSVImporter) SaveTransaction(txImport *TXImport, fields map[string]string) (err error) {
	txType := model.Debit
	if t, ok := fields["type"]; ok {
		txType = t
	}
	var tx grumble.Persistable
	tx, err = imp.Account.MakeTransaction(txType)
	if err != nil {
		return
	}
	for _, mapping := range imp.Mappings {
		var val interface{}
		val, err = mapping.Convert(fields[mapping.Name])
		if err != nil {
			return
		}
		setValueInObject(tx, mapping.Name, val)
	}
	if err = txImport.SetReference(tx, "Contact", model.Contact{}, fields["contact"]); err != nil {
		return
	}
	if err = txImport.SetReference(tx, "Project", model.Project{}, fields["project"]); err != nil {
		return
	}
	if err = txImport.SetReference(tx, "Category", model.Category{}, fields["category"]); err != nil {
		return
	}
	if err = tx.Manager().Put(tx); err != nil {
		return
	}
	return
}

func MakeCSVImporter(account *model.Account) (ret Importer, err error) {
	imp := &CSVImporter{}
	imp.Account = account
	if err = imp.parseTemplate(); err == nil {
		ret = imp
	}
	return
}

var importers = make(map[string]ImporterFactory)

func GetImporter(account *model.Account) (ret Importer, err error) {
	factory, ok := importers[account.Importer]
	if ok {
		ret, err = factory(account)
	}
	return
}

func init() {
	importers["CSV"] = MakeCSVImporter
}
