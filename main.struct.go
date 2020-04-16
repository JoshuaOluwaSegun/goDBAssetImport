package main

import (
	"encoding/xml"
	"sync"
	"time"
)

//----- Constants -----
const version = "1.9.1"
const appServiceManager = "com.hornbill.servicemanager"

//----- Variables -----
var (
	assets            = make(map[string]string)
	maxLogFileSize    int64
	SQLImportConf     sqlImportConfStruct
	Sites             []siteListStruct
	Groups            []groupListStruct
	counters          counterTypeStruct
	configFileName    string
	configMaxRoutines string
	configDebug       bool
	configDryRun      bool
	configVersion     bool
	Customers         []customerListStruct
	TimeNow           string
	APITimeNow        string
	startTime         time.Time
	endTime           time.Duration
	AssetClass        string
	AssetTypeID       int
	BaseSQLQuery      string
	StrAssetType      string
	StrSQLAppend      string
	mutex             = &sync.Mutex{}
	mutexBar          = &sync.Mutex{}
	mutexCounters     = &sync.Mutex{}
	mutexCustomers    = &sync.Mutex{}
	mutexGroup        = &sync.Mutex{}
	mutexSite         = &sync.Mutex{}
	worker            sync.WaitGroup
	maxGoroutines     = 1
	logFilePart       = 0
)

//----- Structures -----

type siteListStruct struct {
	SiteName string
	SiteID   int
}
type groupListStruct struct {
	GroupName string
	GroupType int
	GroupID   string
}
type counterTypeStruct struct {
	updated              uint16
	created              uint16
	updateSkipped        uint16
	updateRelatedFailed  uint16
	updateRelatedSkipped uint16
	createSkipped        uint16
	updateFailed         uint16
	createFailed         uint16
}
type sqlImportConfStruct struct {
	APIKey                   string
	InstanceID               string
	Entity                   string
	LogSizeBytes             int64
	SQLConf                  sqlConfStruct
	AssetTypes               []assetTypesStruct
	AssetGenericFieldMapping map[string]interface{}
	AssetTypeFieldMapping    map[string]interface{}
}

type assetTypesStruct struct {
	AssetType       string
	Query           string
	AssetIdentifier assetIdentifierStruct
}

type assetIdentifierStruct struct {
	DBColumn     string
	Entity       string
	EntityColumn string
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

//Group Structs
type xmlmcGroupListResponse struct {
	MethodResult string      `xml:"status,attr"`
	GroupID      string      `xml:"params>rowData>row>h_id"`
	State        stateStruct `xml:"state"`
}

//----- Customer Structs
type customerListStruct struct {
	CustomerID     string
	CustomerName   string
	CustomerHandle string
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
