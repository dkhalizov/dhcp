package protocol

import (
	"net"
	"time"
)

type OptionInfo struct {
	Name       string
	DataLength string
	Meaning    string
}

const (
	// Commonly used DHCP options
	OptionSubnetMask           byte = 1
	OptionRouter                    = 3
	OptionDomainNameServer          = 6
	OptionHostname                  = 12
	OptionDomainName                = 15
	OptionBroadcastAddress          = 28
	OptionNetworkTimeProtocol       = 42
	OptionVendorSpecific            = 43
	OptionRequestedIPAddress        = 50
	OptionIPAddressLeaseTime        = 51
	OptionDHCPMessageType           = 53
	OptionServerIdentifier          = 54
	OptionParameterRequestList      = 55
	OptionRenewalTime               = 58
	OptionRebindingTime             = 59
	OptionClassIdentifier           = 60
	OptionClientIdentifier          = 61
	OptionTFTPServerName            = 66
	OptionBootfileName              = 67
	OptionUserClass                 = 77
	OptionClientFQDN                = 81
	OptionDHCPAgentOptions          = 82
	OptionDomainSearch              = 119
	OptionClasslessStaticRoute      = 121
	OptionEnd                       = 255
)

var DHCPOptions = map[byte]OptionInfo{
	0:   {"Pad", "0", "None"},
	1:   {"Subnet Mask", "4", "Subnet Mask Value"}, // Server needs to provide the correct subnet mask
	2:   {"Time Offset", "4", "Time Offset in Seconds from UTC	(note: deprecated by 100 and 101)"},
	3:   {"Router", "N", "N/4 Router addresses"},                 // Server needs to provide router addresses
	4:   {"Time Server", "N", "N/4 Timeserver addresses"},        // Server needs to provide time server addresses
	5:   {"Name Server", "N", "N/4 IEN-116 Server addresses"},    // Server needs to provide name server addresses
	6:   {"Domain Server", "N", "N/4 DNS Server addresses"},      // Server needs to provide DNS server addresses
	7:   {"Log Server", "N", "N/4 Logging Server addresses"},     // Server needs to provide log server addresses
	8:   {"Quotes Server", "N", "N/4 Quotes Server addresses"},   // Server needs to provide quotes server addresses
	9:   {"LPR Server", "N", "N/4 Printer Server addresses"},     // Server needs to provide printer server addresses
	10:  {"Impress Server", "N", "N/4 Impress Server addresses"}, // Server needs to provide Impress server addresses
	11:  {"RLP Server", "N", "N/4 RLP Server addresses"},         // Server needs to provide RLP server addresses
	12:  {"Hostname", "N", "Hostname string"},
	13:  {"Boot File Size", "2", "Size of boot file in 512 byte chunks"},
	14:  {"Merit Dump File", "N", "Client to dump and name the file to dump it to"},
	15:  {"Domain Name", "N", "The DNS domain name of the client"}, // Server needs to provide the domain name
	16:  {"Swap Server", "N", "Swap Server address"},               // Server needs to provide swap server address
	17:  {"Root Path", "N", "Path name for root disk"},
	18:  {"Extension File", "N", "Path name for more BOOTP info"},
	19:  {"Forward On/Off", "1", "Enable/Disable IP Forwarding"},
	20:  {"SrcRte On/Off", "1", "Enable/Disable Source Routing"},
	21:  {"Policy Filter", "N", "Routing Policy Filters"},
	22:  {"Max DG Assembly", "2", "Max Datagram Reassembly Size"},
	23:  {"Default IP TTL", "1", "Default IP Time to Live"},
	24:  {"MTU Timeout", "4", "Path MTU Aging Timeout"},
	25:  {"MTU Plateau", "N", "Path MTU  Plateau Table"},
	26:  {"MTU Interface", "2", "Interface MTU Size"},
	27:  {"MTU Subnet", "1", "All Subnets are Local"},
	28:  {"Broadcast Address", "4", "Broadcast Address"}, // Server needs to provide the broadcast address
	29:  {"Mask Discovery", "1", "Perform Mask Discovery"},
	30:  {"Mask Supplier", "1", "Provide Mask to Others"},
	31:  {"Router Discovery", "1", "Perform Router Discovery"},
	32:  {"Router Request", "4", "Router Solicitation Address"},
	33:  {"Static Route", "N", "Static Routing Table"}, // Server needs to provide static routes if used
	34:  {"Trailers", "1", "Trailer Encapsulation"},
	35:  {"ARP Timeout", "4", "ARP Cache Timeout"},
	36:  {"Ethernet", "1", "Ethernet Encapsulation"},
	37:  {"Default TCP TTL", "1", "Default TCP Time to Live"},
	38:  {"Keepalive Time", "4", "TCP Keepalive Interval"},
	39:  {"Keepalive Data", "1", "TCP Keepalive Garbage"},
	40:  {"NIS Domain", "N", "NIS Domain Name"},                     // Server needs to provide NIS domain if used
	41:  {"NIS Servers", "N", "NIS Server Addresses"},               // Server needs to provide NIS server addresses if used
	42:  {"NTP Servers", "N", "NTP Server Addresses"},               // Server needs to provide NTP server addresses
	43:  {"Vendor Specific", "N", "Vendor Specific Information"},    // Server may need to provide vendor-specific information
	44:  {"NETBIOS Name Srv", "N", "NETBIOS Name Servers"},          // Server needs to provide NetBIOS name servers if used
	45:  {"NETBIOS Dist Srv", "N", "NETBIOS Datagram Distribution"}, // Server needs to provide NetBIOS datagram distribution servers if used
	46:  {"NETBIOS Node Type", "1", "NETBIOS Node Type"},
	47:  {"NETBIOS Scope", "N", "NETBIOS Scope"},
	48:  {"X Window Font", "N", "X Window Font Server"},        // Server needs to provide X Window font servers if used
	49:  {"X Window Manager", "N", "X Window Display Manager"}, // Server needs to provide X Window display managers if used
	50:  {"Address Request", "4", "Requested IP Address"},
	51:  {"Address Time", "4", "IP Address Lease Time"}, // Server needs to specify the lease time
	52:  {"Overload", "1", "Overload \"sname\" or \"file\""},
	53:  {"DHCP Msg Type", "1", "DHCP Message Type"},
	54:  {"DHCP Server Id", "4", "DHCP Server Identification"}, // Server needs to provide its identifier
	55:  {"Parameter List", "N", "Parameter Request List"},
	56:  {"DHCP Message", "N", "DHCP Error Message"},
	57:  {"DHCP Max Msg Size", "2", "DHCP Maximum Message Size"},
	58:  {"Renewal Time", "4", "DHCP Renewal (T1) Time"},     // Server needs to specify the renewal time
	59:  {"Rebinding Time", "4", "DHCP Rebinding (T2) Time"}, // Server needs to specify the rebinding time
	60:  {"Class Id", "N", "Class Identifier"},
	61:  {"Client Id", "N", "Client Identifier"},
	62:  {"NetWare/IP Domain", "N", "NetWare/IP Domain Name"},      // Server needs to provide NetWare/IP domain if used
	63:  {"NetWare/IP Option", "N", "NetWare/IP sub Options"},      // Server needs to provide NetWare/IP options if used
	64:  {"NIS-Domain-Name", "N", "NIS+ v3 Client Domain Name"},    // Server needs to provide NIS+ domain if used
	65:  {"NIS-Server-Addr", "N", "NIS+ v3 Server Addresses"},      // Server needs to provide NIS+ server addresses if used
	66:  {"Server-Name", "N", "TFTP Server Name"},                  // Server needs to provide TFTP server name if used
	67:  {"Bootfile-Name", "N", "Boot File Name"},                  // Server needs to provide boot file name if used
	68:  {"Home-Agent-Addrs", "N", "Home Agent Addresses"},         // Server needs to provide home agent addresses if used
	69:  {"SMTP-Server", "N", "Simple Mail Server Addresses"},      // Server needs to provide SMTP server addresses if used
	70:  {"POP3-Server", "N", "Post Office Server Addresses"},      // Server needs to provide POP3 server addresses if used
	71:  {"NNTP-Server", "N", "Network News Server Addresses"},     // Server needs to provide NNTP server addresses if used
	72:  {"WWW-Server", "N", "WWW Server Addresses"},               // Server needs to provide WWW server addresses if used
	73:  {"Finger-Server", "N", "Finger Server Addresses"},         // Server needs to provide Finger server addresses if used
	74:  {"IRC-Server", "N", "Chat Server Addresses"},              // Server needs to provide IRC server addresses if used
	75:  {"StreetTalk-Server", "N", "StreetTalk Server Addresses"}, // Server needs to provide StreetTalk server addresses if used
	76:  {"STDA-Server", "N", "ST Directory Assist. Addresses"},    // Server needs to provide STDA server addresses if used
	77:  {"User-Class", "N", "User Class Information"},
	78:  {"Directory Agent", "N", "directory agent information"}, // Server needs to provide directory agent information if used
	79:  {"Service Scope", "N", "service location agent scope"},
	80:  {"Rapid Commit", "0", "Rapid Commit"},
	81:  {"Client FQDN", "N", "Fully Qualified Domain Name"},
	82:  {"Relay Agent Information", "N", "Relay Agent Information"},
	83:  {"iSNS", "N", "Internet Storage Name Service"}, // Server needs to provide iSNS information if used
	84:  {"REMOVED/Unassigned", "", ""},
	85:  {"NDS Servers", "N", "Novell Directory Services"},   // Server needs to provide NDS server addresses if used
	86:  {"NDS Tree Name", "N", "Novell Directory Services"}, // Server needs to provide NDS tree name if used
	87:  {"NDS Context", "N", "Novell Directory Services"},   // Server needs to provide NDS context if used
	88:  {"BCMCS Controller Domain Name list", "", ""},       // Server needs to provide BCMCS controller domain names if used
	89:  {"BCMCS Controller IPv4 address option", "", ""},    // Server needs to provide BCMCS controller IPv4 addresses if used
	90:  {"Authentication", "N", "Authentication"},
	91:  {"client-last-transaction-time option", "", ""},
	92:  {"associated-ip option", "", ""},
	93:  {"Client System", "N", "Client System Architecture"},
	94:  {"Client NDI", "N", "Client Network Device Interface"},
	95:  {"LDAP", "N", "Lightweight Directory Access Protocol"}, // Server needs to provide LDAP server information if used
	96:  {"REMOVED/Unassigned", "", ""},
	97:  {"UUID/GUID", "N", "UUID/GUID-based Client Identifier"},
	98:  {"User-Auth", "N", "Open Group's User Authentication"},
	99:  {"GEOCONF_CIVIC", "", ""}, // Server needs to provide civic location information if used
	100: {"PCode", "N", "IEEE 1003.1 TZ String"},
	101: {"TCode", "N", "Reference to the TZ Database"},
	108: {"IPv6-Only Preferred", "4", "Number of seconds that DHCPv4 should be disabled"},
	109: {"OPTION_DHCP4O6_S46_SADDR", "16", "DHCPv4 over DHCPv6 Softwire Source Address Option"},
	110: {"REMOVED/Unassigned", "", ""},
	111: {"Unassigned", "", ""},
	112: {"Netinfo Address", "N", "NetInfo Parent Server Address"}, // Server needs to provide NetInfo server address if used
	113: {"Netinfo Tag", "N", "NetInfo Parent Server Tag"},         // Server needs to provide NetInfo server tag if used
	114: {"DHCP Captive-Portal", "N", "DHCP Captive-Portal"},       // Server needs to provide captive portal information if used
	115: {"REMOVED/Unassigned", "", ""},
	116: {"Auto-Config", "N", "DHCP Auto-Configuration"},
	117: {"Name Service Search", "N", "Name Service Search"}, // Server needs to provide name service search order if used
	118: {"Subnet Selection Option", "4", "Subnet Selection Option"},
	119: {"Domain Search", "N", "DNS domain search list"},                        // Server needs to provide DNS search list if used
	120: {"SIP Servers DHCP Option", "N", "SIP Servers DHCP Option"},             // Server needs to provide SIP server information if used
	121: {"Classless Static Route Option", "N", "Classless Static Route Option"}, // Server needs to provide classless static routes if used
	122: {"CCC", "N", "CableLabs Client Configuration"},                          // Server needs to provide CableLabs client configuration if used
	123: {"GeoConf Option", "16", "GeoConf Option"},                              // Server needs to provide GeoConf information if used
	124: {"V-I Vendor Class", "", "Vendor-Identifying Vendor Class"},
	125: {"V-I Vendor-Specific Information", "", "Vendor-Identifying Vendor-Specific Information"}, // Server may need to provide vendor-specific information
	126: {"Removed/Unassigned", "", ""},
	127: {"Removed/Unassigned", "", ""},
	128: {"PXE - undefined (vendor specific)", "", ""},      // Server may need to provide PXE-specific information
	129: {"Kernel options. Variable length	string", "", ""}, // Server may need to provide kernel options for PXE clients
	130: {"Discrimination string (to identify vendor)", "", ""},
	131: {"PXE - undefined (vendor specific)", "", ""},                                                                    // Server may need to provide PXE-specific information
	132: {"PXE - undefined (vendor specific)", "", ""},                                                                    // Server may need to provide PXE-specific information
	133: {"PXE - undefined (vendor specific)", "", ""},                                                                    // Server may need to provide PXE-specific information
	134: {"PXE - undefined (vendor specific)", "", ""},                                                                    // Server may need to provide PXE-specific information
	135: {"PXE - undefined (vendor specific)", "", ""},                                                                    // Server may need to provide PXE-specific information
	136: {"OPTION_PANA_AGENT", "", ""},                                                                                    // Server needs to provide PANA Authentication Agent addresses if used
	137: {"OPTION_V4_LOST", "", ""},                                                                                       // Server needs to provide LoST server information if used
	138: {"OPTION_CAPWAP_AC_V4", "N", "CAPWAP Access Controller addresses"},                                               // Server needs to provide CAPWAP AC addresses if used
	139: {"OPTION-IPv4_Address-MoS", "N", "a series of suboptions"},                                                       // Server needs to provide MoS IPv4 addresses if used
	140: {"OPTION-IPv4_FQDN-MoS", "N", "a series of suboptions"},                                                          // Server needs to provide MoS domain names if used
	141: {"SIP UA Configuration Service Domains", "N", "List of domain names to search for SIP User Agent Configuration"}, // Server needs to provide SIP UA configuration domains if used
	142: {"OPTION-IPv4_Address-ANDSF", "N", "ANDSF IPv4 Address Option for DHCPv4"},                                       // Server needs to provide ANDSF addresses if used
	143: {"OPTION_V4_SZTP_REDIRECT", "N", "This option provides a list of URIs for SZTP bootstrap servers"},               // Server needs to provide SZTP bootstrap server URIs if used
	144: {"GeoLoc", "16", "Geospatial Location with Uncertainty"},                                                         // Server needs to provide geolocation information if used
	145: {"FORCERENEW_NONCE_CAPABLE", "1", "Forcerenew Nonce Capable"},
	146: {"RDNSS Selection", "N", "Information for selecting RDNSS"},                                            // Server needs to provide RDNSS selection information if used
	147: {"OPTION_V4_DOTS_RI", "N", "The name of the peer DOTS agent."},                                         // Server needs to provide DOTS agent information if used
	148: {"OPTION_V4_DOTS_ADDRESS", "N (the minimal length is 4)", "N/4 IPv4 addresses of peer DOTS agent(s)."}, // Server needs to provide DOTS agent addresses if used
	149: {"Unassigned", "", ""},
	150: {"GRUB configuration path name", "", ""}, // Server needs to provide GRUB configuration path if PXE boot is used
	151: {"status-code", "N+1", "Status code and optional N byte text message describing status."},
	152: {"base-time", "4", "Absolute time (seconds since Jan 1, 1970) message was sent."},
	153: {"start-time-of-state", "4", "Number of seconds in the past when client entered current state."},
	154: {"query-start-time", "4", "Absolute time (seconds since Jan 1, 1970) for beginning of query."},
	155: {"query-end-time", "4", "Absolute time (seconds since Jan 1, 1970) for end of query."},
	156: {"dhcp-state", "1", "State of IP address."},
	157: {"data-source", "1", "Indicates information came from local or remote server."},
	158: {"OPTION_V4_PCP_SERVER", "Variable; the minimum length is 5.", "Includes one or multiple lists of PCP server IP addresses; each list is treated as a separate PCP server."}, // Server needs to provide PCP server addresses if used
	159: {"OPTION_V4_PORTPARAMS", "4", "This option is used to configure a set of ports bound to a shared IPv4 address."},
	160: {"Unassigned", "", "Previously assigned by [RFC7710]; known to also be used by Polycom."},
	161: {"OPTION_MUD_URL_V4", "N (variable)", "Manufacturer Usage Descriptions"}, // Server may need to provide MUD URL if used
	162: {"OPTION_V4_DNR", "N", "Encrypted DNS Server"},                           // Server needs to provide encrypted DNS server information if used
	175: {"Etherboot (Tentatively Assigned - 2005-06-23)", "", ""},                // Server may need to provide Etherboot-specific information
	176: {"IP Telephone (Tentatively Assigned - 2005-06-23)", "", ""},             // Server may need to provide IP Telephone-specific information
	177: {"Etherboot (Tentatively Assigned - 2005-06-23)", "", ""},                // Server may need to provide Etherboot-specific information
	208: {"PXELINUX Magic", "4", "magic string = F1:00:74:7E"},                    // Server needs to provide PXELinux magic string for PXE clients
	209: {"Configuration File", "N", "Configuration file"},                        // Server needs to provide configuration file information for PXE clients
	210: {"Path Prefix", "N", "Path Prefix Option"},                               // Server needs to provide path prefix for PXE clients
	211: {"Reboot Time", "4", "Reboot Time"},                                      // Server may need to provide reboot time for PXE clients
	212: {"OPTION_6RD", "18 + N", "OPTION_6RD with N/4 6rd BR addresses"},         // Server needs to provide 6RD configuration if used
	213: {"OPTION_V4_ACCESS_DOMAIN", "N", "Access Network Domain Name"},           // Server needs to provide access network domain name if used
	220: {"Subnet Allocation Option", "N", "Subnet Allocation Option"},            // Server needs to handle subnet allocation if used
	221: {"Virtual Subnet Selection (VSS) Option", "", ""},                        // Server needs to handle virtual subnet selection if used
	255: {"End", "0", "None"},
}

type ReplyOptions struct {
	LeaseTime     time.Duration
	RenewalTime   time.Duration
	RebindingTime time.Duration
	SubnetMask    net.IPMask
	Router        net.IP
	DNS           []net.IP
	ServerIP      net.IP
	DomainName    string
}
