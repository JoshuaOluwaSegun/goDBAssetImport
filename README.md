### SQL Asset Import Go - [GO](https://golang.org/) Import Script to Hornbill

### Quick links
- [Installation](#installation)
- [Config](#config)
    - [Instance Config](#InstanceConfig)
    - [SQL Config](#SQLConf)
    - [Asset Types](#AssetTypes)
    - [Asset Field Mapping (Generic)](#AssetGenericFieldMapping)
    - [Asset Field Mapping (Type Specific)](#AssetTypeFieldMapping)
- [Execute](#execute)
- [Testing](testing)
- [Scheduling](#scheduling)
- [Logging](#logging)
- [Error Codes](#error codes)
- [Change Log](#change log)

# Installation

#### Windows
* Download the archive containing the import executable
* Extract zip into a folder you would like the application to run from e.g. `C:\asset_import\`
* Open '''conf_sccm_assetscomputer.json''' and add in the necessary configration
* Open Command Line Prompt as Administrator
* Change Directory to the folder with db_asset_import.exe `C:\asset_import\`
* Run the command db_asset_import.exe -dryrun=true -file=conf_sccm_assetscomputer.json

# config

Example JSON File:

```json
{
  "UserName": "",
  "Password": "",
  "InstanceID": "",
  "AssetIdentifier":"h_name",
  "SQLConf": {
      "Driver": "mssql",
      "Server": "",
      "Database": "",
      "UserName": "",
      "Password": "",
      "Port": 1433,
      "AssetID": "MachineName",
      "Encrypt": false,
      "Query": "SELECT OARSys.ResourceID AS [AssetID], OARSys.User_Name0 AS [UserName], OARSys.Netbios_Name0 AS [MachineName], OARSys.Resource_Domain_OR_Workgr0 AS [NETDomain], dbo.v_GS_OPERATING_SYSTEM.Caption0 AS [OperatingSystemCaption], OARSys.Operating_System_Name_and0 AS [OperatingSystem], dbo.v_GS_OPERATING_SYSTEM.Version0 AS [OperatingSystemVersion], dbo.v_GS_OPERATING_SYSTEM.CSDVersion0 AS [ServicePackVersion], dbo.v_GS_COMPUTER_SYSTEM.Manufacturer0 AS [SystemManufacturer], dbo.v_GS_COMPUTER_SYSTEM.Model0 AS [SystemModel], dbo.v_GS_PC_BIOS.SerialNumber0 AS [SystemSerialNumber], OAProc.MaxClockSpeed0 AS [ProcessorSpeedGHz], OAProc.Name0 AS [ProcessorName], dbo.v_GS_COMPUTER_SYSTEM.NumberOfProcessors0 AS [NumberofProcessors], dbo.v_GS_X86_PC_MEMORY.TotalPhysicalMemory0 AS [MemoryKB], dbo.v_GS_LOGICAL_DISK.Size0 AS [DiskSpaceMB], dbo.v_GS_LOGICAL_DISK.FreeSpace0 AS [FreeDiskSpaceMB], OAIP.IP_Addresses0 AS [IPAddress], OAMac.MAC_Addresses0 AS [MACAddress], dbo.v_GS_PC_BIOS.Description0 AS [BIOSDescription], dbo.v_GS_PC_BIOS.ReleaseDate0 AS [BIOSReleaseDate], dbo.v_GS_PC_BIOS.SMBIOSBIOSVersion0 AS [SMBIOSVersion], dbo.v_GS_SYSTEM.SystemRole0 AS [SystemType], OASysEncl.ChassisTypes0 AS [ChassisTypes], OASysEncl.TimeStamp AS [ChassisDate], OARSys.AD_Site_Name0 AS [SiteName] FROM dbo.v_R_System OUTER APPLY (SELECT TOP 1 * FROM dbo.v_R_System b WHERE b.Netbios_Name0 = dbo.v_R_System.Netbios_Name0 ORDER BY SMS_UUID_Change_Date0 DESC) OARSys OUTER APPLY (SELECT TOP 1 dbo.v_GS_SYSTEM_ENCLOSURE.* FROM dbo.v_GS_SYSTEM_ENCLOSURE WHERE dbo.v_GS_SYSTEM_ENCLOSURE.ResourceID = dbo.v_R_System.ResourceID ORDER BY TimeStamp DESC) OASysEncl OUTER APPLY (SELECT TOP 1 IP_Addresses0, ROW_NUMBER() OVER (order by (SELECT 0)) AS rowNum FROM dbo.v_RA_System_IPAddresses WHERE dbo.v_RA_System_IPAddresses.ResourceID = dbo.v_R_System.ResourceID ORDER BY rowNum DESC) OAIP OUTER APPLY (SELECT TOP 1 MAC_Addresses0 FROM dbo.v_RA_System_MACAddresses WHERE dbo.v_RA_System_MACAddresses.ResourceID = dbo.v_R_System.ResourceID ) OAMac OUTER APPLY (SELECT TOP 1 MaxClockSpeed0, Name0 FROM dbo.v_GS_PROCESSOR WHERE dbo.v_GS_PROCESSOR.ResourceID = dbo.v_R_System.ResourceID ORDER BY TimeStamp DESC) OAProc LEFT JOIN dbo.v_GS_X86_PC_MEMORY ON dbo.v_GS_X86_PC_MEMORY.ResourceID = dbo.v_R_System.ResourceID LEFT JOIN dbo.v_GS_OPERATING_SYSTEM ON dbo.v_GS_OPERATING_SYSTEM.ResourceID = dbo.v_R_System.ResourceID LEFT JOIN dbo.v_GS_COMPUTER_SYSTEM ON dbo.v_GS_COMPUTER_SYSTEM.ResourceID = dbo.v_R_System.ResourceID LEFT JOIN dbo.v_GS_PC_BIOS ON dbo.v_GS_PC_BIOS.ResourceID = dbo.v_R_System.ResourceID LEFT JOIN dbo.v_GS_LOGICAL_DISK ON dbo.v_GS_LOGICAL_DISK.ResourceID = dbo.v_R_System.ResourceID LEFT JOIN dbo.v_FullCollectionMembership ON (dbo.v_FullCollectionMembership.ResourceID = v_R_System.ResourceID) LEFT JOIN dbo.v_GS_SYSTEM ON dbo.v_GS_SYSTEM.ResourceID = dbo.v_R_System.ResourceID WHERE dbo.v_GS_LOGICAL_DISK.DeviceID0 = 'C:' AND dbo.v_FullCollectionMembership.CollectionID = 'SMS00001' "
  },
  "AssetTypes": {
      "Server": "AND OASysEncl.ChassisTypes0 IN (2, 17, 18, 19, 20, 21, 22, 23)",
      "Laptop": "AND OASysEncl.ChassisTypes0 IN (8, 9, 10, 14)",
      "Desktop": "AND OASysEncl.ChassisTypes0 IN (3, 4, 5, 6, 7, 12, 13, 15, 16, 17)",
      "Virtual Machine":"AND OASysEncl.ChassisTypes0 = 1"
  },
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
      "h_operational_state":"",
      "h_order_date":"",
      "h_order_number":"",
      "h_owned_by":"",
      "h_owned_by_name":"",
      "h_product_id":"",
      "h_received_date":"",
      "h_residual_value":"",
      "h_room":"",
      "h_scheduled_retire_date":"",
      "h_supplier_id":"",
      "h_supported_by":"",
      "h_used_by":"",
      "h_used_by_name":"",
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
* "UserName" - Instance User Name with Create / Update User Rights
* "Password" - Instance Password for the above User
* "InstanceId" - Instance Id
* "Entity" - The extended details entity for the assets being imported (for example, AssetsComputer for all Computer/VM type assets)
* "AssetIdentifier" - The asset attribute that holds the unique asset identifier (so that the code can work out which asset records are to be inserted or updated)

#### SQLConf
* "Driver" the driver to use to connect to the database that holds the asset information:
** mssql = Microsoft SQL Server (2005 or above)
** mysql = MySQL Server 4.1+, MariaDB
** mysql320 = MySQL Server v3.2.0 to v4.0
** swsql = Supportworks SQL (Core Services v3.x)
* "Server" The address of the SQL server
* "UserName" The username for the SQL database
* "Password" Password for above User Name
* "Port" SQL port
* "AssetID" Specifies the unique identifier field from the query below
* "Encrypt" Boolean value to specify wether the connection between the script and the database should be encrypted. ''NOTE'': There is a bug in SQL Server 2008 and below that causes the connection to fail if the connection is encrypted. Only set this to true if your SQL Server has been patched accordingly.
* "Query" The basic SQL query to retrieve asset information from the data source. See "AssetTypes below for further filtering

#### AssetTypes
* The left element contains the Asset Type Name, and the right contains the additional SQL filter to be appended to the Query from SQLConf, to retrieve assets of that asset type. Note: the Asset Type Name needs to match a correct Asset Type Name in your Hornbill Instance.

#### AssetGenericFieldMapping 
* Maps data in to the generic Asset record
* Any value wrapped with [] will be populated with the corresponding response from the SQL Query
* Any Other Value is treated literally as written example:
    * "h_name":"[MachineName]", - the value of MachineName is taken from the SQL output and populated within this field
    * "h_description":"This is a description", - the value of "h_description" would be populated with "This is a description" for ALL imported assets
	* "h_site":"[SiteName]", - When a string is passed to the h_site field, the script attempts to resolve the given site name against the Site entity, and populates this (and h_site_id) with the correct site information. If the site cannot be resolved, the site details are not populated for the Asset record being imported.

#### AssetTypeFieldMapping 
* Maps data in to the type-specific Asset record, so the same rules as AssetGenericFieldMapping

# Execute
Command Line Parameters
* file - Defaults to `conf.json` - Name of the Configuration file to load
* dryrun - Defaults to `false` - Set to True and the XMLMC for Create and Update assets will not be called and instead the XML will be dumped to the log file, this is to aid in debugging the initial connection information.
* zone - Defaults to `eur` - Allows you to change the ZONE used for creating the XMLMC EndPoint URL https://{ZONE}api.hornbill.com/{INSTANCE}/

# Testing
If you run the application with the argument dryrun=true then no assets will be created or updated, the XML used to create or update will be saved in the log file so you can ensure the data mappings are correct before running the import.

'db_asset_import.exe -dryrun=true'

# Scheduling

### Windows
You can schedule db_asset_import.exe to run with any optional command line argument from Windows Task Scheduler.
* Ensure the user account running the task has rights to db_asset_import.exe and the containing folder.
* Make sure the Start In parameter contains the folder where db_asset_import.exe resides in otherwise it will not be able to pick up the correct path.

# Logging
All Logging output is saved in the log directory in the same directory as the executable the file name contains the date and time the import was run 'Asset_Import_2015-11-06T14-26-13Z.log'

# Error Codes
* `100` - Unable to create log File
* `101` - Unable to create log folder
* `102` - Unable to Load Configuration File
