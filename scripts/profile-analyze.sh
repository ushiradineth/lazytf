#!/usr/bin/env bash
# Profile analysis helper script for lazytf
# Usage: ./scripts/profile-analyze.sh [cpu|mem|trace|stats] [profile-file]

set -e

PROFILE_TYPE="${1:-cpu}"
PROFILE_FILE="${2:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${GREEN}=== $1 ===${NC}"
}

print_info() {
    echo -e "${YELLOW}$1${NC}"
}

# Find the most recent profile file if not specified
find_latest_profile() {
    local pattern="$1"
    local latest
    latest=$(ls -t $pattern 2>/dev/null | head -1)
    echo "$latest"
}

case "$PROFILE_TYPE" in
    cpu)
        if [[ -z "$PROFILE_FILE" ]]; then
            PROFILE_FILE=$(find_latest_profile "lazytf-cpu-*.prof")
        fi

        if [[ -z "$PROFILE_FILE" || ! -f "$PROFILE_FILE" ]]; then
            echo -e "${RED}No CPU profile found. Run with: LAZYTF_PROFILE=cpu ./lazytf${NC}"
            exit 1
        fi

        print_header "CPU Profile Analysis: $PROFILE_FILE"
        echo ""
        print_info "Opening interactive pprof..."
        echo "Useful commands:"
        echo "  top10         - Show top 10 CPU consumers"
        echo "  top20 -cum    - Show top 20 by cumulative time"
        echo "  list funcname - Show source for a function"
        echo "  web           - Open flame graph in browser"
        echo "  png           - Generate PNG flame graph"
        echo ""
        go tool pprof "$PROFILE_FILE"
        ;;

    mem|memory|heap)
        if [[ -z "$PROFILE_FILE" ]]; then
            PROFILE_FILE=$(find_latest_profile "lazytf-mem-*.prof")
        fi

        if [[ -z "$PROFILE_FILE" || ! -f "$PROFILE_FILE" ]]; then
            echo -e "${RED}No memory profile found. Run with: LAZYTF_PROFILE=mem ./lazytf${NC}"
            exit 1
        fi

        print_header "Memory Profile Analysis: $PROFILE_FILE"
        echo ""
        print_info "Opening interactive pprof..."
        echo "Useful commands:"
        echo "  top10              - Show top 10 allocators"
        echo "  top10 -cum         - By cumulative allocations"
        echo "  top10 -inuse_space - By bytes in use"
        echo "  top10 -alloc_space - By total allocated"
        echo "  list funcname      - Show source for a function"
        echo "  web                - Open graph in browser"
        echo ""
        go tool pprof "$PROFILE_FILE"
        ;;

    trace)
        if [[ -z "$PROFILE_FILE" ]]; then
            PROFILE_FILE=$(find_latest_profile "lazytf-trace-*.out")
        fi

        if [[ -z "$PROFILE_FILE" || ! -f "$PROFILE_FILE" ]]; then
            echo -e "${RED}No trace file found. Run with: LAZYTF_PROFILE=trace ./lazytf${NC}"
            exit 1
        fi

        print_header "Execution Trace: $PROFILE_FILE"
        echo ""
        print_info "Opening trace viewer in browser..."
        echo "The trace viewer shows:"
        echo "  - Goroutine timeline"
        echo "  - Network/syscall blocking"
        echo "  - GC pauses"
        echo "  - Scheduler latency"
        echo ""
        go tool trace "$PROFILE_FILE"
        ;;

    stats)
        if [[ -z "$PROFILE_FILE" ]]; then
            PROFILE_FILE=$(find_latest_profile "lazytf-stats-*.csv")
        fi

        if [[ -z "$PROFILE_FILE" || ! -f "$PROFILE_FILE" ]]; then
            echo -e "${RED}No stats file found. Run with: LAZYTF_PROFILE=stats ./lazytf${NC}"
            exit 1
        fi

        print_header "Runtime Stats: $PROFILE_FILE"
        echo ""

        # Show basic stats
        echo "File: $PROFILE_FILE"
        echo "Records: $(wc -l < "$PROFILE_FILE")"
        echo ""

        print_info "Memory usage over time (heap_alloc_mb):"
        if command -v awk &> /dev/null; then
            awk -F',' 'NR>1 {print $1, $2 " MB"}' "$PROFILE_FILE" | head -20
            echo "..."
            awk -F',' 'NR>1 {print $1, $2 " MB"}' "$PROFILE_FILE" | tail -5
        else
            head -20 "$PROFILE_FILE"
        fi

        echo ""
        print_info "Summary:"
        if command -v awk &> /dev/null; then
            awk -F',' 'NR>1 {
                if(min_heap=="" || $2<min_heap) min_heap=$2;
                if($2>max_heap) max_heap=$2;
                if(min_gc=="" || $5<min_gc) min_gc=$5;
                if($5>max_gc) max_gc=$5;
                if(min_gor=="" || $6<min_gor) min_gor=$6;
                if($6>max_gor) max_gor=$6;
                total_heap+=$2; count++
            } END {
                print "Heap Alloc: min=" min_heap " MB, max=" max_heap " MB, avg=" total_heap/count " MB"
                print "GC cycles: " min_gc " to " max_gc
                print "Goroutines: " min_gor " to " max_gor
            }' "$PROFILE_FILE"
        fi
        ;;

    list)
        print_header "Available Profile Files"
        echo ""
        for f in lazytf-*.prof lazytf-*.out lazytf-*.csv; do
            if [[ -f "$f" ]]; then
                echo "  $f ($(du -h "$f" | cut -f1))"
            fi
        done
        ;;

    help|*)
        echo "Profile Analysis Helper for lazytf"
        echo ""
        echo "Usage: $0 <type> [file]"
        echo ""
        echo "Types:"
        echo "  cpu    - Analyze CPU profile (default)"
        echo "  mem    - Analyze memory/heap profile"
        echo "  trace  - Open execution trace viewer"
        echo "  stats  - Show runtime stats CSV"
        echo "  list   - List available profile files"
        echo ""
        echo "Profiling:"
        echo "  # Enable via environment variable"
        echo "  LAZYTF_PROFILE=cpu,mem,stats ./lazytf"
        echo ""
        echo "  # Enable via flag"
        echo "  ./lazytf --profile cpu,mem,trace,stats"
        echo ""
        echo "  # Enable all profiling"
        echo "  LAZYTF_PROFILE=all ./lazytf"
        echo ""
        echo "Quick Analysis:"
        echo "  # CPU - find hot functions"
        echo "  go tool pprof -top lazytf-cpu-*.prof"
        echo ""
        echo "  # Memory - find allocators"
        echo "  go tool pprof -top -inuse_space lazytf-mem-*.prof"
        echo ""
        echo "  # Generate flame graph (requires graphviz)"
        echo "  go tool pprof -web lazytf-cpu-*.prof"
        ;;
esac
