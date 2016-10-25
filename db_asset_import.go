package main

//----- Packages -----
import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/hornbill/color"
	"github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	//SQL Package
	"github.com/hornbill/sqlx"
	//SQL Drivers
	_ "github.com/hornbill/go-mssqldb"
	_ "github.com/hornbill/mysql"
	_ "github.com/hornbill/mysql320" //MySQL v3.2.0 to v5 driver - Provides SWSQL (MySQL 4.0.16) support
)

//----- Constants -----
const version = "1.2.1"
const appServiceManager = "com.hornbill.servicemanager"

//----- Variables -----
var (
	SQLImportConf       sqlImportConfStruct
	XmlmcInstanceConfig xmlmcConfig
	Sites               []siteListStruct
	counters            counterTypeStruct
	configFileName      string
	configMaxRoutines   string
	configZone          string
	configDryRun        bool
	Customers           []customerListStruct
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
	mutex               = &sync.Mutex{}
	mutexBar            = &sync.Mutex{}
	mutexCounters       = &sync.Mutex{}
	mutexCustomers      = &sync.Mutex{}
	mutexSite           = &sync.Mutex{}
	worker              sync.WaitGroup
	maxGoroutines       = 6
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
	APIKey                   string
	InstanceID               string
	URL                      string
	Entity                   string
	AssetIdentifier          string
	SQLConf                  sqlConfStruct
	AssetTypes               map[string]interface{}
	AssetGenericFieldMapping map[string]interface{}
	AssetTypeFieldMapping    map[string]interface{}
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

//----- Customer Structs
type customerListStruct struct {
	CustomerID   string
	CustomerName string
}
type xmlmcCustomerListResponse struct {
	MethodResult      string      `xml:"status,attr"`
	CustomerFirstName string      `xml:"params>firstName"`
	CustomerLastName  string      `xml:"params>lastName"`
	State             stateStruct `xml:"state"`
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
	flag.StringVar(&configMaxRoutines, "concurrent", "1", "Maximum number of requests to import concurrently.")
	//-- Parse Flags
	flag.Parse()

	//-- Output
	logger(1, "---- XMLMC Database Asset Import Utility V"+version+" ----", true)
	logger(1, "Flag - Config File "+fmt.Sprintf("%s", configFileName), true)
	logger(1, "Flag - Zone "+fmt.Sprintf("%s", configZone), true)
	logger(1, "Flag - Dry Run "+fmt.Sprintf("%v", configDryRun), true)

	//Check maxGoroutines for valid value
	maxRoutines, err := strconv.Atoi(configMaxRoutines)
	if err != nil {
		color.Red("Unable to convert maximum concurrency of [" + configMaxRoutines + "] to type INT for processing")
		return
	}
	maxGoroutines = maxRoutines

	if maxGoroutines < 1 || maxGoroutines > 10 {
		color.Red("The maximum concurrent requests allowed is between 1 and 10 (inclusive).\n\n")
		color.Red("You have selected " + configMaxRoutines + ". Please try again, with a valid value against ")
		color.Red("the -concurrent switch.")
		return
	}

	//--
	//-- Load Configuration File Into Struct
	SQLImportConf = loadConfig()

	//-- Set Instance ID
	SetInstance(configZone, SQLImportConf.InstanceID)
	//-- Generate Instance XMLMC Endpoint
	SQLImportConf.URL = getInstanceURL()

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
		var boolSQLAssets, arrAssets = queryDatabase(StrSQLAppend, StrAssetType)
		if boolSQLAssets {
			//Process records returned by query
			processAssets(arrAssets)
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

//getAssetClass -- Get Asset Class & Type ID from Asset Type Name
func getAssetClass(confAssetType string) (assetClass string, assetType int) {
	espXmlmc := apiLib.NewXmlmcInstance(SQLImportConf.URL)
	espXmlmc.SetAPIKey(SQLImportConf.APIKey)
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
func queryDatabase(sqlAppend, assetTypeName string) (bool, []map[string]interface{}) {
	//Clear existing Asset Map down
	ArrAssetMaps := make([]map[string]interface{}, 0)
	connString := buildConnectionString()
	if connString == "" {
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
	return true, ArrAssetMaps
}

//processAssets -- Processes Assets from Asset Map
//--If asset already exists on the instance, update
//--If asset doesn't exist, create
func processAssets(arrAssets []map[string]interface{}) {
	bar := pb.StartNew(len(arrAssets))
	logger(1, "Processing Assets", false)

	//Get the identity of the AssetID field from the config
	assetIDIdent := fmt.Sprintf("%v", SQLImportConf.SQLConf.AssetID)

	//-- Loop each asset
	maxGoroutinesGuard := make(chan struct{}, maxGoroutines)

	for _, assetRecord := range arrAssets {
		maxGoroutinesGuard <- struct{}{}
		worker.Add(1)
		assetMap := assetRecord
		//Get the asset ID for the current record
		assetID := fmt.Sprintf("%v", assetMap[assetIDIdent])
		logger(1, "Asset ID: "+fmt.Sprintf("%v", assetID), false)
		espXmlmc := apiLib.NewXmlmcInstance(SQLImportConf.URL)
		espXmlmc.SetAPIKey(SQLImportConf.APIKey)
		go func() {
			defer worker.Done()
			time.Sleep(1 * time.Millisecond)
			mutexBar.Lock()
			bar.Increment()
			mutexBar.Unlock()

			var boolUpdate = false
			boolUpdate, assetIDInstance := getAssetID(assetID, espXmlmc)
			//-- Update or Create Asset
			if boolUpdate {
				logger(1, "Update Asset: "+assetID, false)
				updateAsset(assetMap, assetIDInstance, espXmlmc)
			} else {
				logger(1, "Create Asset: "+assetID, false)
				createAsset(assetMap, espXmlmc)
			}
			<-maxGoroutinesGuard
		}()
	}
	worker.Wait()
	bar.FinishPrint("Processing Complete!")
}

//getAssetID -- Check if asset is on the instance
//-- Returns true, assetid if so
//-- Returns false, "" if not
func getAssetID(assetName string, espXmlmc *apiLib.XmlmcInstStruct) (bool, string) {
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
func createAsset(u map[string]interface{}, espXmlmc *apiLib.XmlmcInstStruct) bool {
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
			siteIsOnInstance, SiteIDInstance := searchSite(siteName, espXmlmc)
			//-- If Returned set output
			if siteIsOnInstance {
				siteID = strconv.Itoa(SiteIDInstance)
			}
		}
	}

	//Get Owned By name
	ownedByName := ""
	ownedByURN := ""
	ownedByMapping := fmt.Sprintf("%v", SQLImportConf.AssetGenericFieldMapping["h_owned_by"])
	ownedByID := getFieldValue("h_owned_by", ownedByMapping, u)
	if ownedByID != "" {
		ownedByIsInCache, ownedByNameCache := customerInCache(ownedByID)
		//-- Check if we have cached the customer already
		if ownedByIsInCache {
			ownedByName = ownedByNameCache
		} else {
			ownedByIsOnInstance, ownedByNameInstance := searchCustomer(ownedByID, espXmlmc)
			//-- If Returned set output
			if ownedByIsOnInstance {
				ownedByName = ownedByNameInstance
			}
		}
	}
	if ownedByName != "" {
		ownedByURN = "urn:sys:0:" + ownedByName + ":" + ownedByID
	}

	//Get Used By name
	usedByName := ""
	usedByURN := ""
	usedByMapping := fmt.Sprintf("%v", SQLImportConf.AssetGenericFieldMapping["h_used_by"])
	usedByID := getFieldValue("h_owned_by", usedByMapping, u)
	if usedByID != "" {
		usedByIsInCache, usedByNameCache := customerInCache(usedByID)
		//-- Check if we have cached the customer already
		if usedByIsInCache {
			usedByName = usedByNameCache
		} else {
			usedByIsOnInstance, usedByNameInstance := searchCustomer(usedByID, espXmlmc)
			//-- If Returned set output
			if usedByIsOnInstance {
				usedByName = usedByNameInstance
			}
		}
	}
	if usedByName != "" {
		usedByURN = "urn:sys:0:" + usedByName + ":" + usedByID
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
		if strAttribute == "h_used_by" && usedByName != "" && usedByURN != "" {
			espXmlmc.SetParam("h_used_by", usedByURN)
			espXmlmc.SetParam("h_used_by_name", usedByName)
		}
		if strAttribute == "h_owned_by" && ownedByName != "" && ownedByURN != "" {
			espXmlmc.SetParam("h_owned_by", ownedByURN)
			espXmlmc.SetParam("h_owned_by_name", ownedByName)
		}
		if strAttribute == "h_site" && siteID != "" && siteName != "" {
			espXmlmc.SetParam("h_site", siteName)
			espXmlmc.SetParam("h_site_id", siteID)
		}
		if strAttribute != "h_site" && strAttribute != "h_used_by" && strAttribute != "h_owned_by" && strMapping != "" && getFieldValue(strAttribute, strMapping, u) != "" {
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
			mutexCounters.Lock()
			counters.createskipped++
			mutexCounters.Unlock()
			logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
			return false
		}
		if xmlRespon.MethodResult != "ok" {
			logger(3, "Unable to add asset: "+xmlRespon.State.ErrorRet, false)
			mutexCounters.Lock()
			counters.createskipped++
			mutexCounters.Unlock()
		} else {
			mutexCounters.Lock()
			counters.created++
			mutexCounters.Unlock()
			return true
		}
	} else {
		//-- DEBUG XML TO LOG FILE
		var XMLSTRING = espXmlmc.GetParam()
		logger(1, "Asset Create XML "+fmt.Sprintf("%s", XMLSTRING), false)
		mutexCounters.Lock()
		counters.createskipped++
		mutexCounters.Unlock()
		espXmlmc.ClearParam()
	}
	return true
}

// updateAsset -- Updates Asset record from the passed through map data and asset ID
func updateAsset(u map[string]interface{}, strAssetID string, espXmlmc *apiLib.XmlmcInstStruct) bool {
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
			siteIsOnInstance, SiteIDInstance := searchSite(siteName, espXmlmc)
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
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
			return false
		}
		if xmlRespon.MethodResult != "ok" && xmlRespon.State.ErrorRet != "There are no values to update" {
			logger(3, "Unable to Update Asset: "+xmlRespon.State.ErrorRet, false)
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
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
					mutexCounters.Lock()
					counters.updatedSkipped++
					mutexCounters.Unlock()
					return false
				}
				if xmlRespon.MethodResult != "ok" && xmlRespon.State.ErrorRet != "There are no values to update" {
					logger(3, "Unable to update Last Updated details for asset: "+xmlRespon.State.ErrorRet, false)
				}
				mutexCounters.Lock()
				counters.updated++
				mutexCounters.Unlock()
			} else {
				logger(3, "There are no values to update.", false)
				mutexCounters.Lock()
				counters.updatedSkipped++
				mutexCounters.Unlock()
			}
		}
	} else {
		//-- Inc Counter
		mutexCounters.Lock()
		counters.updatedSkipped++
		mutexCounters.Unlock()
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
				valFieldMap = ""
			} else {
				//20160215 Check for NULL (<nil>) field value
				//Cannot do this when Scanning SQL data, as we don't now the returned cols - we're using MapScan
				if valFieldMap == "<nil>" {
					valFieldMap = ""
				}
			}
		}
		fieldMap = strings.Replace(fieldMap, val, valFieldMap, 1)
	}
	return fieldMap
}

// siteInCache -- Function to check if passed-thorugh site name has been cached
// if so, pass back the Site ID
func siteInCache(siteName string) (bool, int) {
	boolReturn := false
	intReturn := 0
	mutexSite.Lock()
	//-- Check if in Cache
	for _, site := range Sites {
		if site.SiteName == siteName {
			boolReturn = true
			intReturn = site.SiteID
		}
	}
	mutexSite.Unlock()
	return boolReturn, intReturn
}

// seachSite -- Function to check if passed-through  site  name is on the instance
func searchSite(siteName string, espXmlmc *apiLib.XmlmcInstStruct) (bool, int) {
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
					mutexSite.Lock()
					Sites = append(Sites, name...)
					mutexSite.Unlock()
				}
			}
		}
	}
	return boolReturn, intReturn
}

// customerInCache -- Function to check if passed-thorugh Customer ID has been cached
// if so, pass back the Customer Name
func customerInCache(customerID string) (bool, string) {
	boolReturn := false
	strReturn := ""
	mutexCustomers.Lock()
	//-- Check if in Cache
	for _, customer := range Customers {
		if customer.CustomerID == customerID {
			boolReturn = true
			strReturn = customer.CustomerName
		}
	}
	mutexCustomers.Unlock()
	return boolReturn, strReturn
}

// seachSite -- Function to check if passed-through  site  name is on the instance
func searchCustomer(custID string, espXmlmc *apiLib.XmlmcInstStruct) (bool, string) {
	boolReturn := false
	strReturn := ""
	//Get Analyst Info
	espXmlmc.SetParam("customerId", custID)
	espXmlmc.SetParam("customerType", "0")
	XMLCustomerSearch, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager, "shrGetCustomerDetails")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Customer ["+custID+"]: "+fmt.Sprintf("%v", xmlmcErr), true)
	}

	var xmlRespon xmlmcCustomerListResponse
	err := xml.Unmarshal([]byte(XMLCustomerSearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Customer ["+custID+"]: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			//Customer most likely does not exist
			logger(4, "Unable to Search for Customer ["+custID+"]: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.CustomerFirstName != "" {
				boolReturn = true
				//-- Add Customer to Cache
				var newCustomerForCache customerListStruct
				newCustomerForCache.CustomerID = custID
				newCustomerForCache.CustomerName = xmlRespon.CustomerFirstName + " " + xmlRespon.CustomerLastName
				strReturn = newCustomerForCache.CustomerName
				customerNamedMap := []customerListStruct{newCustomerForCache}
				mutexCustomers.Lock()
				Customers = append(Customers, customerNamedMap...)
				mutexCustomers.Unlock()
			}
		}
	}
	return boolReturn, strReturn
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
	var errorLogPrefix string
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 2:
		errorLogPrefix = "[MESSAGE] "
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 3:
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 4:
		errorLogPrefix = "[ERROR] "
		if outputtoCLI {
			color.Set(color.FgRed)
			defer color.Unset()
		}
	}
	if outputtoCLI {
		fmt.Printf("%v \n", errorLogPrefix+s)
	}
	mutex.Lock()
	// assign it to the standard logger
	log.SetOutput(f)
	log.Println(errorLogPrefix + s)
	mutex.Unlock()
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
	espXmlmc := apiLib.NewXmlmcInstance(SQLImportConf.URL)
	espXmlmc.SetAPIKey(SQLImportConf.APIKey)
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
