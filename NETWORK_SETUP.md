# Network Device Configuration Guide

This guide explains how to configure various network devices to authenticate against your FreeRADIUS Manager.

## Prerequisites

1. Note your server's IP address (where FreeRADIUS Manager is running)
2. Get your RADIUS shared secret from the `.env` file (`RADIUS_SECRET`)
3. Add the device in the web UI under **NAS Devices → Add Device**

---

## MikroTik RouterOS

### WiFi Hotspot
```routeros
/radius
add address=YOUR_SERVER_IP secret=YOUR_SECRET service=hotspot

/ip hotspot profile
set hsprof1 use-radius=yes radius-accounting=yes
```

### 802.1X WiFi (WPA-Enterprise)
```routeros
/radius
add address=YOUR_SERVER_IP secret=YOUR_SECRET service=wireless

/interface wireless
set wlan1 security-profile=radius-auth

/interface wireless security-profiles
set radius-auth mode=dynamic-keys eap-methods=peap
```

### PPPoE / VPN
```routeros
/radius
add address=YOUR_SERVER_IP secret=YOUR_SECRET service=ppp

/ppp aaa
set use-radius=yes accounting=yes
```

---

## Cisco IOS / IOS-XE

```cisco
! Global RADIUS configuration
aaa new-model

radius-server host YOUR_SERVER_IP auth-port 1812 acct-port 1813 key YOUR_SECRET

! For IOS-XE:
radius server RADIUS_MANAGER
 address ipv4 YOUR_SERVER_IP auth-port 1812 acct-port 1813
 key YOUR_SECRET

aaa group server radius RADIUS_GROUP
 server name RADIUS_MANAGER

! Authentication for 802.1X
aaa authentication dot1x default group RADIUS_GROUP
aaa authorization network default group RADIUS_GROUP
aaa accounting dot1x default start-stop group RADIUS_GROUP

! Enable dot1x globally
dot1x system-auth-control
```

---

## Cisco WLC (Wireless LAN Controller)

1. Go to **Security → AAA → RADIUS → Authentication**
2. Click **New** and enter:
   - Server IP: `YOUR_SERVER_IP`
   - Port: `1812`
   - Shared Secret: `YOUR_SECRET`
3. Go to **Security → AAA → RADIUS → Accounting**
4. Add server with port `1813`
5. Apply RADIUS to your WLAN: **WLANs → Edit → Security → AAA Servers**

---

## Ubiquiti UniFi

1. Open UniFi Network Controller
2. Go to **Settings → Profiles → RADIUS**
3. Create new profile:
   - Authentication Server: `YOUR_SERVER_IP`
   - Authentication Port: `1812`
   - Password (Shared Secret): `YOUR_SECRET`
   - Accounting Server: `YOUR_SERVER_IP`
   - Accounting Port: `1813`
4. Apply to WiFi network: **Settings → WiFi → Edit Network → Security → WPA Enterprise**

---

## pfSense / OPNsense

### Captive Portal
1. **Services → Captive Portal → Add Zone**
2. Under **Authentication**, select **RADIUS Authentication**
3. Primary RADIUS server: `YOUR_SERVER_IP`
4. Port: `1812`
5. Shared Key: `YOUR_SECRET`
6. Enable RADIUS accounting: `YOUR_SERVER_IP:1813`

### VPN (OpenVPN with RADIUS)
Install the `freeradius3` package:
```
pkg install pfSense-pkg-freeradius3
```
Configure under **Services → FreeRADIUS → RADIUS → Clients**.

---

## OpenVPN (Linux Server)

Install the radiusplugin:
```bash
apt-get install openvpn-auth-radius
```

`/etc/openvpn/radiusplugin.cnf`:
```ini
NAS-Identifier=vpn-server
NAS-IP-Address=YOUR_VPN_SERVER_IP
server
{
    acctport=1813
    authport=1812
    name=YOUR_SERVER_IP
    retry=3
    wait=3
    sharedsecret=YOUR_SECRET
}
```

`/etc/openvpn/server.conf` — add:
```
plugin /usr/lib/openvpn/radiusplugin.so /etc/openvpn/radiusplugin.cnf
```

---

## WireGuard (with Pre-Auth RADIUS)

WireGuard doesn't natively support RADIUS, but you can use a captive portal or pre-auth script. A common approach is to use `wg-dynamic` with a RADIUS-authenticated portal.

---

## Testing RADIUS Authentication

From the command line (on Linux with `freeradius-utils`):
```bash
radtest USERNAME PASSWORD YOUR_SERVER_IP 0 YOUR_SHARED_SECRET
```

Expected responses:
- `Access-Accept` — Authentication successful ✓
- `Access-Reject` — Authentication failed (wrong password or suspended user) ✗
- `Connection refused` — FreeRADIUS not running or wrong IP/port

---

## VLAN Assignment (Dynamic VLAN)

To assign VLANs based on user group, add RADIUS reply attributes:

In the web UI, users can be assigned to groups. Add group reply attributes in the database:

```sql
INSERT INTO radgroupreply (groupname, attribute, op, value)
VALUES ('staff', 'Tunnel-Type', '=', '13');          -- VLAN

INSERT INTO radgroupreply (groupname, attribute, op, value)
VALUES ('staff', 'Tunnel-Medium-Type', '=', '6');    -- IEEE-802

INSERT INTO radgroupreply (groupname, attribute, op, value)
VALUES ('staff', 'Tunnel-Private-Group-Id', '=', '100');  -- VLAN ID 100
```

Or per-user via radreply:
```sql
INSERT INTO radreply (username, attribute, op, value)
VALUES ('alice', 'Tunnel-Private-Group-Id', '=', '200');
```

---

## Troubleshooting

### Authentication failing
1. Check the NAS device is added in the web UI with the correct shared secret
2. Test with `radtest` from the server
3. Check FreeRADIUS logs: `docker-compose logs freeradius`
4. Ensure the user exists and is not suspended

### "Invalid shared secret"
- The device secret must exactly match what's in NAS Devices in the web UI
- Secrets are case-sensitive
- No leading/trailing spaces

### "Access-Reject" for valid credentials
- User may be suspended → activate in Users page
- Password may be expired → check in Users page
- Device limit may be reached → check active sessions

### NAS not reaching RADIUS server
- Firewall must allow UDP 1812 and 1813 from the NAS device IP
- Check `iptables -L` or firewall rules
- Ensure Docker port mappings are correct: `docker-compose ps`
