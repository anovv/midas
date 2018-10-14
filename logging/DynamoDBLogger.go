package logging

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"os"
	"midas/common"
	"midas/common/arb"
	"log"
	"sync"
	"time"
)

type ArbStateItem struct {
	// Tags should be in sync with ATTR_NAME keys below
	ArbChain string `dynamodbav:"arb_chain"`
	QtyBefore float64 `dynamodbav:"qty_before"`
	QtyAfter float64 `dynamodbav:"qty_after"`
	ProfitRelative float64 `dynamodbav:"relative_profit_percentage"`
	LastedForMs int64 `dynamodbav:"lasted_for_ms"`
	CoinA string `dynamodbav:"coin_a"`
	CoinB string `dynamodbav:"coin_b"`
	CoinC string `dynamodbav:"coin_c"`
	StartTs time.Time `dynamodbav:"started_at"`
	EndTs time.Time `dynamodbav:"ended_at"`
}

// Asia Pacific (Tokyo)
const AWS_REGION = "ap-northeast-1"
const ARB_OPPS_TABLE_NAME = "arb_opps_test_1"

const (
	// ATTR_NAME keys should be in sync with ArbStateItem struct above
	ATTR_NAME_ARB_CHAIN = "arb_chain"
	ATTR_NAME_RELATIVE_PROFIT_PERCENTAGE = "relative_profit_percentage"
	ATTR_NAME_LASTED_FOR_MS = "lasted_for_ms"
	ATTR_NAME_STARTED_AT = "started_at"
)

const (
	INDEX_RELATIVE_PROFIT = "index_relative_profit"
	INDEX_LASTED_FOR = "index_lasted_for"
)

const READ_CAPACITY = 10
const WRITE_CAPACITY = 10

var dbClient *dynamodb.DynamoDB
var once sync.Once

// Getting db client as a thread-safe singleton
func getDBClient() *dynamodb.DynamoDB {
	once.Do(func() {
		session, _ := session.NewSession(&aws.Config{
			Region: aws.String(AWS_REGION)},
		)
		dbClient = dynamodb.New(session)
	})
	return dbClient
}

func CreateTableIfNotExists() {
	listTablesOutput, _ := getDBClient().ListTables(&dynamodb.ListTablesInput{})

	if common.ContainsStrPtr(ARB_OPPS_TABLE_NAME, listTablesOutput.TableNames) {
		log.Println("Table " + ARB_OPPS_TABLE_NAME + " exists")
		return
	}

	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String(ATTR_NAME_ARB_CHAIN),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String(ATTR_NAME_STARTED_AT),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String(ATTR_NAME_RELATIVE_PROFIT_PERCENTAGE),
				AttributeType: aws.String("N"),
			},
			{
				AttributeName: aws.String(ATTR_NAME_LASTED_FOR_MS),
				AttributeType: aws.String("N"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String(ATTR_NAME_ARB_CHAIN),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String(ATTR_NAME_STARTED_AT),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(READ_CAPACITY),
			WriteCapacityUnits: aws.Int64(WRITE_CAPACITY),
		},
		LocalSecondaryIndexes: []*dynamodb.LocalSecondaryIndex{
			{
				IndexName: aws.String(INDEX_RELATIVE_PROFIT),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String(ATTR_NAME_ARB_CHAIN),
						KeyType:       aws.String("HASH"),
					},
					{
						AttributeName: aws.String(ATTR_NAME_RELATIVE_PROFIT_PERCENTAGE),
						KeyType:       aws.String("RANGE"),
					},
				},
				Projection: &dynamodb.Projection{
					ProjectionType: aws.String(dynamodb.ProjectionTypeKeysOnly),
				},
			},
			{
				IndexName: aws.String(INDEX_LASTED_FOR),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String(ATTR_NAME_ARB_CHAIN),
						KeyType:       aws.String("HASH"),
					},
					{
						AttributeName: aws.String(ATTR_NAME_LASTED_FOR_MS),
						KeyType:       aws.String("RANGE"),
					},
				},
				Projection: &dynamodb.Projection{
					ProjectionType: aws.String(dynamodb.ProjectionTypeKeysOnly),
				},
			},
		},
		TableName: aws.String(ARB_OPPS_TABLE_NAME),
	}

	_, err := getDBClient().CreateTable(input)

	if err != nil {
		log.Println("Got error creating " + ARB_OPPS_TABLE_NAME)
		log.Println(err.Error())
		os.Exit(1)
	}

	log.Println("Created table " + ARB_OPPS_TABLE_NAME)
}

func RecordArbState(state *arb.State) {
	arbStateItem := genArbStateItem(state)
	av, err := dynamodbattribute.MarshalMap(arbStateItem)
	if err != nil {
		log.Println("Got error marshaling arb state:")
		log.Println(err.Error())
		return
	}

	input := &dynamodb.PutItemInput{
		Item: av,
		TableName: aws.String(ARB_OPPS_TABLE_NAME),
	}

	_, err = getDBClient().PutItem(input)

	if err != nil {
		log.Println("Got error calling PutItem:")
		log.Println(err.Error())
		return
	}
}

func genArbStateItem(state *arb.State) *ArbStateItem {
	arbChain := state.Triangle.CoinA.CoinSymbol + "->" +
		state.Triangle.CoinB.CoinSymbol + "->" +
		state.Triangle.CoinC.CoinSymbol + "->" +
		state.Triangle.CoinA.CoinSymbol

	lastedForMs := int64(state.LastUpdateTs.Sub(state.StartTs)/time.Millisecond)
	return &ArbStateItem{
		ArbChain: arbChain,
		QtyBefore: state.QtyBefore,
		QtyAfter: state.QtyAfter,
		ProfitRelative: state.ProfitRelative * 100.0,
		LastedForMs: lastedForMs,
		CoinA: state.Triangle.CoinA.CoinSymbol,
		CoinB: state.Triangle.CoinB.CoinSymbol,
		CoinC: state.Triangle.CoinC.CoinSymbol,
		StartTs: state.StartTs,
		EndTs: state.LastUpdateTs,
	}
}