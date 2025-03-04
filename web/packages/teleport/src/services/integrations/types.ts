/**
 * Teleport
 * Copyright (C) 2023  Gravitational, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

import { Label } from 'teleport/types';

import { ResourceLabel } from '../agents';
import { Node } from '../nodes';

/**
 * type Integration v. type Plugin:
 *
 * Before "integration" resource was made, a "plugin" resource existed.
 * They are essentially the same where plugin resource could've
 * been defined with the integration resource. But it's too late for
 * renames/changes. There are small differences between the two resource,
 * so they are separate types.
 *
 * "integration" resource is supported in both OS and Enterprise
 * while "plugin" resource is only supported in enterprise. Plugin
 * type exists in OS for easier typing when combining the resources
 * into one list.
 *
 * Generics:
 *  T is resource type "integration" or "plugin"
 *  K is the kind of integration (eg: aws-oidc) or plugin (eg: okta)
 *  SP is the provider-specific spec of integration or plugin
 *  SD is the provider-specific status containing status details
 *   - currently only defined for plugin resource
 */
export type Integration<
  T extends string = 'integration',
  K extends string = IntegrationKind,
  SP extends Record<string, any> = IntegrationSpecAwsOidc,
  SD extends Record<string, any> = null,
> = {
  resourceType: T;
  kind: K;
  spec?: SP;
  name: string;
  details?: string;
  statusCode: IntegrationStatusCode;
  status?: SD;
};
// IntegrationKind string values should be in sync
// with the backend value for defining the integration
// resource's subKind field.
export enum IntegrationKind {
  AwsOidc = 'aws-oidc',
  AzureOidc = 'azure-oidc',
  ExternalAuditStorage = 'external-audit-storage',
  GitHub = 'github',
}

/**
 * IntegrationAudience defines supported audience value for IntegrationSpecAwsOidc
 * audience field.
 */
export enum IntegrationAudience {
  AwsIdentityCenter = 'aws-identity-center',
}

export type IntegrationSpecAwsOidc = {
  roleArn: string;
  issuerS3Prefix?: string;
  issuerS3Bucket?: string;
  /**
   * audience is used to record name of a plugin or discover services in Teleport
   * that depends on this integration.
   */
  audience?: IntegrationAudience;
};

export type AwsOidcPingRequest = {
  // Define roleArn if the ping request should
  // use this potentially new roleArn to test the
  // connection works, typically used with updates.
  //
  // Leave empty if the ping request should
  // use the roleArn stored in the integration resource,
  // typically used when checking integration still works.
  roleArn?: string;
};

export type AwsOidcPingResponse = {
  accountId: string;
  arn: string;
  userId: string;
};

export enum IntegrationStatusCode {
  Unknown = 0,
  Running = 1,
  OtherError = 2,
  Unauthorized = 3,
  SlackNotInChannel = 10,
  Draft = 100,
}

export function getStatusCodeTitle(code: IntegrationStatusCode): string {
  switch (code) {
    case IntegrationStatusCode.Unknown:
      return 'Unknown';
    case IntegrationStatusCode.Running:
      return 'Running';
    case IntegrationStatusCode.Unauthorized:
      return 'Unauthorized';
    case IntegrationStatusCode.SlackNotInChannel:
      return 'Bot not invited to channel';
    case IntegrationStatusCode.Draft:
      return 'Draft';
    default:
      return 'Unknown error';
  }
}

export function getStatusCodeDescription(
  code: IntegrationStatusCode
): string | null {
  switch (code) {
    case IntegrationStatusCode.Unauthorized:
      return 'The integration was denied access. This could be a result of revoked authorization on the 3rd party provider. Try removing and re-connecting the integration.';

    case IntegrationStatusCode.SlackNotInChannel:
      return 'The Slack integration must be invited to the default channel in order to receive access request notifications.';
    default:
      return null;
  }
}

export type ExternalAuditStorage = {
  integrationName: string;
  policyName: string;
  region: string;
  sessionsRecordingsURI: string;
  athenaWorkgroup: string;
  glueDatabase: string;
  glueTable: string;
  auditEventsLongTermURI: string;
  athenaResultsURI: string;
};

export type ExternalAuditStorageIntegration = Integration<
  'external-audit-storage',
  IntegrationKind.ExternalAuditStorage,
  ExternalAuditStorage
>;

export type Plugin<SP = any, D = any> = Integration<
  'plugin',
  PluginKind,
  SP,
  PluginStatus<D>
>;

export type PluginStatus<D = any> = {
  /**
   * the status code of the plugin
   */
  code: IntegrationStatusCode;
  /**
   * the time the plugin was last run
   */
  lastRun: Date;
  /**
   * the last error message from the plugin
   */
  errorMessage: string;
  /**
   * contains provider-specific status information
   */
  details?: D;
};

export type PluginSpec =
  | PluginOktaSpec
  | PluginSlackSpec
  | PluginMattermostSpec
  | PluginOpsgenieSpec
  | PluginDatadogSpec
  | PluginEmailSpec
  | PluginMsTeamsSpec;

// PluginKind represents the type of the plugin
// and should be the same value as defined in the backend (check master branch for the latest):
// https://github.com/gravitational/teleport/blob/a410acef01e0023d41c18ca6b0a7b384d738bb32/api/types/plugin.go#L27
export type PluginKind =
  | 'slack'
  | 'openai'
  | 'pagerduty'
  | 'email'
  | 'jira'
  | 'discord'
  | 'mattermost'
  | 'msteams'
  | 'opsgenie'
  | 'okta'
  | 'servicenow'
  | 'jamf'
  | 'entra-id'
  | 'datadog'
  | 'aws-identity-center';

export type PluginOktaSpec = {
  // scimBearerToken is the plain text of the bearer token that Okta will use
  // to authenticate SCIM requests
  scimBearerToken: string;
  // oktaAppID is the Okta ID of the SAML App created during the Okta plugin
  // installation
  oktaAppId: string;
  // oktaAppName is the human readable name of the Okta SAML app created
  // during the Okta plugin installation
  oktaAppName: string;
  // teleportSSOConnector is the name of the Teleport SAML SSO connector
  // created by the plugin during installation
  teleportSsoConnector: string;
  // error contains a description of any failures during plugin installation
  // that were deemed not serious enough to fail the plugin installation, but
  // may effect the operation of advanced features like User Sync or SCIM.
  error: string;
  /**
   * is the set of usernames that the integration assigns as
   * owners to any Access Lists that it creates
   */
  defaultOwners: string[];
  /**
   * the Okta org's base URL
   */
  orgUrl: string;
};

export type PluginSlackSpec = {
  fallbackChannel: string;
};

export type PluginMattermostSpec = {
  channel: string;
  team: string;
  reportToEmail: string;
};

export type PluginMsTeamsSpec = {
  appID: string;
  tenantID: string;
  teamsAppID: string;
  region: string;
  defaultRecipient: string;
};

export type PluginOpsgenieSpec = {
  defaultSchedules: string[];
};

export type PluginDatadogSpec = {
  apiEndpoint: string;
  fallbackRecipient: string;
};

export type PluginEmailSpec = {
  sender: string;
  fallbackRecipient: string;
};

export type IntegrationCreateRequest = {
  name: string;
  subKind: IntegrationKind;
  awsoidc?: IntegrationSpecAwsOidc;
};

export type IntegrationListResponse = {
  items: Integration[];
  nextKey?: string;
};

// IntegrationWithSummary describes Integration fields and the fields required to return the summary.
export type IntegrationWithSummary = {
  name: string;
  subKind: string;
  awsoidc: IntegrationSpecAwsOidc;
  // AWSEC2 contains the summary for the AWS EC2 resources for this integration.
  awsec2: ResourceTypeSummary;
  // AWSRDS contains the summary for the AWS RDS resources and agents for this integration.
  awsrds: ResourceTypeSummary;
  // AWSEKS contains the summary for the AWS EKS resources for this integration.
  awseks: ResourceTypeSummary;
};

// ResourceTypeSummary contains the summary of the enrollment rules and found resources by the integration.
export type ResourceTypeSummary = {
  // rulesCount is the number of enrollment rules that are using this integration.
  // A rule is a matcher in a DiscoveryConfig that is being processed by a DiscoveryService.
  // If the DiscoveryService is not reporting any Status, it means it is not being processed and it doesn't count for the number of rules.
  // Example 1: a DiscoveryConfig with a matcher whose Type is "EC2" for two regions count as two EC2 rules.
  // Example 2: a DiscoveryConfig with a matcher whose Types is "EC2,RDS" for one regions count as one EC2 rule.
  // Example 3: a DiscoveryConfig with a matcher whose Types is "EC2,RDS", but has no DiscoveryService using it, it counts as 0 rules.
  rulesCount: number;
  // resourcesFound contains the count of resources found by this integration.
  resourcesFound: number;
  // resourcesEnrollmentFailed contains the count of resources that failed to enroll into the cluster.
  resourcesEnrollmentFailed: number;
  // resourcesEnrollmentSuccess contains the count of resources that succeeded to enroll into the cluster.
  resourcesEnrollmentSuccess: number;
  // discoverLastSync contains the time when this integration tried to auto-enroll resources.
  discoverLastSync: number;
  // ecsDatabaseServiceCount is the total number of DatabaseServices that were deployed into Amazon ECS.
  // Only applicable for AWS RDS resource summary.
  ecsDatabaseServiceCount: number;
};

// awsRegionMap maps the AWS regions to it's region name
// as defined in (omitted gov cloud regions):
// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Concepts.RegionsAndAvailabilityZones.html
export const awsRegionMap = {
  'us-east-2': 'US East (Ohio)',
  'us-east-1': 'US East (N. Virginia)',
  'us-west-1': 'US West (N. California)',
  'us-west-2': 'US West (Oregon)',
  'af-south-1': 'Africa (Cape Town)',
  'ap-east-1': 'Asia Pacific (Hong Kong)',
  'ap-south-2': 'Asia Pacific (Hyderabad)',
  'ap-southeast-3': 'Asia Pacific (Jakarta)',
  'ap-southeast-4': 'Asia Pacific (Melbourne)',
  'ap-south-1': 'Asia Pacific (Mumbai)',
  'ap-northeast-3': 'Asia Pacific (Osaka)',
  'ap-northeast-2': 'Asia Pacific (Seoul)',
  'ap-southeast-1': 'Asia Pacific (Singapore)',
  'ap-southeast-2': 'Asia Pacific (Sydney)',
  'ap-northeast-1': 'Asia Pacific (Tokyo)',
  'ca-central-1': 'Canada (Central)',
  'eu-central-1': 'Europe (Frankfurt)',
  'eu-west-1': 'Europe (Ireland)',
  'eu-west-2': 'Europe (London)',
  'eu-south-1': 'Europe (Milan)',
  'eu-west-3': 'Europe (Paris)',
  'eu-south-2': 'Europe (Spain)',
  'eu-north-1': 'Europe (Stockholm)',
  'eu-central-2': 'Europe (Zurich)',
  'il-central-1': 'Israel (Tel Aviv)',
  'me-south-1': 'Middle East (Bahrain)',
  'me-central-1': 'Middle East (UAE)',
  'sa-east-1': 'South America (São Paulo)',
};

export type Regions = keyof typeof awsRegionMap;

// RdsEngine are the expected backend string values,
// used when requesting lists of rds databases of the
// specified engine.
export type RdsEngine =
  | 'aurora-mysql' // (for MySQL 5.7-compatible and MySQL 8.0-compatible Aurora)
  | 'aurora-postgresql'
  | 'mariadb'
  | 'mysql'
  | 'postgres';

// RdsEngineIdentifier are the name of engines
// used to determine the grouping of similar RdsEngines.
// eg: if `aurora-mysql` then the grouping of RdsEngines
// is 'aurora, aurora-mysql`, they are both mysql but
// refer to different versions. This type is used solely
// for frontend.
export type RdsEngineIdentifier =
  | 'mysql'
  | 'postgres'
  | 'aurora-mysql'
  | 'aurora-postgres';

export type AwsOidcListDatabasesRequest = {
  // engines is used as a filter to get a list of specified engines only.
  engines: RdsEngine[];
  region: Regions;
  // nextToken is the start key for the next page
  nextToken?: string;
  // rdsType describes the type of RDS dbs to request.
  // `cluster` is used for requesting aurora related
  // engines, and `instance` for rest of engines.
  rdsType: 'instance' | 'cluster';
};

export type AwsRdsDatabase = {
  // engine of the database. eg. aurora-mysql
  engine: RdsEngine;
  // name is the Database's name.
  name: string;
  // uri contains the endpoint with port for connecting to this Database.
  uri: string;
  // resourceId is the AWS Region-unique, immutable identifier for the DB.
  resourceId: string;
  // accountId is the AWS account id.
  accountId: string;
  // labels contains this Instance tags.
  labels: Label[];
  // subnets is a list of subnets for the RDS instance.
  subnets: string[];
  // vpcId is the AWS VPC ID for the DB.
  vpcId: string;
  /**
   * securityGroups is a list of AWS security group IDs attached to the DB.
   */
  securityGroups: string[];
  // region is the AWS cloud region that this database is from.
  region: Regions;
  // status contains this Instance status.
  // There is a lot of status states available so only a select few were
  // hard defined to use to determine the status color.
  // https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/accessing-monitoring.html
  status: 'available' | 'failed' | 'deleting';
};

export type ListAwsRdsDatabaseResponse = {
  databases: AwsRdsDatabase[];
  // nextToken is the start key for the next page.
  // Empty value means last page.
  nextToken?: string;
};

export type ListAwsRdsFromAllEnginesResponse = {
  databases: AwsRdsDatabase[];
  /**
   * next page for rds instances.
   */
  instancesNextToken?: string;
  /**
   * next page for rds clusters.
   */
  clustersNextToken?: string;
  /**
   * set if fetching rds instances OR rds clusters
   * returned an error
   */
  oneOfError?: string;
};

export type IntegrationUpdateRequest = {
  awsoidc: {
    roleArn: string;
  };
};

export type AwsOidcDeployServiceRequest = {
  deploymentMode: 'database-service';
  region: Regions;
  subnetIds: string[];
  taskRoleArn: string;
  securityGroups?: string[];
  vpcId: string;
  accountId: string;
};

/**
 * AwsOidcPolicyPreset specifies preset policy to apply
 * to the AWS IAM role created for the OIDC integration.
 */
export enum AwsOidcPolicyPreset {
  Unspecified = '',
  AwsIdentityCenter = 'aws-identity-center',
}

// DeployDatabaseServiceDeployment identifies the required fields to deploy a DatabaseService.
type DeployDatabaseServiceDeployment = {
  // VPCID is the VPCID where the service is going to be deployed.
  vpcId: string;
  // SubnetIDs are the subnets for the network configuration.
  // They must belong to the VPCID above.
  subnetIds: string[];
  // SecurityGroups are the SecurityGroup IDs to associate with this particular deployment.
  // If empty, the default security group for the VPC is going to be used.
  // TODO(lisa): out of scope.
  securityGroups?: string[];
};

// AwsOidcDeployDatabaseServicesRequest contains the required fields to perform a DeployService request.
// Each deployed DatabaseService will be proxying the resources that match the following labels:
// -region: <Region>
// -account-id: <AccountID>s
// -vpc-id: <Deployments[].VPCID>
export type AwsOidcDeployDatabaseServicesRequest = {
  // The AWS account to deploy the db service to.
  accountId: string;
  // Region is the AWS Region for the Service.
  region: string;
  // TaskRoleARN is the AWS Role's ARN used within the Task execution.
  // Ensure the AWS Client's Role has `iam:PassRole` for this Role's ARN.
  // This can be either the ARN or the short name of the AWS Role.
  taskRoleArn: string;
  // Deployments is a list of Services to be deployed.
  // If the target deployment already exists, the deployment is skipped.
  deployments: DeployDatabaseServiceDeployment[];
};

export type AwsEksCluster = {
  name: string;
  region: Regions;
  accountId: string;
  status:
    | 'active'
    | 'pending'
    | 'creating'
    | 'failed'
    | 'updating'
    | 'deleting';
  /**
   * labels contains this cluster's tags.
   */
  labels: Label[];
  /**
   * joinLabels contains labels that should be injected into teleport kube agent, if EKS cluster is being enrolled.
   */
  joinLabels: Label[];

  /**
   * AuthenticationMode is the cluster's configured authentication mode.
   * You can read more about the Authentication Modes here: https://aws.amazon.com/blogs/containers/a-deep-dive-into-simplified-amazon-eks-access-management-controls/
   */
  authenticationMode: 'API' | 'API_AND_CONFIG_MAP' | 'CONFIG_MAP';

  /**
   * EndpointPublicAccess indicates whether this cluster is publicly accessible.
   * This is a requirement for Teleport Cloud tenants because the control plane must be able to access the EKS Cluster
   * in order to deploy the helm chart.
   */
  endpointPublicAccess: boolean;
};

export type EnrollEksClustersRequest = {
  region: string;
  enableAppDiscovery: boolean;
  clusterNames: string[];
  /**
   * User provided labels.
   * Only supported with V2 endpoint
   */
  extraLabels?: ResourceLabel[];
};

export type EnrollEksClustersResponse = {
  results: {
    clusterName: string;
    resourceId: string;
    error: string;
  }[];
};

export type ListEksClustersRequest = {
  region: Regions;
  nextToken?: string;
};

export type ListEksClustersResponse = {
  /**
   * clusters is the list of EKS clusters.
   */
  clusters: AwsEksCluster[];
  nextToken?: string;
};

export type ListEc2InstancesRequest = {
  region: Regions;
  nextToken?: string;
};

export type ListEc2InstancesResponse = {
  // instances is the list of EC2 Instances.
  instances: Node[];
  nextToken?: string;
};

export type ListEc2InstanceConnectEndpointsRequest = {
  region: Regions;
  // VPCIDs is a list of VPCs to filter EC2 Instance Connect Endpoints.
  vpcIds: string[];
  nextToken?: string;
};

export type ListEc2InstanceConnectEndpointsResponse = {
  // endpoints is the list of EC2 Instance Connect Endpoints.
  endpoints: Ec2InstanceConnectEndpoint[];
  nextToken?: string;
  // DashboardLink is the URL for AWS Web Console that
  // lists all the Endpoints for the queries VPCs.
  dashboardLink: string;
};

export type Ec2InstanceConnectEndpoint = {
  name: string;
  // state is the current state of the EC2 Instance Connect Endpoint.
  state: Ec2InstanceConnectEndpointState;
  // stateMessage is an optional message describing the state of the EICE, such as an error message.
  stateMessage?: string;
  // dashboardLink is a URL to AWS Console where the user can see the EC2 Instance Connect Endpoint.
  dashboardLink: string;
  // subnetID is the subnet used by the Endpoint. Please note that the Endpoint should be able to reach any subnet within the VPC.
  subnetId: string;
  // VPCID is the VPC ID where the Endpoint is created.
  vpcId: string;
};

export type Ec2InstanceConnectEndpointState =
  | 'create-in-progress'
  | 'create-complete'
  | 'create-failed'
  | 'delete-in-progress'
  | 'delete-complete'
  | 'delete-failed';

export type AwsOidcDeployEc2InstanceConnectEndpointRequest = {
  // SubnetID is the subnet id for the EC2 Instance Connect Endpoint.
  subnetId: string;
  // SecurityGroupIDs is the list of SecurityGroups to apply to the Endpoint.
  // If not specified, the Endpoint will receive the default SG for the Subnet's VPC.
  securityGroupIds?: string[];
};

export type DeployEc2InstanceConnectEndpointRequest = {
  region: Regions;
  // Endpoints is a list of endpoinst to create.
  endpoints: AwsOidcDeployEc2InstanceConnectEndpointRequest[];
};

export type AwsEc2InstanceConnectEndpoint = {
  // Name is the EC2 Instance Connect Endpoint name.
  name: string;
  // SubnetID is the subnet where this endpoint was created.
  subnetId: string;
};

export type DeployEc2InstanceConnectEndpointResponse = {
  // Endpoints is a list of created endpoints
  endpoints: AwsEc2InstanceConnectEndpoint[];
};

export type Subnet = {
  /**
   * Subnet name.
   * This is just a friendly name and should not be used for further API calls.
   * It can be empty if the subnet was not given a "Name" tag.
   */
  name?: string;
  /**
   * Subnet ID, for example "subnet-0b3ca383195ad2cc7".
   * This is the value that should be used when doing further API calls.
   */
  id: string;
  /**
   *  AWS availability zone of the subnet, for example
   * "us-west-1a".
   */
  availabilityZone: string;
};

export type ListAwsSubnetsRequest = {
  vpcId: string;
  region: Regions;
  nextToken?: string;
};

export type ListAwsSubnetsResponse = {
  subnets: Subnet[];
  nextToken?: string;
};

export type ListAwsSecurityGroupsRequest = {
  // VPCID is the VPC to filter Security Groups.
  vpcId: string;
  region: Regions;
  nextToken?: string;
};

export type ListAwsSecurityGroupsResponse = {
  securityGroups: SecurityGroup[];
  nextToken?: string;
};

export type SecurityGroup = {
  // Name is the Security Group name.
  // This is just a friendly name and should not be used for further API calls
  name: string;
  // ID is the security group ID.
  // This is the value that should be used when doing further API calls.
  id: string;
  description: string;
  // InboundRules describe the Security Group Inbound Rules.
  // The CIDR of each rule represents the source IP that the rule applies to.
  inboundRules: SecurityGroupRule[];
  // OutboundRules describe the Security Group Outbound Rules.
  // The CIDR of each rule represents the destination IP that the rule applies to.
  outboundRules: SecurityGroupRule[];
};

export type SecurityGroupRule = {
  // IPProtocol is the protocol used to describe the rule.
  ipProtocol: string;
  // FromPort is the inclusive start of the Port range for the Rule.
  fromPort: string;
  // ToPort is the inclusive end of the Port range for the Rule.
  toPort: string;
  // CIDRs contains a list of IP ranges that this rule applies to and a description for the value.
  cidrs: Cidr[];
  // Groups is a list of rules that allow another security group referenced
  // by ID.
  groups: GroupIdRule[];
};

export type Cidr = {
  /**
   * CIDR is the IP range using CIDR notation.
   */
  cidr: string;
  /**
   *  Description contains a small text describing the CIDR.
   */
  description: string;
};

export type GroupIdRule = {
  /**
   * GroupId is the ID of the security group that is allowed by the rule.
   */
  groupId: string;
  /**
   * Description contains a small text describing the rule.
   */
  description: string;
};

// IntegrationUrlLocationState define fields to preserve state between
// react routes (eg. in External Audit Storage flow, it is required of user
// to create a AWS OIDC integration which requires changing route
// and then coming back to resume the flow.)
export type IntegrationUrlLocationState = {
  kind: IntegrationKind;
  redirectText: string;
};

export type Vpc = {
  id: string;
  name?: string;
  /**
   * if defined, a database service is already deployed for this vpc.
   */
  ecsServiceDashboardURL?: string;
};

export type AwsDatabaseVpcsResponse = {
  vpcs: Vpc[];
  nextToken: string;
};
