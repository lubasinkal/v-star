// risk.zig - Risk measures (VaR, CTE, Expected Shortfall)
//
// Port of Go pkg/risk. Computes risk metrics from simulated portfolio losses.
const std = @import("std");
const math = std.math;
const Allocator = std.mem.Allocator;

/// RiskReport contains comprehensive risk metrics from a simulation.
pub const RiskReport = struct {
    mean: f64 = 0,
    std_dev: f64 = 0,
    min: f64 = 0,
    max: f64 = 0,
    var_95: f64 = 0,
    var_99: f64 = 0,
    cte_95: f64 = 0,
    cte_99: f64 = 0,
    std_error: f64 = 0,
    confidence_95_lo: f64 = 0,
    confidence_95_hi: f64 = 0,
};

/// VaR computes Value at Risk at the given confidence level.
/// Returns the loss threshold that is not exceeded with the specified probability.
/// The losses slice is sorted in place during computation.
pub fn vaR(losses: []f64, confidence: f64) f64 {
    if (losses.len == 0 or confidence <= 0 or confidence >= 1) return 0.0;
    std.sort.pdq(f64, losses, {}, comptime std.sort.asc(f64));
    return varSorted(losses, confidence);
}

/// CTE computes Conditional Tail Expectation (Expected Shortfall).
/// Returns the average of losses exceeding the VaR threshold.
/// The losses slice is sorted in place during computation.
pub fn cte(losses: []f64, confidence: f64) f64 {
    if (losses.len == 0 or confidence <= 0 or confidence >= 1) return 0.0;
    std.sort.pdq(f64, losses, {}, comptime std.sort.asc(f64));
    return cteSorted(losses, confidence);
}

fn varSorted(sorted: []f64, confidence: f64) f64 {
    var idx: usize = @intFromFloat(confidence * @as(f64, @floatFromInt(sorted.len - 1)));
    if (idx >= sorted.len) idx = sorted.len - 1;
    return sorted[idx];
}

fn cteSorted(sorted: []f64, confidence: f64) f64 {
    var idx: usize = @intFromFloat(confidence * @as(f64, @floatFromInt(sorted.len - 1)));
    if (idx >= sorted.len) idx = sorted.len - 1;
    var sum: f64 = 0;
    for (idx..sorted.len) |i| {
        sum += sorted[i];
    }
    return sum / @as(f64, @floatFromInt(sorted.len - idx));
}

/// Generates a full risk report from simulated losses.
/// The losses slice is sorted in place during computation.
pub fn computeReport(losses: []f64) RiskReport {
    const n: f64 = @floatFromInt(losses.len);
    if (n == 0) return .{};

    var mean: f64 = 0;
    var min_val: f64 = losses[0];
    var max_val: f64 = losses[0];
    for (losses) |l| {
        mean += l;
        if (l < min_val) min_val = l;
        if (l > max_val) max_val = l;
    }
    mean /= n;

    var variance: f64 = 0;
    for (losses) |l| {
        const d = l - mean;
        variance += d * d;
    }
    variance /= n;

    std.sort.pdq(f64, losses, {}, comptime std.sort.asc(f64));

    const std_dev = @sqrt(variance);
    const std_err = std_dev / @sqrt(n);

    return .{
        .mean = mean,
        .std_dev = std_dev,
        .min = min_val,
        .max = max_val,
        .var_95 = varSorted(losses, 0.95),
        .var_99 = varSorted(losses, 0.99),
        .cte_95 = cteSorted(losses, 0.95),
        .cte_99 = cteSorted(losses, 0.99),
        .std_error = std_err,
        .confidence_95_lo = mean - 1.96 * std_err,
        .confidence_95_hi = mean + 1.96 * std_err,
    };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test "VaR basic" {
    const testing = std.testing;
    var losses = [_]f64{ 10, 20, 30, 40, 50, 60, 70, 80, 90, 100 };
    const result = vaR(&losses, 0.95);
    try testing.expect(result >= 90);
    try testing.expect(result <= 100);
}

test "CTE basic" {
    const testing = std.testing;
    var losses = [_]f64{ 10, 20, 30, 40, 50, 60, 70, 80, 90, 100 };
    const result = cte(&losses, 0.95);
    try testing.expect(result >= 90);
    try testing.expect(result <= 100);
}

test "CTE >= VaR" {
    const testing = std.testing;
    var losses1 = [_]f64{ 10, 20, 30, 40, 50, 60, 70, 80, 90, 100 };
    var losses2 = [_]f64{ 10, 20, 30, 40, 50, 60, 70, 80, 90, 100 };
    const var_result = vaR(&losses1, 0.95);
    const cte_result = cte(&losses2, 0.95);
    try testing.expect(cte_result >= var_result);
}

test "ComputeReport" {
    const testing = std.testing;
    var losses = [_]f64{ 10, 20, 30, 40, 50, 60, 70, 80, 90, 100 };
    const report = computeReport(&losses);

    try testing.expectApproxEqAbs(@as(f64, 55), report.mean, 1e-10);
    try testing.expectApproxEqAbs(@as(f64, 10), report.min, 1e-10);
    try testing.expectApproxEqAbs(@as(f64, 100), report.max, 1e-10);
    try testing.expect(report.std_dev > 0);
    try testing.expect(report.var_95 > 0);
    try testing.expect(report.cte_95 >= report.var_95);
}

test "ComputeReport empty" {
    const testing = std.testing;
    var losses: [0]f64 = undefined;
    const report = computeReport(&losses);
    try testing.expectApproxEqAbs(@as(f64, 0), report.mean, 1e-10);
}
