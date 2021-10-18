package dynamodb

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"log"
)

type AttributeValue struct {
	types.AttributeValue
}

type Client interface {
	Scan(tableName string) ([]map[string]interface{}, error)
	GetItem(tableName string, key map[string]AttributeValue) (map[string]interface{}, error)
}

func NewClient(region string) Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region))
	if err != nil {
		log.Fatal(err)
	}
	return &dynamodbClientImpl{
		cfg: cfg,
	}
}

type dynamodbClientImpl struct {
	cfg aws.Config
}

func (d *dynamodbClientImpl) Scan(tableName string) ([]map[string]interface{}, error) {
	ctx := context.TODO()
	dynamo := dynamodb.NewFromConfig(d.cfg)

	params := &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}

	res, err := dynamo.Scan(ctx, params)
	if err != nil {
		return nil, err
	}

	resConv := make([]map[string]interface{}, 0)
	for _, val := range res.Items {
		valConv := make(map[string]interface{})
		for k, v := range val {
			log.Printf("blabla %s -> %+v\n", k, v)
			valConv[k] = convertType(v)
		}
		resConv = append(resConv, valConv)
	}

	return resConv, nil
}

func (d *dynamodbClientImpl) GetItem(tableName string, key map[string]AttributeValue) (map[string]interface{}, error) {
	ctx := context.TODO()
	dynamo := dynamodb.NewFromConfig(d.cfg)

	keyConv := make(map[string]types.AttributeValue)
	for k, v := range key {
		keyConv[k] = v.AttributeValue
	}

	params := &dynamodb.GetItemInput{
		Key:       keyConv,
		TableName: aws.String(tableName),
	}

	res, err := dynamo.GetItem(ctx, params)
	if err != nil {
		return nil, err
	}

	resConv := make(map[string]interface{})
	for k, v := range res.Item {
		resConv[k] = convertType(v)
	}

	return resConv, nil
}

func convertType(i interface{}) interface{} {
	var value interface{}

	switch j := i.(type) {
	case *types.AttributeValueMemberS:
		value = j.Value
	case *types.AttributeValueMemberN:
		value = j.Value
	case *types.AttributeValueMemberB:
		value = j.Value
	case *types.AttributeValueMemberSS:
		value = j.Value
	case *types.AttributeValueMemberNS:
		value = j.Value
	case *types.AttributeValueMemberBS:
		value = j.Value
	case *types.AttributeValueMemberM:
		value = j.Value
	case *types.AttributeValueMemberL:
		value = j.Value
	case *types.AttributeValueMemberNULL:
		value = j.Value
	case *types.AttributeValueMemberBOOL:
		value = j.Value
	default:
		value = "invalid"
	}

	return value
}
