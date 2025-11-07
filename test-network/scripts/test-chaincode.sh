#!/bin/bash
#
# Test Chaincode Script
# Automates chaincode deployment and testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
CHANNEL_NAME="testchannel"
CHAINCODE_NAME="ssm"
CHAINCODE_VERSION="2.0"
CHAINCODE_PATH="../chaincode/go/ssm"
CHAINCODE_LABEL="ssm_2.0"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Blockchain-SSM Chaincode Test Script${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Function to print step
print_step() {
    echo -e "${YELLOW}>>> $1${NC}"
}

# Function to print success
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print error
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Step 1: Build chaincode
print_step "Step 1: Building chaincode"
cd $CHAINCODE_PATH
if go build; then
    print_success "Chaincode built successfully"
else
    print_error "Chaincode build failed"
    exit 1
fi
cd - > /dev/null

# Step 2: Package chaincode
print_step "Step 2: Packaging chaincode"
peer lifecycle chaincode package ${CHAINCODE_NAME}.tar.gz \
    --path ${CHAINCODE_PATH} \
    --lang golang \
    --label ${CHAINCODE_LABEL}

if [ $? -eq 0 ]; then
    print_success "Chaincode packaged"
else
    print_error "Packaging failed"
    exit 1
fi

# Step 3: Install chaincode
print_step "Step 3: Installing chaincode on peer"
peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz

if [ $? -eq 0 ]; then
    print_success "Chaincode installed"
else
    print_error "Installation failed"
    exit 1
fi

# Step 4: Query installed chaincode to get package ID
print_step "Step 4: Querying package ID"
PACKAGE_ID=$(peer lifecycle chaincode queryinstalled | grep ${CHAINCODE_LABEL} | awk '{print $3}' | sed 's/,$//')

if [ -z "$PACKAGE_ID" ]; then
    print_error "Could not determine package ID"
    exit 1
fi

print_success "Package ID: $PACKAGE_ID"

# Step 5: Approve chaincode for organization
print_step "Step 5: Approving chaincode for Org1"
peer lifecycle chaincode approveformyorg \
    -o orderer.example.com:7050 \
    --channelID ${CHANNEL_NAME} \
    --name ${CHAINCODE_NAME} \
    --version ${CHAINCODE_VERSION} \
    --package-id ${PACKAGE_ID} \
    --sequence 1 \
    --tls \
    --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem

if [ $? -eq 0 ]; then
    print_success "Chaincode approved"
else
    print_error "Approval failed"
    exit 1
fi

# Step 6: Check commit readiness
print_step "Step 6: Checking commit readiness"
peer lifecycle chaincode checkcommitreadiness \
    --channelID ${CHANNEL_NAME} \
    --name ${CHAINCODE_NAME} \
    --version ${CHAINCODE_VERSION} \
    --sequence 1 \
    --tls \
    --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
    --output json

# Step 7: Commit chaincode
print_step "Step 7: Committing chaincode"
peer lifecycle chaincode commit \
    -o orderer.example.com:7050 \
    --channelID ${CHANNEL_NAME} \
    --name ${CHAINCODE_NAME} \
    --version ${CHAINCODE_VERSION} \
    --sequence 1 \
    --tls \
    --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
    --peerAddresses peer0.org1.example.com:7051 \
    --tlsRootCertFiles /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt

if [ $? -eq 0 ]; then
    print_success "Chaincode committed"
else
    print_error "Commit failed"
    exit 1
fi

# Step 8: Query committed chaincode
print_step "Step 8: Verifying committed chaincode"
peer lifecycle chaincode querycommitted \
    --channelID ${CHANNEL_NAME} \
    --name ${CHAINCODE_NAME} \
    --tls \
    --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
    --peerAddresses peer0.org1.example.com:7051 \
    --tlsRootCertFiles /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt

if [ $? -eq 0 ]; then
    print_success "Chaincode deployment verified"
else
    print_error "Verification failed"
    exit 1
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Chaincode Deployment Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Channel: ${CHANNEL_NAME}"
echo "Chaincode: ${CHAINCODE_NAME}"
echo "Version: ${CHAINCODE_VERSION}"
echo "Package ID: ${PACKAGE_ID}"
echo ""
echo "You can now run tests or interact with the chaincode."
