package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/nathejk/shared-go/types"
)

type BridgeQueries interface {
	MapTeamID(string) types.TeamID
}

func BridgeTeam(q BridgeQueries) http.HandlerFunc {

	type request struct {
	}
	type response struct {
		TeamID types.TeamID `json:"teamId"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("MApping * -> *")
		oldTeamID := r.URL.Path[12:]
		log.Printf("MApping %q -> *", oldTeamID)
		newTeamID := q.MapTeamID(oldTeamID)
		log.Printf("MApping %q -> %q", oldTeamID, newTeamID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{
			TeamID: newTeamID,
		})
	}
}
