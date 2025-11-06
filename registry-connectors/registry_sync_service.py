"""
Registry Synchronization Service
Syncs carbon credit data between official registries and blockchain
"""

import asyncio
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional
from enum import Enum
import hashlib


logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class ConflictResolutionStrategy(Enum):
    """Strategy for resolving conflicts between registry and blockchain"""
    REGISTRY_WINS = "registry_wins"
    BLOCKCHAIN_WINS = "blockchain_wins"
    MANUAL_REVIEW = "manual_review"
    MERGED = "merged"


class SyncStatus(Enum):
    """Status of synchronization"""
    SUCCESS = "success"
    FAILED = "failed"
    PENDING = "pending"
    IN_PROGRESS = "in_progress"


class CreditUpdate:
    """Represents an update to a carbon credit"""
    def __init__(self, serial_number: str, update_type: str, quantity: float,
                 timestamp: datetime, details: Dict):
        self.serial_number = serial_number
        self.update_type = update_type  # 'issuance', 'retirement', 'cancellation', 'transfer'
        self.quantity = quantity
        self.timestamp = timestamp
        self.details = details


class Conflict:
    """Represents a conflict between registry and blockchain data"""
    def __init__(self, credit_id: str, field: str, registry_value, chain_value):
        self.conflict_id = self._generate_id()
        self.credit_id = credit_id
        self.field = field
        self.registry_value = registry_value
        self.chain_value = chain_value
        self.detected_at = datetime.utcnow()
        self.resolved = False
        self.resolution_strategy = None

    def _generate_id(self) -> str:
        """Generate unique conflict ID"""
        data = f"{datetime.utcnow().isoformat()}"
        return f"conflict_{hashlib.sha256(data.encode()).hexdigest()[:16]}"


class RegistrySyncService:
    """Service for synchronizing carbon credit registries with blockchain"""

    def __init__(self, registry_connectors: Dict, blockchain_client,
                 sync_interval: int = 300):
        """
        Initialize sync service

        Args:
            registry_connectors: Dict of registry_id -> connector instance
            blockchain_client: Client for blockchain interaction
            sync_interval: Sync interval in seconds (default 5 minutes)
        """
        self.connectors = registry_connectors
        self.blockchain = blockchain_client
        self.sync_interval = sync_interval
        self.last_sync = {}
        self.conflicts = []
        self.sync_history = []
        self.is_running = False

    async def start_sync_worker(self):
        """Start background synchronization worker"""
        logger.info("Starting registry sync worker")
        self.is_running = True

        while self.is_running:
            try:
                await self.sync_all_registries()
                await asyncio.sleep(self.sync_interval)
            except Exception as e:
                logger.error(f"Sync worker error: {e}")
                await asyncio.sleep(60)  # Wait 1 minute before retry

    def stop_sync_worker(self):
        """Stop synchronization worker"""
        logger.info("Stopping registry sync worker")
        self.is_running = False

    async def sync_all_registries(self):
        """Sync all configured registries"""
        logger.info("Starting full registry synchronization")

        for registry_id, connector in self.connectors.items():
            try:
                await self.sync_registry(registry_id, connector)
            except Exception as e:
                logger.error(f"Failed to sync registry {registry_id}: {e}")

        logger.info("Full registry synchronization complete")

    async def sync_registry(self, registry_id: str, connector):
        """
        Sync a specific registry

        Args:
            registry_id: Registry identifier
            connector: Registry connector instance
        """
        logger.info(f"Syncing registry: {registry_id}")

        # Get last sync time
        last_sync = self.last_sync.get(registry_id, datetime.utcnow() - timedelta(days=7))

        # Poll for updates since last sync
        try:
            updates = await self._fetch_updates(connector, last_sync)
            logger.info(f"Found {len(updates)} updates for {registry_id}")

            # Process each update
            for update in updates:
                await self.process_update(registry_id, update)

            # Update last sync time
            self.last_sync[registry_id] = datetime.utcnow()

            # Record sync success
            self._record_sync_status(registry_id, SyncStatus.SUCCESS)

        except Exception as e:
            logger.error(f"Sync failed for {registry_id}: {e}")
            self._record_sync_status(registry_id, SyncStatus.FAILED, str(e))

    async def _fetch_updates(self, connector, since: datetime) -> List[CreditUpdate]:
        """Fetch updates from registry since given time"""
        # This would call the connector's poll_updates method
        # Placeholder implementation
        return []

    async def process_update(self, registry_id: str, update: CreditUpdate):
        """
        Process a single credit update

        Args:
            registry_id: Registry identifier
            update: Credit update to process
        """
        logger.info(f"Processing update: {update.update_type} for {update.serial_number}")

        # Fetch registry data
        registry_data = await self._fetch_registry_data(registry_id, update.serial_number)

        # Fetch blockchain data
        chain_data = await self._fetch_chain_data(update.serial_number)

        if chain_data is None:
            # Credit not on blockchain yet, create it
            await self._create_on_chain(registry_data)
            logger.info(f"Created new credit on chain: {update.serial_number}")
            return

        # Check for conflicts
        conflicts = self._detect_conflicts(registry_data, chain_data)

        if conflicts:
            logger.warning(f"Detected {len(conflicts)} conflicts for {update.serial_number}")
            for conflict in conflicts:
                await self._handle_conflict(conflict)
        else:
            # No conflicts, update blockchain
            if self._needs_update(registry_data, chain_data):
                await self._update_chain(registry_data)
                logger.info(f"Updated credit on chain: {update.serial_number}")

    async def _fetch_registry_data(self, registry_id: str, serial_number: str) -> Dict:
        """Fetch credit data from registry"""
        connector = self.connectors.get(registry_id)
        if not connector:
            raise ValueError(f"No connector for registry: {registry_id}")

        # Fetch credit data
        credit = await connector.fetch_credit(serial_number)
        return credit

    async def _fetch_chain_data(self, serial_number: str) -> Optional[Dict]:
        """Fetch credit data from blockchain"""
        try:
            credit = await self.blockchain.get_credit(serial_number)
            return credit
        except Exception:
            return None

    def _detect_conflicts(self, registry_data: Dict, chain_data: Dict) -> List[Conflict]:
        """
        Detect conflicts between registry and blockchain data

        Args:
            registry_data: Data from registry
            chain_data: Data from blockchain

        Returns:
            List of detected conflicts
        """
        conflicts = []
        fields_to_check = ['quantity', 'status', 'current_owner', 'retired_quantity']

        for field in fields_to_check:
            registry_value = registry_data.get(field)
            chain_value = chain_data.get(field)

            if registry_value != chain_value:
                conflict = Conflict(
                    credit_id=chain_data.get('serial_number'),
                    field=field,
                    registry_value=registry_value,
                    chain_value=chain_value
                )
                conflicts.append(conflict)
                self.conflicts.append(conflict)

        return conflicts

    async def _handle_conflict(self, conflict: Conflict):
        """
        Handle a detected conflict

        Args:
            conflict: Conflict to resolve
        """
        logger.warning(f"Handling conflict {conflict.conflict_id}: {conflict.field}")

        # Determine resolution strategy based on field
        strategy = self._get_resolution_strategy(conflict)

        if strategy == ConflictResolutionStrategy.REGISTRY_WINS:
            # Registry is source of truth
            await self._resolve_registry_wins(conflict)

        elif strategy == ConflictResolutionStrategy.BLOCKCHAIN_WINS:
            # Blockchain is authoritative (rare)
            await self._resolve_blockchain_wins(conflict)

        elif strategy == ConflictResolutionStrategy.MANUAL_REVIEW:
            # Flag for manual review
            logger.error(f"Conflict {conflict.conflict_id} requires manual review")
            await self._flag_for_review(conflict)

        elif strategy == ConflictResolutionStrategy.MERGED:
            # Attempt to merge data
            await self._resolve_merged(conflict)

    def _get_resolution_strategy(self, conflict: Conflict) -> ConflictResolutionStrategy:
        """
        Determine resolution strategy for a conflict

        Args:
            conflict: Conflict to analyze

        Returns:
            Resolution strategy
        """
        # Critical fields always defer to registry
        critical_fields = ['status', 'retired_quantity', 'cancelled_quantity']
        if conflict.field in critical_fields:
            return ConflictResolutionStrategy.REGISTRY_WINS

        # Ownership changes may need manual review
        if conflict.field == 'current_owner':
            return ConflictResolutionStrategy.MANUAL_REVIEW

        # Quantity differences need careful handling
        if conflict.field == 'quantity':
            # If close (< 0.01%), consider merged
            if abs(conflict.registry_value - conflict.chain_value) < 0.0001:
                return ConflictResolutionStrategy.MERGED
            return ConflictResolutionStrategy.REGISTRY_WINS

        return ConflictResolutionStrategy.REGISTRY_WINS

    async def _resolve_registry_wins(self, conflict: Conflict):
        """Resolve conflict by using registry value"""
        logger.info(f"Resolving {conflict.conflict_id}: registry wins")

        # Update blockchain with registry value
        await self.blockchain.update_field(
            credit_id=conflict.credit_id,
            field=conflict.field,
            value=conflict.registry_value
        )

        conflict.resolved = True
        conflict.resolution_strategy = ConflictResolutionStrategy.REGISTRY_WINS

    async def _resolve_blockchain_wins(self, conflict: Conflict):
        """Resolve conflict by keeping blockchain value"""
        logger.info(f"Resolving {conflict.conflict_id}: blockchain wins")
        # Just mark as resolved, no action needed
        conflict.resolved = True
        conflict.resolution_strategy = ConflictResolutionStrategy.BLOCKCHAIN_WINS

    async def _resolve_merged(self, conflict: Conflict):
        """Resolve conflict by merging data"""
        logger.info(f"Resolving {conflict.conflict_id}: merged")

        # Take average or most recent, depending on field
        merged_value = (conflict.registry_value + conflict.chain_value) / 2

        await self.blockchain.update_field(
            credit_id=conflict.credit_id,
            field=conflict.field,
            value=merged_value
        )

        conflict.resolved = True
        conflict.resolution_strategy = ConflictResolutionStrategy.MERGED

    async def _flag_for_review(self, conflict: Conflict):
        """Flag conflict for manual review"""
        logger.warning(f"Flagging conflict {conflict.conflict_id} for manual review")

        # Store in database for review
        # Send notification to admins
        # This would integrate with your notification system

        conflict.resolution_strategy = ConflictResolutionStrategy.MANUAL_REVIEW

    def _needs_update(self, registry_data: Dict, chain_data: Dict) -> bool:
        """Check if blockchain data needs updating"""
        # Compare last updated timestamps
        registry_updated = registry_data.get('last_updated')
        chain_updated = chain_data.get('last_sync_date')

        if not chain_updated:
            return True

        if registry_updated and registry_updated > chain_updated:
            return True

        return False

    async def _create_on_chain(self, registry_data: Dict):
        """Create new credit on blockchain"""
        await self.blockchain.create_credit(registry_data)

    async def _update_chain(self, registry_data: Dict):
        """Update existing credit on blockchain"""
        await self.blockchain.update_credit(registry_data)

    def _record_sync_status(self, registry_id: str, status: SyncStatus,
                           error_msg: Optional[str] = None):
        """Record synchronization status"""
        record = {
            'registry_id': registry_id,
            'timestamp': datetime.utcnow(),
            'status': status.value,
            'error': error_msg
        }
        self.sync_history.append(record)

        # Keep only last 1000 records
        if len(self.sync_history) > 1000:
            self.sync_history = self.sync_history[-1000:]

    def get_sync_status(self, registry_id: Optional[str] = None) -> List[Dict]:
        """Get synchronization status"""
        if registry_id:
            return [r for r in self.sync_history if r['registry_id'] == registry_id]
        return self.sync_history

    def get_unresolved_conflicts(self) -> List[Conflict]:
        """Get list of unresolved conflicts"""
        return [c for c in self.conflicts if not c.resolved]

    def get_sync_metrics(self) -> Dict:
        """Get synchronization metrics"""
        return {
            'total_syncs': len(self.sync_history),
            'successful_syncs': len([s for s in self.sync_history if s['status'] == SyncStatus.SUCCESS.value]),
            'failed_syncs': len([s for s in self.sync_history if s['status'] == SyncStatus.FAILED.value]),
            'total_conflicts': len(self.conflicts),
            'unresolved_conflicts': len(self.get_unresolved_conflicts()),
            'last_sync_times': self.last_sync,
        }


# Example usage
async def main():
    """Example usage of registry sync service"""

    # Mock connectors (would be real implementations)
    connectors = {
        'verra': None,  # VerraConnector instance
        'gold_standard': None,  # GoldStandardConnector instance
    }

    # Mock blockchain client
    blockchain_client = None  # Real blockchain client

    # Create sync service
    sync_service = RegistrySyncService(
        registry_connectors=connectors,
        blockchain_client=blockchain_client,
        sync_interval=300  # 5 minutes
    )

    # Start sync worker
    await sync_service.start_sync_worker()


if __name__ == "__main__":
    asyncio.run(main())
