package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

// HyruleWarriorsGetRatings answers DataStore::GetRatings with the
// list-of-slot-lists layout the Wii U DataStore client expects (one slot
// list per requested data ID, then one QResult per data ID). Ratings are
// not stored, so every slot comes back empty - Hyrule Warriors' shared
// objects (Network Links, My Fairy) are not rated content. The handler is
// registered so a stray call decodes cleanly rather than erroring; drop it
// if a capture ever shows the title never issues GetRatings.
func HyruleWarriorsGetRatings(err error, packet nex.PacketInterface, callID uint32, dataIDs types.List[types.UInt64], accessPassword types.UInt64) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	endpoint := packet.Sender().Endpoint()
	ratings := types.NewList[types.List[datastore_types.DataStoreRatingInfoWithSlot]]()
	results := types.NewList[types.QResult]()

	const slotCount = 8
	for range dataIDs {
		slots := types.NewList[datastore_types.DataStoreRatingInfoWithSlot]()
		for slot := 0; slot < slotCount; slot++ {
			rating := datastore_types.NewDataStoreRatingInfoWithSlot()
			rating.Slot = types.NewInt8(int8(slot))
			slots = append(slots, rating)
		}
		ratings = append(ratings, slots)
		results = append(results, types.NewQResultSuccess(nex.ResultCodes.DataStore.Unknown))
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
	ratings.WriteTo(rmcResponseStream)
	results.WriteTo(rmcResponseStream)

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseStream.Bytes())
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetRatings
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
