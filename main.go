package main

//----- Packages -----
import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/fatih/color"
	"github.com/rhysd/go-github-selfupdate/selfupdate"

	//SQL Drivers
	_ "github.com/alexbrainman/odbc" //ODBC Driver
	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/hornbill/mysql320"
)

//----- Main Function -----
func main() {
	//-- Start Time for Duration
	startTime = time.Now()

	//-- Grab Flags
	flag.StringVar(&configFileName, "file", "conf.json", "Name of Configuration File To Load")
	flag.BoolVar(&configDebug, "debug", false, "Output additional debug information to the log")
	flag.BoolVar(&configDryRun, "dryrun", false, "Allow the Import to run without Creating or Updating Assets")
	flag.IntVar(&configMaxRoutines, "concurrent", 1, "Maximum number of Assets to import concurrently.")
	flag.BoolVar(&configVersion, "version", false, "Return version and end")
	flag.BoolVar(&configForceUpdates, "forceupdates", false, "Force updates (ignoring hash calculation; CI only - NOT software (type needs to be set to Update or Both))")
	flag.Parse()

	//-- If configVersion just output version number and die
	if configVersion {
		fmt.Printf("%v \n", version)
		return
	}

	//--
	//-- Load Configuration File Into Struct
	importConf = loadConfig()
	if importConf.LogSizeBytes > 0 {
		maxLogFileSize = importConf.LogSizeBytes
	}

	err := checkConfig()
	if err != nil {
		color.Red("Your configuration file is invalid. Since v2.0.0 of this tool, square-bracket notation has been replaced by Golang templates.")
		color.Red("See the Hornbill Wiki for more information: https://wiki.hornbill.com/index.php?title=Database_Asset_Import")
		fmt.Println()
		fmt.Println("Unsupported mappings:")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	//XMLMC session to perform local caching of instance records with
	initXMLMC()

	//Check version of utility, self-update if appropriate
	doSelfUpdate()

	//-- Output
	logger(3, "---- XMLMC Database Asset Import Utility v"+version+" ----", true, true)
	logger(1, "Flag - Config File "+configFileName, true, true)
	logger(1, "Flag - Dry Run "+fmt.Sprintf("%v", configDryRun), true, true)
	logger(1, "Flag - Concurrent "+strconv.Itoa(configMaxRoutines), true, true)

	if configMaxRoutines < 1 || configMaxRoutines > maxGoRoutines {
		color.Red("The maximum concurrent value allowed is between 1 and 10 (inclusive).\n\n")
		color.Red("You have selected " + strconv.Itoa(configMaxRoutines) + ". Please try again, with a valid value against ")
		color.Red("the -concurrent argument.")
		return
	}

	if importConf.KeysafeKeyID != 0 {
		getKeysafeKey(importConf.KeysafeKeyID)
	}
	//Set SWSQLDriver to mysql320
	if importConf.SourceConfig.Source == "swsql" {
		importConf.SourceConfig.Source = "mysql320"
	}

	processCaching()

	//Get asset types, process accordingly
	configCSV = (strings.ToLower(importConf.SourceConfig.Source) == "csv")
	configNexthink = (strings.ToLower(importConf.SourceConfig.Source) == "nexthink")
	configLDAP = (strings.EqualFold(importConf.SourceConfig.Source, "ldap"))
	configGoogle = (strings.EqualFold(importConf.SourceConfig.Source, "google"))

	if !configCSV && !configNexthink && !configLDAP && !configGoogle {
		//Build DB connection string
		connString = buildConnectionString()
		if connString == "" {
			logger(4, " [DATABASE] Database Connection String Empty. Check the SQLConf section of your configuration.", true, true)
			return
		}
	}
	setTemplateFilters()

	templateFault := checkTemplate()
	if templateFault {
		logger(4, " [Template] Issues were found with the template.", true, true)
		return
	}

	for _, v := range importConf.AssetTypes {
		StrAssetType = v.AssetType
		StrSQLAppend = v.Query
		//Set Asset Class & Type vars from instance
		if !strings.HasPrefix(v.AssetType, "__all__:") {
			AssetClass, AssetTypeID = getAssetClass(StrAssetType)
			v.TypeID = AssetTypeID
			v.Class = AssetClass
			debugLog(nil, "Asset Type and Class:", StrAssetType, strconv.Itoa(AssetTypeID), AssetClass)
		} else {
			if !strings.EqualFold(v.OperationType, "Update") {
				logger(4, "AssetType: "+v.AssetType+" has an unsupported OperationType defined: "+v.OperationType, true, true)
				continue
			}
			v.TypeID = 0
			v.Class = strings.Split(v.AssetType, ":")[1]
		}

		//-- Query Data Source
		boolSQLAssets := false
		var arrAssets map[string]map[string]interface{}
		if configCSV {
			//-- Read CSV
			boolSQLAssets, arrAssets = getAssetsFromCSV(v)
		} else if configNexthink {
			//-- Query Nexthink
			arrAssets, err = getAssetsFromNexthink(v)
			if err != nil {
				logger(4, err.Error(), true, true)
			} else {
				boolSQLAssets = true
			}
		} else if configLDAP {
			//-- Query LDAP
			arrAssets, boolSQLAssets = queryLDAP(v)
		} else if configGoogle {
			//-- Query Nexthink
			arrAssets, err = getAssetsFromGoogle(v)
			if err != nil {
				logger(4, err.Error(), true, true)
			} else {
				boolSQLAssets = true
			}
		} else {
			//-- Query database
			boolSQLAssets, arrAssets = queryAssets(StrSQLAppend, v)
		}
		if boolSQLAssets && len(arrAssets) > 0 {
			//Cache instance asset records of class & optional type
			logger(3, "Caching "+v.AssetType+" Asset Records from Hornbill...", true, true)
			assetCount, err := getAssetCount(v, hornbillImport)
			if err != nil {
				logger(4, "Unable to count asset records: "+err.Error(), true, true)
				continue
			}
			var assetCache map[string]map[string]interface{}
			if assetCount > 0 {
				assetCache, err = getAssetRecords(assetCount, v, hornbillImport)
				if err != nil {
					logger(4, "Unable to cache asset records: "+err.Error(), true, true)
					continue
				}
			}
			//Process records returned by query & cache
			processAssets(arrAssets, assetCache, v)
		}
	}

	//-- End output
	fmt.Println()
	logger(3, "-=-=-= Summary =-=-=-", true, true)
	logger(3, "Created: "+fmt.Sprintf("%d", counters.created), true, true)
	logger(3, "Create Skipped: "+fmt.Sprintf("%d", counters.createSkipped), true, true)
	logger(3, "Create Failed: "+fmt.Sprintf("%d", counters.createFailed), true, true)
	logger(3, "Updated: "+fmt.Sprintf("%d", counters.updated), true, true)
	logger(3, "Update Skipped: "+fmt.Sprintf("%d", counters.updateSkipped), true, true)
	logger(3, "Update Failed: "+fmt.Sprintf("%d", counters.updateFailed), true, true)
	logger(3, "Update Extended Record Skipped: "+fmt.Sprintf("%d", counters.updateRelatedSkipped), true, true)
	logger(3, "Update Extended Record Failed: "+fmt.Sprintf("%d", counters.updateRelatedFailed), true, true)
	logger(3, "Assets Software Inventory Skipped: "+fmt.Sprintf("%d", counters.softwareSkipped), true, true)
	logger(3, "Software Records Created: "+fmt.Sprintf("%d", counters.softwareCreated), true, true)
	logger(3, "Software Records Create Failed: "+fmt.Sprintf("%d", counters.softwareCreateFailed), true, true)
	logger(3, "Software Records Removed: "+fmt.Sprintf("%d", counters.softwareRemoved), true, true)
	logger(3, "Software Records Removal Failed: "+fmt.Sprintf("%d", counters.softwareRemoveFailed), true, true)
	logger(3, "Asset Supplier Associations Success: "+fmt.Sprintf("%d", counters.suppliersAssociatedSuccess), true, true)
	logger(3, "Asset Supplier Associations Failed: "+fmt.Sprintf("%d", counters.suppliersAssociatedFailed), true, true)
	logger(3, "Asset Supplier Associations Skipped: "+fmt.Sprintf("%d", counters.suppliersAssociatedSkipped), true, true)
	logger(3, "Asset Supplier Contract Associations Success: "+fmt.Sprintf("%d", counters.supplierContractsAssociatedSuccess), true, true)
	logger(3, "Asset Supplier Contract Associations Failed: "+fmt.Sprintf("%d", counters.supplierContractsAssociatedFailed), true, true)
	logger(3, "Asset Supplier Contract Associations Skipped: "+fmt.Sprintf("%d", counters.supplierContractsAssociatedSkipped), true, true)

	//-- Show Time Takens
	logger(3, "Time Taken: "+fmt.Sprintf("%v", time.Since(startTime).Round(time.Second)), true, true)
	logger(3, "---- XMLMC Database Asset Import Complete ---- ", true, true)
}

//loadConfig -- Function to Load Configruation File
func loadConfig() importConfStruct {
	//-- Check Config File File Exists
	cwd, _ := os.Getwd()
	configurationFilePath := cwd + "/" + configFileName
	logger(3, "Loading Config File: "+configurationFilePath, false, false)
	if _, fileCheckErr := os.Stat(configurationFilePath); os.IsNotExist(fileCheckErr) {
		logger(4, "No Configuration File", true, false)
		os.Exit(102)
	}
	//-- Load Config File
	file, fileError := os.Open(configurationFilePath)
	//-- Check For Error Reading File
	if fileError != nil {
		logger(4, "Error Opening Configuration File: "+fileError.Error(), true, false)
	}

	//-- New Decoder
	decoder := json.NewDecoder(file)
	//-- New Var based on importConf
	esqlConf := importConfStruct{}
	//-- Decode JSON
	err := decoder.Decode(&esqlConf)
	//-- Error Checking
	if err != nil {
		logger(4, "Error Decoding Configuration File: "+err.Error(), true, false)
	}

	//-- Return New Config
	return esqlConf
}

func processCaching() {

	//only load if any of the user colums are set
	importConf.HornbillUserIDColumn = strings.ToLower(importConf.HornbillUserIDColumn)
	blnHasUserConfigured := false
	if val, ok := importConf.AssetGenericFieldMapping["h_owned_by"]; ok {
		if val != "" {
			blnHasUserConfigured = true
		}
	}
	if val, ok := importConf.AssetGenericFieldMapping["h_used_by"]; ok {
		if val != "" {
			blnHasUserConfigured = true
		}
	}
	if val, ok := importConf.AssetTypeFieldMapping["h_last_logged_on_user"]; ok {
		if val != "" {
			blnHasUserConfigured = true
		}
	}

	if blnHasUserConfigured {
		logger(3, "Caching User Records from Hornbill...", true, true)
		loadUsers()
	}

	//only load if site colum is configured
	if val, ok := importConf.AssetGenericFieldMapping["h_site"]; ok {
		if val != "" {
			logger(3, "Caching Site Records from Hornbill...", true, true)
			loadSites()
		}
	}

	var queryGroups []string
	if val, ok := importConf.AssetGenericFieldMapping["h_company_name"]; ok {
		if val != "" {
			queryGroups = append(queryGroups, "company")
		}
	}
	if val, ok := importConf.AssetGenericFieldMapping["h_department_name"]; ok {
		if val != "" {
			queryGroups = append(queryGroups, "department")
		}
	}

	if len(queryGroups) > 0 {
		logger(3, "Caching Group Records from Hornbill...", true, true)
		loadGroups(queryGroups)
	}
	logger(3, "Caching Application Records from Hornbill...", true, true)
	getApplications()
}

func doSelfUpdate() {
	v := semver.MustParse(version)
	latest, found, err := selfupdate.DetectLatest(repo)
	if err != nil {
		logger(5, "Error occurred while detecting version: "+err.Error(), true, true)
		return
	}
	if !found {
		logger(5, "Could not find Github repo: "+repo, true, true)
		return
	}

	latestMajorVersion := strings.Split(fmt.Sprintf("%v", latest.Version), ".")[0]
	latestMinorVersion := strings.Split(fmt.Sprintf("%v", latest.Version), ".")[1]
	latestPatchVersion := strings.Split(fmt.Sprintf("%v", latest.Version), ".")[2]

	currentMajorVersion := strings.Split(version, ".")[0]
	currentMinorVersion := strings.Split(version, ".")[1]
	currentPatchVersion := strings.Split(version, ".")[2]

	//Useful in dev, customers should never see current version > latest release version
	if currentMajorVersion > latestMajorVersion {
		logger(3, "Current version "+version+" (major) is greater than the latest release version on Github "+fmt.Sprintf("%v", latest.Version), true, true)
		return
	} else {
		if currentMinorVersion > latestMinorVersion {
			logger(3, "Current version "+version+" (minor) is greater than the latest release version on Github "+fmt.Sprintf("%v", latest.Version), true, true)
			return
		} else if currentPatchVersion > latestPatchVersion {
			logger(3, "Current version "+version+" (patch) is greater than the latest release version on Github "+fmt.Sprintf("%v", latest.Version), true, true)
			return
		}
	}
	if latestMajorVersion > currentMajorVersion {
		msg := "v" + version + " is not latest, you should upgrade to " + fmt.Sprintf("%v", latest.Version) + " by downloading the latest package from: https://github.com/" + repo + "/releases/latest"
		logger(5, msg, true, true)
		return
	}

	_, err = selfupdate.UpdateSelf(v, repo)
	if err != nil {
		logger(5, "Binary update failed: "+err.Error(), true, true)
		return
	}
	if latest.Version.Equals(v) {
		// latest version is the same as current version. It means current binary is up to date.
		logger(3, "Current binary is the latest version: "+version, true, true)
	} else {
		logger(3, "Successfully updated to version: "+fmt.Sprintf("%v", latest.Version), true, true)
		logger(3, "Release notes:\n"+latest.ReleaseNotes, true, true)
	}
}
