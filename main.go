package main

//----- Packages -----
import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hornbill/color"

	"time"
	//SQL Drivers
	_ "github.com/alexbrainman/odbc" //ODBC Driver
	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
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
	flag.BoolVar(&configDebug, "debug", false, "Output additional debug information to the log")
	flag.BoolVar(&configDryRun, "dryrun", false, "Allow the Import to run without Creating or Updating Assets")
	flag.StringVar(&configMaxRoutines, "concurrent", "1", "Maximum number of Assets to import concurrently.")
	flag.BoolVar(&configVersion, "version", false, "Return version and end")
	flag.Parse()

	//-- If configVersion just output version number and die
	if configVersion {
		fmt.Printf("%v \n", version)
		return
	}

	//-- Output
	logger(1, "---- XMLMC Database Asset Import Utility V"+version+" ----", true)
	logger(1, "Flag - Config File "+configFileName, true)
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
		debugLog("Asset Type and Class:", StrAssetType, strconv.Itoa(AssetTypeID), AssetClass)
		//-- Query Database
		var boolSQLAssets, arrAssets = queryDatabase(StrSQLAppend, StrAssetType)
		if boolSQLAssets {
			//Process records returned by query
			processAssets(arrAssets, v)
		}
	}

	//-- End output
	logger(1, "Created: "+fmt.Sprintf("%d", counters.created), true)
	logger(1, "Create Skipped: "+fmt.Sprintf("%d", counters.createSkipped), true)
	logger(1, "Create Failed: "+fmt.Sprintf("%d", counters.createFailed), true)
	logger(1, "Updated: "+fmt.Sprintf("%d", counters.updated), true)
	logger(1, "Update Skipped: "+fmt.Sprintf("%d", counters.updateSkipped), true)
	logger(1, "Update Failed: "+fmt.Sprintf("%d", counters.updateFailed), true)
	logger(1, "Update Extended Record Skipped: "+fmt.Sprintf("%d", counters.updateRelatedSkipped), true)
	logger(1, "Update Extended Record Failed: "+fmt.Sprintf("%d", counters.updateRelatedFailed), true)
	//-- Show Time Takens
	endTime = time.Since(startTime)
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
