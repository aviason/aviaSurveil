package agacandidatedemo

import "fmt"

const (
	OverlayOwnerRole  = "preprod_aga_demo_owner"
	OverlayWriterRole = "preprod_aga_demo_writer"
	OverlayReaderRole = "preprod_aga_demo_reader"
	NormalAPIRole     = "preprod_normal_api"
)

// ValidateOverlayDatabaseUser rejects the normal API and reader before any
// write-capable pool is constructed. Bootstrap uses the NOLOGIN owner role;
// the one-shot loader must use the dedicated writer login.
func ValidateOverlayDatabaseUser(user string) error {
	if user != OverlayWriterRole {
		return fmt.Errorf("AGA demo loader requires the dedicated overlay writer role")
	}
	return nil
}
