#!/bin/bash

DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

export PRIMARY_INTERFACE="primary"
export SECONDARY_INTERFACE="secondary"
export IP_ADDRESS="192.168.123.10"
export ARP_TARGET="192.168.123.1"
export PREFIX_LEN="24"
export VLAN_ID=1234
export ARP_INTERVAL=200
export ARP_TIMEOUT=100
export LINK_MONITOR_INTERVAL=150

ip netns del failover-source
ip netns del failover-target

ip netns add failover-source
ip netns add failover-target
ip link add dev "${PRIMARY_INTERFACE}" type veth peer name "${PRIMARY_INTERFACE}"_peer
ip link add dev "${SECONDARY_INTERFACE}" type veth peer name "${SECONDARY_INTERFACE}"_peer
ip link set dev "${PRIMARY_INTERFACE}" netns failover-source
ip link set dev "${PRIMARY_INTERFACE}"_peer netns failover-target
ip link set dev "${SECONDARY_INTERFACE}" netns failover-source
ip link set dev "${SECONDARY_INTERFACE}"_peer netns failover-target

ip netns exec failover-target ip link add br0 type bridge
ip netns exec failover-target ip link set dev "${PRIMARY_INTERFACE}"_peer master br0
ip netns exec failover-target ip link set dev "${SECONDARY_INTERFACE}"_peer master br0
ip netns exec failover-target ip link add name br0."${VLAN_ID}" link br0 type vlan id "${VLAN_ID}"
ip netns exec failover-target ip link set dev lo up
ip netns exec failover-target ip link set dev "${PRIMARY_INTERFACE}"_peer up
ip netns exec failover-target ip link set dev "${SECONDARY_INTERFACE}"_peer up
ip netns exec failover-target ip link set dev br0 up
ip netns exec failover-target ip link set dev br0."${VLAN_ID}" up
ip netns exec failover-target ip address add dev br0."${VLAN_ID}" "${ARP_TARGET}/${PREFIX_LEN}"

ip netns exec failover-source ip link set dev lo up
ip netns exec failover-source ip link set dev "${PRIMARY_INTERFACE}" up
ip netns exec failover-source ip link set dev "${SECONDARY_INTERFACE}" up

# ip netns exec failover-source ip link add name "${PRIMARY_INTERFACE}"."${VLAN_ID}" link "${PRIMARY_INTERFACE}" type vlan id "${VLAN_ID}"
# ip netns exec failover-source ip link set dev "${PRIMARY_INTERFACE}"."${VLAN_ID}" up
# ip netns exec failover-source ip address add dev "${PRIMARY_INTERFACE}"."${VLAN_ID}" "${IP_ADDRESS}/${PREFIX_LEN}"

test() {
  echo "========================="
  echo "Sleeping for 5 seconds"
  echo "========================="
  sleep 5

  echo ""
  
  echo "========================="
  echo "Disabling primary interface"
  echo "========================="
  ip netns exec failover-source ip link set dev "${PRIMARY_INTERFACE}" down
  sleep 2
  ip netns exec failover-source ip a
  
  echo ""

  echo "========================="
  echo "Sleeping for 5 seconds"
  echo "========================="
  sleep 5

  echo ""
  
  echo "========================="
  echo "Disabling secondary interface"
  echo "========================="
  ip netns exec failover-source ip link set dev "${SECONDARY_INTERFACE}" down
  
  echo ""

  echo "========================="
  echo "Sleeping for 5 seconds"
  echo "========================="
  sleep 5

  echo ""
  
  echo "========================="
  echo "Enabling secondary interface"
  echo "========================="
  ip netns exec failover-source ip link set dev "${SECONDARY_INTERFACE}" up
  sleep 2
  ip netns exec failover-source ip a
  
  echo ""

  echo "========================="
  echo "Sleeping for 5 seconds"
  echo "========================="
  sleep 5

  echo ""
  
  echo "========================="
  echo "Enabling primary interface"
  echo "========================="
  ip netns exec failover-source ip link set dev "${PRIMARY_INTERFACE}" up
  sleep 2
  ip netns exec failover-source ip a
  
  echo ""

  echo "========================="
  echo "Disabling secondary interface"
  echo "========================="
  ip netns exec failover-source ip link set dev "${SECONDARY_INTERFACE}" down
  sleep 2
  ip netns exec failover-source ip a

  echo ""
  
  echo "========================="
  echo "Sleeping for 5 seconds"
  echo "========================="
  sleep 5

  echo ""
  
  echo "========================="
  echo "Enabling secondary interface"
  echo "========================="
  ip netns exec failover-source ip link set dev "${SECONDARY_INTERFACE}" up
  sleep 2
  ip netns exec failover-source ip a

  echo ""

  exit 0
}

test&

export IP_ADDRESS="${IP_ADDRESS}/${PREFIX_LEN}"
timeout 60 ip netns exec failover-source "${DIR}"/_output/userspace-failover

