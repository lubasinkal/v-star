// reserves.zig - Policy reserves (net premium, gross premium, prospective)
//
// Port of Go pkg/reserves. Calculates reserves for life insurance policies.
const std = @import("std");
const Allocator = std.mem.Allocator;
const rates = @import("rates.zig");
const mortality = @import("mortality.zig");
const annuities = @import("annuities.zig");

/// PolicySpec defines the parameters for a life insurance policy.
pub const PolicySpec = struct {
    age: i64,
    term: i64,
    sum_assured: f64,
    premium: f64,
};

/// Calculates the net premium reserve using the prospective method.
/// The net annual premium is determined internally so the policy is fair at inception.
pub fn netPremiumReserve(policy: PolicySpec, discount: *rates.RateConverter, mort: *const mortality.MortalityTable) f64 {
    const age = policy.age;
    const term = policy.term;
    const sa = policy.sum_assured;

    if (age < 0 or term <= 0 or sa <= 0) return 0.0;

    const v = discount.discount(1);

    // Build unit annuity-due values recursively
    const unit_annuities = discount.allocator.alloc(f64, @intCast(term + 1)) catch return 0.0;
    defer discount.allocator.free(unit_annuities);
    @memset(unit_annuities, 0.0);

    var k: i64 = term - 1;
    while (k >= 0) : (k -= 1) {
        const k_usize: usize = @intCast(k);
        const p = mort.px(age + k, 1);
        unit_annuities[k_usize] = p * v * (1.0 + unit_annuities[k_usize + 1]);
    }

    if (unit_annuities[0] <= 0) return 0.0;

    const annual_premium = sa / unit_annuities[0];

    var reserve: f64 = 0.0;
    var year: i64 = 1;
    while (year <= term) : (year += 1) {
        const p = mort.px(age + year - 1, 1);
        if (p <= 0) break;
        const net_liability = (sa - annual_premium) * unit_annuities[@intCast(year)];
        reserve = (reserve + net_liability) * v / p - annual_premium;
    }

    return reserve;
}

/// Calculates the gross premium reserve (NPR + expense reserve).
pub fn grossPremiumReserve(policy: PolicySpec, expenses: f64, discount: *rates.RateConverter, mort: *const mortality.MortalityTable) f64 {
    const npr = netPremiumReserve(policy, discount, mort);
    const calc = annuities.AnnuityCalculator.init(discount, mort);
    const expense_annuity = calc.termImmediate(policy.age, policy.term, expenses);
    const expense_reserve = expense_annuity - expenses;
    return npr + expense_reserve;
}

/// Calculates the reserve as future benefits minus future premiums.
pub fn prospectiveReserve(policy: PolicySpec, discount: *rates.RateConverter, mort: *const mortality.MortalityTable) f64 {
    const age = policy.age;
    const term = policy.term;
    const sa = policy.sum_assured;
    const prem = policy.premium;

    if (age < 0 or term <= 0 or sa <= 0 or prem <= 0) return 0.0;

    const calc = annuities.AnnuityCalculator.init(discount, mort);
    const future_benefits = calc.termNSP(age, term, sa);
    const future_premiums = calc.termDue(age, term, prem);

    return future_benefits - future_premiums;
}

/// Calculates the reserve using the retrospective (Fackler) method.
/// V_t = [(V_{t-1} + P) * (1+i) - SA * qx] / px
pub fn retrospectiveReserve(policy: PolicySpec, discount: *rates.RateConverter, mort: *const mortality.MortalityTable) f64 {
    const age = policy.age;
    const term = policy.term;
    const sa = policy.sum_assured;
    const prem = policy.premium;

    if (age < 0 or term <= 0 or sa <= 0) return 0.0;

    const v = discount.discount(1);
    var reserve: f64 = 0.0;
    var year: i64 = 0;
    while (year < term) : (year += 1) {
        const qx = mort.qxAt(age + year);
        const px = 1.0 - qx;
        if (px <= 0) break;
        reserve = (reserve + prem) * v - sa * qx;
        reserve /= px;
    }

    return reserve;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test "NetPremiumReserve" {
    const testing = std.testing;
    const alloc = testing.allocator;

    var rc = try rates.RateConverter.init(alloc, 0.05);
    defer rc.deinit();

    const qx = [_]f64{ 0.001, 0.002, 0.003, 0.004, 0.005 };
    var table = try mortality.MortalityTable.init(alloc, "test", &qx);
    defer table.deinit();

    const policy = PolicySpec{
        .age = 0,
        .term = 5,
        .sum_assured = 100000,
        .premium = 0,
    };

    const reserve = netPremiumReserve(policy, &rc, &table);
    try testing.expect(reserve >= 0);
    try testing.expect(reserve < 1000000);
}

test "ProspectiveReserve" {
    const testing = std.testing;
    const alloc = testing.allocator;

    var rc = try rates.RateConverter.init(alloc, 0.05);
    defer rc.deinit();

    const qx = [_]f64{ 0.001, 0.002, 0.003, 0.004, 0.005 };
    var table = try mortality.MortalityTable.init(alloc, "test", &qx);
    defer table.deinit();

    const policy = PolicySpec{
        .age = 0,
        .term = 5,
        .sum_assured = 100000,
        .premium = 1500,
    };

    const reserve = prospectiveReserve(policy, &rc, &table);
    // Reserve should be reasonable (could be positive or negative depending on premium)
    try testing.expect(reserve > -100000);
    try testing.expect(reserve < 100000);
}
