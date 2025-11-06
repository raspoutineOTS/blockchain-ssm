/**
 * ITMO Article 6 State Lifecycle Example
 *
 * This example demonstrates the complete lifecycle of ITMO state tracking:
 * 1. Create ITMO state record (referencing ITMO in national registry)
 * 2. Perform state transitions (authorize, transfer, hold)
 * 3. Tokenize the state (fractional ownership)
 * 4. Collateralize the state (mint stablecoins)
 * 5. Release collateral
 * 6. Query state history
 *
 * Philosophy: We track STATES, not ITMOs. ITMOs stay in their registries.
 * This is like SWIFT for banking - messages about assets, not asset transfer.
 */

const { Gateway, Wallets } = require('fabric-network');
const path = require('path');
const fs = require('fs');
const crypto = require('crypto');

// Utility: Sign data with private key
function signData(data, privateKeyPem) {
    const sign = crypto.createSign('SHA256');
    sign.update(data);
    sign.end();
    return sign.sign(privateKeyPem, 'base64');
}

async function main() {
    try {
        console.log('=== ITMO Article 6 State Lifecycle Example ===\n');

        // Load connection profile and wallet
        const ccpPath = path.resolve(__dirname, '..', 'connection-profile.json');
        const ccp = JSON.parse(fs.readFileSync(ccpPath, 'utf8'));

        const walletPath = path.join(process.cwd(), 'wallet');
        const wallet = await Wallets.newFileSystemWallet(walletPath);

        // Create gateway
        const gateway = new Gateway();
        await gateway.connect(ccp, {
            wallet,
            identity: 'user1',
            discovery: { enabled: true, asLocalhost: true }
        });

        // Get network and contract
        const network = await gateway.getNetwork('mychannel');
        const contract = network.getContract('ssm');

        // Load user keys for signing
        const privateKey = fs.readFileSync(path.resolve(__dirname, '..', 'keys', 'user1.key'), 'utf8');

        console.log('✓ Connected to Hyperledger Fabric network\n');

        // ========================================================================
        // Step 1: Create ITMO State Record
        // ========================================================================
        console.log('--- Step 1: Create ITMO State Record ---');
        console.log('Remember: The ITMO stays in its national registry (e.g., Brazil)');
        console.log('We only create a STATE RECORD to track it\n');

        const itmoStateRecord = {
            stateRecordId: 'STATE_BRA_001',
            sessionId: 'SESSION_ITMO_001',
            itmoReference: {
                serialNumber: 'BRA-2024-001-0001',
                registryId: 'brazil_national_registry',
                registryType: 'national',
                originCountry: 'BRA',
                currentCountry: 'BRA',
                quantity: 1000.0,
                vintageYear: 2024,
                projectId: 'BRA-AMAZ-001',
                projectName: 'Amazon Rainforest Conservation',
                methodology: 'VM0015',
                sector: 'Forestry',
                registryUrl: 'https://brazil-registry.gov.br/itmo/BRA-2024-001-0001',
                lastVerifiedAt: new Date().toISOString()
            },
            currentState: 'issued',
            article6Type: '6.2',
            stateController: 'user1',
            authorizedUsers: ['user1', 'user2']
        };

        const stateRecordJSON = JSON.stringify(itmoStateRecord);
        const signature1 = signData(stateRecordJSON, privateKey);

        await contract.submitTransaction(
            'CreateITMOStateRecord',
            stateRecordJSON,
            'user1',
            signature1
        );

        console.log('✓ ITMO state record created: STATE_BRA_001');
        console.log('  Serial Number: BRA-2024-001-0001');
        console.log('  Registry: Brazil National Registry');
        console.log('  Quantity: 1000 tCO2e');
        console.log('  Current State: issued\n');

        // ========================================================================
        // Step 2: Authorize ITMO for International Transfer
        // ========================================================================
        console.log('--- Step 2: Authorize for International Transfer ---');
        console.log('This is Article 6.2 authorization by the originating country\n');

        const authorizationData = {
            proof_hash: crypto.createHash('sha256')
                .update('Authorization document from Brazil Government')
                .digest('hex'),
            reason: 'Authorized by Brazil for transfer to Switzerland under bilateral agreement'
        };

        const authJSON = JSON.stringify(authorizationData);
        const signature2 = signData(authJSON, privateKey);

        await contract.submitTransaction(
            'PerformITMOStateTransition',
            'STATE_BRA_001',
            'authorize',
            authJSON,
            'user1',
            signature2
        );

        console.log('✓ ITMO state transitioned: issued → authorized');
        console.log('  Action: authorize');
        console.log('  Authorization proof recorded on blockchain\n');

        // ========================================================================
        // Step 3: Initiate Corresponding Adjustment
        // ========================================================================
        console.log('--- Step 3: Initiate Corresponding Adjustment ---');
        console.log('Article 6.2 requires corresponding adjustments to prevent double counting\n');

        const correspondingAdjustment = {
            transferringParty: 'BRA',
            acquiringParty: 'CHE',
            quantity: 1000.0,
            firstTransferYear: 2024,
            adjustmentStatus: 'pending',
            transferringPartyCA: true,
            acquiringPartyCA: false,
            a6dReported: false,
            proofHash: crypto.createHash('sha256')
                .update('CA documentation Brazil-Switzerland')
                .digest('hex'),
            initiatedAt: new Date().toISOString()
        };

        const caJSON = JSON.stringify(correspondingAdjustment);
        const signature3 = signData(caJSON, privateKey);

        await contract.submitTransaction(
            'UpdateCorrespondingAdjustment',
            'STATE_BRA_001',
            caJSON,
            'user1',
            signature3
        );

        console.log('✓ Corresponding Adjustment initiated');
        console.log('  Transferring Party: Brazil (BRA)');
        console.log('  Acquiring Party: Switzerland (CHE)');
        console.log('  Quantity: 1000 tCO2e');
        console.log('  Status: pending (awaiting acquiring party CA)\n');

        // ========================================================================
        // Step 4: Transfer State (Simulate International Transfer)
        // ========================================================================
        console.log('--- Step 4: Transfer ITMO State ---');
        console.log('The ITMO moves in the registry, we update the state\n');

        const transferData = {
            proof_hash: crypto.createHash('sha256')
                .update('Transfer receipt from Brazil to Switzerland')
                .digest('hex'),
            reason: 'International transfer under Paris Agreement Article 6.2'
        };

        const transferJSON = JSON.stringify(transferData);
        const signature4 = signData(transferJSON, privateKey);

        // First transition to pending_ca, then to transferred after CA
        await contract.submitTransaction(
            'PerformITMOStateTransition',
            'STATE_BRA_001',
            'initiate_ca',
            transferJSON,
            'user1',
            signature4
        );

        console.log('✓ State transitioned: authorized → pending_ca');

        // Simulate CA completion
        const caCompleteData = {
            proof_hash: crypto.createHash('sha256')
                .update('CA confirmation from both parties')
                .digest('hex'),
            reason: 'Both parties applied corresponding adjustments'
        };

        const caCompleteJSON = JSON.stringify(caCompleteData);
        const signature5 = signData(caCompleteJSON, privateKey);

        await contract.submitTransaction(
            'PerformITMOStateTransition',
            'STATE_BRA_001',
            'complete_ca',
            caCompleteJSON,
            'user1',
            signature5
        );

        console.log('✓ State transitioned: pending_ca → transferred');
        console.log('  Corresponding Adjustments complete\n');

        // ========================================================================
        // Step 5: Hold State
        // ========================================================================
        console.log('--- Step 5: Hold ITMO State ---');

        const holdData = {
            proof_hash: crypto.createHash('sha256')
                .update('Hold confirmation')
                .digest('hex'),
            reason: 'Held by Swiss entity for potential collateralization'
        };

        const holdJSON = JSON.stringify(holdData);
        const signature6 = signData(holdJSON, privateKey);

        await contract.submitTransaction(
            'PerformITMOStateTransition',
            'STATE_BRA_001',
            'hold',
            holdJSON,
            'user1',
            signature6
        );

        console.log('✓ State transitioned: transferred → held');
        console.log('  Ready for tokenization and collateralization\n');

        // ========================================================================
        // Step 6: Tokenize ITMO State
        // ========================================================================
        console.log('--- Step 6: Tokenize ITMO State ---');
        console.log('Create tokens representing ownership rights to the state\n');

        const tokenizationData = {
            stateRecordId: 'STATE_BRA_001',
            totalSupply: 1000.0 // 1000 tokens (1 token = 1 tCO2e state right)
        };

        const tokenJSON = JSON.stringify(tokenizationData);
        const signature7 = signData(tokenJSON, privateKey);

        await contract.submitTransaction(
            'TokenizeITMOState',
            tokenJSON,
            'user1',
            signature7
        );

        console.log('✓ ITMO state tokenized');
        console.log('  Token ID: TOKEN_STATE_BRA_001');
        console.log('  Total Supply: 1000 tokens');
        console.log('  Owner: user1');
        console.log('  Enables fractional ownership and trading\n');

        // ========================================================================
        // Step 7: Collateralize State for Stablecoin
        // ========================================================================
        console.log('--- Step 7: Collateralize State (Key Innovation!) ---');
        console.log('Use the STATE as collateral to mint stablecoins');
        console.log('The actual ITMO stays in the registry\n');

        const currentPrice = 15.0; // $15 per tCO2e
        const collateralQty = 1000.0; // Use all 1000 tCO2e
        const collateralValue = currentPrice * collateralQty; // $15,000
        const collateralRatio = 1.5; // 150% collateralization
        const stablecoinMinted = collateralValue / collateralRatio; // $10,000

        const collateralData = {
            positionId: 'POS_STATE_BRA_001',
            stateRecordId: 'STATE_BRA_001',
            collateralQty: collateralQty,
            totalQty: 1000.0,
            collateralPct: 100.0,
            stablecoinMinted: stablecoinMinted,
            stablecoinType: 'USDC',
            collateralRatio: collateralRatio,
            liquidationPrice: 10.0, // Liquidate if price drops to $10
            currentPrice: currentPrice,
            healthFactor: collateralRatio,
            owner: 'user1',
            canTransferState: false,
            canRetireItmo: false
        };

        const collateralJSON = JSON.stringify(collateralData);
        const signature8 = signData(collateralJSON, privateKey);

        await contract.submitTransaction(
            'CollateralizeITMOState',
            collateralJSON,
            'user1',
            signature8
        );

        console.log('✓ ITMO state collateralized');
        console.log('  Position ID: POS_STATE_BRA_001');
        console.log('  Collateral: 1000 tCO2e @ $15 = $15,000');
        console.log('  Stablecoin Minted: 10,000 USDC');
        console.log('  Collateral Ratio: 150%');
        console.log('  Health Factor: 1.5 (healthy)');
        console.log('  Liquidation Price: $10/tCO2e');
        console.log('  State transitioned: held → collateral\n');

        console.log('📊 Key Innovation:');
        console.log('  - The ITMO (BRA-2024-001-0001) is still in Brazil Registry');
        console.log('  - We only collateralized the STATE tracking record');
        console.log('  - This enables DeFi on carbon credits without custody\n');

        // ========================================================================
        // Step 8: Query State History
        // ========================================================================
        console.log('--- Step 8: Query State History ---\n');

        const historyResult = await contract.evaluateTransaction(
            'GetITMOStateHistory',
            'STATE_BRA_001'
        );

        const history = JSON.parse(historyResult.toString());
        console.log('State Transition History:');
        history.forEach((transition, index) => {
            console.log(`  ${index + 1}. ${transition.fromState || 'none'} → ${transition.toState}`);
            console.log(`     Action: ${transition.action}`);
            console.log(`     Actor: ${transition.actor}`);
            console.log(`     Timestamp: ${new Date(transition.timestamp).toLocaleString()}`);
            if (transition.proofHash) {
                console.log(`     Proof: ${transition.proofHash.substring(0, 16)}...`);
            }
        });
        console.log();

        // ========================================================================
        // Step 9: Get Current State
        // ========================================================================
        console.log('--- Step 9: Get Current State ---\n');

        const stateResult = await contract.evaluateTransaction(
            'GetITMOStateRecord',
            'STATE_BRA_001'
        );

        const currentState = JSON.parse(stateResult.toString());
        console.log('Current ITMO State Record:');
        console.log(`  State Record ID: ${currentState.stateRecordId}`);
        console.log(`  Current State: ${currentState.currentState}`);
        console.log(`  ITMO Serial: ${currentState.itmoReference.serialNumber}`);
        console.log(`  Registry: ${currentState.itmoReference.registryId}`);
        console.log(`  Country: ${currentState.itmoReference.currentCountry}`);
        console.log(`  Quantity: ${currentState.itmoReference.quantity} tCO2e`);
        console.log(`  Is Tokenized: ${currentState.isTokenized}`);
        console.log(`  Is Collateral: ${currentState.isCollateral}`);
        if (currentState.collateralDetails) {
            console.log(`  Collateral Position: ${currentState.collateralDetails.positionId}`);
            console.log(`  Stablecoin Minted: ${currentState.collateralDetails.stablecoinMinted} USDC`);
            console.log(`  Health Factor: ${currentState.collateralDetails.healthFactor}`);
        }
        console.log();

        // ========================================================================
        // Step 10: Query Collateralized ITMOs
        // ========================================================================
        console.log('--- Step 10: Query All Collateralized ITMOs ---\n');

        const collateralizedResult = await contract.evaluateTransaction(
            'QueryCollateralizedITMOs'
        );

        const collateralizedITMOs = JSON.parse(collateralizedResult.toString());
        console.log(`Found ${collateralizedITMOs.length} collateralized ITMO state(s):`);
        collateralizedITMOs.forEach((itmo, index) => {
            console.log(`  ${index + 1}. ${itmo.stateRecordId}`);
            console.log(`     Serial: ${itmo.itmoReference.serialNumber}`);
            console.log(`     Quantity: ${itmo.itmoReference.quantity} tCO2e`);
            console.log(`     Collateral Value: $${itmo.collateralDetails.currentPrice * itmo.collateralDetails.collateralQty}`);
        });
        console.log();

        // ========================================================================
        // Disconnect
        // ========================================================================
        await gateway.disconnect();

        console.log('=== Example Complete ===\n');
        console.log('Summary:');
        console.log('1. ✓ Created state record for ITMO (ITMO stays in Brazil)');
        console.log('2. ✓ Authorized for international transfer');
        console.log('3. ✓ Applied corresponding adjustments');
        console.log('4. ✓ Transferred state (with CA)');
        console.log('5. ✓ Held state');
        console.log('6. ✓ Tokenized state (1000 tokens)');
        console.log('7. ✓ Collateralized state ($10,000 USDC minted)');
        console.log('8. ✓ Queried full state history');
        console.log();
        console.log('Key Insight: SWIFT-like Model');
        console.log('- SWIFT: Messages about money, money stays in banks');
        console.log('- This System: States about ITMOs, ITMOs stay in registries');
        console.log('- Result: DeFi on carbon credits without custody issues');

    } catch (error) {
        console.error(`Error: ${error}`);
        process.exit(1);
    }
}

// Run the example
main().then(() => {
    console.log('\n✓ ITMO lifecycle example completed successfully');
}).catch((error) => {
    console.error('\n✗ ITMO lifecycle example failed:');
    console.error(error);
    process.exit(1);
});
