package relation

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
)

type Peer struct {
	ID          fields.ID        `json:"id"`
	ActorID     fields.ID        `json:"actor_id"`
	ChannelID   fields.ID        `json:"channel_id"`
	AvatarURL   fields.URL       `json:"avatar_url"`
	Username    user.Username    `json:"username"`
	DisplayName user.DisplayName `json:"display_name"`
	RelType     Type             `json:"rel_type"`
	Presence    user.Presence    `json:"presence"`
}

func hydratePeer(
	peerID fields.ID,
	rel *Relation,
	u *user.User,
	p user.Presence,
) (Peer, bool) {
	if rel == nil || u == nil {
		return Peer{}, false
	}

	return Peer{
		ID:          peerID,
		ActorID:     rel.ActorID(),
		ChannelID:   rel.ChannelID(),
		AvatarURL:   u.AvatarURL(),
		Username:    u.Username(),
		DisplayName: u.DisplayName(),
		RelType:     rel.Type(),
		Presence:    p,
	}, true
}

func hydratePeers(
	currentUserID fields.ID,
	relations []*Relation,
	userMap map[fields.ID]*user.User,
	presenceMap map[fields.ID]user.Presence,
) []Peer {
	if len(relations) == 0 {
		return nil
	}

	views := make([]Peer, 0, len(relations))

	for _, rel := range relations {
		if rel == nil {
			continue
		}

		peerID := rel.PeerID(currentUserID)
		u := userMap[peerID]
		if u == nil {
			continue
		}

		p, ok := presenceMap[peerID]
		if !ok {
			p = user.NewPresence(user.PresenceOffline)
		}

		if view, ok := hydratePeer(peerID, rel, u, p); ok {
			views = append(views, view)
		}
	}

	sortPeers(views)

	return views
}
