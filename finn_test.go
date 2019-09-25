package main

import (
	"github.com/JanDeVisser/finn/model"
	"github.com/JanDeVisser/finn/tximport"
	"github.com/JanDeVisser/grumble"
	"testing"
)

func TestImportSchema(t *testing.T) {
	var err error
	if err = model.ImportSchema("data/my_schema.json"); err != nil {
		t.Fatal(err)
	}
}

func TestCSVImport(t *testing.T) {
	e, err := grumble.GetKind(model.Account{}).By("AccName", "ManulifeOne")
	if err != nil {
		t.Fatal(err)
	}
	acc := e.(*model.Account)
	importer, err := tximport.GetImporter(acc)
	if err != nil {
		t.Fatal(err)
	}
	if err = importer.Read("initial/02212019_Transactions.csv"); err != nil {
		t.Fatal(err)
	}
}
