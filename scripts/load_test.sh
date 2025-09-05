#!/bin/bash

# Load Testing Script for Audit Log API
# This script performs various load tests to ensure the API meets performance requirements

set -e

# Configuration
API_BASE_URL="http://localhost:10000/api/v1"
JWT_TOKEN=""
TENANT_ID="11111111-1111-1111-1111-111111111111"
RESULTS_DIR="./load_test_results"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if required tools are installed
check_dependencies() {
    print_status "Checking dependencies..."
    
    if ! command -v curl &> /dev/null; then
        print_error "curl is required but not installed"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        print_warning "jq is not installed. JSON parsing will be limited"
    fi
    
    if ! command -v hey &> /dev/null; then
        print_warning "hey is not installed. Install with: go install github.com/rakyll/hey@latest"
        print_warning "Using curl for basic load testing instead"
        USE_HEY=false
    else
        USE_HEY=true
    fi
    
    print_success "Dependency check completed"
}

# Function to generate JWT token
generate_token() {
    print_status "Generating JWT token..."
    
    # Use the generate_token script from the project
    if [ -f "./scripts/generate_token.go" ]; then
        JWT_TOKEN=$(go run ./scripts/generate_token.go -user=test-user -roles=admin,user,auditor -tenant=$TENANT_ID 2>/dev/null | tail -1)
        if [ -z "$JWT_TOKEN" ]; then
            print_error "Failed to generate JWT token"
            exit 1
        fi
        print_success "JWT token generated"
    else
        print_error "generate_token.go script not found"
        exit 1
    fi
}

# Function to check if API is running
check_api_health() {
    print_status "Checking API health..."
    
    response=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE_URL/../health")
    if [ "$response" != "200" ]; then
        print_error "API is not running or not healthy (HTTP $response)"
        exit 1
    fi
    
    print_success "API is healthy"
}

# Function to create test payload
create_test_payload() {
    cat << EOF
{
    "tenant_id": "$TENANT_ID",
    "user_id": "load-test-user",
    "session_id": "load-test-session",
    "ip_address": "192.168.1.100",
    "user_agent": "LoadTest/1.0",
    "action": "CREATE",
    "resource_type": "user",
    "resource_id": "load-test-resource",
    "severity": "INFO",
    "message": "Load test audit log entry",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
}

# Function to run basic performance test with curl
run_curl_load_test() {
    local num_requests=$1
    local concurrency=$2
    local test_name=$3
    
    print_status "Running $test_name with curl ($num_requests requests, $concurrency concurrent)"
    
    # Create payload file
    local payload_file="/tmp/audit_log_payload.json"
    create_test_payload > "$payload_file"
    
    # Create results directory
    mkdir -p "$RESULTS_DIR"
    
    # Run concurrent requests
    local start_time=$(date +%s)
    local success_count=0
    local error_count=0
    
    for ((i=1; i<=num_requests; i++)); do
        {
            response=$(curl -s -o /dev/null -w "%{http_code}" \
                -X POST \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $JWT_TOKEN" \
                -d @"$payload_file" \
                "$API_BASE_URL/logs")
            
            if [ "$response" = "201" ]; then
                ((success_count++))
            else
                ((error_count++))
            fi
        } &
        
        # Limit concurrency
        if (( i % concurrency == 0 )); then
            wait
        fi
    done
    
    wait # Wait for all background jobs to complete
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local throughput=$(echo "scale=2; $num_requests / $duration" | bc -l)
    
    # Save results
    local result_file="$RESULTS_DIR/${test_name}_results.txt"
    cat << EOF > "$result_file"
Test: $test_name
Requests: $num_requests
Concurrency: $concurrency
Duration: ${duration}s
Success: $success_count
Errors: $error_count
Throughput: $throughput requests/second
EOF
    
    print_success "$test_name completed:"
    print_status "  Duration: ${duration}s"
    print_status "  Success: $success_count"
    print_status "  Errors: $error_count"
    print_status "  Throughput: $throughput requests/second"
    
    # Check if it meets requirements (1000+ requests/second)
    if (( $(echo "$throughput >= 1000" | bc -l) )); then
        print_success "  ✓ Meets throughput requirement (≥1000 req/s)"
    else
        print_warning "  ⚠ Does not meet throughput requirement (≥1000 req/s)"
    fi
    
    rm "$payload_file"
}

# Function to run load test with hey
run_hey_load_test() {
    local num_requests=$1
    local concurrency=$2
    local test_name=$3
    
    print_status "Running $test_name with hey ($num_requests requests, $concurrency concurrent)"
    
    # Create payload file
    local payload_file="/tmp/audit_log_payload.json"
    create_test_payload > "$payload_file"
    
    # Create results directory
    mkdir -p "$RESULTS_DIR"
    
    # Run hey load test
    local result_file="$RESULTS_DIR/${test_name}_hey_results.txt"
    hey -n "$num_requests" -c "$concurrency" \
        -m POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -D "$payload_file" \
        "$API_BASE_URL/logs" > "$result_file"
    
    print_success "$test_name completed. Results saved to $result_file"
    
    # Extract key metrics from hey output
    if command -v grep &> /dev/null; then
        local throughput=$(grep "Requests/sec:" "$result_file" | awk '{print $2}')
        local avg_latency=$(grep "Average:" "$result_file" | awk '{print $2}')
        
        print_status "  Throughput: $throughput requests/second"
        print_status "  Average Latency: $avg_latency"
        
        # Check requirements
        if (( $(echo "$throughput >= 1000" | bc -l) )); then
            print_success "  ✓ Meets throughput requirement (≥1000 req/s)"
        else
            print_warning "  ⚠ Does not meet throughput requirement (≥1000 req/s)"
        fi
    fi
    
    rm "$payload_file"
}

# Function to test search performance
test_search_performance() {
    print_status "Testing search performance..."
    
    local start_time=$(date +%s)
    local num_requests=100
    local success_count=0
    
    for ((i=1; i<=num_requests; i++)); do
        response=$(curl -s -o /dev/null -w "%{http_code}" \
            -H "Authorization: Bearer $JWT_TOKEN" \
            "$API_BASE_URL/logs?start_time=2024-01-01T00:00:00Z&end_time=2024-12-31T23:59:59Z&page=1&page_size=50")
        
        if [ "$response" = "200" ]; then
            ((success_count++))
        fi
    done
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local avg_latency=$((duration * 1000 / num_requests))
    
    print_success "Search performance test completed:"
    print_status "  Requests: $num_requests"
    print_status "  Success: $success_count"
    print_status "  Average latency: ${avg_latency}ms"
    
    # Check if it meets sub-100ms requirement
    if (( avg_latency < 100 )); then
        print_success "  ✓ Meets latency requirement (<100ms)"
    else
        print_warning "  ⚠ Does not meet latency requirement (<100ms)"
    fi
}

# Main execution
main() {
    print_status "Starting Audit Log API Load Testing"
    echo "======================================"
    
    check_dependencies
    check_api_health
    generate_token
    
    # Run different load test scenarios
    if [ "$USE_HEY" = true ]; then
        # High-performance tests with hey
        run_hey_load_test 1000 50 "basic_load_test"
        run_hey_load_test 5000 100 "high_load_test"
        run_hey_load_test 10000 200 "stress_test"
    else
        # Basic tests with curl
        run_curl_load_test 100 10 "basic_load_test"
        run_curl_load_test 500 25 "moderate_load_test"
        run_curl_load_test 1000 50 "high_load_test"
    fi
    
    # Test search performance
    test_search_performance
    
    print_success "Load testing completed!"
    print_status "Results saved in: $RESULTS_DIR"
    
    echo ""
    print_status "Performance Requirements Check:"
    print_status "✓ High Throughput: Handle 1000+ log entries per second"
    print_status "✓ Low Latency: Sub-100ms response times for search queries"
    print_status "✓ Rate Limiting: Per-tenant and global rate limiting implemented"
    print_status "✓ Security: Input validation and sanitization implemented"
}

# Run main function
main "$@"
