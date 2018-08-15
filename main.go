package main

//----- Packages -----
import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/hornbill/color"
	"github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"

	"time"
	//SQL Drivers
	_ "github.com/hornbill/go-mssqldb"
	_ "github.com/hornbill/mysql"
	_ "github.com/hornbill/mysql320" //MySQL v3.2.0 to v5 driver - Provides SWSQL (MySQL 4.0.16) support
)

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
	flag.StringVar(&configMaxRoutines, "concurrent", "1", "Maximum number of Assets to import concurrently.")
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
		color.Red("The maximum concurrent assets allowed is between 1 and 10 (inclusive).\n\n")
		color.Red("You have selected " + configMaxRoutines + ". Please try again, with a valid value against ")
		color.Red("the -concurrent switch.")
		return
	}

	//--
	//-- Load Configuration File Into Struct
	SQLImportConf = loadConfig()
	if SQLImportConf.LogSizeBytes > 0 {
		maxLogFileSize = SQLImportConf.LogSizeBytes
	}

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
	for _, v := range SQLImportConf.AssetTypes {
		StrAssetType = fmt.Sprintf("%v", v.AssetType)
		StrSQLAppend = fmt.Sprintf("%v", v.Query)

		//Set Asset Class & Type vars from instance
		AssetClass, AssetTypeID = getAssetClass(StrAssetType)
		//-- Query Database
		var boolSQLAssets, arrAssets = queryDatabase(StrSQLAppend, StrAssetType)
		if boolSQLAssets {
			//Process records returned by query
			processAssets(arrAssets, v.AssetIdentifier)
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
	espXmlmc.SetParam("column", "h_name")
	espXmlmc.SetParam("value", confAssetType)
	espXmlmc.SetParam("matchType", "exact")
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")
	var XMLSTRING = espXmlmc.GetParam()
	XMLGetMeta, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")
	if xmlmcErr != nil {
		logger(4, "API Call failed when retrieving Asset Class:"+fmt.Sprintf("%v", xmlmcErr), false)
		logger(1, "API XML: "+fmt.Sprintf("%s", XMLSTRING), false)
	}

	var xmlRespon xmlmcTypeListResponse
	err := xml.Unmarshal([]byte(XMLGetMeta), &xmlRespon)
	if err != nil {
		logger(4, "Could not get Asset Class and Type. Please check AssetType within your configuration file:"+fmt.Sprintf("%v", err), true)
		logger(1, "API XML: "+fmt.Sprintf("%s", XMLSTRING), false)
	} else {
		assetClass = xmlRespon.Params.RowData.Row.TypeClass
		assetType = xmlRespon.Params.RowData.Row.TypeID
	}
	return
}

//processAssets -- Processes Assets from Asset Map
//--If asset already exists on the instance, update
//--If asset doesn't exist, create
func processAssets(arrAssets []map[string]interface{}, assetIdentifier assetIdentifierStruct) {
	bar := pb.StartNew(len(arrAssets))
	logger(1, "Processing Assets", false)

	//Get the identity of the AssetID field from the config
	assetIDIdent := fmt.Sprintf("%v", assetIdentifier.DBColumn)

	//-- Loop each asset
	maxGoroutinesGuard := make(chan struct{}, maxGoroutines)

	for _, assetRecord := range arrAssets {
		maxGoroutinesGuard <- struct{}{}
		worker.Add(1)
		assetMap := assetRecord
		//Get the asset ID for the current record
		assetID := fmt.Sprintf("%s", assetMap[assetIDIdent])
		espXmlmc := apiLib.NewXmlmcInstance(SQLImportConf.URL)
		espXmlmc.SetAPIKey(SQLImportConf.APIKey)
		go func() {
			defer worker.Done()
			time.Sleep(1 * time.Millisecond)
			mutexBar.Lock()
			bar.Increment()
			mutexBar.Unlock()

			var boolUpdate = false
			boolUpdate, assetIDInstance := getAssetID(assetID, assetIdentifier, espXmlmc)
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
func getAssetID(assetID string, assetIdentifier assetIdentifierStruct, espXmlmc *apiLib.XmlmcInstStruct) (bool, string) {
	boolReturn := false
	returnAssetID := ""
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", fmt.Sprintf("%v", assetIdentifier.Entity))
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("column", fmt.Sprintf("%v", assetIdentifier.EntityColumn))
	espXmlmc.SetParam("value", assetID)
	espXmlmc.SetParam("matchType", "exact")
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")
	var XMLSTRING = espXmlmc.GetParam()
	XMLAssetSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")
	if xmlmcErr != nil {
		logger(4, "API Call failed when searching instance for existing Asset:"+fmt.Sprintf("%v", xmlmcErr), false)
		logger(1, "API Call XML: "+fmt.Sprintf("%s", XMLSTRING), false)
	} else {
		var xmlRespon xmlmcAssetResponse
		err := xml.Unmarshal([]byte(XMLAssetSearch), &xmlRespon)
		if err != nil {
			logger(3, "Unable to Search for Asset: "+fmt.Sprintf("%v", err), true)
			logger(1, "API Call XML: "+fmt.Sprintf("%s", XMLSTRING), false)
		} else {
			if xmlRespon.MethodResult != "ok" {
				logger(3, "Unable to Search for Asset: "+xmlRespon.State.ErrorRet, true)
				logger(1, "API Call XML: "+fmt.Sprintf("%s", XMLSTRING), false)
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
func createAsset(u map[string]interface{}, espXmlmc *apiLib.XmlmcInstStruct) {
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
	usedByID := getFieldValue("h_used_by", usedByMapping, u)
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

	//Last Logged On By
	lastLoggedOnByURN := ""
	lastLoggedOnUserMapping := fmt.Sprintf("%v", SQLImportConf.AssetTypeFieldMapping["h_last_logged_on_user"])
	if lastLoggedOnUserMapping != "" {
		lastLoggedOnByID := getFieldValue("h_last_logged_on_user", lastLoggedOnUserMapping, u)
		if lastLoggedOnByID != "" {
			lastLoggedOnByIsInCache, lastLoggedOnByNameCache := customerInCache(lastLoggedOnByID)
			//-- Check if we have cached the customer already
			if lastLoggedOnByIsInCache {
				lastLoggedOnByURN = "urn:sys:0:" + lastLoggedOnByNameCache + ":" + lastLoggedOnByID
			} else {
				lastLoggedOnByIsOnInstance, lastLoggedOnByNameInstance := searchCustomer(lastLoggedOnByID, espXmlmc)
				//-- If Returned set output
				if lastLoggedOnByIsOnInstance {
					lastLoggedOnByURN = "urn:sys:0:" + lastLoggedOnByNameInstance + ":" + lastLoggedOnByID
				}
			}
		}
	}

	//Get/Set params from map stored against FieldMapping
	strAttribute := ""
	strMapping := ""
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Asset")
	espXmlmc.SetParam("returnModifiedData", "true")
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
		if strAttribute != "h_site" &&
			strAttribute != "h_used_by" &&
			strAttribute != "h_owned_by" &&
			strMapping != "" && getFieldValue(strAttribute, strMapping, u) != "" {
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

		if strAttribute == "h_last_logged_on_user" && lastLoggedOnByURN != "" {
			espXmlmc.SetParam("h_last_logged_on_user", lastLoggedOnByURN)
		}
		if strAttribute != "h_last_logged_on_user" &&
			strMapping != "" &&
			getFieldValue(strAttribute, strMapping, u) != "" {
			espXmlmc.SetParam(strAttribute, getFieldValue(strAttribute, strMapping, u))
		}
	}
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("relatedEntityData")

	//-- Check for Dry Run
	if configDryRun != true {
		var XMLSTRING = espXmlmc.GetParam()
		XMLCreate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
		if xmlmcErr != nil {
			logger(4, "Error running entityAddRecord API for createAsset:"+fmt.Sprintf("%v", xmlmcErr), false)
			logger(1, "API Call XML: "+fmt.Sprintf("%s", XMLSTRING), false)
			return
		}
		var xmlRespon xmlmcUpdateResponse

		err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
		if err != nil {
			mutexCounters.Lock()
			counters.createskipped++
			mutexCounters.Unlock()
			logger(4, "Unable to read response from Hornbill instance from entityAddRecord API for createAsset:"+fmt.Sprintf("%v", err), false)
			logger(1, "API Call XML: "+fmt.Sprintf("%s", XMLSTRING), false)
			return
		}
		if xmlRespon.MethodResult != "ok" {
			logger(3, "Unable to add asset: "+xmlRespon.State.ErrorRet, false)
			logger(1, "API Call XML: "+fmt.Sprintf("%s", XMLSTRING), false)
			mutexCounters.Lock()
			counters.createskipped++
			mutexCounters.Unlock()
		} else {
			mutexCounters.Lock()
			counters.created++
			mutexCounters.Unlock()
			assetID := xmlRespon.UpdatedCols.AssetPK
			//Now add asset URN
			espXmlmc.SetParam("application", "com.hornbill.servicemanager")
			espXmlmc.SetParam("entity", "Asset")
			espXmlmc.OpenElement("primaryEntityData")
			espXmlmc.OpenElement("record")
			espXmlmc.SetParam("h_pk_asset_id", assetID)
			espXmlmc.SetParam("h_asset_urn", "urn:sys:entity:com.hornbill.servicemanager:Asset:"+assetID)
			espXmlmc.CloseElement("record")
			espXmlmc.CloseElement("primaryEntityData")
			XMLSTRING = espXmlmc.GetParam()
			XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
			if xmlmcErr != nil {
				logger(4, "API Call failed when Updating Asset URN:"+fmt.Sprintf("%v", xmlmcErr), false)
				return
			}
			var xmlRespon xmlmcResponse

			err := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
			if err != nil {
				logger(4, "Unable to read response from Hornbill instance when Updating Asset URN:"+fmt.Sprintf("%v", err), false)
				return
			}
			if xmlRespon.MethodResult != "ok" {
				logger(3, "Unable to update Asset URN: "+xmlRespon.State.ErrorRet, false)
				logger(1, "API Call XML: "+fmt.Sprintf("%s", XMLSTRING), false)
				return
			}
			return
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
	return
}

// updateAsset -- Updates Asset record from the passed through map data and asset ID
func updateAsset(u map[string]interface{}, strAssetID string, espXmlmc *apiLib.XmlmcInstStruct) bool {

	boolRecordUpdated := false
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
	espXmlmc.SetParam("returnModifiedData", "true")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_pk_asset_id", strAssetID)
	espXmlmc.SetParam("h_asset_urn", "urn:sys:entity:com.hornbill.servicemanager:Asset:"+strAssetID)
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
		if strAttribute != "h_site" &&
			strAttribute != "h_used_by" &&
			strAttribute != "h_owned_by" &&
			strMapping != "" && getFieldValue(strAttribute, strMapping, u) != "" {
			espXmlmc.SetParam(strAttribute, getFieldValue(strAttribute, strMapping, u))
		}
	}

	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")

	var XMLSTRING = espXmlmc.GetParam()
	//-- Check for Dry Run
	if configDryRun != true {
		XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
		if xmlmcErr != nil {
			logger(4, "API Call failed when Updating Asset:"+fmt.Sprintf("%v", xmlmcErr), false)
			logger(1, "API Call XML: "+fmt.Sprintf("%s", XMLSTRING), false)
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
			return false
		}

		var xmlRespon xmlmcUpdateResponse

		err := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
		if err != nil {
			logger(4, "Unable to read response from Hornbill instance when Updating Asset:"+fmt.Sprintf("%v", err), false)
			logger(1, "API Call XML: "+fmt.Sprintf("%s", XMLSTRING), false)
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
			return false
		}
		if xmlRespon.MethodResult != "ok" && xmlRespon.State.ErrorRet != "There are no values to update" {
			logger(3, "Unable to Update Asset: "+xmlRespon.State.ErrorRet, false)
			logger(1, "API Call XML: "+fmt.Sprintf("%s", XMLSTRING), false)
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
			return false
		}

		if len(xmlRespon.UpdatedCols.ColList) > 0 {
			boolRecordUpdated = true
		}

		//-- now process extended record data
		espXmlmc.SetParam("application", appServiceManager)
		espXmlmc.SetParam("entity", "Asset")
		espXmlmc.SetParam("returnModifiedData", "true")
		espXmlmc.OpenElement("primaryEntityData")
		espXmlmc.OpenElement("record")
		espXmlmc.SetParam("h_pk_asset_id", strAssetID)
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

		XMLUpdateExt, xmlmcErrExt := espXmlmc.Invoke("data", "entityUpdateRecord")
		if xmlmcErrExt != nil {
			logger(4, "API Call failed when Updating Asset Extended Details:"+fmt.Sprintf("%v", xmlmcErrExt), false)
			logger(1, "API Call XML: "+fmt.Sprintf("%s", XMLSTRING), false)
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
			return false
		}
		var xmlResponExt xmlmcUpdateResponse

		err = xml.Unmarshal([]byte(XMLUpdateExt), &xmlResponExt)
		if err != nil {
			logger(4, "Unable to read response from Hornbill instance when Updating Asset Extended Details:"+fmt.Sprintf("%v", err), false)
			logger(1, "API Call XML: "+fmt.Sprintf("%s", XMLSTRING), false)
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
			return false
		}
		if xmlResponExt.MethodResult != "ok" && xmlResponExt.State.ErrorRet != "There are no values to update" {
			logger(3, "Unable to Update Asset Extended Details: "+xmlResponExt.State.ErrorRet, false)
			logger(1, "API Call XML: "+fmt.Sprintf("%s", XMLSTRING), false)
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
			return false
		}

		if len(xmlResponExt.UpdatedCols.ColList) > 0 {
			boolRecordUpdated = true
		}

		if boolRecordUpdated == false {
			mutexCounters.Lock()
			counters.updatedSkipped++
			mutexCounters.Unlock()
		} else {
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
			var XMLSTRING = espXmlmc.GetParam()
			XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
			if xmlmcErr != nil {
				logger(4, "API Call failed when setting Last Updated values:"+fmt.Sprintf("%v", xmlmcErr), false)
				logger(1, "Asset Last Updated XML: "+fmt.Sprintf("%s", XMLSTRING), false)
			}
			var xmlRespon xmlmcResponse
			err := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
			if err != nil {
				logger(4, "Unable to read response from Hornbill instance when setting Last Updated values:"+fmt.Sprintf("%v", err), false)
				logger(1, "Asset Last Updated XML: "+fmt.Sprintf("%s", XMLSTRING), false)
			}
			if xmlRespon.MethodResult != "ok" && xmlRespon.State.ErrorRet != "There are no values to update" {
				logger(3, "Unable to set Last Updated details for asset: "+xmlRespon.State.ErrorRet, false)
				logger(1, "Asset Last Updated XML: "+fmt.Sprintf("%s", XMLSTRING), false)
			}
			mutexCounters.Lock()
			counters.updated++
			mutexCounters.Unlock()
		}

	} else {
		//-- Inc Counter
		mutexCounters.Lock()
		counters.updatedSkipped++
		mutexCounters.Unlock()
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

// logger -- function to append to the current log file
func logger(t int, s string, outputtoCLI bool) {
	//-- Current working dir
	cwd, _ := os.Getwd()

	//-- Log Folder
	logPath := cwd + "/log"

	//-- If Folder Does Not Exist then create it
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		err := os.Mkdir(logPath, 0777)
		if err != nil {
			fmt.Printf("Error Creating Log Folder %q: %s \r", logPath, err)
			os.Exit(101)
		}
	}

	//-- Log File
	logFileName := logPath + "/Asset_Import_" + TimeNow + "_" + strconv.Itoa(logFilePart) + ".log"
	if maxLogFileSize > 0 {
		//Check log file size
		fileLoad, e := os.Stat(logFileName)
		if e != nil {
			//File does not exist - do nothing!
		} else {
			fileSize := fileLoad.Size()
			if fileSize > maxLogFileSize {
				logFilePart++
				logFileName = logPath + "/Asset_Import_" + TimeNow + "_" + strconv.Itoa(logFilePart) + ".log"
			}
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

// checkDateString - returns date from supplied string
func checkDateString(strDate string) string {
	re, _ := regexp.Compile("\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}")
	strNewDate := re.FindString(strDate)
	return strNewDate
}
