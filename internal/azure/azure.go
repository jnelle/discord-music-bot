package db

import (
	"context"
	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type AzureClient struct {
	azcosmos     *azcosmos.Client
	azBlobClient *azblob.Client
}

func NewAzureClient() *AzureClient {
	return &AzureClient{}
}

func (a *AzureClient) NewAzCosmos(clientID, endpointURL string) error {
	// FOR AZURE MANAGED IDENTITY
	// azClientID := azidentity.ClientID(clientID)
	// opts := azidentity.ManagedIdentityCredentialOptions{ID: azClientID}
	// cred, err := azidentity.NewManagedIdentityCredential(&opts)
	// if err != nil {
	// 	slog.Error("[azure.go]", slog.String("error", err.Error()))
	// 	return nil, err
	// }
	cred, err := azcosmos.NewKeyCredential(clientID)
	if err != nil {
		slog.Error("[db.go]", slog.String("error", err.Error()))
		return err
	}
	// client, err := azcosmos.NewClient(endpointURL, cred, nil)
	// if err != nil {
	// 	slog.Error("[db.go]", slog.String("error", err.Error()))
	// 	return nil, err
	// }
	client, err := azcosmos.NewClientWithKey(endpointURL, cred, nil)
	if err != nil {
		slog.Error("[db.go]", slog.String("error", err.Error()))
		return err
	}
	a.azcosmos = client
	return nil
}

func (a *AzureClient) CreateDatabase(ctx context.Context) (azcosmos.DatabaseResponse, error) {
	return a.azcosmos.CreateDatabase(ctx, azcosmos.DatabaseProperties{ID: "data"}, nil)
}

func (a *AzureClient) CreateContainer(ctx context.Context) (*azcosmos.ContainerClient, error) {
	_, _ = a.CreateDatabase(ctx)
	properties := azcosmos.ContainerProperties{
		ID: "video",
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
			Paths: []string{"/id"},
		},
	}

	client, err := a.azcosmos.NewDatabase("data")
	if err != nil {
		return nil, err
	}

	_, err = client.CreateContainer(ctx, properties, nil)
	if err != nil {
		return client.NewContainer("video")
	}

	return client.NewContainer("video")
}

func (a *AzureClient) NewAzBlobStorage(connectionString string) {
	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return
	}
	a.azBlobClient = client
}

func (a *AzureClient) GetAzBlobClient() *azblob.Client {
	return a.azBlobClient
}
