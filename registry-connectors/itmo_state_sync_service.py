"""
ITMO State Synchronization Service
State-only tracking service for ITMOs (like SWIFT for carbon credits)

Philosophy:
- ITMOs stay in national registries (source of truth)
- We only sync and track STATES
- State changes are messages (like SWIFT messages)
- States can be tokenized and collateralized
"""

import asyncio
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional
from enum import Enum
import hashlib
import json


logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class MessageType(Enum):
    """Types of state messages (like SWIFT MT types)"""
    STATE_SYNC = "STATE_SYNC"               # Sync state from registry
    STATE_TRANSITION = "STATE_TRANSITION"   # State change notification
    CA_UPDATE = "CA_UPDATE"                 # Corresponding adjustment update
    AUTHORIZATION = "AUTHORIZATION"         # Authorization notification
    RETIREMENT = "RETIREMENT"               # Retirement notification


class ITMOStateMessage:
    """
    State message (like SWIFT message)
    Contains information about state, not the ITMO itself
    """
    def __init__(self, message_type: MessageType, state_record_id: str,
                 serial_number: str, registry_id: str):
        self.message_id = self._generate_message_id()
        self.message_type = message_type
        self.state_record_id = state_record_id
        self.serial_number = serial_number
        self.registry_id = registry_id
        self.timestamp = datetime.utcnow()
        self.from_state = None
        self.to_state = None
        self.proof_hash = None
        self.ca_details = None
        self.metadata = {}
        self.processed = False

    def _generate_message_id(self) -> str:
        """Generate unique message ID"""
        data = f"{datetime.utcnow().isoformat()}{id(self)}"
        return f"MSG_{hashlib.sha256(data.encode()).hexdigest()[:16]}"

    def to_dict(self) -> Dict:
        """Convert to dictionary"""
        return {
            "message_id": self.message_id,
            "message_type": self.message_type.value,
            "state_record_id": self.state_record_id,
            "serial_number": self.serial_number,
            "registry_id": self.registry_id,
            "timestamp": self.timestamp.isoformat(),
            "from_state": self.from_state,
            "to_state": self.to_state,
            "proof_hash": self.proof_hash,
            "ca_details": self.ca_details,
            "metadata": self.metadata,
            "processed": self.processed,
        }

    def to_json(self) -> str:
        """Convert to JSON"""
        return json.dumps(self.to_dict(), indent=2)


class ITMOStateSyncService:
    """
    Service for syncing ITMO states (not ITMOs themselves)
    Similar to SWIFT messaging network for banking
    """

    def __init__(self, registry_connectors: Dict, blockchain_client,
                 unfccc_connector, sync_interval: int = 300):
        """
        Initialize ITMO state sync service

        Args:
            registry_connectors: Dict of registry_id -> connector (Verra, etc.)
            blockchain_client: Client for blockchain (SSM) interaction
            unfccc_connector: UNFCCC A6D connector
            sync_interval: Sync interval in seconds (default 5 minutes)
        """
        self.registry_connectors = registry_connectors
        self.blockchain = blockchain_client
        self.unfccc = unfccc_connector
        self.sync_interval = sync_interval

        # Message queue (like SWIFT message queue)
        self.message_queue = []
        self.processed_messages = []

        # Sync tracking
        self.last_sync = {}
        self.is_running = False

    async def start_sync_worker(self):
        """Start background state synchronization worker"""
        logger.info("Starting ITMO state sync worker")
        self.is_running = True

        while self.is_running:
            try:
                # Sync states from all sources
                await self.sync_all_states()

                # Process queued messages
                await self.process_message_queue()

                await asyncio.sleep(self.sync_interval)

            except Exception as e:
                logger.error(f"Sync worker error: {e}")
                await asyncio.sleep(60)

    def stop_sync_worker(self):
        """Stop synchronization worker"""
        logger.info("Stopping ITMO state sync worker")
        self.is_running = False

    async def sync_all_states(self):
        """Sync states from all registries and UNFCCC A6D"""
        logger.info("Starting state synchronization")

        # 1. Sync from UNFCCC A6D (primary source for Article 6)
        await self.sync_from_unfccc()

        # 2. Sync from national/voluntary registries
        for registry_id, connector in self.registry_connectors.items():
            try:
                await self.sync_from_registry(registry_id, connector)
            except Exception as e:
                logger.error(f"Failed to sync from {registry_id}: {e}")

        logger.info("State synchronization complete")

    async def sync_from_unfccc(self):
        """Sync states from UNFCCC A6D"""
        logger.info("Syncing states from UNFCCC A6D")

        # Get last sync time
        last_sync = self.last_sync.get("unfccc", datetime.utcnow() - timedelta(hours=1))

        # Poll for state updates
        try:
            updates = await self.unfccc.poll_state_updates(last_sync)
            logger.info(f"Found {len(updates)} state updates from UNFCCC")

            for update in updates:
                # Create state message
                message = self._create_message_from_a6d_update(update)
                self.message_queue.append(message)

            self.last_sync["unfccc"] = datetime.utcnow()

        except Exception as e:
            logger.error(f"UNFCCC sync failed: {e}")

    async def sync_from_registry(self, registry_id: str, connector):
        """Sync states from a specific registry"""
        logger.info(f"Syncing states from registry: {registry_id}")

        # Get tracked ITMOs for this registry
        tracked_itmos = await self.blockchain.get_tracked_itmos(registry_id)
        logger.info(f"Tracking {len(tracked_itmos)} ITMOs in {registry_id}")

        for itmo_ref in tracked_itmos:
            try:
                # Fetch current state from registry
                current_state = await connector.fetch_itmo_state(
                    itmo_ref.serial_number
                )

                # Get blockchain state record
                state_record = await self.blockchain.get_state_record(
                    itmo_ref.serial_number
                )

                # Compare and create message if states differ
                if self._states_differ(current_state, state_record):
                    message = self._create_state_sync_message(
                        state_record.state_record_id,
                        itmo_ref.serial_number,
                        registry_id,
                        current_state,
                        state_record
                    )
                    self.message_queue.append(message)

            except Exception as e:
                logger.error(f"Failed to sync ITMO {itmo_ref.serial_number}: {e}")

    def _create_message_from_a6d_update(self, update) -> ITMOStateMessage:
        """Create message from A6D update notification"""
        message = ITMOStateMessage(
            message_type=MessageType.STATE_TRANSITION,
            state_record_id=f"STATE_{update.serial_number}",
            serial_number=update.serial_number,
            registry_id="unfccc_a6d"
        )

        message.from_state = update.previous_state
        message.to_state = update.new_state
        message.proof_hash = update.proof_hash
        message.metadata = {
            "update_type": update.update_type,
            "notification_id": update.notification_id,
        }

        return message

    def _create_state_sync_message(self, state_record_id: str, serial_number: str,
                                   registry_id: str, registry_state, chain_state) -> ITMOStateMessage:
        """Create state sync message"""
        message = ITMOStateMessage(
            message_type=MessageType.STATE_SYNC,
            state_record_id=state_record_id,
            serial_number=serial_number,
            registry_id=registry_id
        )

        message.from_state = chain_state.current_state
        message.to_state = registry_state.current_state
        message.proof_hash = registry_state.proof_hash
        message.metadata = {
            "registry_updated_at": registry_state.last_updated.isoformat(),
            "chain_updated_at": chain_state.state_updated_at.isoformat(),
        }

        # Include CA details if available
        if hasattr(registry_state, 'corresponding_adjustment'):
            message.ca_details = registry_state.corresponding_adjustment

        return message

    async def process_message_queue(self):
        """Process queued state messages"""
        if not self.message_queue:
            return

        logger.info(f"Processing {len(self.message_queue)} state messages")

        for message in self.message_queue[:]:  # Copy to avoid modification during iteration
            try:
                await self.process_message(message)
                self.message_queue.remove(message)
                self.processed_messages.append(message)

            except Exception as e:
                logger.error(f"Failed to process message {message.message_id}: {e}")

    async def process_message(self, message: ITMOStateMessage):
        """
        Process a single state message
        Similar to processing a SWIFT message
        """
        logger.info(f"Processing message {message.message_id}: {message.message_type.value}")

        if message.message_type == MessageType.STATE_SYNC:
            await self._process_state_sync(message)

        elif message.message_type == MessageType.STATE_TRANSITION:
            await self._process_state_transition(message)

        elif message.message_type == MessageType.CA_UPDATE:
            await self._process_ca_update(message)

        elif message.message_type == MessageType.AUTHORIZATION:
            await self._process_authorization(message)

        elif message.message_type == MessageType.RETIREMENT:
            await self._process_retirement(message)

        message.processed = True
        message.metadata["processed_at"] = datetime.utcnow().isoformat()

    async def _process_state_sync(self, message: ITMOStateMessage):
        """Process state sync message"""
        logger.info(f"Syncing state: {message.from_state} → {message.to_state}")

        # Determine SSM action
        action = self._map_states_to_ssm_action(message.from_state, message.to_state)

        # Prepare SSM context
        context = {
            "session": message.state_record_id,
            "action": action,
            "public": {
                "current_state": message.to_state,
                "registry_proof": {
                    "proof_hash": message.proof_hash,
                    "registry_id": message.registry_id,
                    "timestamp": message.timestamp.isoformat(),
                },
                "metadata": message.metadata,
            }
        }

        # Execute SSM transition
        await self.blockchain.perform_ssm_transition(context)

        logger.info(f"State synced for {message.serial_number}")

    async def _process_state_transition(self, message: ITMOStateMessage):
        """Process state transition message"""
        logger.info(f"Processing transition: {message.from_state} → {message.to_state}")

        # Similar to state sync but may include additional validations
        await self._process_state_sync(message)

    async def _process_ca_update(self, message: ITMOStateMessage):
        """Process corresponding adjustment update"""
        logger.info(f"Processing CA update for {message.serial_number}")

        if not message.ca_details:
            logger.error("No CA details in message")
            return

        # Update CA information in state record
        await self.blockchain.update_corresponding_adjustment(
            state_record_id=message.state_record_id,
            ca_details=message.ca_details
        )

        logger.info(f"CA updated for {message.serial_number}")

    async def _process_authorization(self, message: ITMOStateMessage):
        """Process authorization message"""
        logger.info(f"Processing authorization for {message.serial_number}")

        context = {
            "session": message.state_record_id,
            "action": "authorize",
            "public": {
                "current_state": "authorized",
                "authorization": message.metadata,
            }
        }

        await self.blockchain.perform_ssm_transition(context)

    async def _process_retirement(self, message: ITMOStateMessage):
        """Process retirement message"""
        logger.info(f"Processing retirement for {message.serial_number}")

        context = {
            "session": message.state_record_id,
            "action": "retire",
            "public": {
                "current_state": "retired",
                "retirement": message.metadata,
            }
        }

        await self.blockchain.perform_ssm_transition(context)

    def _states_differ(self, registry_state, chain_state) -> bool:
        """Check if states differ"""
        if not registry_state or not chain_state:
            return True

        return registry_state.current_state != chain_state.current_state

    def _map_states_to_ssm_action(self, from_state: str, to_state: str) -> str:
        """
        Map state changes to SSM actions
        This defines the state machine transitions
        """
        transitions = {
            ("issued", "authorized"): "authorize",
            ("authorized", "transferred"): "transfer",
            ("transferred", "held"): "hold",
            ("held", "collateral"): "collateralize",
            ("collateral", "held"): "release_collateral",
            ("held", "retired"): "retire",
            ("transferred", "retired"): "retire",
            # CA-related transitions
            ("authorized", "pending_ca"): "initiate_ca",
            ("pending_ca", "transferred"): "complete_ca",
            # Article 6.4 transitions
            ("a64_issued", "a64_authorized"): "a64_authorize",
            ("a64_authorized", "a64_first_transfer"): "a64_first_transfer",
            ("a64_first_transfer", "transferred"): "convert_to_itmo",
        }

        action = transitions.get((from_state, to_state), "sync_state")
        return action

    def get_message_queue_status(self) -> Dict:
        """Get status of message queue"""
        return {
            "queued_messages": len(self.message_queue),
            "processed_messages": len(self.processed_messages),
            "total_messages": len(self.message_queue) + len(self.processed_messages),
        }

    def get_sync_metrics(self) -> Dict:
        """Get synchronization metrics"""
        return {
            "last_sync_times": self.last_sync,
            "is_running": self.is_running,
            "message_queue_length": len(self.message_queue),
            "processed_count": len(self.processed_messages),
        }

    async def create_state_record(self, itmo_reference: Dict) -> str:
        """
        Create a new state record for tracking an ITMO
        The ITMO stays in its registry, we just create a state record
        """
        logger.info(f"Creating state record for ITMO {itmo_reference['serial_number']}")

        # Create initial state message
        message = ITMOStateMessage(
            message_type=MessageType.STATE_SYNC,
            state_record_id=f"STATE_{itmo_reference['serial_number']}",
            serial_number=itmo_reference['serial_number'],
            registry_id=itmo_reference['registry_id']
        )

        message.to_state = "issued"
        message.metadata = itmo_reference

        # Create state record on blockchain
        state_record_id = await self.blockchain.create_state_record({
            "state_record_id": message.state_record_id,
            "itmo_reference": itmo_reference,
            "current_state": "issued",
            "created_at": datetime.utcnow().isoformat(),
        })

        logger.info(f"Created state record: {state_record_id}")

        return state_record_id

    async def tokenize_state(self, state_record_id: str, total_supply: float) -> str:
        """
        Tokenize an ITMO state record
        Creates tokens representing rights to the state
        """
        logger.info(f"Tokenizing state record: {state_record_id}")

        token_id = await self.blockchain.create_state_token({
            "state_record_id": state_record_id,
            "total_supply": total_supply,
            "circulating_supply": total_supply,
            "tokenized_at": datetime.utcnow().isoformat(),
        })

        # Update state record
        await self.blockchain.update_state_record(
            state_record_id,
            {"is_tokenized": True, "token_id": token_id}
        )

        logger.info(f"Created state token: {token_id}")

        return token_id


# Example usage
async def main():
    """Example usage of ITMO state sync service"""

    # Mock connectors
    registry_connectors = {
        "verra": None,  # VerraConnector instance
        "gold_standard": None,  # GoldStandardConnector instance
    }

    # Mock clients
    blockchain_client = None  # SSM blockchain client
    unfccc_connector = None  # UNFCCC A6D connector

    # Create sync service
    sync_service = ITMOStateSyncService(
        registry_connectors=registry_connectors,
        blockchain_client=blockchain_client,
        unfccc_connector=unfccc_connector,
        sync_interval=300  # 5 minutes
    )

    # Start sync worker
    await sync_service.start_sync_worker()


if __name__ == "__main__":
    asyncio.run(main())
