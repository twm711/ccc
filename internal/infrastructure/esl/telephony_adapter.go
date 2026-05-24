package esl

import "context"

// TelephonyAdapter adapts an ESL Client to the call.TelephonyProvider interface.
type TelephonyAdapter struct {
	client *Client
}

func NewTelephonyAdapter(c *Client) *TelephonyAdapter {
	return &TelephonyAdapter{client: c}
}

func (a *TelephonyAdapter) Originate(ctx context.Context, dest, callerID, eslContext string) (string, error) {
	return a.client.Originate(ctx, dest, callerID, eslContext)
}

func (a *TelephonyAdapter) Hangup(ctx context.Context, uuid string) error {
	return a.client.HangupCall(ctx, uuid)
}

func (a *TelephonyAdapter) Hold(ctx context.Context, uuid string) error {
	return a.client.HoldCall(ctx, uuid)
}

func (a *TelephonyAdapter) Retrieve(ctx context.Context, uuid string) error {
	return a.client.RetrieveCall(ctx, uuid)
}

func (a *TelephonyAdapter) Transfer(ctx context.Context, uuid, dest string) error {
	return a.client.TransferCall(ctx, uuid, dest)
}

func (a *TelephonyAdapter) SendDTMF(ctx context.Context, uuid, digits string) error {
	return a.client.SendDTMF(ctx, uuid, digits)
}

func (a *TelephonyAdapter) Bridge(ctx context.Context, uuid1, uuid2 string) error {
	return a.client.Bridge(ctx, uuid1, uuid2)
}

func (a *TelephonyAdapter) Eavesdrop(ctx context.Context, spyUUID, targetUUID string) error {
	return a.client.Eavesdrop(ctx, spyUUID, targetUUID)
}

func (a *TelephonyAdapter) Conference(ctx context.Context, uuid, confName string) error {
	return a.client.Conference(ctx, uuid, confName)
}
