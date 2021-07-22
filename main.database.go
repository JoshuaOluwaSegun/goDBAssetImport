package main

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	//SQL Package
	"github.com/jmoiron/sqlx"
)

//buildConnectionString -- Build the connection string for the SQL driver
func buildConnectionString() string {
	if SQLImportConf.SQLConf.Database == "" ||
		SQLImportConf.SQLConf.Authentication == "SQL" && (SQLImportConf.SQLConf.UserName == "" || SQLImportConf.SQLConf.Password == "") {
		//Conf not set - log error and return empty string
		logger(4, "Database configuration not set.", true, true)
		return ""
	}
	if SQLImportConf.SQLConf.Driver != "odbc" {
		logger(1, "Connecting to Database Server: "+SQLImportConf.SQLConf.Server, true, true)
	} else {
		logger(1, "Connecting to ODBC Data Source: "+SQLImportConf.SQLConf.Database, true, true)
	}

	connectString := ""
	switch SQLImportConf.SQLConf.Driver {
	case "mssql":
		connectString = "server=" + SQLImportConf.SQLConf.Server
		connectString = connectString + ";database=" + SQLImportConf.SQLConf.Database
		if SQLImportConf.SQLConf.Authentication == "Windows" {
			connectString = connectString + ";Trusted_Connection=True"
		} else {
			connectString = connectString + ";user id=" + SQLImportConf.SQLConf.UserName
			connectString = connectString + ";password=" + SQLImportConf.SQLConf.Password
		}

		if !SQLImportConf.SQLConf.Encrypt {
			connectString = connectString + ";encrypt=disable"
		}
		if SQLImportConf.SQLConf.Port != 0 {
			dbPortSetting := strconv.Itoa(SQLImportConf.SQLConf.Port)
			connectString = connectString + ";port=" + dbPortSetting
		}
	case "mysql":
		connectString = SQLImportConf.SQLConf.UserName + ":" + SQLImportConf.SQLConf.Password
		connectString = connectString + "@tcp(" + SQLImportConf.SQLConf.Server + ":"
		if SQLImportConf.SQLConf.Port != 0 {
			dbPortSetting := strconv.Itoa(SQLImportConf.SQLConf.Port)
			connectString = connectString + dbPortSetting
		} else {
			connectString = connectString + "3306"
		}
		connectString = connectString + ")/" + SQLImportConf.SQLConf.Database
	case "mysql320":
		dbPortSetting := "3306"
		if SQLImportConf.SQLConf.Port != 0 {
			dbPortSetting = strconv.Itoa(SQLImportConf.SQLConf.Port)
		}
		connectString = "tcp:" + SQLImportConf.SQLConf.Server + ":" + dbPortSetting
		connectString = connectString + "*" + SQLImportConf.SQLConf.Database + "/" + SQLImportConf.SQLConf.UserName + "/" + SQLImportConf.SQLConf.Password
	case "odbc":
		connectString = "DSN=" + SQLImportConf.SQLConf.Database + ";UID=" + SQLImportConf.SQLConf.UserName + ";PWD=" + SQLImportConf.SQLConf.Password
	}
	return connectString
}

func makeDBConnection() (db *sqlx.DB, err error) {
	//Connect to the config specified DB
	db, err = sqlx.Open(SQLImportConf.SQLConf.Driver, connString)
	if err != nil {
		err = errors.New("DB Connection Error: " + err.Error())
		return
	}
	//Check connection is open
	err = db.Ping()
	if err != nil {
		err = errors.New("DB Ping Error: " + err.Error())
		return
	}
	return
}

//queryAssets -- Query Asset Database for assets of current type
//-- Builds map of assets, returns true if successful
func queryAssets(sqlAppend string, assetType assetTypesStruct) (bool, map[string]map[string]interface{}) {
	//Initialise Asset Map
	arrAssetMaps := make(map[string]map[string]interface{})

	db, err := makeDBConnection()
	if err != nil {
		logger(4, "[DATABASE] "+err.Error(), true, true)
		return false, arrAssetMaps
	}
	defer db.Close()
	logger(1, " ", false, false)
	logger(3, "[DATABASE] Running database query for "+assetType.AssetType+" assets. Please wait...", true, true)
	//build query
	sqlAssetQuery := BaseSQLQuery + " " + sqlAppend
	logger(3, "[DATABASE] Query for "+assetType.AssetType+" assets:"+sqlAssetQuery, false, true)
	//Run Query
	rows, err := db.Queryx(sqlAssetQuery)
	if err != nil {
		logger(4, " [DATABASE] Database Query Error: "+err.Error(), true, true)
		return false, arrAssetMaps
	}
	defer rows.Close()

	//Build map full of assets
	intAssetCount := 0
	intAssetSuccess := 0
	for rows.Next() {
		intAssetCount++
		results := make(map[string]interface{})
		err = rows.MapScan(results)
		if err != nil {
			logger(4, " [DATABASE] Data Unmarshal Error: "+err.Error(), true, true)
		} else {
			//Stick marshalled data map in to parent slice
			for k, val := range results {
				if results[k] != nil {
					results[k] = iToS(val)
				}
			}
			arrAssetMaps[fmt.Sprintf("%s", results[assetType.AssetIdentifier.DBColumn])] = results
			intAssetSuccess++
		}
	}
	logger(3, "[DATABASE] "+strconv.Itoa(intAssetSuccess)+" of "+strconv.Itoa(intAssetCount)+" returned assets successfully retrieved ready for processing.", true, true)
	return true, arrAssetMaps
}

func querySoftwareInventoryRecords(assetID string, assetTypeDetails assetTypesStruct, db *sqlx.DB, buffer *bytes.Buffer) (map[string]map[string]interface{}, string, error) {
	var (
		recordMap []map[string]interface{}
		returnMap = make(map[string]map[string]interface{})
		hash      string
		err       error
	)

	buffer.WriteString(loggerGen(3, "[DATABASE] Running database query for software inventory records for asset ["+assetID+"]"))
	//build query
	sqlAssetQuery := strings.ReplaceAll(assetTypeDetails.SoftwareInventory.Query, "{{AssetID}}", assetID)
	buffer.WriteString(loggerGen(3, "[DATABASE] Query: "+sqlAssetQuery))

	//Run Query
	rows, err := db.Queryx(sqlAssetQuery)
	if err != nil {
		err = errors.New("[DATABASE] Database Query Error: " + err.Error())
		return returnMap, hash, err
	}
	defer rows.Close()

	//Build map full of software records
	intAssetCount := 0

	for rows.Next() {
		intAssetCount++
		results := make(map[string]interface{})
		err = rows.MapScan(results)
		if err != nil {
			err = errors.New("[DATABASE] Data Unmarshal Error: " + err.Error())
			return returnMap, hash, err
		} else {
			for k, val := range results {
				if results[k] != nil {
					results[k] = iToS(val)
				}
			}
			//Stick marshalled data map in to parent slice
			recordMap = append(recordMap, results)
		}
	}
	recordsHash := Hash(recordMap)
	hash = fmt.Sprintf("%v", recordsHash)

	//Now process return map
	for _, v := range recordMap {
		returnMap[fmt.Sprintf("%s", v[assetTypeDetails.SoftwareInventory.AppIDColumn])] = v
	}
	buffer.WriteString(loggerGen(3, "[DATABASE] "+strconv.Itoa(len(recordMap))+" of "+strconv.Itoa(intAssetCount)+" returned software inventory records successfully retrieved"))
	return returnMap, hash, err

}
