// Package nex is the Hyrule Warriors (Wii U) NEX server (game_server_id
// 1017cd00 / 269995264). The retail RPX statically links NEX and drives it
// through nn::act::AcquireNexServiceToken with that game server ID - the
// standard Wii U NASC -> nex_token -> PRUDP path. Authentication and secure
// run as separate PRUDP endpoints.
package nex

import (
	"fmt"
	"os"
	"strconv"

	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_ticket_granting "github.com/PretendoNetwork/nex-protocols-common-go/v2/ticket-granting"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting"
	"github.com/Protarium-Network/hyrule-warriors-nex/globals"
)

var AuthenticationServer *nex.PRUDPServer
var AuthenticationEndpoint *nex.PRUDPEndPoint

func StartAuthenticationServer() {
	AuthenticationServer = nex.NewPRUDPServer()

	// PRUDPv1 CONNECT-ACK signature scheme. The 3.4.x Wii U titles (Trine 2,
	// the Mario & Sonic servers) need LegacyConnectionSignature = true: their
	// console signs CONNECT with an empty connection signature and rejects a
	// CONNECT-ACK signed any other way (106-0502, endless CONNECT retransmit).
	// Hyrule Warriors is NEX 3.8.13 and a real console capture shows it
	// rejecting the legacy-signed CONNECT-ACK the same way - it wants the
	// modern scheme (nex-go's default), so leave this false.
	AuthenticationServer.PRUDPV1Settings.LegacyConnectionSignature = false

	AuthenticationEndpoint = nex.NewPRUDPEndPoint(1)
	AuthenticationEndpoint.ServerAccount = globals.AuthenticationServerAccount
	AuthenticationEndpoint.AccountDetailsByPID = globals.AccountDetailsByPID
	AuthenticationEndpoint.AccountDetailsByUsername = globals.AccountDetailsByUsername
	AuthenticationServer.BindPRUDPEndPoint(AuthenticationEndpoint)

	AuthenticationServer.LibraryVersions.SetDefault(nex.NewLibraryVersion(globals.NEXMajor, globals.NEXMinor, globals.NEXPatch))
	AuthenticationServer.AccessKey = globals.AccessKey
	// NEX 3.8.x-era titles write the structure-header version byte before
	// structured RMC parameters (unlike the 3.4.x Mario & Sonic Sochi 2014
	// build this was templated from, which does not). If a real LoginEx
	// capture ever shows Hyrule Warriors omitting it - "Structure content
	// length longer than data size" on login - flip this to false on both
	// endpoints. The raw-parameter dump below is here to make that call.
	AuthenticationServer.ByteStreamSettings.UseStructureHeader = true

	AuthenticationEndpoint.OnData(func(packet nex.PacketInterface) {
		request := packet.RMCMessage()
		if request == nil {
			return
		}
		pid := uint64(packet.Sender().PID())
		fmt.Printf("[HyruleWarriors Auth] PID=%d protocol=0x%02X method=0x%02X\n", pid, request.ProtocolID, request.MethodID)
		// One-off wire-format diagnostic: dump the raw RMC parameter bytes
		// for LoginEx so the AuthenticationInfo layout can be confirmed by
		// hand against a real capture instead of guessing.
		if request.ProtocolID == 0x0A && request.MethodID == 0x02 {
			fmt.Printf("[HyruleWarriors Auth] RAW LoginEx params (%d bytes): %x\n", len(request.Parameters), request.Parameters)
		}
	})

	registerAuthenticationServerProtocols()

	port, _ := strconv.Atoi(os.Getenv("PN_HYRULEWARRIORS_AUTH_PORT"))
	globals.Logger.Successf("[HyruleWarriors] Authentication server listening on UDP %d", port)
	AuthenticationServer.Listen(port)
}

func registerAuthenticationServerProtocols() {
	ticketGrantingProtocol := ticket_granting.NewProtocol()
	AuthenticationEndpoint.RegisterServiceProtocol(ticketGrantingProtocol)
	commonTicketGrantingProtocol := common_ticket_granting.NewCommonProtocol(ticketGrantingProtocol)

	securePort, _ := strconv.Atoi(os.Getenv("PN_HYRULEWARRIORS_SECURE_PORT"))

	// Must stay short (~15 chars): the retail binary truncates this field
	// into a small fixed-size buffer, so use a bare host, not a subdomain.
	secureHost := os.Getenv("PN_HYRULEWARRIORS_SECURE_HOST")
	if secureHost == "" {
		secureHost = "localhost"
	}

	secureStationURL := types.NewStationURL("")
	secureStationURL.SetURLType(constants.StationURLPRUDPS)
	secureStationURL.SetAddress(secureHost)
	secureStationURL.SetPortNumber(uint16(securePort))
	secureStationURL.SetConnectionID(1)
	secureStationURL.SetPrincipalID(types.NewPID(2))
	secureStationURL.SetStreamID(1)
	secureStationURL.SetStreamType(constants.StreamTypeRVSecure)
	secureStationURL.SetType(uint8(constants.StationURLFlagPublic))

	commonTicketGrantingProtocol.ValidateLoginData = globals.ValidateLoginData
	commonTicketGrantingProtocol.SecureStationURL = secureStationURL
	commonTicketGrantingProtocol.BuildName = types.NewString("")
	commonTicketGrantingProtocol.SecureServerAccount = globals.SecureServerAccount
}
