// annuities.zig - Annuity calculations (whole life, term, deferred)
//
// Port of Go pkg/annuities. Computes present values of life-contingent cash flows.
const std = @import("std");
const math = std.math;
const Allocator = std.mem.Allocator;
const rates = @import("rates.zig");
const mortality = @import("mortality.zig");

/// AnnuityCalculator computes annuity values using a discount factor and mortality table.
pub const AnnuityCalculator = struct {
    rate_conv: *rates.RateConverter,
    mort: *const mortality.MortalityTable,

    pub fn init(rate_conv: *rates.RateConverter, mort: *const mortality.MortalityTable) AnnuityCalculator {
        return .{
            .rate_conv = rate_conv,
            .mort = mort,
        };
    }

    /// Present value of a whole life annuity-immediate.
    /// Payments of amount at the end of each year while the annuitant is alive.
    pub fn wholeLifeImmediate(self: *const AnnuityCalculator, age: i64, amount: f64) f64 {
        if (age < 0 or amount <= 0) return 0.0;
        const max_age = self.mort.maxAge();
        var sum: f64 = 0.0;
        var px: f64 = 1.0;
        var t: i64 = 1;
        while (t <= max_age - age) : (t += 1) {
            px *= 1.0 - self.mort.qxAt(age + t - 1);
            if (px <= 0) break;
            sum += px * self.rate_conv.discount(t);
        }
        return amount * sum;
    }

    /// Present value of a term annuity-immediate.
    /// Payments of amount at the end of each year for the specified term.
    pub fn termImmediate(self: *const AnnuityCalculator, age: i64, term: i64, amount: f64) f64 {
        if (age < 0 or term <= 0 or amount <= 0) return 0.0;
        const max_age = self.mort.maxAge();
        const limit = @min(max_age - age, term);
        var sum: f64 = 0.0;
        var px: f64 = 1.0;
        var t: i64 = 1;
        while (t <= limit) : (t += 1) {
            px *= 1.0 - self.mort.qxAt(age + t - 1);
            if (px <= 0) break;
            sum += px * self.rate_conv.discount(t);
        }
        return amount * sum;
    }

    /// Present value of a whole life annuity-due.
    /// Payments of amount at the start of each year while the annuitant is alive.
    pub fn wholeLifeDue(self: *const AnnuityCalculator, age: i64, amount: f64) f64 {
        if (age < 0 or amount <= 0) return 0.0;
        const max_age = self.mort.maxAge();
        var sum: f64 = self.rate_conv.discount(0); // First payment at t=0
        var px: f64 = 1.0;
        var t: i64 = 1;
        while (t <= max_age - age) : (t += 1) {
            px *= 1.0 - self.mort.qxAt(age + t - 1);
            if (px <= 0) break;
            sum += px * self.rate_conv.discount(t);
        }
        return amount * sum;
    }

    /// Present value of a term annuity-due.
    /// Payments of amount at the start of each year for the specified term.
    pub fn termDue(self: *const AnnuityCalculator, age: i64, term: i64, amount: f64) f64 {
        if (age < 0 or term <= 0 or amount <= 0) return 0.0;
        const max_age = self.mort.maxAge();
        const limit = @min(max_age - age, term);
        var sum: f64 = self.rate_conv.discount(0); // First payment at t=0
        var px: f64 = 1.0;
        var t: i64 = 1;
        while (t < limit) : (t += 1) { // Note: < not <= because first payment is at t=0
            px *= 1.0 - self.mort.qxAt(age + t - 1);
            if (px <= 0) break;
            sum += px * self.rate_conv.discount(t);
        }
        return amount * sum;
    }

    /// Present value of a deferred whole life annuity.
    /// Payments begin after deferment years, contingent on survival.
    pub fn deferredWholeLife(self: *const AnnuityCalculator, age: i64, deferment: i64, amount: f64) f64 {
        if (age < 0 or deferment <= 0 or amount <= 0) return 0.0;
        const prob_alive = self.mort.px(age, deferment);
        if (prob_alive <= 0) return 0.0;
        const discount_at_deferment = self.rate_conv.discount(deferment);
        const pv_deferred = prob_alive * discount_at_deferment;
        const annuity_pv = self.wholeLifeImmediate(age + deferment, amount);
        return pv_deferred * annuity_pv;
    }

    /// Present value of a deferred term annuity.
    pub fn deferredTerm(self: *const AnnuityCalculator, age: i64, deferment: i64, term: i64, amount: f64) f64 {
        if (age < 0 or deferment <= 0 or term <= 0 or amount <= 0) return 0.0;
        const prob_alive = self.mort.px(age, deferment);
        if (prob_alive <= 0) return 0.0;
        const discount_at_deferment = self.rate_conv.discount(deferment);
        const pv_deferred = prob_alive * discount_at_deferment;
        const annuity_pv = self.termImmediate(age + deferment, term, amount);
        return pv_deferred * annuity_pv;
    }

    /// Net single premium for whole life insurance: A_x.
    pub fn wholeLifeNSP(self: *const AnnuityCalculator, age: i64, sum_assured: f64) f64 {
        if (age < 0 or sum_assured <= 0) return 0.0;
        const max_age = self.mort.maxAge();
        var nsp: f64 = 0.0;
        var t: i64 = 1;
        while (t <= max_age - age + 1) : (t += 1) {
            const qx = self.mort.qxAt(age + t - 1);
            if (qx <= 0) continue;
            nsp += qx * self.rate_conv.discount(t);
        }
        return sum_assured * nsp;
    }

    /// Net single premium for term life insurance: A^1_{x:n}.
    pub fn termNSP(self: *const AnnuityCalculator, age: i64, term: i64, sum_assured: f64) f64 {
        if (age < 0 or term <= 0 or sum_assured <= 0) return 0.0;
        const max_age = self.mort.maxAge();
        const limit = @min(max_age - age + 1, term);
        var nsp: f64 = 0.0;
        var px: f64 = 1.0;
        var t: i64 = 1;
        while (t <= limit) : (t += 1) {
            const qx = self.mort.qxAt(age + t - 1);
            if (qx > 0) {
                nsp += px * qx * self.rate_conv.discount(t);
            }
            px *= 1.0 - qx;
            if (px <= 0) break;
        }
        return sum_assured * nsp;
    }

    /// Net single premium for endowment: A_{x:n} = A^1_{x:n} + v^n * Px(age, n).
    pub fn endowmentNSP(self: *const AnnuityCalculator, age: i64, term: i64, sum_assured: f64) f64 {
        if (age < 0 or term <= 0 or sum_assured <= 0) return 0.0;
        const term_insurance = self.termNSP(age, term, sum_assured);
        const survival = self.mort.px(age, term);
        const pure_endowment = sum_assured * self.rate_conv.discount(term) * survival;
        return term_insurance + pure_endowment;
    }
};

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test "AnnuityCalculator whole life immediate" {
    const testing = std.testing;
    const alloc = testing.allocator;

    var rc = try rates.RateConverter.init(alloc, 0.05);
    defer rc.deinit();

    const qx = [_]f64{ 0.001, 0.002, 0.003, 0.004, 0.005 };
    var table = try mortality.MortalityTable.init(alloc, "test", &qx);
    defer table.deinit();

    const calc = AnnuityCalculator.init(&rc, &table);
    const pv = calc.wholeLifeImmediate(0, 1000);
    try testing.expect(pv > 0);
    try testing.expect(pv < 5000); // Should be less than sum of all payments
}

test "AnnuityCalculator term immediate" {
    const testing = std.testing;
    const alloc = testing.allocator;

    var rc = try rates.RateConverter.init(alloc, 0.05);
    defer rc.deinit();

    const qx = [_]f64{ 0.001, 0.002, 0.003, 0.004, 0.005 };
    var table = try mortality.MortalityTable.init(alloc, "test", &qx);
    defer table.deinit();

    const calc = AnnuityCalculator.init(&rc, &table);
    const pv = calc.termImmediate(0, 3, 1000);
    try testing.expect(pv > 0);

    // Term annuity should be less than whole life
    const wlpv = calc.wholeLifeImmediate(0, 1000);
    try testing.expect(pv < wlpv);
}

test "AnnuityCalculator whole life due" {
    const testing = std.testing;
    const alloc = testing.allocator;

    var rc = try rates.RateConverter.init(alloc, 0.05);
    defer rc.deinit();

    const qx = [_]f64{ 0.001, 0.002, 0.003, 0.004, 0.005 };
    var table = try mortality.MortalityTable.init(alloc, "test", &qx);
    defer table.deinit();

    const calc = AnnuityCalculator.init(&rc, &table);

    // Due should be greater than immediate (first payment at t=0)
    const due = calc.wholeLifeDue(0, 1000);
    const imm = calc.wholeLifeImmediate(0, 1000);
    try testing.expect(due > imm);
}

test "AnnuityCalculator whole life NSP" {
    const testing = std.testing;
    const alloc = testing.allocator;

    var rc = try rates.RateConverter.init(alloc, 0.05);
    defer rc.deinit();

    const qx = [_]f64{ 0.001, 0.002, 0.003, 0.004, 0.005 };
    var table = try mortality.MortalityTable.init(alloc, "test", &qx);
    defer table.deinit();

    const calc = AnnuityCalculator.init(&rc, &table);
    const nsp = calc.wholeLifeNSP(0, 100000);
    try testing.expect(nsp > 0);
    try testing.expect(nsp < 100000);
}

test "AnnuityCalculator deferred" {
    const testing = std.testing;
    const alloc = testing.allocator;

    var rc = try rates.RateConverter.init(alloc, 0.05);
    defer rc.deinit();

    const qx = [_]f64{ 0.001, 0.002, 0.003, 0.004, 0.005 };
    var table = try mortality.MortalityTable.init(alloc, "test", &qx);
    defer table.deinit();

    const calc = AnnuityCalculator.init(&rc, &table);
    const deferred = calc.deferredWholeLife(0, 2, 1000);
    try testing.expect(deferred > 0);

    // Deferred should be less than immediate
    const immediate = calc.wholeLifeImmediate(0, 1000);
    try testing.expect(deferred < immediate);
}
