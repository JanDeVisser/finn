#!/usr/bin/env bash

curl -D - http://localhost:8080/tools/reset
curl -D - -F 'schema=@../data/my_schema.json' http://localhost:8080/schema/upload
curl -D - -F 'csv=@../data/ManulifeOne/initial/02212019_Transactions.csv' http://localhost:8080/account/upload/1
curl -D - http://localhost:8080/json/account/1
curl -D - http://localhost:8080/json/transaction?accountid=1

