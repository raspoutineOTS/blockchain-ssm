// Copyright Luc Yriarte <luc.yriarte@thingagora.org> 2018
// Modernized 2024 for Hyperledger Fabric 2.5+
// License: Apache-2.0

package main

import (
	"fmt"
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	ssmContract := new(SSMContract)

	chaincode, err := contractapi.NewChaincode(ssmContract)
	if err != nil {
		log.Panicf("Error creating SSM chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting SSM chaincode: %v", err)
	}
}
