package logging

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"os"
	"midas/common"
	"log"
)

// Asia Pacific (Tokyo)
const AWS_REGION = "ap-northeast-1"
const ARB_OPPS_TABLE_NAME = "arb_opps_test_1"

const (
	ATTR_NAME_ARB_CHAIN = "arb_chain"
	ATTR_NAME_COIN_A = "coin_a"
	ATTR_NAME_COIN_B = "coin_b"
	ATTR_NAME_COIN_C = "coin_c"
	ATTR_NAME_QTY_BEFORE = "qty_before"
	ATTR_NAME_QTY_AFTER = "qty_after"
	ATTR_NAME_RELATIVE_PROFIT = "relative_profit"
	ATTR_NAME_LASTED_FOR = "lasted_for"
	ATTR_NAME_STARTED_AT = "started_at"
	ATTR_NAME_FINISHED_AT = "finished_at"
)

const (
	INDEX_RELATIVE_PROFIT = "index_relative_profit"
	INDEX_LASTED_FOR = "index_lasted_for"
)

const READ_CAPACITY = 10
const WRITE_CAPACITY = 10

func CreateTeableIfNotExists() {
	session, _ := session.NewSession(&aws.Config{
		Region: aws.String(AWS_REGION)},
	)

	svc := dynamodb.New(session)

	listTablesOutput, _ := svc.ListTables(&dynamodb.ListTablesInput{})

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
				AttributeName: aws.String(ATTR_NAME_RELATIVE_PROFIT),
				AttributeType: aws.String("N"),
			},
			{
				AttributeName: aws.String(ATTR_NAME_LASTED_FOR),
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
						AttributeName: aws.String(ATTR_NAME_RELATIVE_PROFIT),
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
						AttributeName: aws.String(ATTR_NAME_LASTED_FOR),
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

	_, err := svc.CreateTable(input)

	if err != nil {
		log.Println("Got error creating " + ARB_OPPS_TABLE_NAME)
		log.Println(err.Error())
		os.Exit(1)
	}

	log.Println("Created table " + ARB_OPPS_TABLE_NAME)
}