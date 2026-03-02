import * as pulumi from "@pulumi/pulumi";
import * as hcloud from "@pulumi/hcloud";
import * as path from "path";
import * as fs from "fs";
import {execSync} from "child_process";
import * as os from "os";

const config = new pulumi.Config();
const serverRegion = config.require("serverRegion");
const frpServerToken = config.requireSecret("frpServerToken");
const frpServerPort = config.requireNumber("frpServerPort");
const frpDashboardPort = config.requireNumber("frpDashboardPort");
const frpDashboardUser = config.require("frpDashboardUser");
const frpDashboardPassword = config.requireSecret("frpDashboardPassword");
const exposedTcpPort = config.requireNumber("exposedTcpPort");

function setSecureFilePermissions(filePath: string): void {
  if (os.platform() === 'win32') {
    try {
      execSync(`icacls "${filePath}" /inheritance:r /grant:r "%USERNAME%:F"`, {stdio: 'pipe'});
      console.log(`Set Windows file permissions for ${filePath}`);
    } catch (error) {
      console.warn(`Could not set Windows file permissions for ${filePath}:`, error as Error);
    }
  } else {
    try {
      fs.chmodSync(filePath, 0o600);
    } catch (error) {
      console.warn(`Could not set Unix file permissions for ${filePath}:`, error as Error);
    }
  }
}

const cloudInitScript = fs.readFileSync(path.join(__dirname, "frp.yaml"), "utf8");
const userData = pulumi.all([
  frpServerToken,
  frpDashboardPassword,
]).apply(([serverToken, dashPassword]) => {
  return cloudInitScript
    .replace(/FRP_SERVER_PORT_PLACEHOLDER/g, frpServerPort.toString())
    .replace(/FRP_DASHBOARD_PORT_PLACEHOLDER/g, frpDashboardPort.toString())
    .replace(/FRP_DASHBOARD_USER_PLACEHOLDER/g, frpDashboardUser)
    .replace(/FRP_DASHBOARD_PASSWORD_PLACEHOLDER/g, dashPassword)
    .replace(/FRP_SERVER_TOKEN_PLACEHOLDER/g, serverToken)
    .replace(/FRP_EXPOSED_TCP_PORT_PLACEHOLDER/g, exposedTcpPort.toString());
});

const opensshPrivateKeyPath = path.join(__dirname, "frp-server-key");
try {
  try {
    fs.unlinkSync(opensshPrivateKeyPath);
    fs.unlinkSync(`${opensshPrivateKeyPath}.pub`);
  } catch (error) {
    console.log(`Cannot remove existing SSH key files: ${(error as Error).message}`);
  }

  execSync(`ssh-keygen -t ed25519 -f "${opensshPrivateKeyPath}" -N "" -C "root@frp-server"`, {
    stdio: 'pipe'
  });

} catch (error) {
  console.error("Failed to generate SSH key pair:", error as Error);
  throw new Error("SSH key generation failed. Make sure ssh-keygen is installed and available in PATH.");
}

setSecureFilePermissions(opensshPrivateKeyPath);

const sshPublicKey = fs.readFileSync(`${opensshPrivateKeyPath}.pub`, 'utf8').trim();

const sshKey = new hcloud.SshKey("frp-ssh-key", {
  name: `frp-server-ssh-key`,
  publicKey: sshPublicKey,
});

const frpServer = new hcloud.Server("frp-server", {
  name: "frp-server",
  sshKeys: [sshKey.id],
  serverType: "cx23",
  image: "debian-13",
  location: serverRegion,
  userData: userData,
});

const firewallRules: hcloud.types.input.FirewallRule[] = [
  {
    direction: "in",
    protocol: "tcp",
    port: "22",
    sourceIps: ["0.0.0.0/0", "::/0"],
    description: "Allow SSH access",
  },
  {
    direction: "in",
    protocol: "tcp",
    port: frpServerPort.toString(),
    sourceIps: ["0.0.0.0/0", "::/0"],
    description: "Allow frp server connections",
  },
  {
    direction: "in",
    protocol: "tcp",
    port: exposedTcpPort.toString(),
    sourceIps: ["0.0.0.0/0", "::/0"],
    description: "Allow configured TCP port for web service",
  },
];

const frpFirewall = new hcloud.Firewall("frp-firewall", {
  name: "frp-server-firewall",
  rules: firewallRules,
});

new hcloud.FirewallAttachment("frp-firewall-attachment", {
  firewallId: frpFirewall.id.apply(id => parseInt(id, 10)),
  serverIds: [frpServer.id.apply(id => parseInt(id, 10))],
});

export const serverPublicIp = frpServer.ipv4Address;
export const dashboardUrl = pulumi.interpolate`http://localhost:${frpDashboardPort} (via SSH tunnel)`;
export const frpServerEndpoint = pulumi.interpolate`${frpServer.ipv4Address}:${frpServerPort}`;
export const configuredExposedTcpPort = exposedTcpPort;
export const webServiceUrl = pulumi.interpolate`http://${frpServer.ipv4Address}:${exposedTcpPort}`;
export const sshKeyPath = opensshPrivateKeyPath;
export const sshConnectCommand = pulumi.interpolate`ssh -i ${opensshPrivateKeyPath} root@${frpServer.ipv4Address}`;
export const sshTunnelCommand = pulumi.interpolate`ssh -i ${opensshPrivateKeyPath} -L ${frpDashboardPort}:localhost:${frpDashboardPort} root@${frpServer.ipv4Address}`;
export const caCertPath = "/etc/ssl/private/frp-ca.crt (on server)";
export const tlsEnabled = "true (forced encryption)";
export const getCaCertCommand = pulumi.interpolate`ssh -i ${opensshPrivateKeyPath} root@${frpServer.ipv4Address} "cat /etc/ssl/private/frp-ca.crt" > frp-ca.crt`;
export const getClientCertCommand = pulumi.interpolate`ssh -i ${opensshPrivateKeyPath} root@${frpServer.ipv4Address} "cat /etc/ssl/certs/frpc.crt" > frpc.crt`;
export const getClientKeyCommand = pulumi.interpolate`ssh -i ${opensshPrivateKeyPath} root@${frpServer.ipv4Address} "cat /etc/ssl/private/frpc.key" > frpc.key`;

pulumi.all([serverPublicIp, frpServerToken, frpServerPort]).apply(([ip, token, port]) => {
  if (pulumi.runtime.isDryRun()) {
    return;
  }
  const frpcTemplate = fs.readFileSync(path.join(__dirname, "frpc.toml.template"), "utf8");
  const updatedFrpc = frpcTemplate
    .replace(/your-frps-server-ip/g, ip)
    .replace(/your-frp-server-token/g, token)
    .replace(/SERVER_PORT_PLACEHOLDER/g, port.toString())
    .replace(/FRP_EXPOSED_TCP_PORT_PLACEHOLDER/g, exposedTcpPort.toString());
  
  fs.writeFileSync(path.join(__dirname, "frpc.toml"), updatedFrpc, "utf8");
});
