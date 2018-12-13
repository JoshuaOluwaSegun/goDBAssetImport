package main

import (
	"fmt"
	"strconv"
	//SQL Package
	"github.com/jmoiron/sqlx"
)

//buildConnectionString -- Build the connection string for the SQL driver
func buildConnectionString() string {
	if SQLImportConf.SQLConf.Database == "" ||
		SQLImportConf.SQLConf.Authentication == "SQL" && (SQLImportConf.SQLConf.UserName == "" || SQLImportConf.SQLConf.Password == "") {
		//Conf not set - log error and return empty string
		logger(4, "Database configuration not set.", true)
		return ""
	}
	if SQLImportConf.SQLConf.Driver != "odbc" {
		logger(1, "Connecting to Database Server: "+SQLImportConf.SQLConf.Server, true)
	} else {
		logger(1, "Connecting to ODBC Data Source: "+SQLImportConf.SQLConf.Database, true)
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

//queryDatabase -- Query Asset Database for assets of current type
//-- Builds map of assets, returns true if successful
func queryDatabase(sqlAppend, assetTypeName string) (bool, []map[string]interface{}) {
	//Clear existing Asset Map down
	var ArrAssetMaps []map[string]interface{}
	connString := buildConnectionString()
	if connString == "" {
		logger(4, " [DATABASE] Database Connection String Empty. Check the SQLConf section of your configuration.", true)
		return false, ArrAssetMaps
	}
	//Connect to the JSON specified DB
	db, err := sqlx.Open(SQLImportConf.SQLConf.Driver, connString)
	defer db.Close()
	if err != nil {
		logger(4, " [DATABASE] Database Connection Error: "+fmt.Sprintf("%v", err), true)
		return false, ArrAssetMaps
	}
	//Check connection is open
	err = db.Ping()
	if err != nil {
		logger(4, " [DATABASE] [PING] Database Connection Error: "+fmt.Sprintf("%v", err), true)
		return false, ArrAssetMaps
	}
	logger(3, "[DATABASE] Connection Successful", true)
	logger(3, "[DATABASE] Running database query for "+assetTypeName+" assets. Please wait...", true)
	//build query
	sqlAssetQuery := BaseSQLQuery + " " + sqlAppend
	logger(3, "[DATABASE] Query for "+assetTypeName+" assets:"+sqlAssetQuery, false)
	//Run Query
	rows, err := db.Queryx(sqlAssetQuery)
	if err != nil {
		logger(4, " [DATABASE] Database Query Error: "+fmt.Sprintf("%v", err), true)
		return false, ArrAssetMaps
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
			logger(4, " [DATABASE] Data Unmarshal Error: "+fmt.Sprintf("%v", err), true)
		} else {
			//Stick marshalled data map in to parent slice
			ArrAssetMaps = append(ArrAssetMaps, results)
			intAssetSuccess++
		}
	}
	logger(3, "[DATABASE] "+strconv.Itoa(intAssetSuccess)+" of "+strconv.Itoa(intAssetCount)+" returned assets successfully retrieved ready for processing.", true)
	return true, ArrAssetMaps
}
