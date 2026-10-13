package channel

// type Channel struct {
// 	ID            uuid.UUID  `json:"id"`
// 	Type          int        `json:"type"`
// 	Name          *string    `json:"name,omitempty"`
// 	IconURL       *string    `json:"icon_url,omitempty"`
// 	LastMessageID *uuid.UUID `json:"last_message_id,omitempty"`
// 	LastMessageAt time.Time  `json:"last_message_at"`
// 	CreatedAt     time.Time  `json:"created_at"`
// 	UpdatedAt     time.Time  `json:"updated_at"`
// }

// func ParseChannel(ch *channel.Channel) Channel {
// 	if ch == nil {
// 		return Channel{}
// 	}

// 	return Channel{
// 		ID:            ch.ID,
// 		Type:          int(ch.Type),
// 		Name:          ch.Name,
// 		IconURL:       ch.IconURL,
// 		LastMessageID: ch.LastMessageID,
// 		LastMessageAt: ch.LastMessageAt,
// 		CreatedAt:     ch.CreatedAt,
// 		UpdatedAt:     ch.UpdatedAt,
// 	}
// }

// func (c Channel) ToDomain() *channel.Channel {
// 	return channel.ReconstituteChannel(
// 		c.ID,
// 		channel.ChannelType(c.Type),
// 		c.Name,
// 		c.IconURL,
// 		c.LastMessageID,
// 		c.LastMessageAt,
// 		c.CreatedAt,
// 		c.UpdatedAt,
// 	)
// }

// func unmarshalChannel(data []byte) (*channel.Channel, error) {
// 	if len(data) == 0 {
// 		return nil, nil
// 	}

// 	var dto Channel
// 	if err := json.Unmarshal(data, &dto); err != nil {
// 		return nil, err
// 	}
// 	return dto.ToDomain(), nil
// }

// func marshalChannel(ch *channel.Channel) ([]byte, error) {
// 	if ch == nil {
// 		return nil, nil
// 	}

// 	dto := ParseChannel(ch)
// 	bytes, err := json.Marshal(dto)
// 	if err != nil {
// 		return nil, errs.Internal("Failed to marshal channel json.").
// 			Meta("scope", redis.ScopeChannel.String()).
// 			Wrap(err)
// 	}
// 	return bytes, nil
// }

// type Member struct {
// 	ChannelID         uuid.UUID  `json:"channel_id"`
// 	UserID            uuid.UUID  `json:"user_id"`
// 	LastReadMessageID *uuid.UUID `json:"last_read_message_id,omitempty"`
// 	LastReadMessageAt time.Time  `json:"last_read_message_at"`
// 	PinnedAt          *time.Time `json:"pinned_at,omitempty"`
// 	MutedUntil        *time.Time `json:"muted_until,omitempty"`
// 	MentionCount      int        `json:"mention_count"`
// 	IsVisible         bool       `json:"is_visible"`
// 	CreatedAt         time.Time  `json:"created_at"`
// 	UpdatedAt         time.Time  `json:"updated_at"`
// }

// func ParseMember(m *channel.Member) Member {
// 	if m == nil {
// 		return Member{}
// 	}

// 	return Member{
// 		ChannelID:         m.ChannelID,
// 		UserID:            m.UserID,
// 		LastReadMessageID: m.LastReadMessageID,
// 		LastReadMessageAt: m.LastReadMessageAt,
// 		PinnedAt:          m.PinnedAt,
// 		MutedUntil:        m.MutedUntil,
// 		MentionCount:      m.MentionCount,
// 		IsVisible:         m.IsVisible,
// 		CreatedAt:         m.CreatedAt,
// 		UpdatedAt:         m.UpdatedAt,
// 	}
// }

// func (m Member) ToDomain() *channel.Member {
// 	return channel.ReconstituteMember(
// 		m.ChannelID,
// 		m.UserID,
// 		m.LastReadMessageID,
// 		m.LastReadMessageAt,
// 		m.PinnedAt,
// 		m.MutedUntil,
// 		m.MentionCount,
// 		m.IsVisible,
// 		m.CreatedAt,
// 		m.UpdatedAt,
// 	)
// }

// func unmarshalMember(data []byte) (*channel.Member, error) {
// 	if len(data) == 0 {
// 		return nil, nil
// 	}

// 	var dto Member
// 	if err := json.Unmarshal(data, &dto); err != nil {
// 		return nil, err
// 	}
// 	return dto.ToDomain(), nil
// }

// func marshalMember(m *channel.Member) ([]byte, error) {
// 	if m == nil {
// 		return nil, nil
// 	}

// 	dto := ParseMember(m)
// 	bytes, err := json.Marshal(dto)
// 	if err != nil {
// 		return nil, errs.Internal("Failed to marshal member json.").
// 			Meta("scope", redis.ScopeMember.String()).
// 			Wrap(err)
// 	}
// 	return bytes, nil
// }

// type Message struct {
// 	ID               uuid.UUID       `json:"id"`
// 	ChannelID        uuid.UUID       `json:"channel_id"`
// 	AuthorID         *uuid.UUID      `json:"author_id,omitempty"`
// 	Type             int             `json:"type"`
// 	Content          *string         `json:"content,omitempty"`
// 	Metadata         json.RawMessage `json:"metadata,omitempty"`
// 	ReplyToMessageID *uuid.UUID      `json:"reply_to_message_id,omitempty"`
// 	ForwardMessageID *uuid.UUID      `json:"forward_message_id,omitempty"`
// 	ForwardChannelID *uuid.UUID      `json:"forward_channel_id,omitempty"`
// 	PinnedAt         *time.Time      `json:"pinned_at,omitempty"`
// 	CreatedAt        time.Time       `json:"created_at"`
// 	UpdatedAt        time.Time       `json:"updated_at"`
// 	EditedAt         *time.Time      `json:"edited_at,omitempty"`
// }

// func ParseMessage(msg *channel.Message) Message {
// 	if msg == nil {
// 		return Message{}
// 	}

// 	return Message{
// 		ID:               msg.ID,
// 		ChannelID:        msg.ChannelID,
// 		AuthorID:         msg.AuthorID,
// 		Type:             int(msg.Type),
// 		Content:          msg.Content,
// 		Metadata:         msg.Metadata,
// 		ReplyToMessageID: msg.ReplyToMessageID,
// 		ForwardMessageID: msg.ForwardMessageID,
// 		ForwardChannelID: msg.ForwardChannelID,
// 		PinnedAt:         msg.PinnedAt,
// 		CreatedAt:        msg.CreatedAt,
// 		UpdatedAt:        msg.UpdatedAt,
// 		EditedAt:         msg.EditedAt,
// 	}
// }

// func (m Message) ToDomain() *channel.Message {
// 	return channel.ReconstituteMessage(
// 		m.ID,
// 		m.ChannelID,
// 		m.AuthorID,
// 		channel.MessageType(m.Type),
// 		m.Content,
// 		m.Metadata,
// 		m.ReplyToMessageID,
// 		m.ForwardMessageID,
// 		m.ForwardChannelID,
// 		m.PinnedAt,
// 		m.EditedAt,
// 		m.CreatedAt,
// 		m.UpdatedAt,
// 	)
// }

// func unmarshalMessage(data []byte) (*channel.Message, error) {
// 	if len(data) == 0 {
// 		return nil, nil
// 	}

// 	var dto Message
// 	if err := json.Unmarshal(data, &dto); err != nil {
// 		return nil, err
// 	}
// 	return dto.ToDomain(), nil
// }

// func marshalMessage(msg *channel.Message) ([]byte, error) {
// 	if msg == nil {
// 		return nil, nil
// 	}
// 	dto := ParseMessage(msg)
// 	return json.Marshal(dto)
// }

// const (
// 	channelTTL      = 24 * time.Hour
// 	userChannelsTTL = 24 * time.Hour
// )

// func channelKey(channelID uuid.UUID) string {
// 	return "{channel:" + channelID.String() + "}"
// }

// func userChannelsKey(id uuid.UUID) string {
// 	return "{user:" + id.String() + "}:channels"
// }

// type ChannelCache struct {
// 	client redisdriver.Cmdable
// }

// func NewChannelCache(client redisdriver.Cmdable) *ChannelCache {
// 	return &ChannelCache{
// 		client: client,
// 	}
// }

// func (c *ChannelCache) Get(ctx context.Context, id uuid.UUID) (*channel.Channel, error) {
// 	return getAndUnmarshal(ctx, redis.ScopeChannel, c.client, channelKey(id), unmarshalChannel)
// }

// func (c *ChannelCache) Set(ctx context.Context, ch *channel.Channel) error {
// 	return marshalAndSet(ctx, redis.ScopeChannel, c.client, channelKey(ch.ID), ch, channelTTL, marshalChannel)
// }

// func (c *ChannelCache) Delete(ctx context.Context, id uuid.UUID) error {
// 	if err := c.client.Del(ctx, channelKey(id)).Err(); err != nil {
// 		return redis.NewError(err, redis.ScopeChannel)
// 	}
// 	return nil
// }

// func (c *ChannelCache) GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*channel.Channel, []uuid.UUID, error) {
// 	return getAndUnmarshalBatch(ctx, redis.ScopeChannel, c.client, ids, channelKey, unmarshalChannel)
// }

// func (c *ChannelCache) SetBatch(ctx context.Context, channels map[uuid.UUID]*channel.Channel) error {
// 	return marshalAndSetBatch(ctx, redis.ScopeChannel, c.client, channels, channelKey, channelTTL, marshalChannel)
// }

// func (c *ChannelCache) CreateGroup(ctx context.Context, ch *channel.Channel, members []*channel.Member) error {
// 	chBytes, err := marshalChannel(ch)
// 	if err != nil {
// 		return err
// 	}

// 	memberMap := make(map[string]any, len(members))
// 	for _, m := range members {
// 		if m == nil || m.UserID == uuid.Nil {
// 			continue
// 		}
// 		mBytes, err := marshalMember(m)
// 		if err != nil {
// 			return err
// 		}
// 		memberMap[m.UserID.String()] = mBytes
// 	}

// 	pipe := c.client.Pipeline()

// 	pipe.Set(ctx, channelKey(ch.ID), chBytes, channelTTL)

// 	if len(memberMap) > 0 {
// 		pipe.HSet(ctx, channelMembersKey(ch.ID), memberMap)
// 	}

// 	channelIDStr := ch.ID.String()
// 	for _, m := range members {
// 		if m == nil || m.UserID == uuid.Nil {
// 			continue
// 		}

// 		var score float64
// 		if m.PinnedAt != nil && !m.PinnedAt.IsZero() {
// 			score = 1e12 + float64(m.PinnedAt.Unix())
// 		} else {
// 			score = float64(ch.CreatedAt.Unix())
// 		}

// 		pipe.ZAdd(ctx, userChannelsKey(m.UserID), redisdriver.Z{
// 			Score:  score,
// 			Member: channelIDStr,
// 		})
// 	}

// 	if _, err := pipe.Exec(ctx); err != nil {
// 		return redis.NewError(err, redis.ScopeChannel)
// 	}

// 	return nil
// }

// func (c *ChannelCache) DeleteGroup(ctx context.Context, channelID uuid.UUID, memberIDs []uuid.UUID) error {
// 	pipe := c.client.Pipeline()

// 	pipe.Del(ctx, channelKey(channelID))
// 	pipe.Del(ctx, channelMembersKey(channelID))
// 	pipe.Del(ctx, channelMessagesKey(channelID))

// 	channelIDStr := channelID.String()
// 	for _, userID := range memberIDs {
// 		if userID != uuid.Nil {
// 			pipe.ZRem(ctx, userChannelsKey(userID), channelIDStr)
// 		}
// 	}

// 	if _, err := pipe.Exec(ctx); err != nil {
// 		return redis.NewError(err, redis.ScopeChannel)
// 	}

// 	return nil
// }

// func (c *ChannelCache) GetUserChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, bool, error) {
// 	if userID == uuid.Nil {
// 		return nil, false, nil
// 	}

// 	rawIDs, err := c.client.ZRevRange(ctx, userChannelsKey(userID), 0, -1).Result()
// 	if err != nil {
// 		if redis.IsCacheMiss(err) {
// 			return nil, false, nil
// 		}
// 		return nil, false, redis.NewError(err, redis.ScopeUser)
// 	}

// 	if len(rawIDs) == 0 {
// 		return nil, false, nil
// 	}

// 	ids := make([]uuid.UUID, 0, len(rawIDs))
// 	for _, raw := range rawIDs {
// 		id, err := uuid.Parse(raw)
// 		if err == nil {
// 			ids = append(ids, id)
// 		}
// 	}

// 	return ids, true, nil
// }

// func (c *ChannelCache) SetUserChannelIDs(ctx context.Context, userID uuid.UUID, members []*channel.Member) error {
// 	if userID == uuid.Nil || len(members) == 0 {
// 		return nil
// 	}

// 	pipe := c.client.Pipeline()
// 	zKey := userChannelsKey(userID)

// 	pipe.Del(ctx, zKey)

// 	for _, m := range members {
// 		if m == nil || m.ChannelID == uuid.Nil {
// 			continue
// 		}

// 		var score float64
// 		if m.PinnedAt != nil && !m.PinnedAt.IsZero() {
// 			score = 1e12 + float64(m.PinnedAt.Unix())
// 		} else {
// 			score = float64(m.CreatedAt.Unix())
// 		}

// 		pipe.ZAdd(ctx, zKey, redisdriver.Z{
// 			Score:  score,
// 			Member: m.ChannelID.String(),
// 		})
// 	}

// 	if _, err := pipe.Exec(ctx); err != nil {
// 		return redis.NewError(err, redis.ScopeUser)
// 	}
// 	return nil
// }

// func (c *ChannelCache) RemoveUserChannelID(ctx context.Context, userID, channelID uuid.UUID) error {
// 	if err := c.client.ZRem(ctx, userChannelsKey(userID), channelID.String()).Err(); err != nil {
// 		return redis.NewError(err, redis.ScopeUser)
// 	}

// 	return nil
// }
