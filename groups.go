package main

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/types"
)

/*
 * This function will try to query group information from WhatsApp servers.
 * Upon immediate success, the list of participants is returned.
 * In case of failure (e.g. due to timeout), the function will return nil and retry in background.
 * Upon delayed success, the response is fed to purple asynchronously.
 */
func (handler *Handler) query_group_participants_retry(group_jid types.JID, seconds_backoff int, max_retries int, retry_count int) []types.GroupParticipant {
	group, err := handler.client.GetGroupInfo(context.TODO(), group_jid)
	if err == nil && group != nil {
		if retry_count > 0 {
			purple_update_group(handler.account, group)
		} else {
			return group.Participants
		}
	} else {
		consequence := "Giving up."
		if retry_count < max_retries {
			consequence = "Retrying…"
			go func() {
				time.Sleep(time.Duration(seconds_backoff) * time.Second)
				handler.query_group_participants_retry(group_jid, seconds_backoff, max_retries, retry_count+1)
			}()
		}
		handler.log.Infof("Cannot get group information due to %#v. %s", err, consequence)
	}
	return []types.GroupParticipant{}
}

/*
 * Channel (newsletter) messages carry no name of the channel.
 * Fetch the names of all subscribed channels so their chats do not show the raw JID.
 */
func (handler *Handler) fetch_channel_names() {
	channels, err := handler.client.GetSubscribedNewsletters(context.TODO())
	if err != nil {
		handler.log.Warnf("Failed to fetch subscribed channels: %v", err)
		return
	}
	for _, channel := range channels {
		if channel != nil && channel.ThreadMeta.Name.Text != "" {
			purple_update_name(handler.account, channel.ID.String(), channel.ThreadMeta.Name.Text)
		}
	}
}
