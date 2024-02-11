package playback

import (
	"errors"
)

type PlayerStorage struct {
	services Map[string, *Player]
}

var (
	ErrPlaybackServiceAlreadyExists = errors.New("PLAYBACK_SERVICE_ALREADY_EXISTS")
	ErrPlaybackServiceDoesntExist   = errors.New("PLAYBACK_SERVICE_DOESNT_EXIST")
)

func NewManager() *PlayerStorage {
	return &PlayerStorage{
		services: Map[string, *Player]{},
	}
}

func (m *PlayerStorage) Get(guildID string) *Player {
	var res *Player
	if ps, ok := m.services.Load(guildID); ok {
		res = ps
	}

	return res
}

func (m *PlayerStorage) Add(guildID string, ps *Player) error {
	if _, ok := m.services.Load(guildID); ok {
		return ErrPlaybackServiceAlreadyExists
	}

	m.services.Store(guildID, ps)

	return nil
}

func (m *PlayerStorage) Delete(guildID string) error {
	if _, ok := m.services.Load(guildID); !ok {
		return ErrPlaybackServiceDoesntExist
	}

	m.services.Delete(guildID)

	return nil
}
