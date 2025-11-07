/**
 * Complete DeFi Example for ITMO States
 *
 * This example demonstrates the full DeFi stack:
 * 1. Price Oracle - Multi-source carbon credit pricing
 * 2. Lending Protocol - Supply and earn interest
 * 3. Borrowing - Borrow against ITMO collateral
 * 4. Liquidation - Automatic liquidation of unhealthy positions
 * 5. Liquidity Pools - AMM-style trading
 * 6. Swaps - Token swapping
 *
 * Complete DeFi ecosystem for carbon credits!
 */

const { Gateway, Wallets } = require('fabric-network');
const path = require('path');
const fs = require('fs');
const crypto = require('crypto');

// Utility: Sign data
function signData(data, privateKeyPem) {
    const sign = crypto.createSign('SHA256');
    sign.update(data);
    sign.end();
    return sign.sign(privateKeyPem, 'base64');
}

async function main() {
    try {
        console.log('╔════════════════════════════════════════════════════════════════╗');
        console.log('║   COMPLETE DEFI EXAMPLE FOR ITMO STATES                       ║');
        console.log('║   Full DeFi Stack: Oracle + Lending + AMM + Liquidation       ║');
        console.log('╚════════════════════════════════════════════════════════════════╝');
        console.log();

        // Setup
        const ccpPath = path.resolve(__dirname, '..', 'connection-profile.json');
        const ccp = JSON.parse(fs.readFileSync(ccpPath, 'utf8'));

        const walletPath = path.join(process.cwd(), 'wallet');
        const wallet = await Wallets.newFileSystemWallet(walletPath);

        const gateway = new Gateway();
        await gateway.connect(ccp, {
            wallet,
            identity: 'user1',
            discovery: { enabled: true, asLocalhost: true }
        });

        const network = await gateway.getNetwork('mychannel');
        const contract = network.getContract('ssm');

        const privateKey = fs.readFileSync(path.resolve(__dirname, '..', 'keys', 'user1.key'), 'utf8');

        console.log('✓ Connected to Hyperledger Fabric network\n');

        // ====================================================================
        // PART 1: PRICE ORACLE
        // ====================================================================
        console.log('═══ PART 1: PRICE ORACLE ═══\n');

        console.log('Step 1.1: Create Price Oracle for ITMO states');
        const oracle = {
            assetType: 'itmo_state',
            currentPrice: 15.0,
            priceSources: [
                {
                    sourceId: 'verra_api',
                    price: 15.2,
                    weight: 0.4,
                    isActive: true,
                    reliabilityScore: 0.9
                },
                {
                    sourceId: 'gold_standard',
                    price: 14.8,
                    weight: 0.3,
                    isActive: true,
                    reliabilityScore: 0.85
                },
                {
                    sourceId: 'exchange_coinbase',
                    price: 15.0,
                    weight: 0.3,
                    isActive: true,
                    reliabilityScore: 0.95
                }
            ],
            aggregationMethod: 'weighted_average',
            updateFrequency: 300 // 5 minutes
        };

        const oracleJSON = JSON.stringify(oracle);
        const sig1 = signData(oracleJSON, privateKey);

        await contract.submitTransaction(
            'CreatePriceOracle',
            oracleJSON,
            'admin1',
            sig1
        );

        console.log('✓ Price oracle created');
        console.log(`  Asset Type: ${oracle.assetType}`);
        console.log(`  Sources: ${oracle.priceSources.length}`);
        console.log(`  Aggregation: ${oracle.aggregationMethod}`);
        console.log(`  Initial Price: $${oracle.currentPrice}/tCO2e`);
        console.log();

        console.log('Step 1.2: Update price with new data');
        const priceUpdates = [
            { sourceId: 'verra_api', price: 15.5, isActive: true },
            { sourceId: 'gold_standard', price: 15.3, isActive: true },
            { sourceId: 'exchange_coinbase', price: 15.4, isActive: true }
        ];

        const updateJSON = JSON.stringify(priceUpdates);
        const sig2 = signData(updateJSON, privateKey);

        await contract.submitTransaction(
            'UpdatePrice',
            'itmo_state',
            updateJSON,
            'admin1',
            sig2
        );

        const currentPriceResult = await contract.evaluateTransaction(
            'GetCurrentPrice',
            'itmo_state'
        );
        const currentPrice = parseFloat(currentPriceResult.toString());

        console.log('✓ Price updated');
        console.log(`  New Price: $${currentPrice.toFixed(2)}/tCO2e`);
        console.log(`  Change: +${((currentPrice - 15.0) / 15.0 * 100).toFixed(2)}%`);
        console.log();

        // ====================================================================
        // PART 2: LENDING PROTOCOL - SUPPLY
        // ====================================================================
        console.log('═══ PART 2: LENDING PROTOCOL ═══\n');

        console.log('Step 2.1: Create Lending Pool');
        const lendingPool = {
            poolId: 'ITMO_LENDING_POOL_1',
            assetType: 'itmo_state',
            baseRate: 0.02,        // 2% base rate
            optimalUtilization: 0.80, // 80% optimal
            slope1: 0.05,          // 5% slope before optimal
            slope2: 0.50,          // 50% slope after optimal
            maxLoanToValue: 0.75,  // 75% max LTV
            liquidationThreshold: 0.85, // 85% liquidation
            liquidationPenalty: 0.10,   // 10% penalty
            reserveFactor: 0.10    // 10% to reserve
        };

        const poolJSON = JSON.stringify(lendingPool);
        const sig3 = signData(poolJSON, privateKey);

        await contract.submitTransaction(
            'CreateLendingPool',
            poolJSON,
            'admin1',
            sig3
        );

        console.log('✓ Lending pool created');
        console.log(`  Pool ID: ${lendingPool.poolId}`);
        console.log(`  Max LTV: ${lendingPool.maxLoanToValue * 100}%`);
        console.log(`  Liquidation Threshold: ${lendingPool.liquidationThreshold * 100}%`);
        console.log();

        console.log('Step 2.2: Supply ITMO state to earn interest');
        // First create an ITMO state to supply
        const stateRecord = {
            stateRecordId: 'STATE_SUPPLY_001',
            itmoReference: {
                serialNumber: 'BRA-2024-SUPPLY-001',
                registryId: 'brazil_registry',
                registryType: 'national',
                originCountry: 'BRA',
                currentCountry: 'BRA',
                quantity: 500.0,
                vintageYear: 2024
            },
            currentState: 'held',
            stateController: 'user1',
            authorizedUsers: ['user1']
        };

        const stateJSON = JSON.stringify(stateRecord);
        const sig4 = signData(stateJSON, privateKey);

        await contract.submitTransaction(
            'CreateITMOStateRecord',
            stateJSON,
            'user1',
            sig4
        );

        // Now supply it to lending pool
        const supplyData = {
            poolId: 'ITMO_LENDING_POOL_1',
            stateRecordId: 'STATE_SUPPLY_001',
            amount: 500.0
        };

        const supplyJSON = JSON.stringify(supplyData);
        const sig5 = signData(supplyJSON, privateKey);

        await contract.submitTransaction(
            'Supply',
            supplyJSON,
            'user1',
            sig5
        );

        console.log('✓ ITMO state supplied to lending pool');
        console.log(`  Amount: ${supplyData.amount} tCO2e`);
        console.log(`  Earning: Supply APY (based on utilization)`);
        console.log();

        // ====================================================================
        // PART 3: BORROWING
        // ====================================================================
        console.log('═══ PART 3: BORROWING ═══\n');

        console.log('Step 3.1: Create collateral ITMO states');
        // Create collateral states for borrowing
        const collateralStates = [];
        for (let i = 1; i <= 2; i++) {
            const collateralState = {
                stateRecordId: `STATE_COLLATERAL_00${i}`,
                itmoReference: {
                    serialNumber: `CHE-2024-COLL-00${i}`,
                    registryId: 'switzerland_registry',
                    registryType: 'national',
                    originCountry: 'CHE',
                    currentCountry: 'CHE',
                    quantity: 300.0,
                    vintageYear: 2024
                },
                currentState: 'held',
                stateController: 'user2',
                authorizedUsers: ['user2']
            };

            const collStateJSON = JSON.stringify(collateralState);
            const sigColl = signData(collStateJSON, privateKey);

            await contract.submitTransaction(
                'CreateITMOStateRecord',
                collStateJSON,
                'user2',
                sigColl
            );

            collateralStates.push(collateralState.stateRecordId);
        }

        console.log(`✓ Created ${collateralStates.length} collateral ITMO states`);
        console.log(`  Total Collateral: 600 tCO2e`);
        console.log();

        console.log('Step 3.2: Borrow against collateral');
        const borrowData = {
            poolId: 'ITMO_LENDING_POOL_1',
            borrowAmount: 300.0, // Borrow 300 tCO2e
            collateralStateIds: collateralStates
        };

        // Collateral: 600 tCO2e @ $15.4 = $9,240
        // Borrow: 300 tCO2e @ $15.4 = $4,620
        // LTV: $4,620 / $9,240 = 50% (safe!)

        const borrowJSON = JSON.stringify(borrowData);
        const sig6 = signData(borrowJSON, privateKey);

        await contract.submitTransaction(
            'Borrow',
            borrowJSON,
            'user2',
            sig6
        );

        const collateralValue = 600 * currentPrice;
        const borrowValue = 300 * currentPrice;
        const ltv = borrowValue / collateralValue;
        const healthFactor = (collateralValue * 0.85) / borrowValue;

        console.log('✓ Borrow executed');
        console.log(`  Borrowed: ${borrowData.borrowAmount} tCO2e ($${borrowValue.toFixed(2)})`);
        console.log(`  Collateral: 600 tCO2e ($${collateralValue.toFixed(2)})`);
        console.log(`  LTV: ${(ltv * 100).toFixed(2)}%`);
        console.log(`  Health Factor: ${healthFactor.toFixed(3)} ✓ (healthy)`);
        console.log(`  Liquidation Price: $${(borrowValue / (600 * 0.85)).toFixed(2)}/tCO2e`);
        console.log();

        // ====================================================================
        // PART 4: LIQUIDITY POOL & SWAPS
        // ====================================================================
        console.log('═══ PART 4: LIQUIDITY POOL (AMM) ═══\n');

        console.log('Step 4.1: Create Liquidity Pool');
        const liquidityPool = {
            poolId: 'ITMO_USDC_POOL',
            tokenA: 'ITMO_STATE_TOKEN',
            tokenB: 'USDC',
            swapFee: 0.003,      // 0.3%
            protocolFee: 0.0005  // 0.05%
        };

        const liqPoolJSON = JSON.stringify(liquidityPool);
        const sig7 = signData(liqPoolJSON, privateKey);

        await contract.submitTransaction(
            'CreateLiquidityPool',
            liqPoolJSON,
            'admin1',
            sig7
        );

        console.log('✓ Liquidity pool created');
        console.log(`  Pool: ${liquidityPool.tokenA} / ${liquidityPool.tokenB}`);
        console.log(`  Swap Fee: ${liquidityPool.swapFee * 100}%`);
        console.log();

        console.log('Step 4.2: Add liquidity to pool');
        const addLiquidityData = {
            poolId: 'ITMO_USDC_POOL',
            amountA: 1000.0,  // 1000 ITMO tokens
            amountB: 15000.0, // 15000 USDC (price ~$15)
            minLpTokens: 0    // No slippage protection for first deposit
        };

        const addLiqJSON = JSON.stringify(addLiquidityData);
        const sig8 = signData(addLiqJSON, privateKey);

        await contract.submitTransaction(
            'AddLiquidity',
            addLiqJSON,
            'user1',
            sig8
        );

        const lpTokens = Math.sqrt(1000 * 15000);

        console.log('✓ Liquidity added');
        console.log(`  Token A: ${addLiquidityData.amountA} ITMO`);
        console.log(`  Token B: ${addLiquidityData.amountB} USDC`);
        console.log(`  LP Tokens Minted: ${lpTokens.toFixed(2)}`);
        console.log(`  Initial Price: $15/ITMO`);
        console.log();

        console.log('Step 4.3: Execute swap');
        // Simulate someone swapping USDC for ITMO
        const swapData = {
            poolId: 'ITMO_USDC_POOL',
            tokenIn: 'tokenB',  // Swap USDC
            amountIn: 1500.0,   // 1500 USDC
            minAmountOut: 95.0  // Expect ~100 ITMO, 5% slippage protection
        };

        // Calculate expected output
        // Formula: amountOut = (reserveOut * amountIn * (1-fee)) / (reserveIn + amountIn * (1-fee))
        const reserveITMO = 1000;
        const reserveUSDC = 15000;
        const amountInWithFee = 1500 * (1 - 0.003 - 0.0005);
        const expectedOut = (reserveITMO * amountInWithFee) / (reserveUSDC + amountInWithFee);
        const priceImpact = (expectedOut / reserveITMO) * 100;

        const swapJSON = JSON.stringify(swapData);
        const sig9 = signData(swapJSON, privateKey);

        await contract.submitTransaction(
            'Swap',
            swapJSON,
            'user3',
            sig9
        );

        console.log('✓ Swap executed');
        console.log(`  Swapped: ${swapData.amountIn} USDC`);
        console.log(`  Received: ~${expectedOut.toFixed(2)} ITMO`);
        console.log(`  Execution Price: $${(swapData.amountIn / expectedOut).toFixed(2)}/ITMO`);
        console.log(`  Price Impact: ${priceImpact.toFixed(2)}%`);
        console.log(`  Swap Fee: ${(swapData.amountIn * 0.003).toFixed(2)} USDC`);
        console.log();

        // ====================================================================
        // PART 5: PRICE CRASH & LIQUIDATION
        // ====================================================================
        console.log('═══ PART 5: PRICE CRASH & LIQUIDATION ═══\n');

        console.log('Step 5.1: Simulate price crash');
        console.log('⚠️  Carbon credit price drops due to market conditions');

        const crashPrices = [
            { sourceId: 'verra_api', price: 9.0, isActive: true },
            { sourceId: 'gold_standard', price: 8.8, isActive: true },
            { sourceId: 'exchange_coinbase', price: 9.2, isActive: true }
        ];

        const crashJSON = JSON.stringify(crashPrices);
        const sig10 = signData(crashJSON, privateKey);

        await contract.submitTransaction(
            'UpdatePrice',
            'itmo_state',
            crashJSON,
            'admin1',
            sig10
        );

        const newPriceResult = await contract.evaluateTransaction(
            'GetCurrentPrice',
            'itmo_state'
        );
        const newPrice = parseFloat(newPriceResult.toString());

        console.log(`✓ Price crashed to $${newPrice.toFixed(2)}/tCO2e`);
        console.log(`  Price Drop: ${((newPrice - currentPrice) / currentPrice * 100).toFixed(2)}%`);
        console.log();

        console.log('Step 5.2: Check borrow position health');
        // Recalculate health factor with new price
        const newCollateralValue = 600 * newPrice;
        const newHealthFactor = (newCollateralValue * 0.85) / borrowValue;

        console.log(`  New Collateral Value: $${newCollateralValue.toFixed(2)}`);
        console.log(`  Debt: $${borrowValue.toFixed(2)}`);
        console.log(`  New Health Factor: ${newHealthFactor.toFixed(3)}`);

        if (newHealthFactor < 1.0) {
            console.log(`  ⚠️  Position is UNHEALTHY - Can be liquidated!`);
            console.log();

            console.log('Step 5.3: Liquidate unhealthy position');
            const liquidateData = { positionId: 'BORROW_ITMO_LENDING_POOL_1_user2' };
            const liqDataJSON = JSON.stringify(liquidateData);
            const sig11 = signData(liqDataJSON, privateKey);

            await contract.submitTransaction(
                'Liquidate',
                liquidateData.positionId,
                'liquidator1',
                sig11
            );

            const liquidationBonus = borrowValue * 0.10 / newPrice;

            console.log('✓ Position liquidated');
            console.log(`  Liquidator: liquidator1`);
            console.log(`  Debt Repaid: $${borrowValue.toFixed(2)}`);
            console.log(`  Collateral Seized: ${((borrowValue * 1.10) / newPrice).toFixed(2)} tCO2e`);
            console.log(`  Liquidation Bonus: ${liquidationBonus.toFixed(2)} tCO2e`);
            console.log(`  Protocol Protected: ✓`);
        } else {
            console.log(`  ✓ Position is still healthy`);
        }
        console.log();

        // ====================================================================
        // PART 6: QUERY STATISTICS
        // ====================================================================
        console.log('═══ PART 6: DEFI STATISTICS ═══\n');

        console.log('Lending Pool Stats:');
        const poolStatsResult = await contract.evaluateTransaction(
            'GetLendingPool',
            'ITMO_LENDING_POOL_1'
        );
        const poolStats = JSON.parse(poolStatsResult.toString());

        console.log(`  Total Deposited: ${poolStats.totalDeposited} tCO2e`);
        console.log(`  Total Borrowed: ${poolStats.totalBorrowed} tCO2e`);
        console.log(`  Available Liquidity: ${poolStats.availableLiquidity} tCO2e`);
        console.log(`  Utilization Rate: ${(poolStats.utilizationRate * 100).toFixed(2)}%`);
        console.log(`  Supply APY: ${(poolStats.supplyAPY * 100).toFixed(2)}%`);
        console.log(`  Borrow APY: ${(poolStats.borrowAPY * 100).toFixed(2)}%`);
        console.log(`  Reserve Balance: ${poolStats.reserveBalance} tCO2e`);
        console.log();

        console.log('AMM Pool Stats:');
        const ammStatsResult = await contract.evaluateTransaction(
            'GetPoolStats',
            'ITMO_USDC_POOL'
        );
        const ammStats = JSON.parse(ammStatsResult.toString());

        console.log(`  Reserve ITMO: ${ammStats.reserveA}`);
        console.log(`  Reserve USDC: ${ammStats.reserveB}`);
        console.log(`  Current Price: $${ammStats.price.toFixed(2)}/ITMO`);
        console.log(`  TVL: $${ammStats.tvl.toFixed(2)}`);
        console.log(`  24h Volume: ${ammStats.volume24h.toFixed(2)}`);
        console.log(`  Fees Earned: $${ammStats.feesEarned.toFixed(2)}`);
        console.log(`  LP Token Holders: ${ammStats.numLPHolders}`);
        console.log();

        // Disconnect
        await gateway.disconnect();

        console.log('╔════════════════════════════════════════════════════════════════╗');
        console.log('║              COMPLETE DEFI EXAMPLE FINISHED                    ║');
        console.log('╚════════════════════════════════════════════════════════════════╝');
        console.log();

        console.log('Summary:');
        console.log('✓ Price Oracle - Multi-source aggregation with circuit breaker');
        console.log('✓ Lending Protocol - Supply, borrow, repay with dynamic rates');
        console.log('✓ Liquidation - Automatic protection for protocol');
        console.log('✓ AMM Pool - Constant product formula trading');
        console.log('✓ Swaps - Low slippage token swapping');
        console.log();

        console.log('Key Innovations:');
        console.log('1. DeFi on carbon credits WITHOUT custody');
        console.log('2. State-only tracking (ITMOs stay in registries)');
        console.log('3. Complete lending/borrowing protocol');
        console.log('4. AMM for price discovery and liquidity');
        console.log('5. Automatic liquidations for risk management');
        console.log('6. Multi-source price oracle with protection');
        console.log();

        console.log('🌍 This enables a complete DeFi ecosystem for carbon markets!');

    } catch (error) {
        console.error(`Error: ${error}`);
        process.exit(1);
    }
}

// Run the example
main().then(() => {
    console.log('\n✓ DeFi example completed successfully');
}).catch((error) => {
    console.error('\n✗ DeFi example failed:');
    console.error(error);
    process.exit(1);
});
