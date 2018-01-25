package main

import (
	"encoding/xml"
	"sync"
	"time"
)

//----- Constants -----
const version = "1.4.1"
const appServiceManager = "com.hornbill.servicemanager"

//----- Variables -----
var (
	maxLogFileSize      int64
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
	//espXmlmc            *apiLib.XmlmcInstStruct
	mutex          = &sync.Mutex{}
	mutexBar       = &sync.Mutex{}
	mutexCounters  = &sync.Mutex{}
	mutexCustomers = &sync.Mutex{}
	mutexSite      = &sync.Mutex{}
	worker         sync.WaitGroup
	maxGoroutines  = 1
	logFilePart    = 0
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
	LogSizeBytes             int64
	SQLConf                  sqlConfStruct
	AssetTypes               map[string]interface{}
	AssetGenericFieldMapping map[string]interface{}
	AssetTypeFieldMapping    map[string]interface{}
}

type sqlConfStruct struct {
	Driver         string
	Server         string
	Database       string
	Authentication string
	UserName       string
	Password       string
	Port           int
	Query          string
	Encrypt        bool
	AssetID        string
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

type xmlmcUpdateResponse struct {
	MethodResult string      `xml:"status,attr"`
	UpdatedCols  updatedCols `xml:"params>primaryEntityData>record"`
	State        stateStruct `xml:"state"`
}
type updatedCols struct {
	AssetPK string       `xml:"h_pk_asset_id"`
	ColList []updatedCol `xml:",any"`
}

type updatedCol struct {
	XMLName xml.Name `xml:""`
	Amount  string   `xml:",chardata"`
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
