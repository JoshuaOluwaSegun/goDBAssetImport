package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/hornbill/color"
)

// getFieldValue --Retrieve field value from mapping via SQL record map
func getFieldValue(k string, v string, u map[string]interface{}) string {
	debugLog("getFieldValue:", k, ":", v)
	fieldMap := v
	//-- Match $variable from String
	re1, err := regexp.Compile(`\[(.*?)\]`)
	if err != nil {
		logger(4, err.Error(), false)
		return fieldMap
	}

	result := re1.FindAllString(fieldMap, 100)

	//-- Loop Matches
	for _, val := range result {
		debugLog("val:", val)
		valFieldMap := ""
		valFieldMap = strings.Replace(val, "[", "", 1)
		valFieldMap = strings.Replace(valFieldMap, "]", "", 1)
		debugLog("valFieldMap 1:", valFieldMap)
		if valFieldMap == "HBAssetType" {
			valFieldMap = StrAssetType
		} else {
			interfaceContent := u[valFieldMap]
			switch v := interfaceContent.(type) {
			case []uint8:
				valFieldMap = string(v)
			default:
				valFieldMap = fmt.Sprintf("%v", u[valFieldMap])
			}
		}
		debugLog("valFieldMap 2:", valFieldMap)
		if valFieldMap != "" {
			if strings.Contains(strings.ToLower(k), "date") {
				valFieldMap = checkDateString(valFieldMap)
			}
			if strings.Contains(valFieldMap, "[") {
				valFieldMap = ""
			} else {
				//20160215 Check for NULL (<nil>) field value
				//Cannot do this when Scanning SQL data, as we don't know the returned cols - we're using MapScan
				if valFieldMap == "<nil>" {
					valFieldMap = ""
				}
			}
		}
		debugLog("valFieldMap 3:", valFieldMap)
		debugLog(fieldMap, ":", val, ":", valFieldMap)
		fieldMap = strings.Replace(fieldMap, val, valFieldMap, 1)
		debugLog("fieldMap:", fieldMap)
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

// checkDateString - returns date from supplied string
func checkDateString(strDate string) string {
	re, _ := regexp.Compile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)
	strNewDate := re.FindString(strDate)
	return strNewDate
}

func debugLog(debugStrings ...string) {
	if configDebug {
		logger(1, strings.Join(debugStrings, " "), false)
	}
}
