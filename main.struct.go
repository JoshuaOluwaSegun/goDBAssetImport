package main

import (
	"encoding/xml"
	"sync"
	"time"
)

//----- Constants -----
const version = "1.15.0"
const appServiceManager = "com.hornbill.servicemanager"

//----- Variables -----
var (
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
	TimeNow                string
	APITimeNow             string
	startTime              time.Time
	endTime                time.Duration
	AssetClass             string
	AssetTypeID            int
	BaseSQLQuery           string
	StrAssetType           string
	StrSQLAppend           string
	HInstalledApplications []string
	blnCMInPolicy          bool
	blnContractConnect     bool
	blnSupplierConnect     bool
	mutex                  = &sync.Mutex{}
	mutexBar               = &sync.Mutex{}
	mutexCounters          = &sync.Mutex{}
	mutexCustomers         = &sync.Mutex{}
	mutexGroup             = &sync.Mutex{}
	mutexSite              = &sync.Mutex{}
	worker                 sync.WaitGroup
	maxGoroutines          = 1
	logFilePart            = 0
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
}

type assetIdentifierStruct struct {
	DBContractColumn string
	DBSupplierColumn string
	DBInPolicyColumn string
	DBColumn         string
	Entity           string
	EntityColumn     string
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

type xmlmcCreateResponse struct {
	MethodResult    string      `xml:"status,attr"`
	PrimaryKeyValue string      `xml:"params>primaryEntityKeyValue"`
	State           stateStruct `xml:"state"`
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
	CustomerID        string      `xml:"params>customerId"`
	State             stateStruct `xml:"state"`
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
	HMiddleName string `json:"h_middle_name"`
	HLastName   string `json:"h_last_name"`
	HPhone      string `json:"h_phone"`
	HEmail      string `json:"h_email"`
	HMobile     string `json:"h_mobile"`
	HSnA        string `json:"h_sn_a"`
	HSnB        string `json:"h_sn_b"`
	HAttrib1    string `json:"h_attrib_1"`
	HAttrib2    string `json:"h_attrib_2"`
	HAttrib3    string `json:"h_attrib_3"`
	/*	HJobTitle            string `json:"h_job_title"`
		HLoginCreds          string `json:"h_login_creds"`
		HClass               string `json:"h_class"`
		HAvailStatus         string `json:"h_avail_status"`
		HAvailStatusMsg      string `json:"h_avail_status_msg"`
		HTimezone            string `json:"h_timezone"`
		HCountry             string `json:"h_country"`
		HLanguage            string `json:"h_language"`
		HDateTimeFormat      string `json:"h_date_time_format"`
		HDateFormat          string `json:"h_date_format"`
		HTimeFormat          string `json:"h_time_format"`
		HCurrencySymbol      string `json:"h_currency_symbol"`
		HLastLogon           string `json:"h_last_logon"`
		HSnC                 string `json:"h_sn_c"`
		HSnD                 string `json:"h_sn_d"`
		HSnE                 string `json:"h_sn_e"`
		HSnF                 string `json:"h_sn_f"`
		HSnG                 string `json:"h_sn_g"`
		HSnH                 string `json:"h_sn_h"`
		HIconRef             string `json:"h_icon_ref"`
		HIconChecksum        string `json:"h_icon_checksum"`
		HDob                 string `json:"h_dob"`
		HAccountStatus       string `json:"h_account_status"`
		HFailedAttempts      string `json:"h_failed_attempts"`
		HIdxRef              string `json:"h_idx_ref"`
		HSite                string `json:"h_site"`
		HManager             string `json:"h_manager"`
		HSummary             string `json:"h_summary"`
		HInterests           string `json:"h_interests"`
		HQualifications      string `json:"h_qualifications"`
		HPersonalInterests   string `json:"h_personal_interests"`
		HSkills              string `json:"h_skills"`
		HGender              string `json:"h_gender"`
		HNationality         string `json:"h_nationality"`
		HReligion            string `json:"h_religion"`
		HHomeTelephoneNumber string `json:"h_home_telephone_number"`
		HHomeAddress         string `json:"h_home_address"`
		HBlog                string `json:"h_blog"`
		HAttrib4             string `json:"h_attrib_4"`
		HAttrib5             string `json:"h_attrib_5"`
		HAttrib6             string `json:"h_attrib_6"`
		HAttrib7             string `json:"h_attrib_7"`
		HAttrib8             string `json:"h_attrib_8"`
		HHomeOrg             string `json:"h_home_organization"`
	*/
}

//Asset Structs
type xmlmcAssetResponse struct {
	MethodResult string            `xml:"status,attr"`
	Params       paramsAssetStruct `xml:"params"`
	State        stateStruct       `xml:"state"`
}
type xmlmcAssetDetails struct {
	MethodResult string            `xml:"status,attr"`
	Details      assetObjectStruct `xml:"params>primaryEntityData>record"`
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
	UsedByName string `xml:"h_used_by_name"`
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
