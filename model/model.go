/*
 * Copyright (c) 2019.
 *
 * This file is part of Finn.
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
 * along with Foobar.  If not, see <https://www.gnu.org/licenses/>.
 */

package model

import (
	"errors"
	"fmt"
	"github.com/JanDeVisser/grumble"
	"net/url"
	"strconv"
	"time"
)

type Category struct {
	grumble.Key
	Name           string
	Description    string
	CurrentBalance float64 `grumble:"transient"`
}

type Project struct {
	grumble.Key
	Name           string
	Description    string
	Category       *Category
	CurrentBalance float64 `grumble:"transient"`
}

type Contact struct {
	grumble.Key
	Name           string
	InteracAddress string
	AccountInfo    string
	Category       *Category
	CurrentBalance float64 `grumble:"transient"`
}

type Institution struct {
	grumble.Key
	Name        string
	Description string
	Accounts    []*Account `grumble:"transient"`
}

func GetInstitutions(mgr *grumble.EntityManager) (institutions []*Institution, err error) {
	q := mgr.MakeQuery(&Institution{})
	q.WithDerived = true
	results, err := q.Execute()
	if err != nil {
		return
	}
	institutions = make([]*Institution, len(results))
	for ix, row := range results {
		e := row[0]
		institutions[ix] = e.(*Institution)
	}
	return
}

func (institution *Institution) GetAccounts() (err error) {
	institution.Accounts, err = GetAccounts(institution.Manager(), institution)
	return
}

//type TransactionType string
const (
	Debit          = "D"
	Credit         = "C"
	Transfer       = "T"
	OpeningBalance = "O"
	Adjustment     = "A"
)

type Transaction struct {
	grumble.Key
	Date         time.Time
	TXType       string
	Amt          float64
	Currency     string `grumble:"default=CAD"`
	ForeignAmt   float64
	Debit        float64 `grumble:"verbosename=Out;formula=(CASE WHEN \"Amt\" < 0 THEN -\"Amt\" ELSE 0 END)"`
	Credit       float64 `grumble:"verbosename=In;formula=(CASE WHEN \"Amt\" > 0 THEN \"Amt\" ELSE 0 END)"`
	Description  string
	Consolidated bool
	Category     *Category
	Project      *Project
	Contact      *Contact
}

type OpeningBalanceTx struct {
	Transaction
}

type TransferTx struct {
	Transaction
	CrossPost *TransferTx
	Account   *Account
}

type Account struct {
	grumble.Key
	AccName        string `grumble:"verbosename=Account name;label"`
	AccNr          string `grumble:"verbosename=Account #"`
	Description    string
	Currency       string `grumble:"default=CAD"`
	Importer       string
	OpeningDate    time.Time `grumble:"transient"`
	OpeningBalance float64   `grumble:"transient"`
	CurrentBalance float64   `grumble:"transient"`
	TotalDebit     float64   `grumble:"transient"`
	TotalCredit    float64   `grumble:"transient"`
	InstName       string    `grumble:"transient"` // FIXME
	InstIdent      int       `grumble:"transient"`
}

func (acc *Account) SetOpeningBalance(date time.Time, balance float64) (err error) {
	txp, err := acc.MakeTransaction("O")
	if err != nil {
		return
	}
	tx := txp.(*OpeningBalanceTx)
	tx.Date = date
	tx.Amt = balance
	tx.Description = "Opening Balance"
	return acc.Manager().Put(tx)
}

func (acc *Account) MakeTransaction(txType string) (tx grumble.Persistable, err error) {
	switch txType {
	case Debit, Credit:
		debitCreditTx := &Transaction{TXType: txType}
		tx = debitCreditTx
	case Transfer:
		transfer := &TransferTx{}
		transfer.TXType = txType
		// Not now dear...
		tx = transfer
	case OpeningBalance:
		opening := &OpeningBalanceTx{}
		opening.TXType = txType
		tx = opening
	}
	if tx != nil {
		_ = tx.Initialize(acc, 0)
	}
	return
}

func GetAccounts(mgr *grumble.EntityManager, institution *Institution) (accounts []*Account, err error) {
	return accountQuery(mgr, institution, 0)
}

func GetAccount(mgr *grumble.EntityManager, id int) (account *Account, err error) {
	accounts, err := accountQuery(mgr, nil, id)
	if err != nil {
		return
	}
	switch len(accounts) {
	case 0:
		err = errors.New(fmt.Sprintf("No account with ID %d found", id))
	case 1:
		account = accounts[0]
	}
	return
}

func GetAccountQuery(mgr *grumble.EntityManager, institution *Institution, id int) (q *grumble.Query, err error) {
	q = mgr.MakeQuery(&Account{})
	q.WithDerived = true
	if id != 0 {
		q.AddQueryCondition(grumble.HasId{Id: id})
	}
	if institution != nil {
		q.AddQueryCondition(grumble.HasParent{Parent: institution.AsKey()})
	}
	q = addTransactionJoin(q)
	return
}

func addTransactionJoin(q *grumble.Query) *grumble.Query {
	q.GroupBy = true
	txJoin := grumble.Join{
		QueryTable: grumble.QueryTable{
			Kind:        grumble.GetKind(&Transaction{}),
			WithDerived: true,
			GroupBy:     false,
			Alias:       "tx",
		},
		JoinType:  grumble.Outer,
		Direction: grumble.In,
		FieldName: "_parent",
	}
	txJoin.AddAggregate(grumble.Aggregate{
		Function: "SUM",
		Column:   "Debit",
		Name:     "TotalDebit",
		Default:  "0.0",
		Query:    nil,
	})
	txJoin.AddAggregate(grumble.Aggregate{
		Function: "SUM",
		Column:   "Credit",
		Name:     "TotalCredit",
		Default:  "0.0",
		Query:    nil,
	})
	q.AddJoin(txJoin)
	openingBalance := grumble.SubQuery{
		QueryTable: grumble.QueryTable{
			Kind:        grumble.GetKind(&OpeningBalanceTx{}),
			WithDerived: false,
			GroupBy:     false,
			Alias:       "opening",
		},
		Where: "(opening.\"_parent\"[1]).id = k.\"_id\"",
	}
	openingBalance.AddSubSelect(grumble.Computed{
		Formula: "COALESCE(MIN(opening.\"Date\"), NOW())",
		Name:    "OpeningDate",
	})
	openingBalance.AddSubSelect(grumble.Computed{
		Formula: "COALESCE(SUM(opening.\"Amt\"), 0.0)",
		Name:    "OpeningBalance",
	})
	q.AddSubQuery(openingBalance)
	q.AddGlobalComputedColumn(grumble.Computed{
		Formula: "COALESCE(SUM(tx.\"Amt\"), 0.0)",
		Name:    "CurrentBalance",
		Query:   nil,
	})
	addInstSubSelect := false
	for _, j := range q.Joins {
		if j.FieldName == "_parent" && j.Direction == grumble.Out {
			q.RemoveJoin(j.Alias)
			addInstSubSelect = true
			break
		}
	}

	// FIXME This should be handled in the Query - all 'Outgoing' joins should be added to
	// the GROUP BY clause if the the grouping is on the main QueryTable.
	if addInstSubSelect {
		institution := grumble.SubQuery{
			QueryTable: grumble.QueryTable{
				Kind:        grumble.GetKind(&Institution{}),
				WithDerived: true,
				GroupBy:     false,
				Alias:       "institution",
			},
			Where: "(k.\"_parent\"[1]).id = institution.\"_id\"",
		}
		institution.AddSubSelect(grumble.Computed{
			Formula: "institution.\"Name\"",
			Name:    "InstName",
		})
		institution.AddSubSelect(grumble.Computed{
			Formula: "institution.\"_id\"",
			Name:    "InstIdent",
		})
		q.AddSubQuery(institution)
	}
	return q
}

func (acc *Account) GetQuery(q *grumble.Query) *grumble.Query {
	return addTransactionJoin(q)
}

func (acc *Account) ManyQuery(query *grumble.Query, values url.Values) (ret *grumble.Query) {
	ret = query
	switch {
	case values.Get("institutionid") != "":
		id, err := strconv.ParseInt(values.Get("institutionid"), 0, 0)
		if err != nil {
			return
		}
		k, _ := grumble.CreateKey(nil, grumble.GetKind(&Institution{}), int(id))
		query.AddCondition(grumble.HasParent{Parent: k})
	}
	return addTransactionJoin(query)
}

func accountQuery(mgr *grumble.EntityManager, institution *Institution, id int) (accounts []*Account, err error) {
	q, err := GetAccountQuery(mgr, institution, id)
	if err != nil {
		return
	}
	results, err := q.Execute()
	if err != nil {
		return
	}
	accounts = make([]*Account, len(results))
	for ix, row := range results {
		e := row[0]
		accounts[ix] = e.(*Account)
	}
	return
}

// --------------------------------------------------------------------------

func (tx *Transaction) ManyQuery(query *grumble.Query, values url.Values) (ret *grumble.Query) {
	ret = query
	switch {
	case values.Get("accountid") != "":
		id, err := strconv.ParseInt(values.Get("accountid"), 0, 0)
		if err == nil {
			query = makeTXQuery(query, nil, int(id))
		}
	}
	return
}

func makeTXQuery(query *grumble.Query, acc *Account, accountid int) *grumble.Query {
	query.WithDerived = true
	var err error
	if acc == nil && accountid > 0 {
		var e grumble.Persistable
		e, err = query.Manager.Get(Account{}, accountid)
		acc = e.(*Account)
	}
	if err == nil && acc != nil {
		query.AddCondition(grumble.HasParent{Parent: acc.AsKey()})
	} else {
		query.AddCondition(grumble.SimpleCondition{SQL: "FALSE"})
	}
	query.AddReferenceJoins()
	query.AddSort(grumble.Sort{Column: "Date"})
	return query
}

func (acc *Account) GetTransactions() (txs []*Transaction, err error) {
	q := acc.Manager().MakeQuery(&Transaction{})
	q = makeTXQuery(q, acc, 0)
	results, err := q.Execute()
	if err != nil {
		return
	}
	txs = make([]*Transaction, len(results))
	for ix, row := range results {
		e := row[0]
		switch tx := e.(type) {
		case *OpeningBalanceTx:
			txs[ix] = &(tx.Transaction)
		case *TransferTx:
			txs[ix] = &(tx.Transaction)
		case *Transaction:
			txs[ix] = tx
		}
	}
	return
}

func init() {
	grumble.GetKind(&Category{})
	grumble.GetKind(&Contact{})
	grumble.GetKind(&Project{})
	grumble.GetKind(&Institution{})
	grumble.GetKind(&Account{})
	grumble.GetKind(&Transaction{})
	grumble.GetKind(&OpeningBalanceTx{})
	grumble.GetKind(&TransferTx{})
}
