package agentmail

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/google/uuid"
)

type memoryClient struct {
	pods       map[string]string
	inboxes    map[string]remoteInbox
	deleted    map[string]struct{}
	podKeys    map[string]string
	rejectKeys map[string]struct{}
	threads    map[string][]ThreadDetail
	minted     []string
	ensurePodN atomic.Int32
	createKeyN atomic.Int32
	keySeq     atomic.Int32
}

func newMemoryClient() *memoryClient {
	return &memoryClient{
		pods:       map[string]string{},
		inboxes:    map[string]remoteInbox{},
		deleted:    map[string]struct{}{},
		podKeys:    map[string]string{},
		rejectKeys: map[string]struct{}{},
		threads:    map[string][]ThreadDetail{},
	}
}

func (f *memoryClient) ensurePod(_ context.Context, _, clientID string) (string, error) {
	f.ensurePodN.Add(1)
	if id, ok := f.pods[clientID]; ok {
		return id, nil
	}
	id := "pod_" + clientID
	f.pods[clientID] = id
	return id, nil
}

func (f *memoryClient) ensureInbox(_ context.Context, _ clientCred, clientID, _ string) (remoteInbox, error) {
	if inbox, ok := f.inboxes[clientID]; ok {
		if _, gone := f.deleted[inbox.id]; !gone {
			return inbox, nil
		}
	}
	id := "inb_" + uuid.NewString()
	short := clientID
	if len(short) > 8 {
		short = short[:8]
	}
	inbox := remoteInbox{id: id, address: short + "@agentmail.to"}
	f.inboxes[clientID] = inbox
	delete(f.deleted, id)
	return inbox, nil
}

func (f *memoryClient) createInboxKey(_ context.Context, _ clientCred, _ string) (string, error) {
	f.createKeyN.Add(1)
	seq := f.keySeq.Add(1)
	key := fmt.Sprintf("am_inbox_%d_%s", seq, uuid.NewString()[:8])
	f.minted = append(f.minted, key)
	return key, nil
}

func (f *memoryClient) deleteInbox(_ context.Context, _ clientCred, inboxID string) error {
	if inboxID == "" {
		return errRemoteNotFound
	}
	found := false
	for _, inbox := range f.inboxes {
		if inbox.id == inboxID {
			found = true
			break
		}
	}
	if !found {
		return errRemoteNotFound
	}
	f.deleted[inboxID] = struct{}{}
	return nil
}

func (f *memoryClient) deletePod(_ context.Context, _, podID string) error {
	if podID == "" {
		return errRemoteNotFound
	}
	for clientID, id := range f.pods {
		if id == podID {
			delete(f.pods, clientID)
			return nil
		}
	}
	return errRemoteNotFound
}

func (f *memoryClient) validateOrgKey(_ context.Context, orgKey string) error {
	if orgKey == "" {
		return ErrBadOrgKey
	}
	if _, reject := f.rejectKeys[orgKey]; reject {
		return ErrBadOrgKey
	}
	return nil
}

func (f *memoryClient) seedThread(inboxID string, thread ThreadDetail) {
	f.threads[inboxID] = append(f.threads[inboxID], thread)
}

func (f *memoryClient) listThreads(_ context.Context, _, inboxID, _ string) (ThreadPage, error) {
	if inboxID == "" {
		return ThreadPage{}, errRemoteNotFound
	}
	rows := f.threads[inboxID]
	page := ThreadPage{Threads: make([]Thread, 0, len(rows))}
	for _, row := range rows {
		page.Threads = append(page.Threads, row.Thread)
	}
	return page, nil
}

func (f *memoryClient) getThread(_ context.Context, _, inboxID, threadID string) (ThreadDetail, error) {
	if inboxID == "" || threadID == "" {
		return ThreadDetail{}, errRemoteNotFound
	}
	for _, row := range f.threads[inboxID] {
		if row.ID == threadID {
			return row, nil
		}
	}
	return ThreadDetail{}, errRemoteNotFound
}

func (f *memoryClient) inspectKey(ctx context.Context, orgKey string) (inspectedKey, error) {
	if err := f.validateOrgKey(ctx, orgKey); err != nil {
		return inspectedKey{}, err
	}
	if podID, ok := f.podKeys[orgKey]; ok {
		return inspectedKey{authorityKind: authorityBYOPod, podID: podID}, nil
	}
	return inspectedKey{authorityKind: authorityBYOOrg}, nil
}
