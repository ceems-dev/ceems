#!/bin/sh

# Step 1, get systemd version
# systemd_version=$(systemctl --version | sed -nE "s/systemd ([0-9]+).*/\1/p")

uid=ceems
gid=ceems

setPermissions() {
    # Step 3 (clean install), setup setuid bit on cacct
    chmod u+s /usr/local/bin/cacct
}

createLogDirectory() {
    # Create log file directory
    mkdir -p /var/log/ceems
    chown ${uid}:${gid} /var/log/ceems
    chmod o-rwx /var/log/ceems
}

cleanInstall() {
    printf "\033[32m Post Install of an clean install\033[0m\n"
    
    setPermissions
    createLogDirectory
}

upgrade() {
    printf "\033[32m Post Install of an upgrade\033[0m\n"
    
    setPermissions
    createLogDirectory
}

# Step 2, check if this is a clean install or an upgrade
action="$1"
if  [ "$1" = "configure" ] && [ -z "$2" ]; then
    # Alpine linux does not pass args, and deb passes $1=configure
    action="install"
elif [ "$1" = "configure" ] && [ -n "$2" ]; then
    # deb passes $1=configure $2=<current version>
    action="upgrade"
fi

case "$action" in
  "1" | "install")
    cleanInstall
    ;;
  "2" | "upgrade")
    cleanInstall
    ;;
  *)
    # $1 == version being installed
    printf "\033[32m Alpine\033[0m"
    cleanInstall
    ;;
esac
