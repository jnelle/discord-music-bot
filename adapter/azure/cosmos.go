package azure

import (
	"context"
	"encoding/json"
	"jnelle/discord-music-bot/common"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

type CosmosDBRepository struct {
	db *azcosmos.ContainerClient
}

func NewCosmosDB(db *azcosmos.ContainerClient) *CosmosDBRepository {
	return &CosmosDBRepository{db: db}
}

func (c *CosmosDBRepository) Create(ctx context.Context, media *common.Media) error {
	b, err := json.Marshal(media)
	if err != nil {
		return err
	}
	pk := azcosmos.NewPartitionKeyString(media.ID)
	_, err = c.db.CreateItem(ctx, pk, b, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *CosmosDBRepository) Read(ctx context.Context, id string) (*common.Media, error) {
	result, err := c.db.ReadItem(ctx, azcosmos.NewPartitionKeyString(id), id, nil)
	if err != nil {
		return nil, err
	}

	var media *common.Media
	err = json.Unmarshal(result.Value, &media)
	if err != nil {
		return nil, err
	}

	return media, nil
}
