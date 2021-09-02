// Package types implements different core types of the API.
package types

import (
	"github.com/ethereum/go-ethereum/p2p/enode"
	"go.mongodb.org/mongo-driver/bson"
	"time"
)

// NetworkNode represents an information about single network node
// discovered by the active node crawler.
type NetworkNode struct {
	// Node holds information about the node ID and ENR record.
	Node *enode.Node

	// Score tracks accuracy of the node record. It's incremented
	// every time the node is detected and halved if a check fails.
	Score int `json:"score"`

	// Found represents the date and time the node was found.
	Found time.Time `json:"found,omitempty"`

	// LastResponse represents the latest date and time the node was successfully contacted.
	LastResponse time.Time `json:"last_response,omitempty"`

	// LastCheck represents the latest date and time the node was tested.
	LastCheck time.Time `json:"last_check,omitempty"`
}

// MarshalBSON creates a BSON representation of a network node record.
func (nn *NetworkNode) MarshalBSON() ([]byte, error) {
	// prep the structure for saving
	pom := struct {
		Scheme    string    `bson:"scheme"`
		URL       string    `bson:"url"`
		Score     int       `bson:"score"`
		Found     time.Time `bson:"found"`
		Responded time.Time `bson:"valid"`
		Checked   time.Time `bson:"check"`
	}{
		Scheme:    "v4",
		URL:       nn.Node.String(),
		Score:     nn.Score,
		Found:     nn.Found,
		Responded: nn.LastResponse,
		Checked:   nn.LastCheck,
	}
	return bson.Marshal(pom)
}

// UnmarshalBSON updates the record of the network node from BSON source.
func (nn *NetworkNode) UnmarshalBSON(data []byte) (err error) {
	// try to decode the BSON data
	var row struct {
		Scheme    string    `bson:"scheme"`
		URL       string    `bson:"url"`
		Score     int       `bson:"score"`
		Found     time.Time `bson:"found"`
		Responded time.Time `bson:"valid"`
		Checked   time.Time `bson:"check"`
	}
	if err = bson.Unmarshal(data, &row); err != nil {
		return err
	}

	// parse the node URL
	nn.Node, err = enode.Parse(enode.ValidSchemes, row.URL)
	if err != nil {
		return err
	}

	// copy score and timestamps
	nn.Score = row.Score
	nn.Found = row.Found
	nn.LastResponse = row.Responded
	nn.LastCheck = row.Checked
	return nil
}
