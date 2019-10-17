package main

import (
	"database/sql"
	"github.com/JanDeVisser/finn/model"
	"github.com/JanDeVisser/finn/tximport"
	"github.com/JanDeVisser/grumble"
	"testing"
)

var mgr *grumble.EntityManager

func init() {
	var err error
	mgr, err = grumble.MakeEntityManager()
	if err != nil {
		panic(err)
	}
}

func TestImportSchema(t *testing.T) {
	var err error
	err = mgr.TX(func(db *sql.DB) error {
		return model.ImportSchema(mgr, "data/my_schema.json")
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCSVImport(t *testing.T) {
	var err error
	err = mgr.TX(func(db *sql.DB) (err error) {
		e, err := mgr.By(grumble.GetKind(model.Account{}), "AccName", "ManulifeOne")
		if err != nil {
			return
		}
		acc := e.(*model.Account)
		txImport, err := tximport.MakeTXImport(acc, "data/ManulifeOne/initial/02212019_Transactions.csv")
		if err != nil {
			return
		}
		return txImport.Read()
	})
	if err != nil {
		t.Fatal(err)
	}
}
