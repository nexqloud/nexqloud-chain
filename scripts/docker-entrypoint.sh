#!/bin/bash
set -e

echo "🚀 Starting NexQloud Chain Node"
echo "================================================"
echo "Node Type: ${NODE_TYPE:-peer}"
echo "Moniker: ${MONIKER:-NexQloudNode}"
echo "Chain ID: ${CHAINID:-nxqd_90009-1}"
echo "Keyring: ${KEYRING:-test}"
echo "================================================"

# Function to wait for a service to be ready
wait_for_service() {
    local service=$1
    local max_attempts=30
    local attempt=0
    
    echo "⏳ Waiting for $service to be ready..."
    
    while [ $attempt -lt $max_attempts ]; do
        if wget -q --spider "http://$service:26657/status" 2>/dev/null; then
            echo "✅ $service is ready!"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo "   Attempt $attempt/$max_attempts - $service not ready yet..."
        sleep 2
    done
    
    echo "⚠️  Warning: $service did not become ready, continuing anyway..."
    return 1
}

# Wait for dependencies based on node type
case "$NODE_TYPE" in
    seed)
        echo "🌱 First Seed Node - No dependencies to wait for"
        ;;
    multi-seed)
        # Wait for first seed to be ready
        if [ -n "$FIRST_SEED_SERVICE" ]; then
            wait_for_service "$FIRST_SEED_SERVICE"
        fi
        ;;
    peer|persistent-peer)
        # Wait for at least one seed to be ready
        seed_ready=false
        for seed in $SEED_SERVICES; do
            if wait_for_service "$seed"; then
                seed_ready=true
                break
            fi
        done
        
        if [ "$seed_ready" = false ]; then
            echo "❌ No seed nodes are ready, cannot start peer node"
            exit 1
        fi
        ;;
esac

# Run appropriate initialization and start
case "$NODE_TYPE" in
    seed)
        echo "🌱 Initializing First Seed Node"
        exec /usr/local/bin/seed_node_prod.sh start
        ;;
    multi-seed)
        echo "🌱 Initializing Additional Seed Node"
        exec /usr/local/bin/multi_seed_node_prod.sh start
        ;;
    peer|persistent-peer)
        echo "🔗 Initializing Peer Node"
        exec /usr/local/bin/peer_node_prod.sh start
        ;;
    *)
        echo "❌ Unknown NODE_TYPE: $NODE_TYPE"
        echo "Valid types: seed, multi-seed, peer, persistent-peer"
        exit 1
        ;;
esac

