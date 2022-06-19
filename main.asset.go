package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	apiLib "github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"
	"github.com/jmoiron/sqlx"
)

func getAssetCount(assetType assetTypesStruct, espXmlmc *apiLib.XmlmcInstStruct) (assetCount uint64, err error) {
	hornbillImport.SetParam("application", appServiceManager)
	hornbillImport.SetParam("queryName", "getAssetsListForImport")
	hornbillImport.OpenElement("queryParams")
	hornbillImport.SetParam("classId", assetType.Class)
	if assetType.TypeID != 0 {
		hornbillImport.SetParam("typeId", strconv.Itoa(assetType.TypeID))
	}
	hornbillImport.CloseElement("queryParams")
	hornbillImport.OpenElement("queryOptions")
	hornbillImport.SetParam("queryType", "count")
	hornbillImport.CloseElement("queryOptions")

	RespBody, err := hornbillImport.Invoke("data", "queryExec")

	var JSONResp xmlmcCountResponse
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		return
	}
	if JSONResp.State.Error != "" {
		err = errors.New(JSONResp.State.Error)
		return
	}

	//-- return Count
	assetCount, err = strconv.ParseUint(JSONResp.Params.RowData.Row[0].Count, 10, 16)
	return
}

func getAssetRecords(assetCount uint64, assetType assetTypesStruct, espXmlmc *apiLib.XmlmcInstStruct) (map[string]map[string]interface{}, error) {
	var (
		loopCount uint64
		queryType string
		recordMap = make(map[string]map[string]interface{})
		err       error
	)
	pageSize = 1000
	switch assetType.Class {
	case "basic":
		queryType = "recordsBasic"
	case "computer":
		queryType = "recordsComputer"
	case "computerPeripheral":
		queryType = "recordsComputerPeripheral"
	case "mobileDevice":
		queryType = "recordsMobileDevice"
	case "networkDevice":
		queryType = "recordsNetworkDevice"
	case "printer":
		queryType = "recordsPrinter"
	case "software":
		queryType = "recordsSoftware"
	case "telecoms":
		queryType = "recordsTelecoms"
	}
	//-- Init Map
	//-- Load Results in pages of pageSize
	bar := pb.StartNew(int(assetCount))
	RespBody := ""
	for loopCount < assetCount {
		logger(1, "Loading Asset List Offset: "+fmt.Sprintf("%d", loopCount)+"\n", false, false)
		hornbillImport.SetParam("application", appServiceManager)
		hornbillImport.SetParam("queryName", "getAssetsListForImport")
		hornbillImport.OpenElement("queryParams")
		hornbillImport.SetParam("rowstart", strconv.FormatUint(loopCount, 10))
		hornbillImport.SetParam("limit", strconv.Itoa(pageSize))
		hornbillImport.SetParam("classId", assetType.Class)
		if assetType.TypeID != 0 {
			hornbillImport.SetParam("typeId", strconv.Itoa(assetType.TypeID))
		}
		hornbillImport.CloseElement("queryParams")
		hornbillImport.OpenElement("queryOptions")
		hornbillImport.SetParam("queryType", queryType)
		if assetType.InPolicyField != "" {
			hornbillImport.SetParam("inPolicyInclusion", "true")
		}
		hornbillImport.CloseElement("queryOptions")

		RespBody, err = hornbillImport.Invoke("data", "queryExec")
		var JSONResp xmlmcAssetRecordsResponse
		if err != nil {
			logger(4, "Error returning page of asset records: "+err.Error(), false, true)
			break
		}
		err = json.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			logger(4, "Error returning page of asset records: "+err.Error(), false, true)
			break
		}
		if JSONResp.State.Error != "" {
			err = errors.New(JSONResp.State.Error)
			logger(4, "Error returning page of asset records: "+JSONResp.State.Error, false, true)
			break
		}

		// Add page size
		loopCount += uint64(pageSize)

		//-- Check for empty result set
		if len(JSONResp.Params.RowData.Row) == 0 {
			break
		}
		for _, v := range JSONResp.Params.RowData.Row {
			bar.Add(1)
			if v[assetType.AssetIdentifier.EntityColumn] != nil {
				keyVal := fmt.Sprintf("%s", v[assetType.AssetIdentifier.EntityColumn])
				recordMap[keyVal] = v
			}
		}
	}
	bar.FinishPrint("Hornbill " + assetType.AssetType + " Asset Records Cached \n")

	return recordMap, err
}

//getAssetClass -- Get Asset Class & Type ID from Asset Type Name
func getAssetClass(confAssetType string) (assetClass string, assetType int) {
	espXmlmc := apiLib.NewXmlmcInstance(importConf.InstanceID)
	espXmlmc.SetAPIKey(importConf.APIKey)
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
		logger(4, "API Call failed when retrieving Asset Class:"+xmlmcErr.Error(), false, true)
		logger(1, "API XML: "+XMLSTRING, false, true)
	}

	var xmlRespon xmlmcTypeListResponse
	err := xml.Unmarshal([]byte(XMLGetMeta), &xmlRespon)
	if err != nil {
		logger(4, "Could not get Asset Class and Type. Please check AssetType within your configuration file:"+err.Error(), true, true)
		logger(1, "API XML: "+XMLSTRING, false, true)
	} else {
		assetClass = xmlRespon.Params.Row.TypeClass
		assetType = xmlRespon.Params.Row.TypeID
	}
	return
}


// addInPolicy --
func addInPolicy(assetId string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) bool {
	boolReturn := false

	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "ConfigurationItemsInPolicy")
	espXmlmc.SetParam("returnModifiedData", "false")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_entity_id", assetId)
	espXmlmc.SetParam("h_entity_name", "asset")
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")

	var XMLSTRING = espXmlmc.GetParam()
	if !configDryRun {
		XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(4, "API Call failed when trying to bring asset in policy: "+xmlmcErr.Error()))
			buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
			return boolReturn
		}
		var xmlRespon xmlmcResponse

		err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
		if err != nil {
			buffer.WriteString(loggerGen(4, "Failed to bring asset in policy: "+err.Error()))
			buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
			return boolReturn
		} else {
			if xmlRespon.MethodResult != "ok" {
				buffer.WriteString(loggerGen(4, "Failed to bring asset in policy: "+xmlRespon.State.ErrorRet))
				buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
				return boolReturn
			} else {
				boolReturn = true
			}
		}
	} else {
		buffer.WriteString(loggerGen(1, "Asset Policy XML: "+XMLSTRING))
		espXmlmc.ClearParam()
		boolReturn = true
	}
	return boolReturn
}

// removeInPolicy --
func removeInPolicy(inPolicyId string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) bool {
	boolReturn := false

	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "ConfigurationItemsInPolicy")
	espXmlmc.SetParam("keyValue", inPolicyId)
	espXmlmc.SetParam("preserveOneToOneData", "true")
	espXmlmc.SetParam("preserveOneToManyData", "true")

	var XMLSTRING = espXmlmc.GetParam()
	if !configDryRun {
		XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityDeleteRecord")
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(4, "API Call failed when trying to remove the asset out of policy: "+xmlmcErr.Error()))
			buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
			return boolReturn
		}
		var xmlRespon xmlmcResponse

		err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
		if err != nil {
			buffer.WriteString(loggerGen(4, "Failed to bring asset out of policy: "+err.Error()))
			buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
			return boolReturn
		} else {
			if xmlRespon.MethodResult != "ok" {
				buffer.WriteString(loggerGen(4, "Failed to bring asset out of policy: "+xmlRespon.State.ErrorRet))
				buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
				return boolReturn
			} else {
				boolReturn = true
			}
		}
	} else {
		buffer.WriteString(loggerGen(1, "Asset Policy XML: "+XMLSTRING))
		espXmlmc.ClearParam()
		boolReturn = true
	}
	return boolReturn
}

//processAssets -- Processes Assets from Asset Map
//--If asset already exists on the instance, update
//--If asset doesn't exist, create
func processAssets(arrAssets map[string]map[string]interface{}, assetsCache map[string]map[string]interface{}, assetType assetTypesStruct) {
	logger(3, "Processing "+strconv.Itoa(len(arrAssets))+" of "+assetType.AssetType+" Type Assets...", true, true)
	bar := pb.StartNew(len(arrAssets))

	//Get the identity of the AssetID field from the config
	assetIDIdent := fmt.Sprintf("%v", assetType.AssetIdentifier.SourceColumn)
	debugLog(nil, "Asset Identifier:", assetType.AssetIdentifier.Entity, assetType.AssetIdentifier.EntityColumn, assetType.AssetIdentifier.SourceColumn, assetIDIdent)
	blnContractConnect := supplierManagerInstalled() && assetType.AssetIdentifier.SourceContractColumn != ""
	blnSupplierConnect := supplierManagerInstalled() && assetType.AssetIdentifier.SourceSupplierColumn != ""

	//-- Loop each asset
	maxGoroutinesGuard := make(chan struct{}, configMaxRoutines)
	for _, assetRecord := range arrAssets {
		maxGoroutinesGuard <- struct{}{}
		worker.Add(1)
		var (
			assetIDInstance string
			hbRecordHash    string
			hbSIRecordHash  string
			dbRecordHash    string
			assetForHash    []map[string]interface{}
			assetMap        = assetRecord
		)

		//Get the asset ID for the current record
		assetID := iToS(assetMap[assetIDIdent])

		dbRecordHash = Hash(append(assetForHash, assetRecord))

		go func() {
			defer worker.Done()
			mutexBar.Lock()
			bar.Increment()
			mutexBar.Unlock()

			var (
				boolUpdate          = false
				boolUpdateSI        = false
				boolCreate          = false
				boolActioned        = false
				err                 error
				db                  *sqlx.DB
				buffer              bytes.Buffer
				softwareRecords     map[string]map[string]interface{}
				softwareRecordsHash string
			)

			if !configCSV && !configNexthink && !configLDAP {
				//One DB connection per worker
				db, err = makeDBConnection()
				if err != nil {
					logger(4, "[DATABASE] "+err.Error(), false, true)
				}
				defer db.Close()
			}

			//One XMLMC connection per worker
			espXmlmc := apiLib.NewXmlmcInstance(importConf.InstanceID)
			espXmlmc.SetAPIKey(importConf.APIKey)

			buffer.WriteString(loggerGen(1, "    "))
			buffer.WriteString(loggerGen(1, "Processing Asset: "+assetID))

			if asset, ok := assetsCache[assetID]; ok {
				//Asset exists
				assetIDInstance = fmt.Sprintf("%v", asset["h_pk_asset_id"])
				debugLog(&buffer, "Asset ID Instance"+assetIDInstance)
				debugLog(&buffer, "Asset Class: "+assetType.Class)
				switch assetType.Class {
				case "computer":
					//Main asset record
					hbRecordHash = fmt.Sprintf("%v", asset["h_dsc_cf_fingerprint"])
					debugLog(&buffer, "Database Asset Record Hash: "+dbRecordHash)
					debugLog(&buffer, "Hornbill Asset Record Hash: "+hbRecordHash)
					if hbRecordHash != dbRecordHash || configForceUpdates {
						boolUpdate = true
					} else {
						mutexCounters.Lock()
						counters.updateSkipped++
						mutexCounters.Unlock()
					}

					if !configCSV {
						//Software inventory records
						hbSIRecordHash = fmt.Sprintf("%v", asset["h_dsc_sw_fingerprint"])
						softwareRecords, softwareRecordsHash, err = getSoftwareRecords(assetMap, assetType, espXmlmc, db, &buffer)
						debugLog(&buffer, "Hornbill Asset Software Inventory Record Hash: "+hbSIRecordHash)
						debugLog(&buffer, "Database Asset Software Inventory Record Hash: "+softwareRecordsHash)

						if err != nil {
							buffer.WriteString(loggerGen(4, err.Error()))
							mutexCounters.Lock()
							counters.softwareCreateFailed++
							mutexCounters.Unlock()
						}
						if len(softwareRecords) > 0 && hbSIRecordHash != softwareRecordsHash {
							boolUpdateSI = true
						} else {
							buffer.WriteString(loggerGen(1, "Asset match found, no software inventory updates required"))
							mutexCounters.Lock()
							counters.softwareSkipped++
							mutexCounters.Unlock()
						}
					}

				case "mobileDevice":
					//Main asset record
					hbRecordHash = fmt.Sprintf("%v", asset["h_dsc_fingerprint"])
					debugLog(&buffer, "Database Asset Record Hash: "+dbRecordHash)
					debugLog(&buffer, "Hornbill Asset Record Hash: "+hbRecordHash)
					if hbRecordHash != dbRecordHash || configForceUpdates {
						boolUpdate = true
					} else {
						mutexCounters.Lock()
						counters.updateSkipped++
						mutexCounters.Unlock()
					}

					if !configCSV {
						//Software inventory records
						hbSIRecordHash = fmt.Sprintf("%v", asset["h_dsc_sw_fingerprint"])
						debugLog(&buffer, "Hornbill Asset Software Inventory Record Hash: "+hbSIRecordHash)
						softwareRecords, softwareRecordsHash, err = getSoftwareRecords(assetMap, assetType, espXmlmc, db, &buffer)
						if err != nil {
							buffer.WriteString(loggerGen(4, err.Error()))
							mutexCounters.Lock()
							counters.softwareCreateFailed++
							mutexCounters.Unlock()
						}
						if len(softwareRecords) > 0 && hbSIRecordHash != softwareRecordsHash {
							boolUpdateSI = true
						} else {
							buffer.WriteString(loggerGen(1, "Asset match found, no software inventory updates required"))
							mutexCounters.Lock()
							counters.softwareSkipped++
							mutexCounters.Unlock()
						}
					}

				case "printer":
					hbRecordHash = fmt.Sprintf("%v", asset["h_dsc_siid"])
					debugLog(&buffer, "Database Asset Record Hash: "+dbRecordHash)
					debugLog(&buffer, "Hornbill Asset Record Hash: "+hbRecordHash)
					if hbRecordHash != dbRecordHash || configForceUpdates {
						boolUpdate = true
					} else {
						mutexCounters.Lock()
						counters.updateSkipped++
						mutexCounters.Unlock()
					}
				default:
					//basic
					//computerPeripheral
					//networkDevice
					//software
					//telecoms
					hbRecordHash = fmt.Sprintf("%v", asset["h_dsc_fingerprint"])
					debugLog(&buffer, "Database Asset Record Hash: "+dbRecordHash)
					debugLog(&buffer, "Hornbill Asset Record Hash: "+hbRecordHash)
					if hbRecordHash != dbRecordHash || configForceUpdates {
						boolUpdate = true
					} else {
						mutexCounters.Lock()
						counters.updateSkipped++
						mutexCounters.Unlock()
					}
					boolUpdate = true
				}
				if !boolUpdate {
					buffer.WriteString(loggerGen(1, "Asset match found, no details require updating"))
				}
			} else {
				debugLog(&buffer, "Asset Match Doesn't Exist - Create")
				boolCreate = true
			}

			//-- Update or Create Asset
			if boolUpdate {
				if assetType.OperationType == "" || strings.ToLower(assetType.OperationType) == "both" || strings.ToLower(assetType.OperationType) == "update" {
					usedBy := ""
					if assetType.PreserveShared {
						usedBy = iToS(assetMap["h_used_by_name"])
					}
					buffer.WriteString(loggerGen(1, "Update Asset: "+assetID))
					boolActioned = updateAsset(assetType, assetMap, assetIDInstance, assetID, usedBy, espXmlmc, &buffer)
					if strings.ToLower(assetType.InPolicyField) == "yes" {
						inPolicyId, ok := assetMap["h_pk_confiteminpolicyid"]
						var strIPID string
						if ok {
							strIPID = fmt.Sprintf("%v", inPolicyId)
						}
						if strIPID != "" && strIPID != "0" {
							// in policy exists, so no need to do anything
						} else {
							addInPolicy(assetIDInstance, espXmlmc, &buffer)
						}
					} else if assetType.InPolicyField == "__clear__" {
						inPolicyId, ok := assetMap["h_pk_confiteminpolicyid"];
						if ok {
							var strIPID string
							strIPID = fmt.Sprintf("%v", inPolicyId)
							if strIPID != "" && strIPID != "0" {
								removeInPolicy(strIPID, espXmlmc, &buffer)
							}
						}
					}
				} else {
					buffer.WriteString(loggerGen(1, "Asset match found, but OperationType not set to Both or Update"))
				}
			}
			if boolCreate {
				if assetType.OperationType == "" || strings.ToLower(assetType.OperationType) == "both" || strings.ToLower(assetType.OperationType) == "create" {
					buffer.WriteString(loggerGen(1, "Create Asset: "+assetID))
					assetIDInstance, boolActioned = createAsset(assetType, assetMap, assetID, espXmlmc, db, &buffer)
					if strings.ToLower(assetType.InPolicyField) == "yes" {
						addInPolicy(assetIDInstance, espXmlmc, &buffer)
					}
					
				} else {
					buffer.WriteString(loggerGen(1, "Asset match not found, but OperationType not set to Both or Create"))
				}
			}
			if boolUpdateSI && !configDryRun {
				err = updateAssetSI(assetIDInstance, softwareRecords, softwareRecordsHash, assetType, espXmlmc, &buffer)
				if err != nil {
					buffer.WriteString(loggerGen(4, err.Error()))
				}
			}

			// additional stuff
			if boolActioned && assetIDInstance != "" {
				if blnSupplierConnect {
					supplierID := iToS(assetMap[assetType.AssetIdentifier.SourceSupplierColumn])
					if supplierID != "" {

						err = addSupplierToAsset(assetIDInstance, supplierID, espXmlmc, &buffer)
						if err != nil {
							counters.suppliersAssociatedFailed++
							buffer.WriteString(loggerGen(4, "Unable to associate Supplier ["+supplierID+"] to Asset ["+assetID+"]: "+err.Error()))
						}
					}
				}
				if blnContractConnect {
					contractID := iToS(assetMap[assetType.AssetIdentifier.SourceContractColumn])
					if contractID != "" {
						err = addSupplierContractToAsset(assetIDInstance, contractID, espXmlmc, &buffer)
						if err != nil {
							counters.supplierContractsAssociatedFailed++
							buffer.WriteString(loggerGen(4, "Unable to associate Contract ["+contractID+"] to Asset ["+assetID+"]: "+err.Error()))
						}
					}
				}
			}
			mutexBuffer.Lock()
			loggerWriteBuffer(buffer.String())
			mutexBuffer.Unlock()
			buffer.Reset()
			<-maxGoroutinesGuard
		}()
	}
	worker.Wait()
	bar.FinishPrint(assetType.AssetType + " Asset Type Processing Complete!")
}

// createAsset -- Creates Asset record from the passed through map data
func createAsset(assetType assetTypesStruct, u map[string]interface{}, strNewAssetID string, espXmlmc *apiLib.XmlmcInstStruct, db *sqlx.DB, buffer *bytes.Buffer) (string, bool) {

	var (
		newAssetHash        string
		err                 error
		softwareRecords     map[string]map[string]interface{}
		softwareRecordsHash string
	)

	var assetForHash []map[string]interface{}
	newAssetHash = Hash(append(assetForHash, u))
	if assetType.Class == "computer" || assetType.Class == "mobileDevice" {

		if !configCSV {
			softwareRecords, softwareRecordsHash, err = getSoftwareRecords(u, assetType, espXmlmc, db, buffer)
			if err != nil {
				buffer.WriteString(loggerGen(4, err.Error()))
				mutexCounters.Lock()
				counters.softwareCreateFailed++
				mutexCounters.Unlock()
			}
		}
	}

	//Get site ID
	siteID, siteName := getSiteID(u, buffer)

	//Get Company ID
	companyID, companyName := getGroupID(u, "company", buffer)

	//Get Department ID
	departmentID, departmentName := getGroupID(u, "department", buffer)

	//Get Owned By details
	_, ownedByURN, ownedByName := getUserID(u, "h_owned_by", buffer)

	//Get Used By details
	_, usedByURN, usedByName := getUserID(u, "h_used_by", buffer)

	//Get Last Logged On details
	_, lastLoggedOnByURN, _ := getUserID(u, "h_last_logged_on_user", buffer)

	//Get/Set params from map stored against FieldMapping
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Asset")
	espXmlmc.SetParam("returnModifiedData", "true")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_class", AssetClass)
	espXmlmc.SetParam("h_type", strconv.Itoa(AssetTypeID))

	espXmlmc.SetParam("h_last_updated", time.Now().Format("2006-01-02 15:04:05"))
	espXmlmc.SetParam("h_last_updated_by", "Import - Add")

	//Get asset field mapping
	debugLog(buffer, "Asset Field Mapping")
	for k, v := range importConf.AssetGenericFieldMapping {
		strMapping := fmt.Sprintf("%v", v)
		value := getFieldValue(k, strMapping, u, buffer)
		debugLog(buffer, k, ":", strMapping, ":", value)

		if value == "__clear__" {
			continue
		}

		if k == "h_used_by" && usedByName != "" && usedByURN != "" {
			espXmlmc.SetParam("h_used_by", usedByURN)
			espXmlmc.SetParam("h_used_by_name", usedByName)
		}

		if k == "h_owned_by" && ownedByName != "" && ownedByURN != "" {
			espXmlmc.SetParam("h_owned_by", ownedByURN)
			espXmlmc.SetParam("h_owned_by_name", ownedByName)
		}

		if k == "h_site" && siteID != 0 && siteName != "" {
			espXmlmc.SetParam("h_site", siteName)
			espXmlmc.SetParam("h_site_id", strconv.Itoa(siteID))
		}

		if k == "h_company_name" && companyID != "" && companyName != "" {
			espXmlmc.SetParam("h_company_name", companyName)
			espXmlmc.SetParam("h_company_id", companyID)
		}

		if k == "h_department_name" && departmentID != "" && departmentName != "" {
			espXmlmc.SetParam("h_department_name", departmentName)
			espXmlmc.SetParam("h_department_id", departmentID)
		}

		if k != "h_site" &&
			k != "h_used_by" &&
			k != "h_owned_by" &&
			k != "h_company_name" &&
			k != "h_department_name" &&
			strMapping != "" && value != "" {
			espXmlmc.SetParam(k, value)
		}
	}
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")

	//Add extended asset type field mapping
	espXmlmc.OpenElement("relatedEntityData")
	espXmlmc.SetParam("relationshipName", "AssetClass")
	espXmlmc.SetParam("entityAction", "insert")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_type", strconv.Itoa(AssetTypeID))
	switch assetType.Class {
	case "computer":
		espXmlmc.SetParam("h_dsc_cf_fingerprint", newAssetHash)
		if softwareRecordsHash != "" {
			espXmlmc.SetParam("h_dsc_sw_fingerprint", softwareRecordsHash)
		}
	case "printer":
		espXmlmc.SetParam("h_dsc_cf_fingerprint", newAssetHash)
	case "mobileDevice":
		espXmlmc.SetParam("h_dsc_fingerprint", newAssetHash)
		if softwareRecordsHash != "" {
			espXmlmc.SetParam("h_dsc_sw_fingerprint", softwareRecordsHash)
		}
	default:
		espXmlmc.SetParam("h_dsc_fingerprint", newAssetHash)
	}
	debugLog(buffer, "Asset Type Field Mapping")

	//Get asset field mapping
	for k, v := range importConf.AssetTypeFieldMapping {
		strMapping := fmt.Sprintf("%v", v)
		value := getFieldValue(k, strMapping, u, buffer)
		debugLog(buffer, k, ":", strMapping, ":", value)

		if k == "h_last_logged_on_user" && lastLoggedOnByURN != "" {
			espXmlmc.SetParam("h_last_logged_on_user", lastLoggedOnByURN)
		}

		if k != "h_last_logged_on_user" &&
			strMapping != "" &&
			value != "" {
			espXmlmc.SetParam(k, value)
		}
	}

	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("relatedEntityData")

	//-- Check for Dry Run
	if !configDryRun {
		var XMLSTRING = espXmlmc.GetParam()
		debugLog(buffer, "Asset Create XML:", XMLSTRING)
		XMLCreate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(4, "Error running entityAddRecord API for createAsset: "+xmlmcErr.Error()))
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			return "", false
		}

		var xmlRespon xmlmcUpdateResponse
		debugLog(buffer, "API Call Response:", XMLCreate)

		err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
		if err != nil {
			mutexCounters.Lock()
			counters.createFailed++
			mutexCounters.Unlock()
			buffer.WriteString(loggerGen(4, "Unable to read response from Hornbill instance from entityAddRecord API for createAsset:"+err.Error()))
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			return "", false
		}

		if xmlRespon.MethodResult != "ok" {
			buffer.WriteString(loggerGen(4, "Unable to add asset: "+xmlRespon.State.ErrorRet))
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			mutexCounters.Lock()
			counters.createFailed++
			mutexCounters.Unlock()
		} else {
			mutexCounters.Lock()
			counters.created++
			mutexCounters.Unlock()
			assetID := xmlRespon.UpdatedCols.AssetPK
			mutexAssets.Lock()
			assets[strNewAssetID] = assetID
			mutexAssets.Unlock()
			buffer.WriteString(loggerGen(1, "Asset record created successfully: "+assetID))

			//Now add asset URN
			espXmlmc.SetParam("application", appServiceManager)
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
				buffer.WriteString(loggerGen(3, "API Call failed when Updating Asset URN:"+xmlmcErr.Error()))
				buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
				return assetID, true
			}

			var xmlRespon xmlmcResponse
			err := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
			if err != nil {
				buffer.WriteString(loggerGen(3, "Unable to read response from Hornbill instance when Updating Asset URN:"+err.Error()))
				buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
				return assetID, true
			}

			if xmlRespon.MethodResult != "ok" {
				buffer.WriteString(loggerGen(3, "Unable to update Asset URN: "+xmlRespon.State.ErrorRet))
				buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
				return assetID, true
			}
			buffer.WriteString(loggerGen(1, "Asset URN updated successfully: "+assetID))

			if (assetType.Class == "computer" || assetType.Class == "mobile") && len(softwareRecords) > 0 {
				buildSoftwareInventory(softwareRecords, assetType, assetID, espXmlmc, buffer)
			}

			return assetID, true
		}
	} else {
		//-- DEBUG XML TO LOG FILE
		var XMLSTRING = espXmlmc.GetParam()
		buffer.WriteString(loggerGen(1, "API Create XML: "+XMLSTRING))
		mutexCounters.Lock()
		counters.createSkipped++
		mutexCounters.Unlock()
		espXmlmc.ClearParam()
	}
	return "", false
}

// updateAsset -- Updates Asset record from the passed through map data and asset ID
//func updateAsset(assetType assetTypesStruct, u map[string]interface{}, strAssetID, strNewAssetID, usedBy string, espXmlmc *apiLib.XmlmcInstStruct, db *sqlx.DB, buffer *bytes.Buffer) bool {
func updateAsset(assetType assetTypesStruct, u map[string]interface{}, strAssetID, strNewAssetID, usedBy string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) bool {

	var (
		newAssetHash      string
		boolRecordUpdated = false
	)

	//Shared clearAttrib array
	var nilAttrib []apiLib.ParamAttribStruct
	attrib := apiLib.ParamAttribStruct{}
	attrib.Name = "nil"
	attrib.Value = "true"
	nilAttrib = append(nilAttrib, attrib)

	var assetForHash []map[string]interface{}
	newAssetHash = Hash(append(assetForHash, u))

	//Get site ID
	siteID, siteName := getSiteID(u, buffer)

	//Get Company ID
	companyID, companyName := getGroupID(u, "company", buffer)

	//Get Department ID
	departmentID, departmentName := getGroupID(u, "department", buffer)

	//Get Owned By details
	ownedByID, ownedByURN, ownedByName := getUserID(u, "h_owned_by", buffer)

	//Get Used By details
	usedByID, usedByURN, usedByName := getUserID(u, "h_used_by", buffer)

	//Get Last Logged On details
	_, lastLoggedOnByURN, _ := getUserID(u, "h_last_logged_on_user", buffer)

	//Get/Set params from map stored against FieldMapping
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Asset")
	espXmlmc.SetParam("returnModifiedData", "true")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_pk_asset_id", strAssetID)
	espXmlmc.SetParam("h_asset_urn", "urn:sys:entity:com.hornbill.servicemanager:Asset:"+strAssetID)
	debugLog(buffer, "Asset Field Mapping")

	//Get asset field mapping
	for k, v := range importConf.AssetGenericFieldMapping {
		strMapping := fmt.Sprintf("%v", v)
		value := getFieldValue(k, strMapping, u, buffer)
		debugLog(buffer, k, ":", strMapping, ":", value)

		if k == "h_operational_state" && assetType.PreserveOperationalState {
			//Skip updating op state
			continue
		}

		if k == "h_record_state" && assetType.PreserveState {
			//Skip updating state
			continue
		}

		if (k == "h_substate_id" || k == "h_substate_name") && assetType.PreserveSubState {
			//Skip updating subState
			continue
		}

		if k == "h_used_by" && usedByID != "" {
			if usedByID == "__clear__" {
				espXmlmc.SetParamAttr("h_used_by", "", nilAttrib)
				espXmlmc.SetParamAttr("h_used_by_name", "", nilAttrib)
			} else if usedByName != "" && usedByURN != "" {
				espXmlmc.SetParam("h_used_by", usedByURN)
				espXmlmc.SetParam("h_used_by_name", usedByName)
			}
			continue
		}

		if k == "h_owned_by" && ownedByID != "" {
			if ownedByID == "__clear__" {
				espXmlmc.SetParamAttr("h_owned_by", "", nilAttrib)
				espXmlmc.SetParamAttr("h_owned_by_name", "", nilAttrib)
			} else if ownedByName != "" && ownedByURN != "" {
				espXmlmc.SetParam("h_owned_by", ownedByURN)
				espXmlmc.SetParam("h_owned_by_name", ownedByName)
			}
			continue
		}

		if k == "h_site" && siteName != "" {
			if siteName == "__clear__" {
				espXmlmc.SetParamAttr("h_site", "", nilAttrib)
				espXmlmc.SetParamAttr("h_site_id", "", nilAttrib)
			} else if siteID != 0 {
				espXmlmc.SetParam("h_site", siteName)
				espXmlmc.SetParam("h_site_id", strconv.Itoa(siteID))
			}
			continue
		}

		if k == "h_company_name" && companyName != "" {
			if companyName == "__clear__" {
				espXmlmc.SetParamAttr("h_company_name", "", nilAttrib)
				espXmlmc.SetParamAttr("h_company_id", "", nilAttrib)
			} else if companyID != "" {
				espXmlmc.SetParam("h_company_name", companyName)
				espXmlmc.SetParam("h_company_id", companyID)
			}
			continue
		}

		if k == "h_department_name" && departmentName != "" {
			if departmentName == "__clear__" {
				espXmlmc.SetParamAttr("h_department_name", "", nilAttrib)
				espXmlmc.SetParamAttr("h_department_id", "", nilAttrib)
			} else if departmentID != "" {
				espXmlmc.SetParam("h_department_name", departmentName)
				espXmlmc.SetParam("h_department_id", departmentID)
			}
			continue
		}

		if value == "__clear__" {
			espXmlmc.SetParamAttr(k, "", nilAttrib)
		} else if strMapping != "" && value != "" {
			espXmlmc.SetParam(k, value)
		}
	}

	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")

	var XMLSTRING = espXmlmc.GetParam()

	if !configDryRun {
		debugLog(buffer, "Asset Update XML:", XMLSTRING)

		XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(4, "API Call failed when Updating Asset:"+xmlmcErr.Error()))
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			mutexCounters.Lock()
			counters.updateFailed++
			mutexCounters.Unlock()
			return false
		}

		var xmlRespon xmlmcUpdateResponse

		err := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
		if err != nil {
			buffer.WriteString(loggerGen(4, "Unable to read response from Hornbill instance when Updating Asset:"+err.Error()))
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLUpdate))
			mutexCounters.Lock()
			counters.updateFailed++
			mutexCounters.Unlock()
			return false
		}

		if xmlRespon.MethodResult != "ok" && xmlRespon.State.ErrorRet != "There are no values to update" && !strings.Contains(xmlRespon.State.ErrorRet, "Superfluous entity record update detected") {
			buffer.WriteString(loggerGen(4, "Unable to Update Asset: "+xmlRespon.State.ErrorRet))
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLUpdate))
			mutexCounters.Lock()
			counters.updateFailed++
			mutexCounters.Unlock()
			return false
		}

		if xmlRespon.MethodResult != "ok" && (xmlRespon.State.ErrorRet == "There are no values to update" || strings.Contains(xmlRespon.State.ErrorRet, "Superfluous entity record update detected")) {
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLUpdate))
			mutexCounters.Lock()
			counters.updateSkipped++
			mutexCounters.Unlock()
		}

		if xmlRespon.MethodResult == "ok" {
			buffer.WriteString(loggerGen(1, "Asset record updated successfully: "+strAssetID))
			boolRecordUpdated = true
		}

		assets[strNewAssetID] = strAssetID

		//-- now process extended record data
		espXmlmc.SetParam("application", appServiceManager)
		espXmlmc.SetParam("entity", "Asset")
		espXmlmc.SetParam("returnModifiedData", "true")
		espXmlmc.OpenElement("primaryEntityData")
		espXmlmc.OpenElement("record")
		espXmlmc.SetParam("h_pk_asset_id", strAssetID)
		espXmlmc.CloseElement("record")
		espXmlmc.CloseElement("primaryEntityData")
		espXmlmc.OpenElement("relatedEntityData")
		espXmlmc.SetParam("relationshipName", "AssetClass")
		espXmlmc.SetParam("entityAction", "update")
		espXmlmc.OpenElement("record")
		espXmlmc.SetParam("h_pk_asset_id", strAssetID)
		switch assetType.Class {
		case "basic":
			espXmlmc.SetParam("h_dsc_fingerprint", newAssetHash)
		case "computer":
			espXmlmc.SetParam("h_dsc_cf_fingerprint", newAssetHash)
		case "printer":
			espXmlmc.SetParam("h_dsc_cf_fingerprint", newAssetHash)
		case "software":
			espXmlmc.SetParam("h_dsc_fingerprint", newAssetHash)
		}
		debugLog(buffer, "Asset Field Mapping")

		//Get asset field mapping
		for k, v := range importConf.AssetTypeFieldMapping {
			strMapping := fmt.Sprintf("%v", v)
			value := getFieldValue(k, strMapping, u, buffer)
			debugLog(buffer, k, ":", strMapping, ":", value)
			if value == "__clear__" {
				espXmlmc.SetParamAttr(k, "", nilAttrib)
			} else {
				if k == "h_last_logged_on_user" && lastLoggedOnByURN != "" {
					espXmlmc.SetParam("h_last_logged_on_user", lastLoggedOnByURN)
				}
				if k != "h_last_logged_on_user" && strMapping != "" && value != "" {
					espXmlmc.SetParam(k, value)
				}
			}
		}

		espXmlmc.CloseElement("record")
		espXmlmc.CloseElement("relatedEntityData")
		XMLMCRequest := espXmlmc.GetParam()
		debugLog(buffer, "Asset Extended Update XML:", XMLMCRequest)

		XMLUpdateExt, xmlmcErrExt := espXmlmc.Invoke("data", "entityUpdateRecord")
		if xmlmcErrExt != nil {
			buffer.WriteString(loggerGen(4, "API Call failed when Updating Asset Extended Details:"+xmlmcErrExt.Error()))
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			mutexCounters.Lock()
			counters.updateFailed++
			mutexCounters.Unlock()
			return false
		}
		var xmlResponExt xmlmcUpdateResponse

		err = xml.Unmarshal([]byte(XMLUpdateExt), &xmlResponExt)
		if err != nil {
			buffer.WriteString(loggerGen(4, "Unable to read response from Hornbill instance when Updating Asset Extended Details:"+err.Error()))
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLUpdateExt))
			mutexCounters.Lock()
			counters.updateRelatedFailed++
			mutexCounters.Unlock()
			return false
		}

		if xmlResponExt.MethodResult != "ok" && xmlResponExt.State.ErrorRet != "There are no values to update" && !strings.Contains(xmlResponExt.State.ErrorRet, "Superfluous entity record update detected") {
			buffer.WriteString(loggerGen(4, "Unable to Update Asset Extended Details: "+xmlResponExt.State.ErrorRet))
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLUpdateExt))
			mutexCounters.Lock()
			counters.updateRelatedFailed++
			mutexCounters.Unlock()
			return false
		}

		if xmlResponExt.MethodResult != "ok" && (xmlResponExt.State.ErrorRet == "There are no values to update" || strings.Contains(xmlResponExt.State.ErrorRet, "Superfluous entity record update detected")) {
			mutexCounters.Lock()
			counters.updateRelatedSkipped++
			mutexCounters.Unlock()
		}

		if xmlResponExt.MethodResult == "ok" {
			boolRecordUpdated = true
			buffer.WriteString(loggerGen(1, "Asset record extended details updated successfully: "+strAssetID))
		}

		if boolRecordUpdated {
			//-- Asset Updated!
			//-- Need to run another update against the Asset for LAST UPDATED and LAST UPDATE BY!
			espXmlmc.SetParam("application", appServiceManager)
			espXmlmc.SetParam("entity", "Asset")
			espXmlmc.OpenElement("primaryEntityData")
			espXmlmc.OpenElement("record")
			espXmlmc.SetParam("h_pk_asset_id", strAssetID)
			espXmlmc.SetParam("h_last_updated", time.Now().Format("2006-01-02 15:04:05"))
			espXmlmc.SetParam("h_last_updated_by", "Import - Update")
			espXmlmc.CloseElement("record")
			espXmlmc.CloseElement("primaryEntityData")
			var XMLSTRING = espXmlmc.GetParam()
			debugLog(buffer, "Asset Update LAST UPDATE XML:", XMLSTRING)

			XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
			if xmlmcErr != nil {
				buffer.WriteString(loggerGen(4, "API Call failed when setting Last Updated values:"+xmlmcErr.Error()))
				buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			} else {
				var xmlRespon xmlmcResponse
				err := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
				if err != nil {
					buffer.WriteString(loggerGen(4, "Unable to read response from Hornbill instance when setting Last Updated values:"+err.Error()))
					buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
				} else {
					if xmlRespon.MethodResult != "ok" && xmlRespon.State.ErrorRet != "There are no values to update" {
						buffer.WriteString(loggerGen(4, "Unable to set Last Updated details for asset: "+xmlRespon.State.ErrorRet))
						buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
					} else {
						buffer.WriteString(loggerGen(1, "Asset Last Updated date & user updated successfully: "+strAssetID))
					}
				}
			}
			mutexCounters.Lock()
			counters.updated++
			mutexCounters.Unlock()
		}

	} else {
		//-- Inc Counter
		mutexCounters.Lock()
		counters.updateSkipped++
		mutexCounters.Unlock()
		buffer.WriteString(loggerGen(1, "Asset Update XML "+XMLSTRING))
		espXmlmc.ClearParam()
	}
	return true
}
