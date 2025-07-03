# CMD-Deploy Tool Pseudocode

## Overview

The `cmd-deploy` tool is a CLI that manages infrastructure deployment through staged microstacks. Each stage (vpc, core, apps) contains multiple microstacks that are executed in sequence.

## Program Flow

```pseudocode
PROGRAM cmd-deploy
BEGIN
    // Command Line Interface
    PARSE command_line_arguments
    
    // Extract common parameters
    config_file = GET config_file_from_args() OR "configs/sample-config.yaml"
    
    SWITCH stage:
        CASE "init":
            CALL initialize_infrastructure(config_file)
            
        CASE "vpc", "core", "apps":
            action = GET action_from_args()
            CALL deploy_stage(stage, action, config_file)
            
        DEFAULT:
            PRINT usage_help()
    END SWITCH
END

FUNCTION initialize_infrastructure(config_file):
BEGIN
    config = LOAD yaml_config(config_file)
    region = GET environment_variable("AWS_REGION")
    
    REQUIRE region is not empty
    REQUIRE config.domain is not empty
    
    PRINT "🚀 Starting infrastructure initialization..."
    
    // Step 1: Create S3 state bucket
    PRINT "📦 Creating S3 state bucket..."
    bucket_name = FORMAT("pulumi-state-{env}-{customer}")
    
    IF bucket_exists(bucket_name):
        PRINT "✅ S3 bucket already exists: " + bucket_name
    ELSE:
        CREATE s3_bucket(bucket_name, region)
        ENABLE versioning(bucket_name)
        ENABLE encryption(bucket_name)
        PRINT "✅ Created S3 bucket: " + bucket_name
    
    // Step 2: Create AWS Secrets Manager secrets
    PRINT "🔐 Creating AWS Secrets Manager secrets..."
    CALL create_secrets_manager_secrets(config, region)
    
    // Step 3: Create Route53 hosted zone
    PRINT "🌐 Creating Route53 hosted zone..."
    hosted_zone_id = CALL create_route53_hosted_zone(config.domain, region)
    
    // Step 4: Configure Pulumi backend
    PRINT "⚙️ Configuring Pulumi backend..."
    RUN_COMMAND("pulumi login s3://" + bucket_name)
    
    // Step 5: Create stacks for each stage
    PRINT "📚 Creating Pulumi stacks..."
    FOR each stage in ["vpc", "core", "apps"]:
        microstacks = GET_MICROSTACKS_FOR_STAGE(stage)
        
        FOR each microstack in microstacks:
            stack_name = FORMAT("{customer}-{stage}-{microstack}-{region}")
            TRY:
                RUN_COMMAND("pulumi stack init " + stack_name)
                PRINT "✅ Created stack: " + stack_name
            CATCH:
                PRINT "⚠️ Stack " + stack_name + " may already exist, continuing..."
    
    // Step 6: Display critical next steps
    PRINT ""
    PRINT "🎯 CRITICAL: Manual NS Configuration Required"
    PRINT "=============================================="
    PRINT "Before proceeding with deployment, you MUST configure the following:"
    PRINT ""
    PRINT "1. 📋 Copy these NS records from Route53 hosted zone '" + config.domain + "':"
    ns_records = GET route53_ns_records(hosted_zone_id)
    FOR each ns_record in ns_records:
        PRINT "   " + ns_record
    PRINT ""
    PRINT "2. 🌐 Configure these NS records with your domain registrar for: " + config.domain
    PRINT "3. ⏱️ Wait for DNS propagation (can take up to 48 hours)"
    PRINT "4. ✅ Verify NS propagation: dig NS " + config.domain
    PRINT ""
    PRINT "⚠️ WARNING: DNS must propagate before running certificate deployment!"
    PRINT ""
    PRINT "✅ Initialization complete - Ready for stage deployment"
    PRINT "Next steps:"
    PRINT "  ./cmd-deploy vpc up --config " + config_file
    PRINT "  ./cmd-deploy core up --config " + config_file  
    PRINT "  ./cmd-deploy apps up --config " + config_file
END

FUNCTION deploy_stage(stage, action, config_file):
BEGIN
    config = LOAD yaml_config(config_file)
    region = GET environment_variable("AWS_REGION")
    
    REQUIRE region is not empty
    
    bucket_name = FORMAT("pulumi-state-{env}-{customer}")
    
    // Connect to Pulumi backend
    RUN_COMMAND("pulumi login s3://" + bucket_name)
    
    // Get microstacks for this stage
    microstacks = GET_MICROSTACKS_FOR_STAGE(stage)
    
    PRINT "Deploying stage: " + stage + " with action: " + action
    PRINT "Microstacks to process: " + JOIN(microstacks, ", ")
    
    // Execute action on each microstack in order
    FOR each microstack in microstacks:
        stack_name = FORMAT("{customer}-{stage}-{microstack}-{region}")
        
        PRINT "Processing microstack: " + microstack + " (" + stack_name + ")"
        
        // Select the microstack
        RUN_COMMAND("pulumi stack select " + stack_name)
        
        // Pass config to Pulumi program
        SET environment_variable("CONFIG_FILE", config_file)
        RUN_COMMAND("pulumi config set microstack " + microstack)
        
        // Execute Pulumi action on this microstack
        IF action in ["up", "down", "preview"]:
            RUN_COMMAND("pulumi " + action + " --yes")
        ELSE:
            RUN_COMMAND("pulumi " + action)
        
        PRINT "✅ Completed " + microstack + " " + action
    
    PRINT "✅ Successfully completed stage " + stage + " " + action
END

FUNCTION GET_MICROSTACKS_FOR_STAGE(stage):
BEGIN
    SWITCH stage:
        CASE "vpc":
            RETURN ["networking", "acls"]
            
        CASE "core":
            RETURN ["s3", "route53", "rds", "eks", "opensearch", "cloudfront", "certificates"]
            
        CASE "apps":
            RETURN ["eks-addons", "helm-charts", "storage-classes", "ingress-classes"]
            
        DEFAULT:
            RETURN []
    END SWITCH
END
```

## Architecture

### Microstack Organization

```
Stage: VPC
├── microstack: networking       → dimarmen-vpc-networking-us-east-1
└── microstack: acls             → dimarmen-vpc-acls-us-east-1

Stage: CORE  
├── microstack: s3               → dimarmen-core-s3-us-east-1
├── microstack: route53          → dimarmen-core-route53-us-east-1
├── microstack: rds              → dimarmen-core-rds-us-east-1
├── microstack: eks              → dimarmen-core-eks-us-east-1
├── microstack: opensearch       → dimarmen-core-opensearch-us-east-1
├── microstack: cloudfront       → dimarmen-core-cloudfront-us-east-1
└── microstack: certificates     → dimarmen-core-certificates-us-east-1

Stage: APPS
├── microstack: eks-addons       → dimarmen-apps-eks-addons-us-east-1
├── microstack: helm-charts      → dimarmen-apps-helm-charts-us-east-1
├── microstack: storage-classes  → dimarmen-apps-storage-classes-us-east-1
└── microstack: ingress-classes  → dimarmen-apps-ingress-classes-us-east-1
```

### Naming Conventions

- **S3 Bucket**: `pulumi-state-{environment}-{customer}`
- **Stack**: `{customer}-{stage}-{microstack}-{region}`

### Examples

```bash
# S3 Bucket
pulumi-state-dev-dimarmen

# Stack Names
dimarmen-vpc-networking-us-east-1
dimarmen-core-rds-us-east-1
dimarmen-apps-eks-addons-us-east-1
```

## Usage Examples

### Initialize Infrastructure

```bash
./cmd-deploy init --config configs/sample-config.yaml
```

This creates:
- S3 state bucket with versioning and encryption
- AWS Secrets Manager secrets
- Route53 hosted zone
- All microstack stacks for all stages

### Deploy Stages

```bash
# Deploy VPC stage (processes: networking → acls)
./cmd-deploy vpc up --config configs/sample-config.yaml

# Deploy CORE stage (processes: s3 → route53 → rds → eks → opensearch → cloudfront → certificates)  
./cmd-deploy core up --config configs/sample-config.yaml

# Deploy APPS stage (processes: eks-addons → helm-charts → storage-classes → ingress-classes)
./cmd-deploy apps up --config configs/sample-config.yaml
```

### Other Actions

```bash
# Preview changes
./cmd-deploy vpc preview --config configs/sample-config.yaml

# Destroy infrastructure  
./cmd-deploy apps down --config configs/sample-config.yaml

# Use default config file
./cmd-deploy vpc up

# Use environment-specific config
./cmd-deploy core up --config configs/production-config.yaml
```

## Benefits of Microstack Architecture

- **Granular Control**: Each microstack can be managed independently
- **Dependency Management**: Microstacks within a stage execute in order  
- **Isolated State**: Each microstack has its own Pulumi state file
- **Fault Isolation**: Failure in one microstack doesn't affect others
- **Stage-based Organization**: Pulumi program organized by stages, not individual microstacks
- **Centralized Definitions**: Microstack definitions only in CLI tool
- **Selective Deployment**: Can target specific microstacks for updates
- **Clear Separation**: CLI handles orchestration, Pulumi handles resources

## Configuration Flow

1. **CLI Tool** loads YAML config file and iterates microstacks
2. **Environment Variables** pass config file path to Pulumi program  
3. **Pulumi Config** stores individual microstack information
4. **Pulumi Program** determines stage from microstack and routes to appropriate function
5. **Stage Functions** deploy the actual AWS resources

```
Config YAML → CLI Tool → Microstack Iteration → Pulumi Program → Stage Router → Microstack Function → AWS Resources
```

## Pulumi Program Architecture

The Pulumi program is organized by stages, not individual microstacks:

```go
// CLI iterates and sets: microstack = "networking"
// Pulumi program receives microstack and determines stage:
stage := getStageFromMicrostack("networking") // Returns "vpc"
switch stage {
case "vpc":
    return deployVPCStage(ctx, cfg, "networking") // Routes to deployNetworking()
case "core":
    return deployCoreStage(ctx, cfg, microstack)
case "apps": 
    return deployAppsStage(ctx, cfg, microstack)
}
```

### Key Design Principles

1. **Single Source of Truth**: Microstack definitions only exist in CLI tool
2. **Stage-based Routing**: Pulumi program routes by stage, then microstack
3. **Clean Separation**: CLI orchestrates, Pulumi deploys
4. **Maintainable**: Add new microstacks by updating only the CLI tool

This design keeps the microstack definitions centralized in the CLI tool while allowing the Pulumi program to focus on stage-based resource deployment.

## Implementation Status

### ✅ Completed Features

1. **CLI Tool (`cmd-deploy`)**:
   - ✅ Configuration loading from YAML
   - ✅ S3 state bucket creation with native Go AWS SDK
   - ✅ Microstack iteration and orchestration
   - ✅ Pulumi stack management
   - ✅ Error handling for existing resources

2. **Pulumi Program (`main.go`)**:
   - ✅ Stage-based architecture (vpc/core/apps)
   - ✅ Microstack routing and dispatch
   - ✅ Configuration loading from environment variables
   - ✅ All 13 microstack function placeholders

3. **Project Structure**:
   - ✅ Clean separation of CLI tool and Pulumi program
   - ✅ Centralized configuration management
   - ✅ Comprehensive documentation

### 🔄 Ready for Implementation

- **AWS Resource Creation**: All microstack functions are placeholders ready for AWS resource implementation
- **Dependencies**: Cross-microstack resource references (e.g., VPC outputs to Core inputs)
- **Validation**: Enhanced error checking and configuration validation
- **Testing**: Integration tests with actual AWS resources

### 🎯 Next Steps

1. Implement actual AWS resources in microstack functions
2. Add cross-stack resource references
3. Enhance error handling and validation
4. Add comprehensive testing
