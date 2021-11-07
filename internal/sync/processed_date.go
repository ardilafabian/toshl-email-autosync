package sync

import (
	"fmt"
	"github.com/Philanthropists/toshl-email-autosync/internal/dynamodb"
	synctypes "github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"log"
	"time"
)

func GetLastProcessedDate() time.Time {
	const dateField = "LastProcessedDate"
	const tableName = "toshl-data"
	defaultDate := time.Now().Add(-365 * 24 * time.Hour) // from 1 year in the past by default

	var selectedDate time.Time

	client, err := dynamodb.NewClient("us-east-1")
	if err != nil {
		log.Fatalf("error creating dynamodb client: %s", err)
	}

	res, err := client.Scan(tableName)
	if err != nil {
		selectedDate = defaultDate
		log.Printf("connect to dynamodb unsuccessfull: %s\n", err)
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
			log.Printf("%s field is not defined in item", dateField)
		}
	} else {
		selectedDate = defaultDate
		log.Printf("something is wrong, the len was not 1: [%+v]", res)
	}

	log.Printf("selected date: %s", selectedDate.Format(time.RFC822Z))

	return selectedDate
}

func UpdateLastProcessedDate(failedTxs []*synctypes.TransactionInfo) error {
	newDate := getEarliestDateFromTxs(failedTxs)

	const idField = "Id"
	const dateField = "LastProcessedDate"
	const tableName = "toshl-data"

	client, err := dynamodb.NewClient("us-east-1")
	if err != nil {
		log.Fatalf("error creating dynamodb client: %s", err)
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
