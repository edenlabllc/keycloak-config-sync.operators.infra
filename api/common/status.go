package common

const (
	FinalizerName string = "config.idp.edenlab.io/finalizer"

	// PhasePending indicates that the keycloak config sync process has pending.
	PhasePending string = "Pending"

	// PhaseCompleted indicates that the keycloak config sync process has finished successfully.
	PhaseCompleted string = "Completed"

	// PhaseFailed indicates that the keycloak config sync process has encountered an error and stopped.
	PhaseFailed string = "Failed"
)
