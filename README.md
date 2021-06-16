# SQL Asset Import Go - [GO](https://golang.org/) Asset Import Script to Hornbill

## Installation

### Windows Installation

- Download the archive containing the import executable
- Extract zip into a folder you would like the application to run from e.g. `C:\asset_import\`
- Open '''conf_sccm_assetscomputer.json''' and add in the necessary configration
- Open Command Line Prompt as Administrator
- Change Directory to the folder with goDBAssetImport.exe `C:\asset_import\`
- Run the command :
  - For Windows Systems: goDBAssetImport.exe -dryrun=true -file=conf_sccm_assetscomputer.json

### Configuration

Example JSON File:

```json
{
  "APIKey": "",
  "InstanceId": "",
  "LogSizeBytes":1000000,
  "SQLConf": {
      "Driver": "mssql",
      "Server": "",
      "Database": "",
      "Authentication":"Windows",
      "UserName": "",
      "Password": "",
      "Port": 1433,
      "Encrypt": false,
      "Query": "SELECT OARSys.ResourceID AS [AssetID], OARSys.User_Name0 AS [UserName], OARSys.Netbios_Name0 AS [MachineName], OARSys.Resource_Domain_OR_Workgr0 AS [NETDomain], dbo.v_GS_OPERATING_SYSTEM.Caption0 AS [OperatingSystemCaption], OARSys.Operating_System_Name_and0 AS [OperatingSystem], dbo.v_GS_OPERATING_SYSTEM.Version0 AS [OperatingSystemVersion], dbo.v_GS_OPERATING_SYSTEM.CSDVersion0 AS [ServicePackVersion], dbo.v_GS_COMPUTER_SYSTEM.Manufacturer0 AS [SystemManufacturer], dbo.v_GS_COMPUTER_SYSTEM.Model0 AS [SystemModel], dbo.v_GS_PC_BIOS.SerialNumber0 AS [SystemSerialNumber], OAProc.MaxClockSpeed0 AS [ProcessorSpeedGHz], OAProc.Name0 AS [ProcessorName], dbo.v_GS_COMPUTER_SYSTEM.NumberOfProcessors0 AS [NumberofProcessors], dbo.v_GS_X86_PC_MEMORY.TotalPhysicalMemory0 AS [MemoryKB], dbo.v_GS_LOGICAL_DISK.Size0 AS [DiskSpaceMB], dbo.v_GS_LOGICAL_DISK.FreeSpace0 AS [FreeDiskSpaceMB], OAIP.IP_Addresses0 AS [IPAddress], OAMac.MAC_Addresses0 AS [MACAddress], dbo.v_GS_PC_BIOS.Description0 AS [BIOSDescription], dbo.v_GS_PC_BIOS.ReleaseDate0 AS [BIOSReleaseDate], dbo.v_GS_PC_BIOS.SMBIOSBIOSVersion0 AS [SMBIOSVersion], dbo.v_GS_SYSTEM.SystemRole0 AS [SystemType], OASysEncl.ChassisTypes0 AS [ChassisTypes], OASysEncl.TimeStamp AS [ChassisDate], OARSys.AD_Site_Name0 AS [SiteName] FROM dbo.v_R_System OUTER APPLY (SELECT TOP 1 * FROM dbo.v_R_System b WHERE b.Netbios_Name0 = dbo.v_R_System.Netbios_Name0 ORDER BY SMS_UUID_Change_Date0 DESC) OARSys OUTER APPLY (SELECT TOP 1 dbo.v_GS_SYSTEM_ENCLOSURE.* FROM dbo.v_GS_SYSTEM_ENCLOSURE WHERE dbo.v_GS_SYSTEM_ENCLOSURE.ResourceID = dbo.v_R_System.ResourceID ORDER BY TimeStamp DESC) OASysEncl OUTER APPLY (SELECT TOP 1 IP_Addresses0, ROW_NUMBER() OVER (order by (SELECT 0)) AS rowNum FROM dbo.v_RA_System_IPAddresses WHERE dbo.v_RA_System_IPAddresses.ResourceID = dbo.v_R_System.ResourceID ORDER BY rowNum DESC) OAIP OUTER APPLY (SELECT TOP 1 MAC_Addresses0 FROM dbo.v_RA_System_MACAddresses WHERE dbo.v_RA_System_MACAddresses.ResourceID = dbo.v_R_System.ResourceID ) OAMac OUTER APPLY (SELECT TOP 1 MaxClockSpeed0, Name0 FROM dbo.v_GS_PROCESSOR WHERE dbo.v_GS_PROCESSOR.ResourceID = dbo.v_R_System.ResourceID ORDER BY TimeStamp DESC) OAProc LEFT JOIN dbo.v_GS_X86_PC_MEMORY ON dbo.v_GS_X86_PC_MEMORY.ResourceID = dbo.v_R_System.ResourceID LEFT JOIN dbo.v_GS_OPERATING_SYSTEM ON dbo.v_GS_OPERATING_SYSTEM.ResourceID = dbo.v_R_System.ResourceID LEFT JOIN dbo.v_GS_COMPUTER_SYSTEM ON dbo.v_GS_COMPUTER_SYSTEM.ResourceID = dbo.v_R_System.ResourceID LEFT JOIN dbo.v_GS_PC_BIOS ON dbo.v_GS_PC_BIOS.ResourceID = dbo.v_R_System.ResourceID LEFT JOIN dbo.v_GS_LOGICAL_DISK ON dbo.v_GS_LOGICAL_DISK.ResourceID = dbo.v_R_System.ResourceID LEFT JOIN dbo.v_FullCollectionMembership ON (dbo.v_FullCollectionMembership.ResourceID = v_R_System.ResourceID) LEFT JOIN dbo.v_GS_SYSTEM ON dbo.v_GS_SYSTEM.ResourceID = dbo.v_R_System.ResourceID WHERE dbo.v_GS_LOGICAL_DISK.DeviceID0 = 'C:' AND dbo.v_FullCollectionMembership.CollectionID = 'SMS00001' "
  },
  "AssetTypes": [
      {
          "AssetType": "Server",
          "OperationType": "Both",
          "PreserveShared": false,
          "Query": "AND OASysEncl.ChassisTypes0 IN (2, 17, 18, 19, 20, 21, 22, 23) AND dbo.v_R_System.Obsolete0 = 0 ORDER BY dbo.v_R_System.ResourceID ASC",
          "AssetIdentifier": {
              "DBColumn": "MachineName",
              "Entity": "Asset",
              "EntityColumn": "h_name"
          },
          "SoftwareInventory": {
              "AssetIDColumn": "AssetID",
              "AppIDColumn": "AppID",
              "Query": "SELECT AppID = CASE WHEN Publisher0 IS NULL AND Version0 IS NULL THEN DisplayName0 WHEN Publisher0 IS NOT NULL AND Version0 IS NULL THEN Publisher0+DisplayName0 ELSE Publisher0+DisplayName0+Version0 END, DisplayName0 , Version0, FCM.Name, convert(datetime, InstallDate0, 112) AS InstallDate0, Publisher0, ProdID0, FCM.ResourceID FROM v_Add_Remove_Programs AS ARP JOIN v_FullCollectionMembership As FCM on ARP.ResourceID=FCM.ResourceID WHERE FCM.CollectionID = 'SMS00001' AND FCM.ResourceID = '{{AssetID}}' AND DisplayName0 IS NOT NULL AND DisplayName0 != '' AND DisplayName0 NOT LIKE '%Update for Windows%' ORDER BY ProdID0 ASC",
              "Mapping": {
                  "h_app_id":"[AppID]",
                  "h_app_name": "[DisplayName0]",
                  "h_app_vendor":"[Publisher0]",
                  "h_app_version":"[Version0]",
                  "h_app_install_date":"[InstallDate0]",
                  "h_app_help":"",
                  "h_app_info":""
              }
          }
      },
      {
          "AssetType": "Virtual Machine",
          "OperationType": "Both",
          "PreserveShared": false,
          "Query": "AND OASysEncl.ChassisTypes0 = 1 AND dbo.v_R_System.Obsolete0 = 0 ORDER BY dbo.v_R_System.ResourceID ASC",
          "AssetIdentifier": {
              "DBColumn": "MachineName",
              "Entity": "Asset",
              "EntityColumn": "h_name"
          },
          "SoftwareInventory": {
              "AssetIDColumn": "AssetID",
              "AppIDColumn": "AppID",
              "Query": "SELECT AppID = CASE WHEN Publisher0 IS NULL AND Version0 IS NULL THEN DisplayName0 WHEN Publisher0 IS NOT NULL AND Version0 IS NULL THEN Publisher0+DisplayName0 ELSE Publisher0+DisplayName0+Version0 END, DisplayName0 , Version0, FCM.Name, convert(datetime, InstallDate0, 112) AS InstallDate0, Publisher0, ProdID0, FCM.ResourceID FROM v_Add_Remove_Programs AS ARP JOIN v_FullCollectionMembership As FCM on ARP.ResourceID=FCM.ResourceID WHERE FCM.CollectionID = 'SMS00001' AND FCM.ResourceID = '{{AssetID}}' AND DisplayName0 IS NOT NULL AND DisplayName0 != '' AND DisplayName0 NOT LIKE '%Update for Windows%' ORDER BY ProdID0 ASC",
              "Mapping": {
                  "h_app_id":"[AppID]",
                  "h_app_name": "[DisplayName0]",
                  "h_app_vendor":"[Publisher0]",
                  "h_app_version":"[Version0]",
                  "h_app_install_date":"[InstallDate0]",
                  "h_app_help":"",
                  "h_app_info":""
              }
          }
      },
      {
          "AssetType": "Laptop",
          "OperationType": "Both",
          "PreserveShared": false,
          "Query": "AND OASysEncl.ChassisTypes0 IN (8, 9, 10, 14) AND dbo.v_R_System.Obsolete0 = 0 ORDER BY dbo.v_R_System.ResourceID ASC",
          "AssetIdentifier": {
              "DBColumn": "MachineName",
              "Entity": "Asset",
              "EntityColumn": "h_name"
          },
          "SoftwareInventory": {
              "AssetIDColumn": "AssetID",
              "AppIDColumn": "AppID",
              "Query": "SELECT AppID = CASE WHEN Publisher0 IS NULL AND Version0 IS NULL THEN DisplayName0 WHEN Publisher0 IS NOT NULL AND Version0 IS NULL THEN Publisher0+DisplayName0 ELSE Publisher0+DisplayName0+Version0 END, DisplayName0 , Version0, FCM.Name, convert(datetime, InstallDate0, 112) AS InstallDate0, Publisher0, ProdID0, FCM.ResourceID FROM v_Add_Remove_Programs AS ARP JOIN v_FullCollectionMembership As FCM on ARP.ResourceID=FCM.ResourceID WHERE FCM.CollectionID = 'SMS00001' AND FCM.ResourceID = '{{AssetID}}' AND DisplayName0 IS NOT NULL AND DisplayName0 != '' AND DisplayName0 NOT LIKE '%Update for Windows%' ORDER BY ProdID0 ASC",
              "Mapping": {
                  "h_app_id":"[AppID]",
                  "h_app_name": "[DisplayName0]",
                  "h_app_vendor":"[Publisher0]",
                  "h_app_version":"[Version0]",
                  "h_app_install_date":"[InstallDate0]",
                  "h_app_help":"",
                  "h_app_info":""
              }
          }
      }
  ],
  "AssetGenericFieldMapping":{
      "h_name":"[MachineName]",
      "h_site":"[SiteName]",
      "h_asset_tag":"[MachineName]",
      "h_acq_method":"",
      "h_actual_retired_date":"",
      "h_beneficiary":"",
      "h_building":"",
      "h_cost":"",
      "h_cost_center":"",
      "h_country":"",
      "h_created_date":"",
      "h_deprec_method":"",
      "h_deprec_start":"",
      "h_description":"[MachineName] ([SystemModel])",
      "h_disposal_price":"",
      "h_disposal_reason":"",
      "h_floor":"",
      "h_geo_location":"",
      "h_invoice_number":"",
      "h_location":"",
      "h_location_type":"",
      "h_maintenance_cost":"",
      "h_maintenance_ref":"",
      "h_notes":"",
      "h_operational_state":"1",
      "h_order_date":"",
      "h_order_number":"",
      "h_owned_by":"[UserName]",
      "h_product_id":"",
      "h_received_date":"",
      "h_record_state": "1",
      "h_residual_value":"",
      "h_room":"",
      "h_scheduled_retire_date":"",
      "h_substate_id": "",
      "h_substate_name": "",
      "h_supplier_id":"",
      "h_supported_by":"",
      "h_used_by":"[UserName]",
      "h_version":"",
      "h_warranty_expires":"",
      "h_warranty_start":""
  },
  "AssetTypeFieldMapping":{
      "h_name":"[MachineName]",
      "h_mac_address":"[MACAddress]",
      "h_net_ip_address":"[IPAddress]",
      "h_net_computer_name":"[MachineName]",
      "h_net_win_domain":"[NETDomain]",
      "h_model":"[SystemModel]",
      "h_manufacturer":"[SystemManufacturer]",
      "h_cpu_info":"[ProcessorName]",
      "h_description":"[SystemModel]",
      "h_last_logged_on":"",
      "h_last_logged_on_user":"",
      "h_memory_info":"[MemoryKB]",
      "h_net_win_dom_role":"",
      "h_optical_drive":"",
      "h_os_description":"[OperatingSystem]",
      "h_os_registered_to":"",
      "h_os_serial_number":"",
      "h_os_service_pack":"[ServicePackVersion]",
      "h_os_type":"",
      "h_os_version":"[OperatingSystemVersion]",
      "h_physical_disk_size":"[DiskSpaceMB]",
      "h_serial_number":"[SystemSerialNumber]",
      "h_cpu_clock_speed":"[ProcessorSpeedGHz]",
      "h_physical_cpus":"[NumberofProcessors]",
      "h_logical_cpus":"",
      "h_bios_name":"[BIOSDescription]",
      "h_bios_manufacturer":"",
      "h_bios_serial_number":"",
      "h_bios_release_date":"[BIOSReleaseDate]",
      "h_bios_version":"[SMBIOSVersion]",
      "h_max_memory_capacity":"",
      "h_number_memory_slots":"",
      "h_net_name":"",
      "h_subnet_mask":""
  }
}
```

#### InstanceConfig

- "APIKey" - a Hornbill API key for a user account with the correct permissions to carry out all of the required API calls
- "InstanceId" - Instance Id
- "LogSizeBytes" - The maximum size that the generated Log Files should be, in bytes. Setting this value to 0 will cause the tool to create one log file only and not split the results between multiple logs.

#### SQLConf

- "Driver" the driver to use to connect to the database that holds the asset information:
  - mssql = Microsoft SQL Server (2005 or above)
  - mysql = MySQL Server 4.1+, MariaDB
  - mysql320 = MySQL Server v3.2.0 to v4.0
  - swsql = Supportworks SQL (Core Services v3.x)
  - odbc = ODBC Data Source using SQL Server driver
    - When using ODBC as a data source, the `Database`, `UserName`, `Password` and `Query` parameters should be populated accordingly:
      - Database - this should be populated with the  name of the ODBC connection on the PC that is running the tool
      - UserName - this should be the SQL authentication Username to connect to the Database
      - Password - this should be the password for the above username
      - Query - this should be the SQL query to retrieve the asset records
- "Server" The address of the SQL server
- "Database" The name of the Database to connect to
- "Authentication" - The tupe of authentication to use to connect to the SQL server. Can be either:
  - Windows - Windows Account authentication, uses the logged-in Windows account to authenticate
  - SQL - uses SQL Server authentication, and requires the Username and Password parameters (below) to be populated
- "UserName" The username for the SQL database - only used when Authentication is set to SQL: for Windows authentication this field can be left as an empty string
- "Password" Password for above User Name - only used when Authentication is set to SQL: for Windows authentication this field can be left as an empty string
- "Port" SQL port
- "Encrypt" Boolean value to specify wether the connection between the script and the database should be encrypted. ''NOTE'': There is a bug in SQL Server 2008 and below that causes the connection to fail if the connection is encrypted. Only set this to true if your SQL Server has been patched accordingly.
- "Query" The basic SQL query to retrieve asset information from the data source. See "AssetTypes below for further filtering

#### AssetTypes

- An array of objects details the asset types to import:
  - AssetType - the Asset Type Name which needs to match a correct Asset Type Name in your Hornbill Instance
  - OperationType - The type of operation that should be performed on discovered assets - can be Create, Update or Both. Defaults to Both if no value is provided
  - PreserveShared - If set to true, when updating assets that are Shared, then the Used By fields will not be updated. Defaults to false
  - PreserveState - If set to true then the State field will not be updated. Defaults to false
  - PreserveSubState - If set to true then the SubState fields will not be updated. Defaults to false
  - PreserveOperationalState - If set to true then the Operational State field will not be updated. Defaults to false
  - Query - additional SQL filter to be appended to the Query from SQLConf, to retrieve assets of that asset type.
  - AssetIdentifier - an object containing details to help in the identification of existing asset records in the Hornbill instance. If value in an imported records DBColumn matches the value in the EntityColumn of an asset in Hornbill (within the defined Entity), then the asset record will be updated rather than a new asset being created:
    - DBColumn - specifies the unique identifier column from the database query
    - Entity - the Hornbill entity where data is stored
    - EntityColumn - specifies the unique identifier column from the Hornbill entity
  - SoftwareInventory - an object containing details pertaining to the import of software inventory records for the specified asset type:
    - AssetIDColumn - the column from the asset type query that contains its primary key
    - AppIDColumn - the column from the Software Inventory that holds the software unique ID
    - Query - the query that will be run per asset, to return its software invemtory records. {{AssetID}} in the query will be replaced by each assets primary key value, whose column is defined in the AssetIDColumn property
    - Mapping - maps data into the software invemtory records   

#### AssetGenericFieldMapping

- Maps data in to the generic Asset record
- Any value wrapped with [] will be populated with the corresponding response from the SQL Query
- Providing a value of `__clear__` will NULL that column for the record in the database when assets are being updated ONLY. This can either be hard-coded in the config, or sent as a string column within the SQL query resultset (`SELECT '__clear__' AS clearColumn` in the query and `[clearColumn]` in the mapping for example)
- Any Other Value is treated literally as written example:
  - "h_name":"[MachineName]", - the value of MachineName is taken from the SQL output and populated within this field
  - "h_description":"This is a description", - the value of "h_description" would be populated with "This is a description" for ALL imported assets
  - "h_site":"[SiteName]", - When a string is passed to the h_site field, the script attempts to resolve the given site name against the Site entity, and populates this (and h_site_id) with the correct site information. If the site cannot be resolved, the site details are not populated for the Asset record being imported.
  - "h_owned_by":"[UserName]" - when a valid Hornbill User ID (for a Full or Basic User) is passed to this field, the user is verified on your Hornbill instance, and the tool will complete the h_owned_by and h_owned_by_name columns appropriately.
  - "h_used_by":"[UserName]" - when a valid Hornbill User ID (for a Full or Basic User) is passed to this field, the user is verified on your Hornbill instance, and the tool will complete the h_used_by and h_used_by_name columns appropriately.
  - "h_company_name":"[CompanyName]" - when a valid Hornbill Company group name is passed to this field, the company is verified on your Hornbill instance, and the tool will complete the h_company_id and h_company_name columns appropriately.

#### AssetTypeFieldMapping

- Maps data in to the type-specific Asset record, so the same rules as AssetGenericFieldMapping
- For the computer asset class:
  - "h_last_logged_on_user":"[UserName]" - when a valid Hornbill User ID (for a Full or Basic User) is passed to this field, the user is verified on your Hornbill instance, and the tool will complete the h_last_logged_on_user column with an appropriate URN value for the user.

## Execute

Command Line Parameters

- file - Defaults to `conf.json` - Name of the Configuration file to load
- dryrun - Defaults to `false` - Set to True and the XMLMC for Create and Update assets will not be called and instead the XML will be dumped to the log file, this is to aid in debugging the initial connection information.
- concurrent - defaults to `1`. This is to specify the number of assets that should be imported concurrently, and can be an integer between 1 and 10 (inclusive). 1 is the slowest level of import, but does not affect performance of your Hornbill instance, and 10 will process the import much more quickly but could affect instance performance while the import is running.
- debug - defaults to `false` = Set to true to enable debug mode, which will output debugging information to the log

## Testing

If you run the application with the argument dryrun=true then no assets will be created or updated, the XML used to create or update will be saved in the log file so you can ensure the data mappings are correct before running the import.

'goDBAssetImport.exe -dryrun=true'

## Scheduling

### Windows

You can schedule goDBAssetImport.exe to run with any optional command line argument from Windows Task Scheduler:

- Ensure the user account running the task has rights to goDBAssetImport.exe and the containing folder.
- Make sure the Start In parameter contains the folder where goDBAssetImport.exe resides in otherwise it will not be able to pick up the correct path.

## Logging

All Logging output is saved in the log directory in the same directory as the executable the file name contains the date and time the import was run 'Asset_Import_2015-11-06T14-26-13Z.log'

## Error Codes

- `100` - Unable to create log File
- `101` - Unable to create log folder
- `102` - Unable to Load Configuration File
