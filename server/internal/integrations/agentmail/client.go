package agentmail

import (
	"context"
	"errors"
)

// Read, send, and draft only. Management scopes stay off this list.
var inboxKeyPermissions = []string{
	"thread_list",
	"thread_get",
	"message_get",
	"message_send",
	"draft_create",
	"draft_get",
	"draft_send",
}

var errRemoteNotFound = errors.New("agentmail: remote not found")

type clientCred struct {
	apiKey string
	podID  string
}

type remoteInbox struct {
	id      string
	address string
}

type inspectedKey struct {
	authorityKind string
	podID         string
	domain        string
}

type apiClient interface {
	ensurePod(ctx context.Context, orgKey, clientID string) (string, error)
	ensureInbox(ctx context.Context, cred clientCred, clientID, displayName string) (remoteInbox, error)
	createInboxKey(ctx context.Context, cred clientCred, inboxID string) (string, error)
	deleteInbox(ctx context.Context, cred clientCred, inboxID string) error
	deletePod(ctx context.Context, orgKey, podID string) error
	validateOrgKey(ctx context.Context, orgKey string) error
	inspectKey(ctx context.Context, orgKey string) (inspectedKey, error)
}

type liveClient struct{}

func (liveClient) ensurePod(context.Context, string, string) (string, error) {
	return "", errors.New("agentmail: live client not implemented")
}

func (liveClient) ensureInbox(context.Context, clientCred, string, string) (remoteInbox, error) {
	return remoteInbox{}, errors.New("agentmail: live client not implemented")
}

func (liveClient) createInboxKey(context.Context, clientCred, string) (string, error) {
	return "", errors.New("agentmail: live client not implemented")
}

func (liveClient) deleteInbox(context.Context, clientCred, string) error {
	return errors.New("agentmail: live client not implemented")
}

func (liveClient) deletePod(context.Context, string, string) error {
	return errors.New("agentmail: live client not implemented")
}

func (liveClient) validateOrgKey(context.Context, string) error {
	return errors.New("agentmail: live client not implemented")
}

func (liveClient) inspectKey(context.Context, string) (inspectedKey, error) {
	return inspectedKey{}, errors.New("agentmail: live client not implemented")
}
