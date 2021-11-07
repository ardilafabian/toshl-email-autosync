package sync

import (
	"fmt"
	"github.com/Philanthropists/toshl-email-autosync/internal/dynamodb"
	"github.com/Philanthropists/toshl-email-autosync/internal/logger"
	synctypes "github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"time"
)

func GetLastProcessedDate() time.Time {
	logger := logger.GetLogger()
	const dateField = "LastProcessedDate"
	const tableName = "toshl-data"
	defaultDate := time.Now().Add(-365 * 24 * time.Hour) // from 1 year in the past by default

	var selectedDate time.Time

	client, err := dynamodb.NewClient("us-east-1")
	if err != nil {
		logger.Fatalw("error creating dynamodb client",
			"error", err)
	}

	res, err := client.Scan(tableName)
	if err != nil {
		selectedDate = defaultDate
		logger.Errorw("connection to dynamodb as unsuccessful",
			"error", err)
	} else if len(res) == 1 {
		resValue := res[0]
		value, ok := resValue[dateField]
		if ok {
			switch j := value.(type) {
			case string:
				selectedDate, err = time.Parse(time.RFC822Z, j)
				if err != nil {
					selectedDate = defaultDate
				}
			}
		} else {
			selectedDate = defaultDate
			logger.Warnw("field is not defined in dynamodb item",
				"field", dateField)
		}
	} else {
		selectedDate = defaultDate
		logger.Warnw("something is wrong, the number of items retrieved was not 1",
			"response", res)
	}

	logger.Infow("selected date",
		"date", selectedDate.Format(time.RFC822Z))

	return selectedDate
}

func UpdateLastProcessedDate(failedTxs []*synctypes.TransactionInfo) error {
	logger := logger.GetLogger()
	newDate := getEarliestDateFromTxs(failedTxs)

	const idField = "Id"
	const dateField = "LastProcessedDate"
	const tableName = "toshl-data"

	client, err := dynamodb.NewClient("us-east-1")
	if err != nil {
		logger.Fatalw("error creating dynamodb client",
			"error", err)
	}

	key := map[string]dynamodb.AttributeValue{
		idField: {
			AttributeValue: &types.AttributeValueMemberN{Value: "1"},
		},
	}

	expressionAttributeValues := map[string]dynamodb.AttributeValue{
		":r": {
			AttributeValue: &types.AttributeValueMemberS{Value: newDate.Format(time.RFC822Z)},
		},
	}

	updateExpression := fmt.Sprintf("set %s = :r", dateField)

	err = client.UpdateItem(tableName, key, expressionAttributeValues, updateExpression)
	return err
}