#!/usr/bin/env bash
# Release public tools.
#
# Publish to Canonistack, HP, AWS, and Azure.
# This script requires that the user has credentials to upload the tools
# to Canonistack, HP Cloud, AWS, and Azure

set -e

SCRIPT_DIR=$(cd $(dirname "${BASH_SOURCE[0]}") && pwd )


usage() {
    echo "usage: $0 PURPOSE DIST_DIRECTORY"
    echo "  PURPOSE: 'RELEASE' or  'TESTING'"
    echo "    RELEASE installs tools/ at the top of juju-dist/tools."
    echo "    TESTING installs tools/ at juju-dist/testing/tools."
    echo "  DIST_DIRECTORY: The directory to the assembled tools."
    echo "    This is the juju-dist dir created by assemble-public-tools.bash."
    exit 1
}


check_deps() {
    echo "Phase 0: Checking requirements."
    has_deps=1
    which swift || has_deps=0
    which s3cmd || has_deps=0
    test -f $JUJU_DIR/canonistacktoolsrc || has_deps=0
    test -f $JUJU_DIR/hptoolsrc || has_deps=0
    test -f $JUJU_DIR/s3cfg || has_deps=0
    test -f $JUJU_DIR/azuretoolsrc || has_deps=0
    if [[ $has_deps == 0 ]]; then
        echo "Install python-swiftclient, and s3cmd"
        echo "Your $JUJU_DIR dir must contain rc files to publish:"
        echo "  canonistacktoolsrc, hptoolsrc, s3cfg, azuretoolsrc"
        exit 2
    fi
}


publish_to_aws() {
    [[ $JT_IGNORE_AWS == '1' ]] && return 0
    if [[ $PURPOSE == "RELEASE" ]]; then
        local event="Publish"
        local destination="s3://juju-dist/"
    else
        local event="Testing"
        local destination="s3://juju-dist/testing/"
    fi
    echo "Phase 1: $event to AWS."
    s3cmd -c $JUJU_DIR/s3cfg sync --exclude '*mirror*' \
        ${JUJU_DIST}/tools $destination
}


publish_to_canonistack() {
    [[ $JT_IGNORE_CANONISTACK == '1' ]] && return 0
    if [[ $PURPOSE == "RELEASE" ]]; then
        local event="Publish"
        local destination="tools"
    else
        local event="Testing"
        local destination="testing/tools"
    fi
    echo "Phase 2: $event to canonistack."
    source $JUJU_DIR/canonistacktoolsrc
    cd $JUJU_DIST/tools/releases/
    ${SCRIPT_DIR}/swift_sync.py $destination/releases/ *.tgz
    cd $JUJU_DIST/tools/streams/v1
    ${SCRIPT_DIR}/swift_sync.py $destination/streams/v1/ {index,com}*
}


publish_to_hp() {
    [[ $JT_IGNORE_HP == '1' ]] && return 0
    if [[ $PURPOSE == "RELEASE" ]]; then
        local event="Publish"
        local destination="tools"
    else
        local event="Testing"
        local destination="testing/tools"
    fi
    echo "Phase 3: $event to HP Cloud."
    source $JUJU_DIR/hptoolsrc
    cd $JUJU_DIST/tools/releases/
    ${SCRIPT_DIR}/swift_sync.py $destination/releases/ *.tgz
    cd $JUJU_DIST/tools/streams/v1
    ${SCRIPT_DIR}/swift_sync.py $destination/streams/v1/ {index,com}*
}


publish_to_azure() {
    [[ $JT_IGNORE_AZURE == '1' ]] && return 0
    if [[ $PURPOSE == "RELEASE" ]]; then
        local event="Publish"
        local destination="release"
    else
        local event="Testing"
        local destination="testing"
    fi
    echo "Phase 4: $event to Azure."
    source $JUJU_DIR/azuretoolsrc
    ${SCRIPT_DIR}/azure_publish_tools.py publish $destination ${JUJU_DIST}
}


publish_to_streams() {
    [[ -f $JUJU_DIR/streamsrc ]] || return 0
    [[ $JT_IGNORE_STREAMS == '1' ]] && return 0
    if [[ $PURPOSE == "RELEASE" ]]; then
        local event="Publish"
        local destination=$STREAMS_OFFICIAL_DEST
    else
        local event="Testing"
        local destination=$STREAMS_TESTING_DEST
    fi
    echo "Phase 5: $event to streams.canonical.com."
    source $JUJU_DIR/streamsrc
    rsync -avzh $JUJU_DIST/ $destination
}


# The location of environments.yaml and rc files.
JUJU_DIR=${JUJU_HOME:-$HOME/.juju}

test $# -eq 2 || usage

PURPOSE=$1
if [[ $PURPOSE != "RELEASE" && $PURPOSE != "TESTING" ]]; then
    usage
fi

JUJU_DIST=$(cd $2; pwd)
if [[ ! -d $JUJU_DIST/tools/releases && ! -d $JUJU_DIST/tools/streams ]]; then
    usage
fi


check_deps
publish_to_aws
publish_to_canonistack
publish_to_hp
publish_to_azure
publish_to_streams
echo "Release data published to all CPCs."

