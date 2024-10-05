package pkg

const (
	// DHCP Message Types
	DHCPDISCOVER = 1
	DHCPREQUEST  = 3
	DHCPDECLINE  = 4
	DHCPRELEASE  = 7
	DHCPINFORM   = 8

	// Hadware Types
	ETHERNET     = 1
	IEEE802      = 6
	ARCNET       = 7
	LOCALTALK    = 11
	LocalNet     = 12
	SMDS         = 14
	FRAMERELAY   = 15
	ATM          = 16
	HDLC         = 17
	FIBRECHANNEL = 18
	ATM2         = 19
	SERIALLINE   = 20

	// positions in the packet
	DHCPOptionMessageType = 53
	DHCPOptionsStart      = 236
)
