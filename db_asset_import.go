package main

//----- Packages -----
import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	//SQL Package
	"github.com/hornbill/sqlx"
	//SQL Drivers
	_ "github.com/hornbill/go-mssqldb"
	_ "github.com/hornbill/mysql"
	_ "github.com/hornbill/mysql320" //MySQL v3.2.0 to v5 driver - Provides SWSQL (MySQL 4.0.16) support
)

//----- Constants -----
const version = 1.0
const appServiceManager = "com.hornbill.servicemanager"

//----- Variables -----
var (
	SQLImportConf       sqlImportConfStruct
	XmlmcInstanceConfig xmlmcConfig
	ArrAssetMaps        = make([]map[string]interface{}, 0)
	Sites               []siteListStruct
	counters            counterTypeStruct
	configFileName      string
	configZone          string
	configDryRun        bool
	TimeNow             string
	APITimeNow          string
	startTime           time.Time
	endTime             time.Duration
	AssetClass          string
	AssetTypeID         int
	BaseSQLQuery        string
	StrAssetType        string
	StrSQLAppend        string
	espXmlmc            *apiLib.XmlmcInstStruct
)

//----- Structures -----
type siteListStruct struct {
	SiteName string
	SiteID   int
}
type xmlmcConfig struct {
	instance string
	zone     string
	url      string
}
type counterTypeStruct struct {
	updated        uint16
	created        uint16
	updatedSkipped uint16
	createskipped  uint16
}
type sqlImportConfStruct struct {
	UserName                 string
	Password                 string
	InstanceID               string
	URL                      string
	Entity                   string
	AssetIdentifier          string
	SQLConf                  sqlConfStruct
	AssetTypes               map[string]interface{}
	AssetGenericFieldMapping map[string]interface{}
	AssetTypeFieldMapping    map[string]interface{}
	SiteLookup               siteLookupStruct
	TypeLookup               typeLookupStruct
}
type sqlConfStruct struct {
	Driver   string
	Server   string
	UserName string
	Password string
	Port     int
	Query    string
	Database string
	Encrypt  bool
	AssetID  string
}
type siteLookupStruct struct {
	Enabled  bool
	QueryCol string
}
type typeLookupStruct struct {
	Enabled   bool
	Attribute string
}
type xmlmcResponse struct {
	MethodResult string       `xml:"status,attr"`
	Params       paramsStruct `xml:"params"`
	State        stateStruct  `xml:"state"`
}

//Site Structs
type xmlmcSiteListResponse struct {
	MethodResult string               `xml:"status,attr"`
	Params       paramsSiteListStruct `xml:"params"`
	State        stateStruct          `xml:"state"`
}
type paramsSiteListStruct struct {
	RowData paramsSiteRowDataListStruct `xml:"rowData"`
}
type paramsSiteRowDataListStruct struct {
	Row siteObjectStruct `xml:"row"`
}
type siteObjectStruct struct {
	SiteID      int    `xml:"h_id"`
	SiteName    string `xml:"h_site_name"`
	SiteCountry string `xml:"h_country"`
}

//Asset Structs
type xmlmcAssetResponse struct {
	MethodResult string            `xml:"status,attr"`
	Params       paramsAssetStruct `xml:"params"`
	State        stateStruct       `xml:"state"`
}
type paramsAssetStruct struct {
	RowData paramsAssetRowDataStruct `xml:"rowData"`
}
type paramsAssetRowDataStruct struct {
	Row assetObjectStruct `xml:"row"`
}
type assetObjectStruct struct {
	AssetID    string `xml:"h_pk_asset_id"`
	AssetClass string `xml:"h_class"`
	AssetType  string `xml:"h_country"`
}

//Asset Type Structures
type xmlmcTypeListResponse struct {
	MethodResult string               `xml:"status,attr"`
	Params       paramsTypeListStruct `xml:"params"`
	State        stateStruct          `xml:"state"`
}
type paramsTypeListStruct struct {
	RowData paramsTypeRowDataListStruct `xml:"rowData"`
}
type paramsTypeRowDataListStruct struct {
	Row assetTypeObjectStruct `xml:"row"`
}
type assetTypeObjectStruct struct {
	Type      string `xml:"h_name"`
	TypeClass string `xml:"h_class"`
	TypeID    int    `xml:"h_pk_type_id"`
}
type stateStruct struct {
	Code     string `xml:"code"`
	ErrorRet string `xml:"error"`
}
type paramsStruct struct {
	SessionID string `xml:"sessionId"`
}

//----- Main Function -----
func main() {
	//-- Start Time for Durration
	startTime = time.Now()
	//-- Start Time for Log File
	TimeNow = time.Now().Format(time.RFC3339)
	APITimeNow = strings.Replace(TimeNow, "T", " ", 1)
	APITimeNow = strings.Replace(APITimeNow, "Z", "", 1)
	//-- Remove :
	TimeNow = strings.Replace(TimeNow, ":", "-", -1)
	//-- Grab Flags
	flag.StringVar(&configFileName, "file", "conf.json", "Name of Configuration File To Load")
	flag.StringVar(&configZone, "zone", "eur", "Override the default Zone the instance sits in")
	flag.BoolVar(&configDryRun, "dryrun", false, "Allow the Import to run without Creating or Updating Assets")
	//-- Parse Flags
	flag.Parse()

	//-- Output
	logger(1, "---- XMLMC Database Asset Import Utility V"+fmt.Sprintf("%v", version)+" ----", true)
	logger(1, "Flag - Config File "+fmt.Sprintf("%s", configFileName), true)
	logger(1, "Flag - Zone "+fmt.Sprintf("%s", configZone), true)
	logger(1, "Flag - Dry Run "+fmt.Sprintf("%v", configDryRun), true)
	//--
	//-- Load Configuration File Into Struct
	SQLImportConf = loadConfig()

	//-- Set Instance ID
	SetInstance(configZone, SQLImportConf.InstanceID)
	//-- Generate Instance XMLMC Endpoint
	SQLImportConf.URL = getInstanceURL()

	//-- Login
	var boolLogin = login()
	if boolLogin != true {
		logger(4, "Unable to Login ", true)
		return
	}
	//-- Logout */
	defer logout()

	//Set SWSQLDriver to mysql320
	if SQLImportConf.SQLConf.Driver == "swsql" {
		SQLImportConf.SQLConf.Driver = "mysql320"
	}

	//Get asset types, process accordingly
	BaseSQLQuery = SQLImportConf.SQLConf.Query
	for k, v := range SQLImportConf.AssetTypes {
		StrAssetType = fmt.Sprintf("%v", k)
		StrSQLAppend = fmt.Sprintf("%v", v)
		//Set Asset Class & Type vars from instance
		AssetClass, AssetTypeID = getAssetClass(StrAssetType)
		//-- Query Database
		var boolSQLAssets = queryDatabase(StrSQLAppend, StrAssetType)
		if boolSQLAssets {
			//Process records returned by query
			processAssets()
		}
	}

	//-- End output
	logger(1, "Updated: "+fmt.Sprintf("%d", counters.updated), true)
	logger(1, "Updated Skipped: "+fmt.Sprintf("%d", counters.updatedSkipped), true)
	logger(1, "Created: "+fmt.Sprintf("%d", counters.created), true)
	logger(1, "Created Skipped: "+fmt.Sprintf("%d", counters.createskipped), true)
	//-- Show Time Takens
	endTime = time.Now().Sub(startTime)
	logger(1, "Time Taken: "+fmt.Sprintf("%v", endTime), true)
	logger(1, "---- XMLMC Database Asset Import Complete ---- ", true)
}

//loadConfig -- Function to Load Configruation File
func loadConfig() sqlImportConfStruct {
	//-- Check Config File File Exists
	cwd, _ := os.Getwd()
	configurationFilePath := cwd + "/" + configFileName
	logger(1, "Loading Config File: "+configurationFilePath, false)
	if _, fileCheckErr := os.Stat(configurationFilePath); os.IsNotExist(fileCheckErr) {
		logger(4, "No Configuration File", true)
		os.Exit(102)
	}
	//-- Load Config File
	file, fileError := os.Open(configurationFilePath)
	//-- Check For Error Reading File
	if fileError != nil {
		logger(4, "Error Opening Configuration File: "+fmt.Sprintf("%v", fileError), true)
	}

	//-- New Decoder
	decoder := json.NewDecoder(file)
	//-- New Var based on SQLImportConf
	esqlConf := sqlImportConfStruct{}
	//-- Decode JSON
	err := decoder.Decode(&esqlConf)
	//-- Error Checking
	if err != nil {
		logger(4, "Error Decoding Configuration File: "+fmt.Sprintf("%v", err), true)
	}
	//-- Return New Congfig
	return esqlConf
}

//login -- XMLMC Login
//-- start ESP user session
func login() bool {
	logger(1, "Logging Into: "+SQLImportConf.URL, false)
	logger(1, "UserName: "+SQLImportConf.UserName, false)
	espXmlmc = apiLib.NewXmlmcInstance(SQLImportConf.URL)

	espXmlmc.SetParam("userId", SQLImportConf.UserName)
	espXmlmc.SetParam("password", base64.StdEncoding.EncodeToString([]byte(SQLImportConf.Password)))
	XMLLogin, xmlmcErr := espXmlmc.Invoke("session", "userLogon")
	if xmlmcErr != nil {
		log.Fatal(xmlmcErr)
	}

	var xmlRespon xmlmcResponse
	err := xml.Unmarshal([]byte(XMLLogin), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Login: "+fmt.Sprintf("%v", err), true)
		return false
	}
	if xmlRespon.MethodResult != "ok" {
		logger(4, "Unable to Login: "+xmlRespon.State.ErrorRet, true)
		return false
	}
	espLogger("---- XMLMC Database Asset Import Utility V"+fmt.Sprintf("%v", version)+" ----", "debug")
	espLogger("Logged In As: "+SQLImportConf.UserName, "debug")
	return true
}

//logout -- XMLMC Logout
//-- Adds details to log file, ends user ESP session
func logout() {
	//-- End output
	espLogger("Updated: "+fmt.Sprintf("%d", counters.updated), "debug")
	espLogger("Updated Skipped: "+fmt.Sprintf("%d", counters.updatedSkipped), "debug")
	espLogger("Created: "+fmt.Sprintf("%d", counters.created), "debug")
	espLogger("Created Skipped: "+fmt.Sprintf("%d", counters.createskipped), "debug")
	espLogger("Time Taken: "+fmt.Sprintf("%v", endTime), "debug")
	espLogger("---- XMLMC Database Asset Import Complete ---- ", "debug")
	logger(1, "Logout", true)
	espXmlmc.Invoke("session", "userLogoff")
}

//getAssetClass -- Get Asset Class & Type ID from Asset Type Name
func getAssetClass(confAssetType string) (assetClass string, assetType int) {

	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "AssetsTypes")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_name", confAssetType)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")
	XMLGetMeta, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		log.Fatal(xmlmcErr)
	}

	var xmlRespon xmlmcTypeListResponse
	err := xml.Unmarshal([]byte(XMLGetMeta), &xmlRespon)
	if err != nil {
		logger(4, "Could not get Asset Class and Type. Please check AssetType within your configuration file.", true)
		fmt.Println(err)
	} else {
		assetClass = xmlRespon.Params.RowData.Row.TypeClass
		assetType = xmlRespon.Params.RowData.Row.TypeID
	}
	return
}

//buildConnectionString -- Build the connection string for the SQL driver
func buildConnectionString() string {
	if SQLImportConf.SQLConf.Server == "" || SQLImportConf.SQLConf.Database == "" || SQLImportConf.SQLConf.UserName == "" || SQLImportConf.SQLConf.Password == "" {
		//Conf not set - log error and return empty string
		logger(4, "Database configuration not set.", true)
		return ""
	}
	logger(1, "Connecting to Database Server: "+SQLImportConf.SQLConf.Server, true)
	connectString := ""
	switch SQLImportConf.SQLConf.Driver {
	case "mssql":
		connectString = "server=" + SQLImportConf.SQLConf.Server
		connectString = connectString + ";database=" + SQLImportConf.SQLConf.Database
		connectString = connectString + ";user id=" + SQLImportConf.SQLConf.UserName
		connectString = connectString + ";password=" + SQLImportConf.SQLConf.Password
		if SQLImportConf.SQLConf.Encrypt == false {
			connectString = connectString + ";encrypt=disable"
		}
		if SQLImportConf.SQLConf.Port != 0 {
			var dbPortSetting string
			dbPortSetting = strconv.Itoa(SQLImportConf.SQLConf.Port)
			connectString = connectString + ";port=" + dbPortSetting
		}
	case "mysql":
		connectString = SQLImportConf.SQLConf.UserName + ":" + SQLImportConf.SQLConf.Password
		connectString = connectString + "@tcp(" + SQLImportConf.SQLConf.Server + ":"
		if SQLImportConf.SQLConf.Port != 0 {
			var dbPortSetting string
			dbPortSetting = strconv.Itoa(SQLImportConf.SQLConf.Port)
			connectString = connectString + dbPortSetting
		} else {
			connectString = connectString + "3306"
		}
		connectString = connectString + ")/" + SQLImportConf.SQLConf.Database
	case "mysql320":
		var dbPortSetting string
		if SQLImportConf.SQLConf.Port != 0 {
			dbPortSetting = strconv.Itoa(SQLImportConf.SQLConf.Port)
		} else {
			dbPortSetting = "3306"
		}
		connectString = "tcp:" + SQLImportConf.SQLConf.Server + ":" + dbPortSetting
		connectString = connectString + "*" + SQLImportConf.SQLConf.Database + "/" + SQLImportConf.SQLConf.UserName + "/" + SQLImportConf.SQLConf.Password
	}
	return connectString
}

//queryDatabase -- Query Asset Database for assets of current type
//-- Builds map of assets, returns true if successful
func queryDatabase(sqlAppend, assetTypeName string) bool {
	connString := buildConnectionString()
	if connString == "" {
		return false
	}
	//Connect to the JSON specified DB
	db, err := sqlx.Open(SQLImportConf.SQLConf.Driver, connString)
	defer db.Close()
	if err != nil {
		logger(4, " [DATABASE] Database Connection Error: "+fmt.Sprintf("%v", err), true)
		return false
	}
	//Check connection is open
	err = db.Ping()
	if err != nil {
		logger(4, " [DATABASE] [PING] Database Connection Error: "+fmt.Sprintf("%v", err), true)
		return false
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
		return false
	}
	//Clear existing Asset Map down
	ArrAssetMaps = nil
	//Build map full of assets
	intAssetCount := 0
	for rows.Next() {
		intAssetCount++
		results := make(map[string]interface{})
		err = rows.MapScan(results)
		//Stick marshalled data map in to parent slice
		ArrAssetMaps = append(ArrAssetMaps, results)
	}
	defer rows.Close()
	return true
}

//processAssets -- Processes Assets from Asset Map
//--If asset already exists on the instance, update
//--If asset doesn't exist, create
func processAssets() {
	bar := pb.StartNew(len(ArrAssetMaps))
	logger(1, "Processing Assets", false)
	//-- Loop each asset
	for _, assetRecord := range ArrAssetMaps {
		bar.Increment()
		var boolUpdate = false
		//Get the identity of the AssetID field from the config
		assetIDIdent := fmt.Sprintf("%v", SQLImportConf.SQLConf.AssetID)
		//Get the asset ID for the current record
		assetID := fmt.Sprintf("%v", assetRecord[assetIDIdent])
		logger(1, "Asset ID: "+fmt.Sprintf("%v", assetID), false)
		//-- For Each Asset, check if it already exists
		//boolUpdate = checkAssetOnInstance(assetID)
		boolUpdate, assetIDInstance := getAssetID(assetID)
		//-- Update or Create Asset
		if boolUpdate {
			logger(1, "Update Asset: "+assetID, false)
			updateAsset(assetRecord, assetIDInstance)
		} else {
			logger(1, "Create Asset: "+assetID, false)
			createAsset(assetRecord)
		}
	}
	bar.FinishPrint("Processing Complete!")
}

//getAssetID -- Check if asset is on the instance
//-- Returns true, assetid if so
//-- Returns false, "" if not
func getAssetID(assetName string) (bool, string) {
	boolReturn := false
	returnAssetID := ""

	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Asset")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_name", assetName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")
	XMLAssetSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		log.Fatal(xmlmcErr)
	} else {
		var xmlRespon xmlmcAssetResponse
		err := xml.Unmarshal([]byte(XMLAssetSearch), &xmlRespon)
		if err != nil {
			logger(3, "Unable to Search for Asset: "+fmt.Sprintf("%v", err), true)
		} else {
			if xmlRespon.MethodResult != "ok" {
				logger(3, "Unable to Search for Asset: "+xmlRespon.State.ErrorRet, true)
			} else {
				returnAssetID = xmlRespon.Params.RowData.Row.AssetID
				//-- Check Response
				if returnAssetID != "" {
					boolReturn = true
				}
			}
		}
	}
	return boolReturn, returnAssetID
}

// createAsset -- Creates Asset record from the passed through map data
func createAsset(u map[string]interface{}) bool {
	//Get site ID
	siteID := ""
	siteNameMapping := fmt.Sprintf("%v", SQLImportConf.AssetGenericFieldMapping["h_site"])
	siteName := getFieldValue("h_site", siteNameMapping, u)
	if siteName != "" {
		siteIsInCache, SiteIDCache := siteInCache(siteName)
		//-- Check if we have cached the site already
		if siteIsInCache {
			siteID = strconv.Itoa(SiteIDCache)
		} else {
			siteIsOnInstance, SiteIDInstance := searchSite(siteName)
			//-- If Returned set output
			if siteIsOnInstance {
				siteID = strconv.Itoa(SiteIDInstance)
			}
		}
	}

	//Get/Set params from map stored against FieldMapping
	strAttribute := ""
	strMapping := ""
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Asset")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	//Set Class & TypeID
	espXmlmc.SetParam("h_class", AssetClass)
	espXmlmc.SetParam("h_type", strconv.Itoa(AssetTypeID))

	espXmlmc.SetParam("h_last_updated", APITimeNow)
	espXmlmc.SetParam("h_last_updated_by", "Import - Add")

	//Get asset field mapping
	for k, v := range SQLImportConf.AssetGenericFieldMapping {
		strAttribute = fmt.Sprintf("%v", k)
		strMapping = fmt.Sprintf("%v", v)
		if strAttribute == "h_site" && siteID != "" && siteName != "" {
			espXmlmc.SetParam("h_site", siteName)
			espXmlmc.SetParam("h_site_id", siteID)
		}
		if strAttribute != "h_site" && strMapping != "" && getFieldValue(strAttribute, strMapping, u) != "" {
			espXmlmc.SetParam(strAttribute, getFieldValue(strAttribute, strMapping, u))
		}
	}
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")

	//Add extended asset type field mapping
	espXmlmc.OpenElement("relatedEntityData")
	//Set Class & TypeID
	espXmlmc.SetParam("relationshipName", "AssetClass")
	espXmlmc.SetParam("entityAction", "insert")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_type", strconv.Itoa(AssetTypeID))
	//Get asset field mapping
	for k, v := range SQLImportConf.AssetTypeFieldMapping {
		strAttribute = fmt.Sprintf("%v", k)
		strMapping = fmt.Sprintf("%v", v)
		if strMapping != "" && getFieldValue(strAttribute, strMapping, u) != "" {
			espXmlmc.SetParam(strAttribute, getFieldValue(strAttribute, strMapping, u))
		}
	}
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("relatedEntityData")

	//-- Check for Dry Run
	if configDryRun != true {
		XMLCreate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
		if xmlmcErr != nil {
			log.Fatal(xmlmcErr)
		}
		var xmlRespon xmlmcResponse

		err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
		if err != nil {
			counters.createskipped++
			logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
			return false
		}
		if xmlRespon.MethodResult != "ok" {
			logger(3, "Unable to add asset: "+xmlRespon.State.ErrorRet, false)
			counters.createskipped++
		} else {
			counters.created++
			return true
		}
	} else {
		//-- DEBUG XML TO LOG FILE
		var XMLSTRING = espXmlmc.GetParam()
		logger(1, "Asset Create XML "+fmt.Sprintf("%s", XMLSTRING), false)
		counters.createskipped++
		espXmlmc.ClearParam()
	}
	return true
}

// updateAsset -- Updates Asset record from the passed through map data and asset ID
func updateAsset(u map[string]interface{}, strAssetID string) bool {
	//Get site ID
	siteID := ""
	siteNameMapping := fmt.Sprintf("%v", SQLImportConf.AssetGenericFieldMapping["h_site"])
	siteName := getFieldValue("h_site", siteNameMapping, u)
	if siteName != "" {
		siteIsInCache, SiteIDCache := siteInCache(siteName)
		//-- Check if we have cached the site already
		if siteIsInCache {
			siteID = strconv.Itoa(SiteIDCache)
		} else {
			siteIsOnInstance, SiteIDInstance := searchSite(siteName)
			//-- If Returned set output
			if siteIsOnInstance {
				siteID = strconv.Itoa(SiteIDInstance)
			}
		}
	}

	//Get/Set params from map stored against FieldMapping
	strAttribute := ""
	strMapping := ""
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Asset")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_pk_asset_id", strAssetID)

	//Get asset field mapping
	for k, v := range SQLImportConf.AssetGenericFieldMapping {
		strAttribute = fmt.Sprintf("%v", k)
		strMapping = fmt.Sprintf("%v", v)
		if strAttribute == "h_site" && siteID != "" && siteName != "" {
			espXmlmc.SetParam("h_site", siteName)
			espXmlmc.SetParam("h_site_id", siteID)
		}
		if strAttribute != "h_site" && strMapping != "" && getFieldValue(strAttribute, strMapping, u) != "" {
			espXmlmc.SetParam(strAttribute, getFieldValue(strAttribute, strMapping, u))
		}
	}
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")

	//Add extended asset type field mapping
	espXmlmc.OpenElement("relatedEntityData")
	//Set Class & TypeID
	espXmlmc.SetParam("relationshipName", "AssetClass")
	espXmlmc.SetParam("entityAction", "update")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_pk_asset_id", strAssetID)
	//Get asset field mapping
	for k, v := range SQLImportConf.AssetTypeFieldMapping {
		strAttribute = fmt.Sprintf("%v", k)
		strMapping = fmt.Sprintf("%v", v)
		if strMapping != "" && getFieldValue(strAttribute, strMapping, u) != "" {
			espXmlmc.SetParam(strAttribute, getFieldValue(strAttribute, strMapping, u))
		}
	}
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("relatedEntityData")
	//-- Check for Dry Run
	if configDryRun != true {
		XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
		if xmlmcErr != nil {
			log.Fatal(xmlmcErr)
		}
		var xmlRespon xmlmcResponse

		err := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
		if err != nil {
			logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
			counters.updatedSkipped++
			return false
		}
		if xmlRespon.MethodResult != "ok" && xmlRespon.State.ErrorRet != "There are no values to update" {
			logger(3, "Unable to Update Asset: "+xmlRespon.State.ErrorRet, false)
			counters.updatedSkipped++
		} else {
			if xmlRespon.State.ErrorRet != "There are no values to update" {
				//-- Asset Updated!
				//-- Need to run another update against the Asset for LAST UPDATED and LAST UPDATE BY!
				espXmlmc.SetParam("application", appServiceManager)
				espXmlmc.SetParam("entity", "Asset")
				espXmlmc.OpenElement("primaryEntityData")
				espXmlmc.OpenElement("record")
				espXmlmc.SetParam("h_pk_asset_id", strAssetID)
				espXmlmc.SetParam("h_last_updated", APITimeNow)
				espXmlmc.SetParam("h_last_updated_by", "Import - Update")
				espXmlmc.CloseElement("record")
				espXmlmc.CloseElement("primaryEntityData")
				XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
				if xmlmcErr != nil {
					log.Fatal(xmlmcErr)
				}
				var xmlRespon xmlmcResponse
				err := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
				if err != nil {
					logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
					counters.updatedSkipped++
					return false
				}
				if xmlRespon.MethodResult != "ok" && xmlRespon.State.ErrorRet != "There are no values to update" {
					logger(3, "Unable to update Last Updated details for asset: "+xmlRespon.State.ErrorRet, false)
				}
				counters.updated++
			} else {
				logger(3, "There are no values to update.", false)
				counters.updatedSkipped++
			}
		}
	} else {
		//-- Inc Counter
		counters.updatedSkipped++
		//-- DEBUG XML TO LOG FILE
		var XMLSTRING = espXmlmc.GetParam()
		logger(1, "Asset Update XML "+fmt.Sprintf("%s", XMLSTRING), false)
		espXmlmc.ClearParam()
	}
	return true
}

// getFieldValue --Retrieve field value from mapping via SQL record map
func getFieldValue(k string, v string, u map[string]interface{}) string {
	fieldMap := v
	//-- Match $variable from String
	re1, err := regexp.Compile(`\[(.*?)\]`)
	if err != nil {
		fmt.Printf("[ERROR] %v", err)
	}

	result := re1.FindAllString(fieldMap, 100)
	valFieldMap := ""
	//-- Loop Matches
	for _, val := range result {
		valFieldMap = ""
		valFieldMap = strings.Replace(val, "[", "", 1)
		valFieldMap = strings.Replace(valFieldMap, "]", "", 1)
		if valFieldMap == "HBAssetType" {
			valFieldMap = StrAssetType
		} else {
			if SQLImportConf.SQLConf.Driver == "mysql320" {
				valFieldMap = fmt.Sprintf("%s", u[valFieldMap])
			} else {
				valFieldMap = fmt.Sprintf("%v", u[valFieldMap])
			}
		}
		if valFieldMap != "" {
			if strings.Contains(strings.ToLower(k), "date") == true {
				valFieldMap = checkDateString(valFieldMap)
			}
			if strings.Contains(valFieldMap, "[") == true {
				fieldMap = strings.Replace(fieldMap, val, "", 1)
			} else {
				fieldMap = strings.Replace(fieldMap, val, valFieldMap, 1)
			}
		} else {
			fieldMap = strings.Replace(fieldMap, val, "", 1)
		}
	}
	return fieldMap
}

// siteInCache -- Function to check if passed-thorugh site name has been cached
// if so, pass back the Site ID
func siteInCache(siteName string) (bool, int) {
	boolReturn := false
	intReturn := 0
	//-- Check if in Cache
	for _, site := range Sites {
		if site.SiteName == siteName {
			boolReturn = true
			intReturn = site.SiteID
		}
	}
	return boolReturn, intReturn
}

// seachSite -- Function to check if passed-through  site  name is on the instance
func searchSite(siteName string) (bool, int) {
	boolReturn := false
	intReturn := 0
	//-- ESP Query for site

	espXmlmc.SetParam("entity", "Site")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_site_name", siteName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcSiteListResponse

	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		logger(3, "Unable to Search for Site: "+fmt.Sprintf("%v", err), true)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(3, "Unable to Search for Site: "+xmlRespon.State.ErrorRet, true)
		} else {
			//-- Check Response
			if xmlRespon.Params.RowData.Row.SiteName != "" {
				if strings.ToLower(xmlRespon.Params.RowData.Row.SiteName) == strings.ToLower(siteName) {
					intReturn = xmlRespon.Params.RowData.Row.SiteID
					boolReturn = true
					//-- Add Site to Cache
					var newSiteForCache siteListStruct
					newSiteForCache.SiteID = intReturn
					newSiteForCache.SiteName = siteName
					name := []siteListStruct{newSiteForCache}
					Sites = append(Sites, name...)
				}
			}
		}
	}
	return boolReturn, intReturn
}

// logger -- function to append to the current log file
func logger(t int, s string, outputtoCLI bool) {
	//-- Current working dir
	cwd, _ := os.Getwd()

	//-- Log Folder
	logPath := cwd + "/log"
	//-- Log File
	logFileName := logPath + "/Asset_Import_" + TimeNow + ".log"

	//-- If Folder Does Not Exist then create it
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		err := os.Mkdir(logPath, 0777)
		if err != nil {
			fmt.Printf("Error Creating Log Folder %q: %s \r", logPath, err)
			os.Exit(101)
		}
	}

	//-- Open Log File
	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		fmt.Printf("Error Creating Log File %q: %s \n", logFileName, err)
		os.Exit(100)
	}
	// don't forget to close it
	defer f.Close()
	// assign it to the standard logger
	log.SetOutput(f)
	var errorLogPrefix string
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
	case 2:
		errorLogPrefix = "[MESSAGE] "
	case 4:
		errorLogPrefix = "[ERROR] "
	}
	if outputtoCLI {
		fmt.Printf("%v \n", errorLogPrefix+s)
	}
	log.Println(errorLogPrefix + s)
}

// SetInstance sets the Zone and Instance config from the passed-through strZone and instanceID values
func SetInstance(strZone string, instanceID string) {
	//-- Set Zone
	SetZone(strZone)
	//-- Set Instance
	XmlmcInstanceConfig.instance = instanceID
	return
}

// SetZone - sets the Instance Zone to Overide current live zone
func SetZone(zone string) {
	XmlmcInstanceConfig.zone = zone
	return
}

// espLogger -- Log to ESP
func espLogger(message string, severity string) {

	espXmlmc.SetParam("fileName", "SQL_Asset_Import")
	espXmlmc.SetParam("group", "general")
	espXmlmc.SetParam("severity", severity)
	espXmlmc.SetParam("message", message)
	espXmlmc.Invoke("system", "logMessage")
}

// getInstanceURL -- Function to build XMLMC End Point
func getInstanceURL() string {
	XmlmcInstanceConfig.url = "https://"
	XmlmcInstanceConfig.url += XmlmcInstanceConfig.zone
	XmlmcInstanceConfig.url += "api.hornbill.com/"
	XmlmcInstanceConfig.url += XmlmcInstanceConfig.instance
	XmlmcInstanceConfig.url += "/xmlmc/"
	return XmlmcInstanceConfig.url
}

// checkDateString - returns date from supplied string
func checkDateString(strDate string) string {
	re, _ := regexp.Compile("\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}")
	strNewDate := re.FindString(strDate)
	return strNewDate
}
