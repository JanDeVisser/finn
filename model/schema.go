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
	"encoding/json"
	"github.com/JanDeVisser/grumble"
	"io/ioutil"
	"time"
)

func JSONToTime(data map[string]interface{}) (t time.Time, err error) {
	y := 2019
	if data["year"] != nil {
		y = int(data["year"].(float64))
	}
	m := time.Month(1)
	if data["month"] != nil {
		m = time.Month(data["month"].(float64))
	}
	d := 1
	if data["day"] != nil {
		d = int(data["day"].(float64))
	}
	t = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return
}

func ImportInstitutions(data interface{}) (err error) {
	if data == nil {
		return
	}
	institutions := data.([]interface{})
	for _, data = range institutions {
		inst := data.(map[string]interface{})
		i := &Institution{}
		i.Name = inst["inst_name"].(string)
		if err = grumble.Put(i); err != nil {
			return
		}
		accounts := inst["accounts"].([]interface{})
		for _, v := range accounts {
			account := v.(map[string]interface{})
			a := &Account{}
			a.Initialize(i.AsKey(), 0)
			a.AccName = account["acc_name"].(string)
			a.AccNr = account["acc_nr"].(string)
			a.Description = account["description"].(string)
			a.Importer = account["importer"].(string)
			if err = grumble.Put(a); err != nil {
				return err
			}
			var openingDate time.Time
			openingDate, err = JSONToTime(account["opening_date"].(map[string]interface{}))
			if err != nil {
				return
			}
			if err = a.SetOpeningBalance(openingDate, account["opening_balance"].(float64)); err != nil {
				return
			}
		}
	}
	return
}

type Factory func(string) (grumble.Persistable, error)

func categoryFactory(name string) (ret grumble.Persistable, err error) {
	cat := &Category{}
	cat.Name = name
	ret = cat
	return
}

func projectFactory(name string) (ret grumble.Persistable, err error) {
	prj := &Project{}
	prj.Name = name
	ret = prj
	return
}

func importSubTree(parent grumble.Persistable, tree map[string]interface{}, factory Factory) (err error){
	for subName, subTree := range tree {
		var sub grumble.Persistable;
		sub, err = factory(subName)
		if err != nil {
			return
		}
		var pk *grumble.Key = nil
		if parent != nil {
			pk = parent.AsKey()
		}
		sub.Initialize(pk, 0)
		if err = grumble.Put(sub); err != nil {
			return
		}
		subTreeMap := subTree.(map[string]interface{})
		if err = importSubTree(sub, subTreeMap, factory); err != nil {
			return
		}
	}
	return
}

func ImportTree(data interface{}, factory Factory) (err error) {
	if data == nil {
		return
	}
	subTree := data.(map[string]interface{})
	err = importSubTree(nil, subTree, factory)
	return
}

func ImportSchema(fileName string) (err error) {
	var jsonText []byte
	if jsonText, err = ioutil.ReadFile(fileName); err != nil {
		return
	}
	var jsonData interface{}
	err = json.Unmarshal(jsonText, &jsonData)
	if err != nil {
		return
	}
	schema := jsonData.(map[string]interface{})
	if err = ImportInstitutions(schema["institutions"]); err != nil {
		return
	}
	if err = ImportTree(schema["categories"], categoryFactory); err != nil {
		return
	}
	if err = ImportTree(schema["projects"], projectFactory); err != nil {
		return
	}
	return
}
