package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/guregu/dynamo/v2"

	"github.com/ccpeng/kube-replay/internal/data"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		panic(fmt.Sprint("usage: go run setup.go <tablename>"))
	}
	table := args[1]

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("us-west-2"))
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}

	db := dynamo.New(cfg)

	fmt.Printf("checking if table %s exists\n", table)
	tables, _ := db.ListTables().All(context.Background())
	create := true
	for _, t := range tables {
		if t == table {
			fmt.Printf("table %s exists\n", table)
			create = false
		}
	}

	if create {
		fmt.Printf("creating table %s\n", table)
		if err := db.CreateTable(table, data.NodeMeta{}).Wait(context.Background()); err != nil {
			panic(fmt.Errorf("error when creating table %s: %w", table, err))
		}
	}

	fmt.Printf("checking for table TTL setup...\n")
	result, err := db.Table(table).DescribeTTL().Run(context.Background())
	if err != nil {
		panic(fmt.Errorf("error when describing table %s: %w", table, err))
	}
	if result.Attribute != "ExpireAt" {
		fmt.Printf("table needs TTL to be set up...\n")
		if err := db.Table(table).UpdateTTL("ExpireAt", true).Run(context.Background()); err != nil {
			panic(fmt.Errorf("error when updating TTL for table %s: %w", table, err))
		}
	}

	fmt.Printf("table %s with TTL exists or has been created\n", table)
}
