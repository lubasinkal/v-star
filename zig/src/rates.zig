// rates.zig - Interest rate calculations, discount factors, present value
//
// Port of Go pkg/rates. Provides pre-computed discount table for O(1) lookups
// and actuarial rate conversion functions.
const std = @import("std");
const math = std.math;

/// RateConverter performs interest rate conversions and present value calculations.
/// Pre-computes a dynamically-growing discount table for fast lookups.
pub const RateConverter = struct {
    discount_table: []f64,
    effective_rate: f64,
    allocator: std.mem.Allocator,

    /// Creates a RateConverter for the given effective annual rate.
    /// Pre-computes discount factors v^t for t in [0, 100] where v = 1/(1+i).
    pub fn init(allocator: std.mem.Allocator, effective_rate: f64) !RateConverter {
        const v_factor = 1.0 / (1.0 + effective_rate);
        const table = try allocator.alloc(f64, 101);
        table[0] = 1.0;
        for (1..101) |i| {
            table[i] = table[i - 1] * v_factor;
        }
        return .{
            .discount_table = table,
            .effective_rate = effective_rate,
            .allocator = allocator,
        };
    }

    pub fn deinit(self: *RateConverter) void {
        self.allocator.free(self.discount_table);
    }

    /// Returns the discount factor v^term.
    /// Uses pre-computed table that grows as needed.
    pub fn discount(self: *RateConverter, term: i64) f64 {
        if (term <= 0) return 1.0;
        const t: usize = @intCast(term);
        if (t >= self.discount_table.len) {
            self.growTable(t);
        }
        return self.discount_table[t];
    }

    fn growTable(self: *RateConverter, min_size: usize) void {
        const new_size: usize = @max(min_size + 1, self.discount_table.len * 2 + 1);
        const new_table = self.allocator.realloc(self.discount_table, new_size) catch {
            // If realloc fails, just recompute into a new allocation
            const buf = self.allocator.alloc(f64, new_size) catch return;
            self.allocator.free(self.discount_table);
            self.discount_table = buf;
            self.recomputeTable(new_size);
            return;
        };
        self.discount_table = new_table;
        self.recomputeTable(new_size);
    }

    fn recomputeTable(self: *RateConverter, size: usize) void {
        const v_factor = 1.0 / (1.0 + self.effective_rate);
        self.discount_table[0] = 1.0;
        for (1..size) |i| {
            self.discount_table[i] = self.discount_table[i - 1] * v_factor;
        }
    }

    /// Returns the one-period discount factor v = 1/(1+i).
    pub fn v(self: *const RateConverter) f64 {
        return 1.0 / (1.0 + self.effective_rate);
    }

    /// Returns the v-star factor v* = (1+j) * v,
    /// used when premiums compound at rate j while being discounted at rate i.
    pub fn vStar(self: *const RateConverter, j: f64) f64 {
        return (1.0 + j) * self.v();
    }

    /// Returns sumAssured * v^term.
    pub fn presentValue(self: *RateConverter, sum_assured: f64, term: i64) f64 {
        if (term <= 0) return sum_assured;
        const t: usize = @intCast(term);
        if (t >= self.discount_table.len) {
            self.growTable(t);
        }
        return sum_assured * self.discount_table[t];
    }

    /// Returns sumAssured * (v*)^term using the v-star discount factor.
    pub fn presentValueStar(self: *const RateConverter, sum_assured: f64, term: i64, j: f64) f64 {
        if (term <= 0) return sum_assured;
        return sum_assured * math.pow(f64, self.vStar(j), @floatFromInt(term));
    }
};

// ---------------------------------------------------------------------------
// Rate conversion functions
// ---------------------------------------------------------------------------

/// Converts a nominal rate compounded m times per period to effective annual rate.
/// i = (1 + im/m)^m - 1
pub fn nominalToEffective(im: f64, m: i64) f64 {
    const mf: f64 = @floatFromInt(m);
    return math.pow(f64, 1.0 + im / mf, mf) - 1.0;
}

/// Converts an effective annual rate to nominal rate compounded m times per period.
/// im = m * ((1+i)^(1/m) - 1)
pub fn effectiveToNominal(i: f64, m: i64) f64 {
    const mf: f64 = @floatFromInt(m);
    return mf * (math.pow(f64, 1.0 + i, 1.0 / mf) - 1.0);
}

/// Returns the force of interest delta = ln(1+i).
pub fn forceOfInterest(i: f64) f64 {
    return @log(1.0 + i);
}

/// Converts a force of interest to an effective annual rate: i = e^delta - 1.
pub fn interestFromForce(delta: f64) f64 {
    return @exp(delta) - 1.0;
}

/// Returns the present value of an annuity-certain-immediate: a_angle_n = (1 - v^n) / i.
pub fn annuityCertainImmediate(i: f64, n: i64) f64 {
    if (n <= 0 or i <= 0) return 0.0;
    const v = 1.0 / (1.0 + i);
    return (1.0 - math.pow(f64, v, @floatFromInt(n))) / i;
}

/// Returns the present value of an annuity-certain-due: adbl_angle_n = (1 - v^n) / d.
pub fn annuityCertainDue(i: f64, n: i64) f64 {
    if (n <= 0 or i <= 0) return 0.0;
    const v = 1.0 / (1.0 + i);
    const d = i / (1.0 + i);
    return (1.0 - math.pow(f64, v, @floatFromInt(n))) / d;
}

/// Macaulay duration of a cash flow stream.
/// Returns sum(t * PV_t) / sum(PV_t).
pub fn macaulayDuration(i: f64, cash_flows: []const f64) f64 {
    if (i <= 0 or cash_flows.len == 0) return 0.0;
    const v = 1.0 / (1.0 + i);
    var v_pow = v;
    var pv_total: f64 = 0.0;
    var duration: f64 = 0.0;
    for (cash_flows, 0..) |cf, t| {
        if (cf <= 0) {
            v_pow *= v;
            continue;
        }
        const pv = cf * v_pow;
        pv_total += pv;
        duration += @as(f64, @floatFromInt(t + 1)) * pv;
        v_pow *= v;
    }
    if (pv_total <= 0) return 0.0;
    return duration / pv_total;
}

/// Modified duration: MacaulayDuration / (1+i).
pub fn modifiedDuration(i: f64, cash_flows: []const f64) f64 {
    return macaulayDuration(i, cash_flows) / (1.0 + i);
}

/// Convexity of a cash flow stream.
pub fn convexity(i: f64, cash_flows: []const f64) f64 {
    if (i <= 0 or cash_flows.len == 0) return 0.0;
    const v = 1.0 / (1.0 + i);
    var v_pow = v;
    var pv_total: f64 = 0.0;
    var conv: f64 = 0.0;
    for (cash_flows, 0..) |cf, t| {
        if (cf <= 0) {
            v_pow *= v;
            continue;
        }
        const pv = cf * v_pow;
        pv_total += pv;
        conv += @as(f64, @floatFromInt(t + 1)) * @as(f64, @floatFromInt(t + 2)) * pv;
        v_pow *= v;
    }
    if (pv_total <= 0) return 0.0;
    return conv / (pv_total * (1.0 + i) * (1.0 + i));
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test "NominalToEffective" {
    const testing = std.testing;
    try testing.expectApproxEqAbs(@as(f64, 0.05), nominalToEffective(0.05, 1), 1e-10);
    // Monthly compounding of 12%
    const me = nominalToEffective(0.12, 12);
    try testing.expect(me > 0.12);
    try testing.expect(me < 0.13);
}

test "EffectiveToNominal" {
    const testing = std.testing;
    try testing.expectApproxEqAbs(@as(f64, 0.05), effectiveToNominal(0.05, 1), 1e-10);
}

test "ForceOfInterest" {
    const testing = std.testing;
    const delta = forceOfInterest(0.05);
    try testing.expectApproxEqAbs(@as(f64, 0.04879), delta, 1e-3);
    try testing.expectApproxEqAbs(@as(f64, 0.05), interestFromForce(delta), 1e-10);
}

test "AnnuityCertain" {
    const testing = std.testing;
    const a_imd = annuityCertainImmediate(0.05, 10);
    try testing.expect(a_imd > 0);
    try testing.expect(a_imd < 10.0);

    const a_dbl = annuityCertainDue(0.05, 10);
    try testing.expect(a_dbl > a_imd);
}

test "RateConverter basics" {
    const testing = std.testing;
    var rc = try RateConverter.init(testing.allocator, 0.05);
    defer rc.deinit();

    try testing.expectApproxEqAbs(@as(f64, 1.0 / 1.05), rc.v(), 1e-10);
    try testing.expectApproxEqAbs(@as(f64, 1.0), rc.discount(0), 1e-10);
    try testing.expectApproxEqAbs(@as(f64, 1.0 / 1.05), rc.discount(1), 1e-10);

    // Present value
    const pv = rc.presentValue(1000, 1);
    try testing.expectApproxEqAbs(@as(f64, 952.38), pv, 0.01);

    // v-star
    const vs = rc.vStar(0.02);
    try testing.expectApproxEqAbs(@as(f64, 1.02 / 1.05), vs, 1e-10);
}

test "PresentValue" {
    const testing = std.testing;
    var rc = try RateConverter.init(testing.allocator, 0.05);
    defer rc.deinit();

    try testing.expectApproxEqAbs(@as(f64, 1000.0), rc.presentValue(1000, 0), 0.01);
    try testing.expectApproxEqAbs(@as(f64, 952.38), rc.presentValue(1000, 1), 0.01);
    try testing.expectApproxEqAbs(@as(f64, 783.53), rc.presentValue(1000, 5), 0.01);
}

test "PresentValueStar" {
    const testing = std.testing;
    var rc = try RateConverter.init(testing.allocator, 0.05);
    defer rc.deinit();
    const j: f64 = 0.02;

    try testing.expectApproxEqAbs(@as(f64, 1000.0), rc.presentValueStar(1000, 0, j), 0.01);
    try testing.expectApproxEqAbs(@as(f64, 971.43), rc.presentValueStar(1000, 1, j), 0.01);
    try testing.expectApproxEqAbs(@as(f64, 865.08), rc.presentValueStar(1000, 5, j), 0.01);
}

test "Macaulay duration" {
    const testing = std.testing;
    const cash_flows = [_]f64{ 100, 100, 100, 100, 1100 };
    const md = macaulayDuration(0.05, &cash_flows);
    try testing.expect(md > 4.0);
    try testing.expect(md < 5.0);
}
