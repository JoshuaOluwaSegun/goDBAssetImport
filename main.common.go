package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/fatih/color"
	apiLib "github.com/hornbill/goApiLib"
)

func initXMLMC() {
	endpoint := apiLib.GetEndPointFromName(importConf.InstanceID)
	if endpoint == "" {
		logger(4, "Unable to retrieve endpoint information for the supplied InstanceID: "+importConf.InstanceID, true, false)
		os.Exit(1)
	}

	hornbillImport = apiLib.NewXmlmcInstance(importConf.InstanceID)
	hornbillImport.SetAPIKey(importConf.APIKey)
	hornbillImport.SetTimeout(60)
	hornbillImport.SetJSONResponse(true)

	if pageSize == 0 {
		pageSize = 100
	}
}

// getFieldValue --Retrieve field value from mapping via SQL record map
func getFieldValue(k string, v string, u map[string]interface{}, buffer *bytes.Buffer) string {
	debugLog(buffer, "getFieldValue:", k, ":", v)
	fieldMap := v
	if fieldMap == "__hbassettype__" {
		debugLog(buffer, "Returning Asset Type:", StrAssetType)
		return StrAssetType
	}

	t := template.New(fieldMap).Funcs(TemplateFilters)
	tmpl, _ := t.Parse(fieldMap)
	buf := bytes.NewBufferString("")
	tmpl.Execute(buf, u)

	value := ""
	if buf != nil {
		value = buf.String()
	}
	debugLog(buffer, "value:", value)

	if value == "<no value>" {
		value = ""
	}
	fieldMap = value
	if fieldMap != "" {
		if strings.Contains(strings.ToLower(k), "date") || strings.ToLower(k) == "h_last_logged_on" {
			fieldMap = checkDateString(fieldMap)
		}
	}
	debugLog(buffer, "returning:", fieldMap)
	return fieldMap
}

// logger -- function to append to the current log file
func logger(t int, s string, outputtoCLI bool, outputToEsp bool) {
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
	logFileName := logPath + "/Asset_Import_" + startTime.Format("20060102150405") + "_" + strconv.Itoa(logFilePart) + ".log"
	if maxLogFileSize > 0 {
		//Check log file size
		fileLoad, e := os.Stat(logFileName)
		if e != nil {
			//File does not exist - do nothing!
		} else {
			fileSize := fileLoad.Size()
			if fileSize > maxLogFileSize {
				logFilePart++
				logFileName = logPath + "/Asset_Import_" + startTime.Format("20060102150405") + "_" + strconv.Itoa(logFilePart) + ".log"
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
	var (
		errorLogPrefix string
		espLogType     string
	)
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
		espLogType = "debug"
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 2:
		errorLogPrefix = "[MESSAGE] "
		espLogType = "notice"
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 3:
		espLogType = "notice"
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 4:
		errorLogPrefix = "[ERROR] "
		espLogType = "error"
		if outputtoCLI {
			color.Set(color.FgRed)
			defer color.Unset()
		}
	case 5:
		errorLogPrefix = "[WARNING] "
		espLogType = "warn"
		if outputtoCLI {
			color.Set(color.FgYellow)
			defer color.Unset()
		}
	}
	if outputtoCLI {
		fmt.Printf("%v \n", errorLogPrefix+s)
	}
	if outputToEsp {
		espLogger(s, espLogType)
	}
	// assign it to the standard logger
	log.SetOutput(f)
	log.Println(errorLogPrefix + s)
}

func loggerGen(t int, s string) string {
	//-- Create Log Entry
	var errorLogPrefix = ""
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
	case 2:
		errorLogPrefix = "[MESSAGE] "
	case 3:
		errorLogPrefix = ""
	case 4:
		errorLogPrefix = "[ERROR] "
	case 5:
		errorLogPrefix = "[WARNING] "
	}
	return errorLogPrefix + s + "\n\r"
}
func loggerWriteBuffer(s string) {
	if s != "" {
		logLines := strings.Split(s, "\n\r")
		for _, line := range logLines {
			if line != "" {
				logger(0, line, false, false)
			}
		}
	}
}

// checkDateString - returns date from supplied string
func checkDateString(strDate string) string {
	re, _ := regexp.Compile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)
	strNewDate := re.FindString(strDate)
	return strNewDate
}

func debugLog(buffer *bytes.Buffer, debugStrings ...string) {
	if configDebug {
		if buffer == nil {
			logger(1, strings.Join(debugStrings, " "), false, false)
		} else {
			buffer.WriteString(loggerGen(1, strings.Join(debugStrings, " ")))
		}
	}
}

func iToS(interfaceVal interface{}) (strVal string) {
	if interfaceVal == nil {
		return
	}

	switch v := interfaceVal.(type) {
	case []uint8:
		strVal = string(v)
	default:
		strVal = fmt.Sprintf("%v", interfaceVal)
	}
	return
}

func Hash(arr []map[string]interface{}) string {
	arrBytes := []byte{}
	for _, item := range arr {
		jsonBytes, _ := json.Marshal(item)
		arrBytes = append(arrBytes, jsonBytes...)
	}
	has := md5.Sum(arrBytes)

	md5str := fmt.Sprintf("%x", has)
	return md5str
}

// getKeysafeKey - returns key details
func getKeysafeKey(keyId int) {
	//API Call to get the key data
	hornbillImport.SetParam("keyId", strconv.Itoa(keyId))
	hornbillImport.SetParam("wantKeyData", "true")
	RespBody, xmlmcErr := hornbillImport.Invoke("admin", "keysafeGetKey")
	var JSONResp xmlmcKeyResponse
	if xmlmcErr != nil {
		logger(4, "Unable to retrieve key information from Keysafe: "+xmlmcErr.Error(), true, true)
		os.Exit(1)
	}
	//Unmarashal the API response
	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		logger(4, "Unable to unmarshal key information from Keysafe: "+err.Error(), true, true)
		os.Exit(1)
	}
	if JSONResp.State.Error != "" {
		logger(4, "API call to retrieve key information from Keysafe failed: "+JSONResp.State.Error, true, true)
		os.Exit(1)
	}

	// Now we need to unmarshal the key data itself
	err = json.Unmarshal([]byte(JSONResp.Params.Data), &key)
	if err != nil {
		logger(4, "Unable to unmarshal Keysafe key data JSON: "+err.Error(), true, true)
		os.Exit(1)
	}
}

// espLogger -- Log to ESP
func espLogger(message string, severity string) {
	if configDryRun {
		message = "[DRYRUN] " + message
	}
	hornbillImport.SetParam("fileName", appName)
	hornbillImport.SetParam("group", "general")
	hornbillImport.SetParam("severity", severity)
	hornbillImport.SetParam("message", message)
	hornbillImport.Invoke("system", "logMessage")
}

func checkConfig() (err error) {
	//Checks:
	// AssetGenericFieldMapping
	// AssetTypeFieldMapping
	// SoftwareInventory - Mapping
	var (
		regex    = `.*\[[A-Za-z0-9]{0,}\].*`
		errorArr []string
	)
	r, _ := regexp.Compile(regex)

	for k, v := range importConf.AssetGenericFieldMapping {
		if r.MatchString(v.(string)) {
			errorArr = append(errorArr, "AssetGenericFieldMapping - "+k+":"+v.(string))
		}
	}

	for k, v := range importConf.AssetTypeFieldMapping {
		if r.MatchString(v.(string)) {
			errorArr = append(errorArr, "AssetTypeFieldMapping - "+k+":"+v.(string))
		}
	}

	for _, assetType := range importConf.AssetTypes {
		for k, v := range assetType.SoftwareInventory.Mapping {
			if r.MatchString(v.(string)) {
				errorArr = append(errorArr, assetType.AssetType+" SoftwareInventory.Mapping  - "+k+":"+v.(string))
			}
		}
	}
	if len(errorArr) > 0 {
		err = errors.New(strings.Join(errorArr[:], "\n"))
	}
	return
}

func supplierManagerInstalled() bool {
	return isAppInstalled("com.hornbill.suppliermanager")
}
func isAppInstalled(app string) (appInstalled bool) {
	_, appInstalled = HInstalledApplications[app]
	return
}
func getApplications() {
	XMLAppList, xmlmcErr := hornbillImport.Invoke("session", "getApplicationList")
	if xmlmcErr != nil {
		logger(4, "API Call failed when trying to get application list:"+xmlmcErr.Error(), true, true)
		return
	}
	var apiResponse xmlmcApplicationResponse
	err := json.Unmarshal([]byte(XMLAppList), &apiResponse)
	if err != nil {
		logger(3, "Failed to read applications: "+err.Error(), true, true)
		return
	}
	if !apiResponse.Status {
		logger(3, "Failed to return applications list: "+apiResponse.State.Error, true, true)
		return
	}
	for i := 0; i < len(apiResponse.Params.Applications); i++ {
		HInstalledApplications[apiResponse.Params.Applications[i].Name] = true
	}
}

func printOnly(r rune) rune {
	if unicode.IsPrint(r) {
		return r
	}
	return -1
}
