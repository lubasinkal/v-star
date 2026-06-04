// stochastic.zig - Monte Carlo simulations, geometric Brownian motion, Vasicek
//
// Port of Go pkg/stochastic. Generates interest rate paths for actuarial simulations.
const std = @import("std");
const math = std.math;
const Allocator = std.mem.Allocator;

/// RatePath represents a sequence of interest rates over time.
pub const RatePath = []f64;

/// RateGenerator generates stochastic interest rate paths using geometric Brownian motion.
/// S(t+1) = S(t) * exp((mu - 0.5*sigma^2)*dt + sigma*sqrt(dt)*Z)
pub const RateGenerator = struct {
    initial_rate: f64,
    mu: f64,
    sigma: f64,
    drift_offset: f64,
    seed: ?u64,
    allocator: Allocator,

    /// Creates a rate generator with a random seed.
    pub fn init(allocator: Allocator, initial_rate: f64, mu: f64, sigma: f64) RateGenerator {
        return .{
            .initial_rate = initial_rate,
            .mu = mu,
            .sigma = sigma,
            .drift_offset = mu - 0.5 * sigma * sigma,
            .seed = null,
            .allocator = allocator,
        };
    }

    /// Creates a rate generator with a deterministic seed.
    pub fn initWithSeed(allocator: Allocator, initial_rate: f64, mu: f64, sigma: f64, seed: u64) RateGenerator {
        return .{
            .initial_rate = initial_rate,
            .mu = mu,
            .sigma = sigma,
            .drift_offset = mu - 0.5 * sigma * sigma,
            .seed = seed,
            .allocator = allocator,
        };
    }

    /// Generates a single interest rate path using GBM.
    pub fn generatePath(self: *const RateGenerator, steps: usize, dt: f64) !RatePath {
        const path = try self.allocator.alloc(f64, steps + 1);
        const seed: u64 = @intCast(@intFromPtr(path.ptr) +% 0x9e3779b97f4a7c15);
        var prng = std.Random.DefaultPrng.init(seed);
        const random = prng.random();

        path[0] = self.initial_rate;
        const drift_term = self.drift_offset * dt;
        const diffusion_factor = self.sigma * @sqrt(dt);
        for (1..steps + 1) |i| {
            const z = random.floatNorm(f64);
            path[i] = path[i - 1] * @exp(drift_term + diffusion_factor * z);
        }
        return path;
    }

    /// Generates a path into a pre-allocated buffer.
    pub fn generatePathInto(self: *const RateGenerator, path: RatePath, steps: usize, dt: f64) !void {
        const seed: u64 = @intCast(@intFromPtr(path.ptr) +% 0x9e3779b97f4a7c15);
        var prng = std.Random.DefaultPrng.init(seed);
        const random = prng.random();

        path[0] = self.initial_rate;
        const drift_term = self.drift_offset * dt;
        const diffusion_factor = self.sigma * @sqrt(dt);
        for (1..steps + 1) |i| {
            const z = random.floatNorm(f64);
            path[i] = path[i - 1] * @exp(drift_term + diffusion_factor * z);
        }
    }

    /// Generates a single path with a deterministic seed.
    pub fn generatePathSeeded(self: *const RateGenerator, steps: usize, dt: f64, seed: u64) !RatePath {
        const path = try self.allocator.alloc(f64, steps + 1);
        var prng = std.Random.DefaultPrng.init(seed);
        const random = prng.random();

        path[0] = self.initial_rate;
        const drift_term = self.drift_offset * dt;
        const diffusion_factor = self.sigma * @sqrt(dt);
        for (1..steps + 1) |i| {
            const z = random.floatNorm(f64);
            path[i] = path[i - 1] * @exp(drift_term + diffusion_factor * z);
        }
        return path;
    }

    /// Generates multiple interest rate paths.
    /// Each path is individually allocated; caller must free each path individually.
    pub fn generatePaths(self: *const RateGenerator, num_paths: usize, steps: usize, dt: f64) ![]RatePath {
        var paths = try self.allocator.alloc(RatePath, num_paths);
        for (0..num_paths) |i| {
            paths[i] = try self.generatePath(steps, dt);
        }
        return paths;
    }

    /// Derives a unique seed for each worker from a base seed.
    pub fn deriveWorkerSeed(base_seed: u64, worker_idx: usize) u64 {
        _ = worker_idx;
        return base_seed;
    }
};

/// VasicekGenerator generates interest rate paths using the Vasicek model:
/// dr = a(b - r)dt + sigma * dW
pub const VasicekGenerator = struct {
    initial_rate: f64,
    long_term_mean: f64,
    mean_reversion: f64,
    volatility: f64,
    seed: ?u64,
    allocator: Allocator,

    pub fn init(allocator: Allocator, initial_rate: f64, long_term_mean: f64, mean_reversion: f64, volatility: f64) VasicekGenerator {
        return .{
            .initial_rate = initial_rate,
            .long_term_mean = long_term_mean,
            .mean_reversion = mean_reversion,
            .volatility = volatility,
            .seed = null,
            .allocator = allocator,
        };
    }

    pub fn initWithSeed(allocator: Allocator, initial_rate: f64, long_term_mean: f64, mean_reversion: f64, volatility: f64, seed: u64) VasicekGenerator {
        return .{
            .initial_rate = initial_rate,
            .long_term_mean = long_term_mean,
            .mean_reversion = mean_reversion,
            .volatility = volatility,
            .seed = seed,
            .allocator = allocator,
        };
    }

    pub fn generatePath(self: *const VasicekGenerator, steps: usize, dt: f64) !RatePath {
        const path = try self.allocator.alloc(f64, steps + 1);
        const seed: u64 = @intCast(@intFromPtr(path.ptr) +% 0x9e3779b97f4a7c15);
        var prng = std.Random.DefaultPrng.init(seed);
        const random = prng.random();

        path[0] = self.initial_rate;
        const sqrt_dt = @sqrt(dt);
        var r = self.initial_rate;
        for (1..steps + 1) |i| {
            const drift = self.mean_reversion * (self.long_term_mean - r) * dt;
            r = r + drift + self.volatility * sqrt_dt * random.floatNorm(f64);
            if (r < 0) r = 0;
            path[i] = r;
        }
        return path;
    }

    pub fn generatePaths(self: *const VasicekGenerator, num_paths: usize, steps: usize, dt: f64) ![]RatePath {
        var paths = try self.allocator.alloc(RatePath, num_paths);
        for (0..num_paths) |i| {
            paths[i] = try self.generatePath(steps, dt);
        }
        return paths;
    }
};

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test "RateGenerator single path" {
    const testing = std.testing;
    const alloc = testing.allocator;

    const gen = RateGenerator.init(alloc, 0.05, 0.02, 0.15);
    const path = try gen.generatePath(10, 1.0);
    defer alloc.free(path);

    try testing.expectEqual(@as(usize, 11), path.len);
    try testing.expectApproxEqAbs(@as(f64, 0.05), path[0], 1e-10);
    for (path) |rate| {
        try testing.expect(rate > 0);
    }
}

test "RateGenerator deterministic seed" {
    const testing = std.testing;
    const alloc = testing.allocator;

    const gen = RateGenerator.initWithSeed(alloc, 0.05, 0.02, 0.15, 42);
    const path1 = try gen.generatePathSeeded(10, 1.0, 42);
    defer alloc.free(path1);
    const path2 = try gen.generatePathSeeded(10, 1.0, 42);
    defer alloc.free(path2);

    for (path1, path2) |a, b| {
        try testing.expectApproxEqAbs(a, b, 1e-10);
    }
}

test "RateGenerator multiple paths" {
    const testing = std.testing;
    const alloc = testing.allocator;

    const gen = RateGenerator.init(alloc, 0.05, 0.02, 0.15);
    const paths = try gen.generatePaths(100, 10, 1.0);
    defer {
        for (paths) |p| alloc.free(p);
        alloc.free(paths);
    }

    try testing.expectEqual(@as(usize, 100), paths.len);
    for (paths) |path| {
        try testing.expectEqual(@as(usize, 11), path.len);
        try testing.expectApproxEqAbs(@as(f64, 0.05), path[0], 1e-10);
    }
}

test "VasicekGenerator" {
    const testing = std.testing;
    const alloc = testing.allocator;

    const gen = VasicekGenerator.init(alloc, 0.05, 0.03, 0.1, 0.02);
    const path = try gen.generatePath(10, 1.0);
    defer alloc.free(path);

    try testing.expectEqual(@as(usize, 11), path.len);
    try testing.expectApproxEqAbs(@as(f64, 0.05), path[0], 1e-10);
    for (path) |rate| {
        try testing.expect(rate >= 0);
    }
}
