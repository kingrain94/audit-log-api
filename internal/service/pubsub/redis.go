package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"

	"github.com/kingrain94/audit-log-api/internal/api/dto"
	"github.com/kingrain94/audit-log-api/pkg/logger"
)

const (
	channelPrefix = "audit_logs:"
)

type RedisPubSub struct {
	client       *redis.Client
	logger       *logger.Logger
	subscribers  map[string]*redis.PubSub // Map of tenant ID to subscriber
	subscriberMu sync.RWMutex
}

func NewRedisPubSub(client *redis.Client, logger *logger.Logger) *RedisPubSub {
	return &RedisPubSub{
		client:      client,
		logger:      logger,
		subscribers: make(map[string]*redis.PubSub),
	}
}

func (ps *RedisPubSub) getChannelName(tenantID string) string {
	return channelPrefix + tenantID
}

// Publish publishes an audit log to the tenant's Redis channel
func (ps *RedisPubSub) Publish(ctx context.Context, log *dto.AuditLogResponse) error {
	message, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("failed to marshal audit log: %w", err)
	}

	channel := ps.getChannelName(log.TenantID)
	if err := ps.client.Publish(ctx, channel, message).Err(); err != nil {
		return fmt.Errorf("failed to publish to Redis channel %s: %w", channel, err)
	}

	return nil
}

// Subscribe subscribes to audit logs for a specific tenant
func (ps *RedisPubSub) Subscribe(ctx context.Context, tenantID string, callback func(*dto.AuditLogResponse)) error {
	channel := ps.getChannelName(tenantID)

	// Check if we're already subscribed to this tenant's channel
	ps.subscriberMu.RLock()
	_, exists := ps.subscribers[tenantID]
	ps.subscriberMu.RUnlock()
	if exists {
		ps.logger.Infof("Already subscribed to tenant channel: %s", channel)
		return nil
	}

	// Create new subscription
	pubsub := ps.client.Subscribe(ctx, channel)

	// Store the subscriber
	ps.subscriberMu.Lock()
	ps.subscribers[tenantID] = pubsub
	ps.subscriberMu.Unlock()

	// Start receiving messages
	go func() {
		defer func() {
			ps.logger.Infof("Closing subscription for tenant channel: %s", channel)
			pubsub.Close()
			ps.subscriberMu.Lock()
			delete(ps.subscribers, tenantID)
			ps.subscriberMu.Unlock()
		}()

		ch := pubsub.Channel()
		for {
			select {
			case msg := <-ch:
				var log dto.AuditLogResponse
				if err := json.Unmarshal([]byte(msg.Payload), &log); err != nil {
					ps.logger.Errorf("Failed to unmarshal audit log from channel %s: %v", channel, err)
					continue
				}
				callback(&log)

			case <-ctx.Done():
				return
			}
		}
	}()

	ps.logger.Infof("Subscribed to tenant channel: %s", channel)
	return nil
}

// Unsubscribe removes subscription for a tenant
func (ps *RedisPubSub) Unsubscribe(tenantID string) {
	ps.subscriberMu.Lock()
	defer ps.subscriberMu.Unlock()

	if pubsub, exists := ps.subscribers[tenantID]; exists {
		pubsub.Close()
		delete(ps.subscribers, tenantID)
		ps.logger.Infof("Unsubscribed from tenant channel: %s", ps.getChannelName(tenantID))
	}
}

func (ps *RedisPubSub) Close() {
	ps.subscriberMu.Lock()
	defer ps.subscriberMu.Unlock()

	for tenantID, pubsub := range ps.subscribers {
		pubsub.Close()
		delete(ps.subscribers, tenantID)
		ps.logger.Infof("Closed subscription for tenant channel: %s", ps.getChannelName(tenantID))
	}
}
