"""
ITMO State Synchronization Example

Demonstrates how to use the ITMO State Sync Service to:
1. Connect to UNFCCC A6D and national registries
2. Poll for state updates
3. Sync states to blockchain
4. Handle corresponding adjustments
5. Process state messages (SWIFT-like)

This service runs continuously to keep blockchain states in sync with registries.
"""

import asyncio
import sys
import os
import logging
from datetime import datetime, timedelta

# Add parent directory to path for imports
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'registry-connectors'))

from itmo_state_sync_service import (
    ITMOStateSyncService,
    ITMOStateMessage,
    MessageType
)

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class MockBlockchainClient:
    """Mock blockchain client for demonstration"""

    def __init__(self):
        self.state_records = {}
        self.tracked_itmos = []

    async def get_tracked_itmos(self, registry_id):
        """Get ITMOs being tracked from a specific registry"""
        return [itmo for itmo in self.tracked_itmos if itmo.registry_id == registry_id]

    async def get_state_record(self, serial_number):
        """Get blockchain state record for an ITMO"""
        return self.state_records.get(serial_number)

    async def perform_ssm_transition(self, context):
        """Perform SSM state transition"""
        logger.info(f"SSM Transition: {context['action']} for session {context['session']}")
        logger.info(f"New state: {context['public']['current_state']}")

    async def create_state_record(self, record_data):
        """Create new state record"""
        state_record_id = record_data['state_record_id']
        self.state_records[state_record_id] = record_data
        logger.info(f"Created state record: {state_record_id}")
        return state_record_id

    async def update_state_record(self, state_record_id, updates):
        """Update existing state record"""
        if state_record_id in self.state_records:
            self.state_records[state_record_id].update(updates)
            logger.info(f"Updated state record: {state_record_id}")

    async def create_state_token(self, token_data):
        """Create state token"""
        token_id = f"TOKEN_{token_data['state_record_id']}"
        logger.info(f"Created state token: {token_id}")
        return token_id

    async def update_corresponding_adjustment(self, state_record_id, ca_details):
        """Update corresponding adjustment information"""
        logger.info(f"Updated CA for {state_record_id}")
        logger.info(f"Transferring: {ca_details.get('transferringParty')} -> "
                   f"Acquiring: {ca_details.get('acquiringParty')}")


class MockUNFCCCConnector:
    """Mock UNFCCC A6D connector for demonstration"""

    async def poll_state_updates(self, last_sync):
        """Poll for state updates from UNFCCC A6D"""
        # Simulate finding some updates
        logger.info(f"Polling UNFCCC A6D for updates since {last_sync}")

        # Return mock updates
        class MockUpdate:
            def __init__(self):
                self.serial_number = "BRA-2024-001-0001"
                self.previous_state = "issued"
                self.new_state = "authorized"
                self.proof_hash = "abc123def456"
                self.update_type = "authorization"
                self.notification_id = "A6D_NOTIF_001"

        return [MockUpdate()]


class MockVerraConnector:
    """Mock Verra registry connector for demonstration"""

    async def fetch_itmo_state(self, serial_number):
        """Fetch ITMO state from Verra"""
        logger.info(f"Fetching state for {serial_number} from Verra")

        # Return mock state
        class MockState:
            def __init__(self):
                self.current_state = "held"
                self.last_updated = datetime.utcnow()
                self.proof_hash = "verra_proof_123"

        return MockState()


async def demonstrate_sync_service():
    """Demonstrate ITMO state synchronization"""

    print("=" * 80)
    print("ITMO STATE SYNCHRONIZATION SERVICE DEMONSTRATION")
    print("=" * 80)
    print()

    print("Philosophy: Track STATES, not ITMOs (like SWIFT for carbon credits)")
    print()

    # Create mock clients
    blockchain_client = MockBlockchainClient()
    unfccc_connector = MockUNFCCCConnector()

    registry_connectors = {
        "verra": MockVerraConnector(),
        "unfccc_a6d": unfccc_connector,
    }

    # Create sync service
    print("Step 1: Initialize ITMO State Sync Service")
    print("-" * 80)
    sync_service = ITMOStateSyncService(
        registry_connectors=registry_connectors,
        blockchain_client=blockchain_client,
        unfccc_connector=unfccc_connector,
        sync_interval=60  # 1 minute for demo
    )
    print("✓ Sync service initialized")
    print(f"  Sync interval: 60 seconds")
    print(f"  Connected registries: {list(registry_connectors.keys())}")
    print()

    # Create an ITMO state record
    print("Step 2: Create Initial ITMO State Record")
    print("-" * 80)
    print("Remember: The ITMO stays in Brazil's registry")
    print("We only create a state record to track it")
    print()

    itmo_reference = {
        "serial_number": "BRA-2024-001-0001",
        "registry_id": "brazil_national_registry",
        "origin_country": "BRA",
        "current_country": "BRA",
        "quantity": 1000.0,
        "vintage_year": 2024,
        "project_id": "BRA-AMAZ-001",
    }

    state_record_id = await sync_service.create_state_record(itmo_reference)
    print(f"✓ State record created: {state_record_id}")
    print(f"  ITMO Serial: {itmo_reference['serial_number']}")
    print(f"  Registry: {itmo_reference['registry_id']}")
    print(f"  Initial State: issued")
    print()

    # Sync from UNFCCC A6D
    print("Step 3: Sync States from UNFCCC A6D")
    print("-" * 80)
    print("Polling UNFCCC Article 6 Database for state updates...")
    print()

    await sync_service.sync_from_unfccc()

    print(f"✓ Found {len(sync_service.message_queue)} state update message(s)")
    print()

    # Process messages
    print("Step 4: Process State Messages (SWIFT-like)")
    print("-" * 80)
    print("Processing queued state messages...")
    print()

    await sync_service.process_message_queue()

    print(f"✓ Processed {len(sync_service.processed_messages)} message(s)")
    print()

    # Create manual state message for transfer
    print("Step 5: Simulate International Transfer")
    print("-" * 80)
    print("Brazil → Switzerland transfer with Corresponding Adjustment")
    print()

    transfer_message = ITMOStateMessage(
        message_type=MessageType.STATE_TRANSITION,
        state_record_id=state_record_id,
        serial_number="BRA-2024-001-0001",
        registry_id="brazil_national_registry"
    )
    transfer_message.from_state = "authorized"
    transfer_message.to_state = "transferred"
    transfer_message.proof_hash = "transfer_proof_xyz789"
    transfer_message.ca_details = {
        "transferringParty": "BRA",
        "acquiringParty": "CHE",
        "quantity": 1000.0,
        "firstTransferYear": 2024,
        "adjustmentStatus": "complete",
    }

    sync_service.message_queue.append(transfer_message)
    print("✓ Transfer message created")
    print(f"  From: Brazil (BRA)")
    print(f"  To: Switzerland (CHE)")
    print(f"  Quantity: 1000 tCO2e")
    print(f"  State: authorized → transferred")
    print()

    await sync_service.process_message_queue()
    print("✓ Transfer message processed")
    print()

    # Tokenize state
    print("Step 6: Tokenize ITMO State")
    print("-" * 80)
    print("Create token representing rights to the state")
    print()

    token_id = await sync_service.tokenize_state(
        state_record_id=state_record_id,
        total_supply=1000.0
    )

    print(f"✓ State tokenized: {token_id}")
    print(f"  Total Supply: 1000 tokens")
    print(f"  Enables fractional ownership and DeFi")
    print()

    # Get metrics
    print("Step 7: View Sync Metrics")
    print("-" * 80)
    metrics = sync_service.get_sync_metrics()
    print(f"Sync Status:")
    print(f"  Running: {metrics['is_running']}")
    print(f"  Messages Processed: {metrics['processed_count']}")
    print(f"  Queue Length: {metrics['message_queue_length']}")
    print(f"  Last Sync Times: {metrics['last_sync_times']}")
    print()

    # Message queue status
    queue_status = sync_service.get_message_queue_status()
    print(f"Message Queue:")
    print(f"  Queued: {queue_status['queued_messages']}")
    print(f"  Processed: {queue_status['processed_messages']}")
    print(f"  Total: {queue_status['total_messages']}")
    print()

    # Show processed messages
    print("Step 8: Review Processed Messages")
    print("-" * 80)
    for i, msg in enumerate(sync_service.processed_messages, 1):
        print(f"{i}. Message ID: {msg.message_id}")
        print(f"   Type: {msg.message_type.value}")
        print(f"   Serial: {msg.serial_number}")
        print(f"   Transition: {msg.from_state} → {msg.to_state}")
        if msg.ca_details:
            print(f"   CA: {msg.ca_details['transferringParty']} → "
                  f"{msg.ca_details['acquiringParty']}")
        print()

    print("=" * 80)
    print("DEMONSTRATION COMPLETE")
    print("=" * 80)
    print()

    print("Key Takeaways:")
    print("1. ITMOs remain in their national/international registries")
    print("2. Blockchain only tracks STATE changes (like SWIFT messages)")
    print("3. States can be tokenized and used as collateral")
    print("4. Corresponding Adjustments tracked without moving ITMOs")
    print("5. Real-time sync with UNFCCC A6D and registries")
    print("6. Enables DeFi on carbon credits without custody")
    print()


async def demonstrate_continuous_sync():
    """Demonstrate continuous synchronization (abbreviated)"""

    print("\n" + "=" * 80)
    print("CONTINUOUS SYNC MODE (Press Ctrl+C to stop)")
    print("=" * 80)
    print()

    # Create mock clients
    blockchain_client = MockBlockchainClient()
    unfccc_connector = MockUNFCCCConnector()

    registry_connectors = {
        "verra": MockVerraConnector(),
    }

    # Create sync service
    sync_service = ITMOStateSyncService(
        registry_connectors=registry_connectors,
        blockchain_client=blockchain_client,
        unfccc_connector=unfccc_connector,
        sync_interval=10  # 10 seconds for demo
    )

    print("Starting continuous sync worker...")
    print("The worker will:")
    print("  - Poll UNFCCC A6D every 10 seconds")
    print("  - Poll registries for tracked ITMOs")
    print("  - Process state messages")
    print("  - Update blockchain state records")
    print()

    # Start sync worker (this runs indefinitely)
    try:
        await sync_service.start_sync_worker()
    except KeyboardInterrupt:
        print("\n\nStopping sync worker...")
        sync_service.stop_sync_worker()
        print("✓ Sync worker stopped")


async def main():
    """Main demonstration"""

    print("""
╔════════════════════════════════════════════════════════════════════════════╗
║                                                                            ║
║          ITMO ARTICLE 6 STATE SYNCHRONIZATION SERVICE                     ║
║                                                                            ║
║  Like SWIFT for Banking → State Tracking for Carbon Credits               ║
║                                                                            ║
║  - ITMOs stay in national registries (source of truth)                    ║
║  - Blockchain tracks STATES only                                          ║
║  - States can be tokenized and collateralized                             ║
║  - Full Paris Agreement Article 6 compliance                              ║
║                                                                            ║
╚════════════════════════════════════════════════════════════════════════════╝
    """)

    print("\nChoose demonstration mode:")
    print("1. Step-by-step demonstration")
    print("2. Continuous sync mode")
    print()

    try:
        choice = input("Enter choice (1 or 2): ").strip()

        if choice == "1":
            await demonstrate_sync_service()
        elif choice == "2":
            await demonstrate_continuous_sync()
        else:
            print("Invalid choice. Running step-by-step demonstration.")
            await demonstrate_sync_service()

    except KeyboardInterrupt:
        print("\n\nDemonstration interrupted by user")


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\n✓ Example terminated")
