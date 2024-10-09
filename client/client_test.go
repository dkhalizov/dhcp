package client

import (
	"fmt"
	"net"
	"testing"
)

func TestClient(t *testing.T) {
	mac, err := net.ParseMAC("04:ec:d8:40:f8:12")
	if err != nil {
		fmt.Println("Error parsing MAC address:", err)
		return
	}

	discoveryPacket := createDHCPPacket(mac, 1, nil, nil)
	fmt.Println("Sending DHCP Discovery packet...")
	offerPackets, err := sendAndReceiveMultiple(discoveryPacket, 2)
	if err != nil {
		fmt.Println("Error receiving DHCP Offers:", err)
		return
	}

	fmt.Printf("Received %d DHCP Offer(s)\n", len(offerPackets))
	for i, offer := range offerPackets {
		fmt.Printf("Offer %d: %v from server %v\n", i+1, offer.YIAddr[:], getServerIP(offer))
	}

	chosenOffer := chooseOffer(offerPackets)
	if chosenOffer == nil {
		fmt.Println("No valid offers received")
		return
	}

	fmt.Printf("Chosen offer: %v from server %v\n", chosenOffer.YIAddr[:], getServerIP(chosenOffer))

	// DHCP Request
	requestPacket := createDHCPPacket(mac, 3, chosenOffer.YIAddr[:], nil)
	fmt.Println("Sending DHCP Request packet...")
	ackPackets, err := sendAndReceiveMultiple(requestPacket, 1)
	if err != nil {
		fmt.Println("Error receiving DHCP Acknowledgement:", err)
		return
	}

	if len(ackPackets) == 0 {
		fmt.Println("No DHCP Acknowledgement received")
		return
	}

	ackPacket := ackPackets[0]
	fmt.Printf("Received DHCP Acknowledgement: %v\n", ackPacket.YIAddr[:])
	fmt.Printf("Assigned IP: %v\n", ackPacket.YIAddr[:])
}
