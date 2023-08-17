#!/bin/bash

DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

export PRIMARY_INTERFACE="primary"
export SECONDARY_INTERFACE="secondary"
export IP_ADDRESS="192.168.123.10"
export ARP_TARGET="192.168.123.1"
export PREFIX_LEN="24"
export VLAN_ID=1234

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
ip netns exec failover-source ip link add name "${PRIMARY_INTERFACE}"."${VLAN_ID}" link "${PRIMARY_INTERFACE}" type vlan id "${VLAN_ID}"

# ip netns exec failover-source ip link set dev "${PRIMARY_INTERFACE}"."${VLAN_ID}" up
# ip netns exec failover-source ip address add dev "${PRIMARY_INTERFACE}"."${VLAN_ID}" "${IP_ADDRESS}/${PREFIX_LEN}"

export IP_ADDRESS="${IP_ADDRESS}/${PREFIX_LEN}"
ip netns exec failover-source "${DIR}"/_output/userspace-failover