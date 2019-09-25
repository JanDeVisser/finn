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
	"time"
)

type Category struct {
	grumble.Key
	Name string
	Description string
	CurrentBalance float64 `grumble:"transient"`
}

type Project struct {
	grumble.Key
	Name string
	Description string
	Category *Category
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
	Name string
	Description string
	Accounts []*Account `grumble:"transient"`
}

func GetInstitutions() (institutions []*Institution, err error) {
	q := grumble.MakeQuery(&Institution{})
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
	institution.Accounts, err = GetAccounts(institution)
	return
}

//type TransactionType string
const (
	Debit = "D"
	Credit = "C"
	Transfer = "T"
	OpeningBalance = "O"
	Adjustment = "A"
)

type Transaction struct {
	grumble.Key
	Date time.Time
	TXType string
	Amt float64
	Currency string `grumble:"default=CAD"`
	ForeignAmt float64
	Debit float64 `grumble:"verbosename=Out;formula=(CASE WHEN \"Amt\" < 0 THEN -\"Amt\" ELSE 0 END)"`
	Credit float64 `grumble:"verbosename=In;formula=(CASE WHEN \"Amt\" > 0 THEN \"Amt\" ELSE 0 END)"`
	Description string
	Consolidated bool
	Category *Category
	Project *Project
	Contact *Contact
}

type OpeningBalanceTx struct {
	Transaction
}

type TransferTx struct {
	Transaction
	CrossPost *TransferTx
	Account *Account
}

type Account struct {
	grumble.Key
	AccName string `grumble:"verbosename=Account name;label"`
	AccNr string `grumble:"verbosename=Account #"`
	Description string
	Currency string `grumble:"default=CAD"`
	Importer string
	OpeningDate time.Time   `grumble:"transient"`
	OpeningBalance float64  `grumble:"transient"`
	CurrentBalance float64  `grumble:"transient"`
	TotalDebit float64      `grumble:"transient"`
	TotalCredit float64     `grumble:"transient"`
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
	return grumble.Put(tx)
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
		_ = tx.Initialize(acc.AsKey(), 0)
	}
	return
}

func GetAccounts(institution *Institution) (accounts []*Account, err error) {
	return accountQuery(institution, 0)
}

func GetAccount(id int) (account *Account, err error) {
	accounts, err := accountQuery(nil, id)
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

func accountQuery(institution *Institution, id int) (accounts []*Account, err error) {
	q := grumble.MakeQuery(&Account{})
	q.WithDerived = true
	q.GroupBy = true
	if institution != nil {
		q.AddCondition(grumble.HasParent{Parent: institution.AsKey()})
	}
	if id != 0 {
		q.AddCondition(grumble.HasId{Id: id})
	}
	txJoin := grumble.Join{
		QueryTable: grumble.QueryTable{
			Kind:        grumble.GetKind(&Transaction{}),
			WithDerived: true,
			GroupBy:     false,
		},
		JoinType:   grumble.Outer,
		FieldName:  "_parent",
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

func (acc *Account) GetTransactions() (txs []*Transaction, err error) {
	q := grumble.MakeQuery(&Transaction{})
	q.WithDerived = true
	q.AddReferenceJoins()
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