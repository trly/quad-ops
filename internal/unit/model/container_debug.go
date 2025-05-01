package model

import (
	"fmt"
	"log"

	"github.com/compose-spec/compose-go/v2/types"
)

// DebugVolumeTypes prints the source, target, and type of all volumes in a service config.
func DebugVolumeTypes(service types.ServiceConfig) {
	for _, vol := range service.Volumes {
		fmt.Printf("Volume: source=%s, target=%s, type=%s\n", vol.Source, vol.Target, vol.Type)
		fmt.Printf("Volume is type=bind?: %v\n", vol.Type == "bind")
		log.Printf("Volume full details: %+v\n", vol)
	}
}
