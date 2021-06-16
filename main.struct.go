package main

import (
	"encoding/xml"
	"sync"
	"time"

	apiLib "github.com/hornbill/goApiLib"
)

//----- Constants -----
const (
	version           = "1.16.0"
	appServiceManager = "com.hornbill.servicemanager"
	appName           = "goDBAssetImport"
)

//----- Variables -----
var (
	connString             string
	assets                 = make(map[string]string)
	maxLogFileSize         int64
	SQLImportConf          sqlImportConfStruct
	Sites                  []siteListStruct
	Groups                 []groupListStruct
	counters               counterTypeStruct
	configFileName         string
	configMaxRoutines      string
	configDebug            bool
	configDryRun           bool
	configVersion          bool
	Customers              []customerListStruct
	startTime              time.Time
	AssetClass             string
	AssetTypeID            int
	BaseSQLQuery           string
	StrAssetType           string
	StrSQLAppend           string
	HInstalledApplications []string
	mutex                  = &sync.Mutex{}
	mutexAssets            = &sync.Mutex{}
	mutexBar               = &sync.Mutex{}
	mutexBuffer            = &sync.Mutex{}
	mutexCounters          = &sync.Mutex{}
	mutexCustomers         = &sync.Mutex{}
	mutexGroup             = &sync.Mutex{}
	mutexSite              = &sync.Mutex{}
	worker                 sync.WaitGroup
	maxGoroutines          = 1
	logFilePart            = 0

	hornbillImport *apiLib.XmlmcInstStruct
	pageSize       int
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
	softwareCreated      uint32
	softwareRemoved      uint32
	softwareSkipped      uint32
	softwareCreateFailed uint32
	softwareRemoveFailed uint32
}
type sqlImportConfStruct struct {
	APIKey                   string
	InstanceID               string
	Entity                   string
	HornbillUserIDColumn     string
	LogSizeBytes             int64
	SQLConf                  sqlConfStruct
	AssetTypes               []assetTypesStruct
	AssetGenericFieldMapping map[string]interface{}
	AssetTypeFieldMapping    map[string]interface{}
}

type assetTypesStruct struct {
	AssetType                string
	OperationType            string
	PreserveShared           bool
	PreserveState            bool
	PreserveSubState         bool
	PreserveOperationalState bool
	Query                    string
	AssetIdentifier          assetIdentifierStruct
	SoftwareInventory        softwareInventoryStruct
	Class                    string
	TypeID                   int
}

type assetIdentifierStruct struct {
	DBContractColumn string
	DBSupplierColumn string
	DBInPolicyColumn string
	DBColumn         string
	Entity           string
	EntityColumn     string
}

type softwareInventoryStruct struct {
	AssetIDColumn string
	AppIDColumn   string
	Query         string
	Mapping       map[string]interface{}
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
type xmlmcSiteResponse struct {
	Params struct {
		Sites string `json:"sites"`
		Count string `json:"count"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}

type xmlmcSitesReader struct {
	Row []struct {
		ID   string `json:"h_id"`
		Name string `json:"h_site_name"`
	} `json:"row"`
}
type xmlmcIndySite struct {
	Row struct {
		ID   string `json:"h_id"`
		Name string `json:"h_site_name"`
	} `json:"row"`
}

//Group Structs
type xmlmcGroupResponse struct {
	Params struct {
		Group []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"group"`
		MaxPages int `json:"maxPages"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}

//----- Customer Structs
type customerListStruct struct {
	CustomerID     string
	CustomerName   string
	CustomerHandle string
}

type xmlmcAssetRecordsResponse struct {
	Params struct {
		RowData struct {
			Row []map[string]interface{} `json:"row"`
		} `json:"rowData"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}

type xmlmcSoftwareRecordsResponse struct {
	Params struct {
		RowData struct {
			Row []softwareRecordDetailsStruct `xml:"row"`
		} `xml:"rowData"`
	} `xml:"params"`
	State stateJSONStruct `xml:"state"`
}

type softwareRecordDetailsStruct struct {
	Count      uint64 `xml:"count"`
	HPKID      int    `xml:"h_pk_id"`
	HFKAssetID int    `xml:"h_fk_asset_id"`
	HAppName   string `xml:"h_app_name"`
	HAppID     string `xml:"h_app_id"`
}

type xmlmcCountResponse struct {
	Params struct {
		RowData struct {
			Row []struct {
				Count string `json:"count"`
			} `json:"row"`
		} `json:"rowData"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}
type stateJSONStruct struct {
	Code      string `json:"code"`
	Service   string `json:"service"`
	Operation string `json:"operation"`
	Error     string `json:"error"`
}

type xmlmcUserListResponse struct {
	Params struct {
		RowData struct {
			Row []userAccountStruct `json:"row"`
		} `json:"rowData"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}

type userAccountStruct struct {
	HUserID     string `json:"h_user_id"`
	HLoginID    string `json:"h_login_id"`
	HEmployeeID string `json:"h_employee_id"`
	HName       string `json:"h_name"`
	HFirstName  string `json:"h_first_name"`
	HLastName   string `json:"h_last_name"`
	HEmail      string `json:"h_email"`
	HAttrib1    string `json:"h_attrib_1"`
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
	HPKID     int    `xml:"primaryEntityData>record>h_pk_id"`
}
