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
	if key.Database == "" ||
		importConf.SourceConfig.Database.Authentication == "SQL" && (key.Username == "" || key.Password == "") {
		//Conf not set - log error and return empty string
		logger(4, "Database configuration not set.", true, true)
		return ""
	}
	if importConf.SourceConfig.Source != "odbc" {
		logger(3, "Connecting to Database Server: "+key.Server, true, true)
	} else {
		logger(3, "Connecting to ODBC Data Source: "+key.Database, true, true)
	}

	connectString := ""
	switch importConf.SourceConfig.Source {
	case "mssql":
		connectString = "server=" + key.Server
		connectString = connectString + ";database=" + key.Database
		if importConf.SourceConfig.Database.Authentication == "Windows" {
			connectString = connectString + ";Trusted_Connection=True"
		} else {
			connectString = connectString + ";user id=" + key.Username
			connectString = connectString + ";password=" + key.Password
		}

		if !importConf.SourceConfig.Database.Encrypt {
			connectString = connectString + ";encrypt=disable"
		}
		if key.Port != 0 {
			dbPortSetting := strconv.Itoa(int(key.Port))
			connectString = connectString + ";port=" + dbPortSetting
		}
	case "mysql":
		connectString = key.Username + ":" + key.Password
		connectString = connectString + "@tcp(" + key.Server + ":"
		if key.Port != 0 {
			dbPortSetting := strconv.Itoa(int(key.Port))
			connectString = connectString + dbPortSetting
		} else {
			connectString = connectString + "3306"
		}
		connectString = connectString + ")/" + key.Database
	case "mysql320":
		dbPortSetting := "3306"
		if key.Port != 0 {
			dbPortSetting = strconv.Itoa(int(key.Port))
		}
		connectString = "tcp:" + key.Server + ":" + dbPortSetting
		connectString = connectString + "*" + key.Database + "/" + key.Username + "/" + key.Password
	case "odbc":
		connectString = "DSN=" + key.Database + ";UID=" + key.Username + ";PWD=" + key.Password
	}
	
	BaseSQLQuery = importConf.SourceConfig.Database.Query
	
	return connectString
}

func makeDBConnection() (db *sqlx.DB, err error) {
	//Connect to the config specified DB
	db, err = sqlx.Open(importConf.SourceConfig.Source, connString)
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
	logger(3, " ", false, false)
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
			arrAssetMaps[fmt.Sprintf("%s", results[assetType.AssetIdentifier.SourceColumn])] = results
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
