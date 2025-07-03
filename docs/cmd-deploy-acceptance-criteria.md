# CMD-Deploy Tool - Acceptance Criteria

## Overview
The `cmd-deploy` tool is a CLI orchestrator that manages staged infrastructure deployment using Pulumi microstacks.

## Acceptance Criteria

### 🚀 Initialize Infrastructure (`cmd-deploy init`)
**AC-01**: GIVEN I run `cmd-deploy init`  
**WHEN** the command executes  
**THEN** it SHALL:
- ✅ Create S3 state bucket with versioning and encryption
- ✅ Create AWS Secrets Manager secrets from config
- ✅ Create Route53 hosted zone for the domain
- ✅ Display NS records that must be manually configured
- ✅ Configure Pulumi backend to use S3 bucket
- ✅ Initialize all Pulumi stacks for all stages/microstacks
- ✅ Warn user about DNS propagation requirements

### 🏗️ Deploy Stages (`cmd-deploy [vpc|core|apps] [up|down|preview]`)
**AC-02**: GIVEN I run `cmd-deploy vpc up`  
**WHEN** the command executes  
**THEN** it SHALL:
- ✅ Load configuration from YAML file
- ✅ Connect to S3 Pulumi backend
- ✅ Process VPC microstacks in sequence: networking → acls
- ✅ Execute `pulumi up` for each microstack
- ✅ Pass microstack context to Pulumi program
- ✅ Report success/failure for each microstack

**AC-03**: GIVEN I run `cmd-deploy core up`  
**WHEN** the command executes  
**THEN** it SHALL:
- ✅ Process CORE microstacks in sequence: s3 → route53 → rds → eks → opensearch → cloudfront → certificates
- ✅ Handle cross-stack dependencies automatically
- ✅ Fail fast if prerequisites are missing

**AC-04**: GIVEN I run `cmd-deploy apps up`  
**WHEN** the command executes  
**THEN** it SHALL:
- ✅ Process APPS microstacks in sequence: eks-addons → helm-charts → storage-classes → ingress-classes
- ✅ Validate EKS cluster exists before deployment

### 📋 Configuration Management
**AC-05**: GIVEN a valid YAML config file  
**WHEN** the tool loads configuration  
**THEN** it SHALL:
- ✅ Parse customer, environment, domain, and region settings
- ✅ Validate required fields are present
- ✅ Pass configuration to Pulumi program via environment variables

### 🎯 Stack Management  
**AC-06**: GIVEN microstacks are defined  
**WHEN** stacks are created  
**THEN** they SHALL:
- ✅ Follow naming convention: `{customer}-{stage}-{microstack}-{region}`
- ✅ Use shared S3 backend: `pulumi-state-{env}-{customer}`
- ✅ Maintain isolated state per microstack
- ✅ Support parallel team development

### 🔧 Error Handling
**AC-07**: GIVEN invalid inputs or AWS errors  
**WHEN** the tool encounters failures  
**THEN** it SHALL:
- ✅ Display clear error messages
- ✅ Exit with appropriate status codes
- ✅ Handle existing resources gracefully
- ✅ Provide guidance for manual intervention

### 📚 CLI Interface
**AC-08**: GIVEN the CLI tool  
**WHEN** users interact with it  
**THEN** it SHALL:
- ✅ Support `--config` flag for custom config files
- ✅ Respect `AWS_REGION` environment variable
- ✅ Display helpful usage information with `--help`
- ✅ Validate stage names: init, vpc, core, apps
- ✅ Support actions: up, down, preview

## Success Metrics

### Functional Requirements
- ✅ Successfully deploys 13 microstacks across 3 stages
- ✅ Creates and manages 40+ AWS resources
- ✅ Supports multiple environments (dev, staging, prod)
- ✅ Handles cross-stack resource dependencies

### Non-Functional Requirements  
- ✅ Zero-downtime updates for individual microstacks
- ✅ Isolated failure domains (one microstack failure doesn't affect others)
- ✅ Auditable state management via S3 versioning
- ✅ Team-friendly parallel development workflow

## Out of Scope
- ❌ Resource-level configuration (handled by Pulumi program)
- ❌ AWS credential management (uses existing AWS CLI setup)
- ❌ Automatic DNS propagation verification
- ❌ Resource cost optimization recommendations

## Definition of Done
- [ ] All acceptance criteria pass
- [ ] Integration tests with real AWS resources
- [ ] Documentation includes usage examples
- [ ] Error scenarios are tested and documented
- [ ] Performance meets requirements (<5 min per stage)
