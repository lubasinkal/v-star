// bench.zig - Performance benchmark suite
//
// Port of Go cmd/v-star/commands/bench.go.
// Outputs benchmark results via std.Io.Writer.
const std = @import("std");
const mem = std.mem;
const Allocator = std.mem.Allocator;

const rates = @import("rates.zig");
const reader = @import("reader.zig");
const stochastic = @import("stochastic.zig");
const risk = @import("risk.zig");
const concurrency = @import("concurrency.zig");

/// Runs the full benchmark suite, writing results to `out`.
pub fn bench(out: anytype, err: anytype, gpa: Allocator, io: std.Io) !void {
    _ = err;
    _ = gpa;
    try out.print("=== V-star Benchmark Suite ===\n\n", .{});

    try benchPresentValue(out, io);
    try benchMonteCarlo(out, io);
    try benchRiskMeasures(out, io);
    try benchCSVParsing(out, io);

    try out.print("=== Benchmarks Complete ===\n", .{});
}

fn printBenchResult(out: anytype, name: []const u8, ops: usize, elapsed_ns: u64) !void {
    const elapsed_sec = @as(f64, @floatFromInt(elapsed_ns)) / 1_000_000_000.0;
    const throughput = if (elapsed_sec > 0) @as(f64, @floatFromInt(ops)) / elapsed_sec else 0;
    try out.print("  {s: <30}: {d: >12} ops, {d:.3}s, {d:.0} ops/sec\n", .{ name, ops, elapsed_sec, throughput });
}

// ---------------------------------------------------------------------------
// Individual benchmarks
// ---------------------------------------------------------------------------

fn benchPresentValue(out: anytype, io: std.Io) !void {
    try out.print("Present Value Benchmarks:\n", .{});
    const allocator = std.heap.page_allocator;

    var rc = try rates.RateConverter.init(allocator, 0.05);
    defer rc.deinit();

    // Benchmark RateConverter.PresentValue
    const pv_count: usize = 10_000_000;
    const start = std.Io.Timestamp.now(io, .real);
    var pv_i: usize = 0;
    while (pv_i < pv_count) : (pv_i += 1) {
        _ = rc.presentValue(100000.0, @intCast(pv_i % 30 + 1));
    }
    const end = std.Io.Timestamp.now(io, .real);
    const elapsed: u64 = @intCast(end.nanoseconds - start.nanoseconds);
    try printBenchResult(out, "PresentValue (10M calls)", pv_count, elapsed);

    // Benchmark v-star discount factor
    const pv2_count: usize = 10_000_000;
    const start2 = std.Io.Timestamp.now(io, .real);
    var pv2_i: usize = 0;
    while (pv2_i < pv2_count) : (pv2_i += 1) {
        _ = rc.presentValueStar(100000.0, @intCast(pv2_i % 30 + 1), 0.02);
    }
    const end2 = std.Io.Timestamp.now(io, .real);
    const elapsed2: u64 = @intCast(end2.nanoseconds - start2.nanoseconds);
    try printBenchResult(out, "PresentValueStar (10M calls)", pv2_count, elapsed2);

    // Discount factor table benchmark (via RateConverter.discount)
    const dt_count: usize = 10_000_000;
    const start3 = std.Io.Timestamp.now(io, .real);
    var dt_i: usize = 0;
    while (dt_i < dt_count) : (dt_i += 1) {
        _ = rc.discount(@intCast(dt_i % 100));
    }
    const end3 = std.Io.Timestamp.now(io, .real);
    const elapsed3: u64 = @intCast(end3.nanoseconds - start3.nanoseconds);
    try printBenchResult(out, "DiscountTable lookups (10M)", dt_count, elapsed3);

    try out.print("\n", .{});
}

fn benchMonteCarlo(out: anytype, io: std.Io) !void {
    try out.print("Monte Carlo Benchmarks:\n", .{});
    const allocator = std.heap.page_allocator;

    // GBM path generation
    const gen = stochastic.RateGenerator.init(allocator, 0.05, 0.02, 0.15);
    const gbm_path = try gen.generatePath(100, 1.0);
    defer allocator.free(gbm_path);

    const mc_count: usize = 10_000;
    const start_gbm = std.Io.Timestamp.now(io, .real);
    var mc_i: usize = 0;
    while (mc_i < mc_count) : (mc_i += 1) {
        const p = try gen.generatePath(100, 1.0);
        allocator.free(p);
    }
    const end_gbm = std.Io.Timestamp.now(io, .real);
    const elapsed: u64 = @intCast(end_gbm.nanoseconds - start_gbm.nanoseconds);
    try printBenchResult(out, "GBM single path (10K x 100 steps)", mc_count, elapsed);

    // Vasicek path generation
    const vgen = stochastic.VasicekGenerator.init(allocator, 0.05, 0.03, 0.1, 0.02);
    const v_path = try vgen.generatePath(100, 1.0);
    defer allocator.free(v_path);

    const vc_count: usize = 10_000;
    const start_v = std.Io.Timestamp.now(io, .real);
    var vc_i: usize = 0;
    while (vc_i < vc_count) : (vc_i += 1) {
        const p = try vgen.generatePath(100, 1.0);
        allocator.free(p);
    }
    const end_v = std.Io.Timestamp.now(io, .real);
    const elapsed2: u64 = @intCast(end_v.nanoseconds - start_v.nanoseconds);
    try printBenchResult(out, "Vasicek single path (10K x 100 steps)", vc_count, elapsed2);

    try out.print("\n", .{});
}

fn benchRiskMeasures(out: anytype, io: std.Io) !void {
    try out.print("Risk Measure Benchmarks:\n", .{});
    const allocator = std.heap.page_allocator;

    // Generate data
    const data = try allocator.alloc(f64, 1_000_000);
    defer allocator.free(data);

    var rm_prng = std.Random.DefaultPrng.init(42);
    const random = rm_prng.random();
    for (data) |*d| {
        d.* = random.float(f64);
    }

    // VaR benchmark
    const var_count: usize = 100;
    const start_var = std.Io.Timestamp.now(io, .real);
    var var_i: usize = 0;
    while (var_i < var_count) : (var_i += 1) {
        const var_copy = try allocator.dupe(f64, data);
        defer allocator.free(var_copy);
        _ = risk.vaR(var_copy, 0.95);
    }
    const end_var = std.Io.Timestamp.now(io, .real);
    const elapsed: u64 = @intCast(end_var.nanoseconds - start_var.nanoseconds);
    try printBenchResult(out, "VaR 95% (100 x 1M elements)", var_count, elapsed);

    // CTE benchmark
    const cte_count: usize = 100;
    const start_cte = std.Io.Timestamp.now(io, .real);
    var cte_i: usize = 0;
    while (cte_i < cte_count) : (cte_i += 1) {
        const cte_copy = try allocator.dupe(f64, data);
        defer allocator.free(cte_copy);
        _ = risk.cte(cte_copy, 0.95);
    }
    const end_cte = std.Io.Timestamp.now(io, .real);
    const elapsed2: u64 = @intCast(end_cte.nanoseconds - start_cte.nanoseconds);
    try printBenchResult(out, "CTE 95% (100 x 1M elements)", cte_count, elapsed2);

    // Full risk report
    const report_count: usize = 10;
    const start_report = std.Io.Timestamp.now(io, .real);
    var report_i: usize = 0;
    while (report_i < report_count) : (report_i += 1) {
        const rep_copy = try allocator.dupe(f64, data);
        defer allocator.free(rep_copy);
        _ = risk.computeReport(rep_copy);
    }
    const end_report = std.Io.Timestamp.now(io, .real);
    const elapsed3: u64 = @intCast(end_report.nanoseconds - start_report.nanoseconds);
    try printBenchResult(out, "Full RiskReport (10 x 1M elements)", report_count, elapsed3);

    try out.print("\n", .{});
}

fn benchCSVParsing(out: anytype, io: std.Io) !void {
    try out.print("CSV Parsing Benchmarks:\n", .{});

    // In-memory parse benchmark
    const csv_line = "30,M,term,100000.50,20\r";
    const parse_count: usize = 10_000_000;
    const start_census = std.Io.Timestamp.now(io, .real);
    var census_i: usize = 0;
    while (census_i < parse_count) : (census_i += 1) {
        _ = try reader.parseCensusFastBytes(csv_line, ',');
    }
    const end_census = std.Io.Timestamp.now(io, .real);
    const elapsed: u64 = @intCast(end_census.nanoseconds - start_census.nanoseconds);
    try printBenchResult(out, "parseCensusFastBytes (10M calls)", parse_count, elapsed);

    // parseFastInt benchmark
    const int_str = "123456789";
    const int_count: usize = 20_000_000;
    const start_int = std.Io.Timestamp.now(io, .real);
    var int_i: usize = 0;
    while (int_i < int_count) : (int_i += 1) {
        _ = try reader.parseFastInt(int_str);
    }
    const end_int = std.Io.Timestamp.now(io, .real);
    const elapsed2: u64 = @intCast(end_int.nanoseconds - start_int.nanoseconds);
    try printBenchResult(out, "parseFastInt (20M calls)", int_count, elapsed2);

    // parseFastFloat benchmark
    const float_str = "12345.6789";
    const float_count: usize = 10_000_000;
    const start_float = std.Io.Timestamp.now(io, .real);
    var float_i: usize = 0;
    while (float_i < float_count) : (float_i += 1) {
        _ = try reader.parseFastFloat(float_str);
    }
    const end_float = std.Io.Timestamp.now(io, .real);
    const elapsed3: u64 = @intCast(end_float.nanoseconds - start_float.nanoseconds);
    try printBenchResult(out, "parseFastFloat (10M calls)", float_count, elapsed3);

    // parseFields benchmark
    var pf_line = [_]u8{ 'a', ',', 'b', ',', 'c', ',', 'd', ',', 'e' };
    const pf_count: usize = 10_000_000;
    const start_pf = std.Io.Timestamp.now(io, .real);
    var pf_i: usize = 0;
    while (pf_i < pf_count) : (pf_i += 1) {
        _ = reader.parseFields(&pf_line, ',');
    }
    const end_pf = std.Io.Timestamp.now(io, .real);
    const elapsed4: u64 = @intCast(end_pf.nanoseconds - start_pf.nanoseconds);
    try printBenchResult(out, "parseFields (10M calls)", pf_count, elapsed4);

    try out.print("\n", .{});
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test "bench module tests" {
    const testing = std.testing;
    try testing.expect(true);
}
