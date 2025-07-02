# CMD-Deploy Tool Pseudocode

## Overview

The `cmd-deploy` tool is a CLI that manages infrastructure deployment through staged microstacks. Each stage (vpc, core, apps) contains multiple microstacks that are executed in sequence.

## Program Flow

```pseudocode
PROGRAM cmd-deploy
BEGIN
    // Command Line Interface
    PARSE command_line_arguments
    
    SWITCH stage:
        CASE "init":
            CALL initialize_infrastructure()
            
        CASE "vpc", "core", "apps":
            CALL deploy_stage(stage, action, config_file)
            
        DEFAULT:
            PRINT usage_help()
    END SWITCH
END

FUNCTION initialize_infrastructure():
BEGIN
    config_file = GET config_file_from_args()
    config = LOAD yaml_config(config_file)
    region = GET environment_variable("AWS_REGION")
    
    REQUIRE region is not empty
    
    // Create S3 state bucket
    bucket_name = FORMAT("pulumi-state-{env}-{customer}")
    
    IF bucket_exists(bucket_name):
        PRINT "Bucket already exists"
    ELSE:
        CREATE s3_bucket(bucket_name, region)
        ENABLE versioning(bucket_name)
    
    // Configure Pulumi backend
    RUN_COMMAND("pulumi login s3://" + bucket_name)
    
    // Create stacks for each stage
    FOR each stage in ["vpc", "core", "apps"]:
        microstacks = GET_MICROSTACKS_FOR_STAGE(stage)
        
        FOR each microstack in microstacks:
            stack_name = FORMAT("{customer}-{stage}-{microstack}-{region}")
            TRY:
                RUN_COMMAND("pulumi stack init " + stack_name)
                PRINT "Created stack: " + stack_name
            CATCH:
                PRINT "Stack " + stack_name + " may already exist, continuing..."
    
    PRINT "✅ Initialization complete"
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
- S3 state bucket with versioning
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
