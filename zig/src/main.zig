// main.zig - CLI entry point for v-star actuarial engine
//
// Port of Go cmd/v-star/main.go. Full CLI with subcommands.
// Uses std.Io.Writer for all output via init.io.
const std = @import("std");
const mem = std.mem;
const process = std.process;
const fmt = std.fmt;
const Allocator = std.mem.Allocator;

const rates = @import("rates.zig");
const reader = @import("reader.zig");
const stochastic = @import("stochastic.zig");
const mortality = @import("mortality.zig");
const annuities = @import("annuities.zig");
const risk = @import("risk.zig");
const writer = @import("writer.zig");
const bench = @import("bench.zig");
const concurrency = @import("concurrency.zig");

const usage_general =
\\v-star: High-performance zero-dependency actuarial engine
\\
\\Usage: v-star [flags] [subcommand] [args]
\\
\\Subcommands:
\\  read        Process CSV file and calculate valuations
\\  montecarlo  Generate Monte Carlo interest rate paths
\\  bench       Run performance benchmark suite
\\  serve       Start HTTP API server (for Python/R/Excel)
\\
\\Flags:
\\  -i float    effective annual interest rate (default 0.05)
\\  -j float    compounding growth rate for v-star (default 0.02)
\\  -version    show version and exit
\\  -h, help    show this help message
\\
\\Default behavior (no subcommand):
\\  Calculate discount factors using specified rates
\\
\\Examples:
\\  v-star -i 0.05 -j 0.02
\\  v-star read policies.csv --benchmark
\\  v-star read policies.csv --table=mortality.csv --output=json
\\  v-star montecarlo --paths=100000 --steps=10 --seed=42
\\  v-star bench
\\  v-star serve --port=8080
\\
\\Run 'v-star help <subcommand>' for detailed subcommand help.
;

const usage_read =
\\Usage: v-star read <file.csv> [flags]
\\
\\Process a CSV file and calculate present values for each record.
\\
\\Arguments:
\\  file.csv    Path to CSV file (required)
\\
\\Flags:
\\  -i, --interest=FLOAT   interest rate (default 0.05)
\\  -t, --table=PATH       mortality table CSV path
\\  -o, --output=STRING    output format: console, json, csv, report (default console)
\\  -b, --benchmark        show benchmark results
\\  -l, --limit=N          limit number of rows to process
\\  -H, --header           file has header row (default true)
\\  -h, help               show this help
\\
\\CSV Format (without mortality table):
\\  age,term,sum_assured
\\
\\CSV Format (with mortality table):
\\  age,sex,policy_type,term,sum_assured
;

const usage_montecarlo =
\\Usage: v-star montecarlo [flags]
\\
\\Generate Monte Carlo interest rate paths using Geometric Brownian Motion.
\\
\\Flags:
\\  -p, --paths=N            number of paths to generate (default 100000)
\\  -s, --steps=N            number of time steps (default 10)
\\  -d, --drift=FLOAT        drift parameter (default 0.02)
\\  -v, --volatility=FLOAT   volatility (default 0.15)
\\  -S, --seed=N             random seed (default random, -1 for deterministic use 42)
\\  -i, --interest=FLOAT     initial interest rate (default 0.05)
\\  -h, help                 show this help
;

const usage_bench =
\\Usage: v-star bench
\\
\\Run performance benchmark suite including:
\\  - CSV parsing (streaming, parallel, raw)
\\  - Present value calculations
\\  - Monte Carlo path generation
\\  - Risk measure computation
\\
\\No flags. Output includes detailed timing and throughput.
;

pub fn main(init: process.Init) !void {
    const io = init.io;
    const arena = init.arena;
    const gpa = init.gpa;

    // Get stdout/stderr writers
    const stdout_file = std.Io.File.stdout();
    const stderr_file = std.Io.File.stderr();
    var stdout_buf: [4096]u8 = undefined;
    var stderr_buf: [4096]u8 = undefined;
    var stdout_fw = std.Io.File.Writer.initStreaming(stdout_file, io, &stdout_buf);
    var stderr_fw = std.Io.File.Writer.initStreaming(stderr_file, io, &stderr_buf);
    // Use .interface pointer (embedded in File.Writer) for vtable-based dispatch
    var stdout = &stdout_fw.interface;
    var stderr = &stderr_fw.interface;

    // Parse command-line args
    const args_slice = try init.minimal.args.toSlice(arena.allocator());
    const args = args_slice[1..]; // skip program name

    // Parse global flags (interest rate).
    var interest: f64 = 0.05;
    var growth: f64 = 0.02;
    var show_version: bool = false;

    for (args) |arg| {
        if (mem.eql(u8, arg, "--version")) {
            show_version = true;
        } else if (mem.startsWith(u8, arg, "-i")) {
            const val_str = if (mem.eql(u8, arg, "-i")) blk: {
                break :blk "";
            } else arg[2..];
            if (val_str.len > 0) {
                interest = fmt.parseFloat(f64, val_str) catch interest;
            }
        } else if (mem.startsWith(u8, arg, "--interest=")) {
            interest = fmt.parseFloat(f64, arg["--interest=".len..]) catch interest;
        } else if (mem.startsWith(u8, arg, "-j")) {
            const val_str = if (mem.eql(u8, arg, "-j")) blk: {
                break :blk "";
            } else arg[2..];
            if (val_str.len > 0) {
                growth = fmt.parseFloat(f64, val_str) catch growth;
            }
        } else if (mem.startsWith(u8, arg, "--growth=")) {
            growth = fmt.parseFloat(f64, arg["--growth=".len..]) catch growth;
        }
    }

    if (show_version) {
        try stdout.print("v-star 0.1.0\n", .{});
        try stdout_fw.flush();
        return;
    }

    // Find subcommand (first non-flag arg)
    var subcmd: ?[]const u8 = null;
    for (args) |arg| {
        if (!mem.startsWith(u8, arg, "-")) {
            subcmd = arg;
            break;
        }
    }

    if (subcmd) |cmd| {
        if (mem.eql(u8, cmd, "help") or mem.eql(u8, cmd, "--help") or mem.eql(u8, cmd, "-h")) {
            try printUsage(stdout, args);
        } else if (mem.eql(u8, cmd, "read")) {
            try runRead(stdout, stderr, gpa, io, args);
        } else if (mem.eql(u8, cmd, "montecarlo")) {
            try runMonteCarlo(stdout, stderr, gpa, io, args, interest);
        } else if (mem.eql(u8, cmd, "bench")) {
            try bench.bench(stdout, stderr, gpa, io);
        } else if (mem.eql(u8, cmd, "serve")) {
            try stdout.print("serve: not yet implemented\n", .{});
        } else {
            try stderr.print("Error: unknown subcommand '{s}'\n\n", .{cmd});
            try stderr_fw.flush();
            try stdout.writeAll(usage_general);
        }
    } else {
        // No subcommand: print help and run default
        try stdout.writeAll(usage_general);
        try runDefault(stdout, interest, growth);
    }

    try stdout_fw.flush();
}

fn printUsage(w: anytype, args: []const [:0]const u8) !void {
    // Find first non-flag arg after "help"
    var i: usize = 1;
    while (i < args.len and (args[i][0] == '-' or mem.eql(u8, args[i], "help"))) : (i += 1) {}

    if (i < args.len) {
        const sub = args[i];
        if (mem.eql(u8, sub, "read")) {
            try w.writeAll(usage_read);
        } else if (mem.eql(u8, sub, "montecarlo")) {
            try w.writeAll(usage_montecarlo);
        } else if (mem.eql(u8, sub, "bench")) {
            try w.writeAll(usage_bench);
        } else {
            try w.print("No help available for '{s}'\n\n", .{sub});
            try w.writeAll(usage_general);
        }
    } else {
        try w.writeAll(usage_general);
    }
}

fn runDefault(w: anytype, interest: f64, growth: f64) !void {
    const allocator = std.heap.page_allocator;
    var converter = try rates.RateConverter.init(allocator, interest);
    defer converter.deinit();
    try w.print("--- V-star Actuarial Engine ---\n", .{});
    try w.print("Effective Rate (i): {d:.2}%\n", .{interest * 100});
    try w.print("Growth Rate (j): {d:.2}%\n", .{growth * 100});
    try w.print("Standard Discount (v): {d:.6}\n", .{converter.v()});
    try w.print("V-Star (v*):           {d:.6}\n", .{converter.vStar(growth)});
    try w.print("-------------------------------\n", .{});
}

// ---------------------------------------------------------------------------
// Subcommand: read
// ---------------------------------------------------------------------------

fn runRead(w: anytype, err: anytype, _gpa: Allocator, io: std.Io, args: []const [:0]const u8) !void {
    _ = _gpa;
    // Find filepath (first non-flag argument after "read")
    var filepath: ?[]const u8 = null;
    var args_start: usize = 1;
    for (args, 0..) |arg, i| {
        if (i == 0) continue; // skip "read"
        if (!mem.startsWith(u8, arg, "-")) {
            filepath = arg;
            args_start = i;
            break;
        }
    }

    const fp = filepath orelse {
        try err.writeAll("Error: missing filepath argument\n\n");
        try w.writeAll(usage_read);
        return;
    };

    // Parse flags
    var interest: f64 = 0.05;
    var benchmark: bool = false;
    var header: bool = true;
    var limit: usize = 0;
    var output: []const u8 = "console";
    var table_path: ?[]const u8 = null;

    for (args[args_start + 1 ..]) |arg| {
        if (mem.eql(u8, arg, "--benchmark") or mem.eql(u8, arg, "-benchmark")) {
            benchmark = true;
        } else if (mem.startsWith(u8, arg, "--limit=") or mem.startsWith(u8, arg, "-limit=")) {
            const eq_idx = mem.indexOfScalar(u8, arg, '=') orelse continue;
            limit = fmt.parseUnsigned(usize, arg[eq_idx + 1 ..], 10) catch continue;
        } else if (mem.startsWith(u8, arg, "--header=") or mem.startsWith(u8, arg, "-header=")) {
            const eq_idx = mem.indexOfScalar(u8, arg, '=') orelse continue;
            header = mem.eql(u8, arg[eq_idx + 1 ..], "true");
        } else if (mem.startsWith(u8, arg, "--output=") or mem.startsWith(u8, arg, "-output=")) {
            const eq_idx = mem.indexOfScalar(u8, arg, '=') orelse continue;
            output = arg[eq_idx + 1 ..];
        } else if (mem.startsWith(u8, arg, "--interest=")) {
            const eq_idx = mem.indexOfScalar(u8, arg, '=') orelse continue;
            interest = fmt.parseFloat(f64, arg[eq_idx + 1 ..]) catch interest;
        } else if (mem.startsWith(u8, arg, "--table=") or mem.startsWith(u8, arg, "-table=")) {
            const eq_idx = mem.indexOfScalar(u8, arg, '=') orelse continue;
            table_path = arg[eq_idx + 1 ..];
        }
    }

    // Run the valuation
    if (table_path) |tp| {
        try runReadWithMortality(w, err, io, fp, interest, tp, header, limit, output, benchmark);
    } else {
        try runReadDirect(w, err, io, fp, interest, header, limit, output, benchmark);
    }
}

fn runReadDirect(w: anytype, err: anytype, io: std.Io, filepath: []const u8, interest: f64, header: bool, limit: usize, output: []const u8, benchmark: bool) !void {
    _ = err;
    _ = output;
    const allocator = std.heap.page_allocator;
    const opts = reader.CSVOptions{ .header = header, .limit = limit };
    const start = std.Io.Timestamp.now(io, .real);

    var total_pv: f64 = 0;
    var results = std.ArrayList(writer.Record).empty;
    defer results.deinit(allocator);

    const Context = struct {
        total_pv: *f64,
        results: *std.ArrayList(writer.Record),
        allocator: Allocator,
        converter: rates.RateConverter,

        fn callback(rec: reader.CensusRecord, ctx_ptr: ?*anyopaque) void {
            const self = @as(*@This(), @ptrCast(@alignCast(ctx_ptr.?)));
            const pv = self.converter.presentValue(rec.sum_assured, @max(rec.term, 0));
            self.total_pv.* += pv;
            self.results.append(self.allocator, writer.Record{
                .age = rec.age,
                .sex = rec.sex,
                .policy_type = rec.policy_type,
                .sum_assured = rec.sum_assured,
                .term = rec.term,
                .present_value = pv,
            }) catch {};
        }
    };

    var ctx = Context{
        .total_pv = &total_pv,
        .results = &results,
        .allocator = allocator,
        .converter = try rates.RateConverter.init(allocator, interest),
    };
    try reader.streamCensus(io, filepath, opts, Context.callback, &ctx);

    const end = std.Io.Timestamp.now(io, .real);
    const elapsed: u64 = @intCast(end.nanoseconds - start.nanoseconds);
    const count = results.items.len;

    try writeReadOutput(w, results.items, total_pv, @intCast(count), interest, null, benchmark, elapsed);
}

fn runReadWithMortality(w: anytype, err: anytype, io: std.Io, filepath: []const u8, interest: f64, table_path: []const u8, header: bool, limit: usize, output: []const u8, benchmark: bool) !void {
    _ = output;
    const allocator = std.heap.page_allocator;

    // Load mortality table
    var table = mortality.loadQxFromCSV(io, allocator, table_path) catch {
        try err.print("Error loading mortality table: {s}\n", .{table_path});
        return;
    };
    defer table.deinit();

    var rc = try rates.RateConverter.init(allocator, interest);
    defer rc.deinit();
    const calc = annuities.AnnuityCalculator.init(&rc, &table);
    const opts = reader.CSVOptions{ .header = header, .limit = limit };
    const start = std.Io.Timestamp.now(io, .real);

    var total_pv: f64 = 0;
    var results = std.ArrayList(writer.Record).empty;
    defer results.deinit(allocator);

    const Context = struct {
        total_pv: *f64,
        results: *std.ArrayList(writer.Record),
        allocator: Allocator,
        calc: annuities.AnnuityCalculator,

        fn callback(rec: reader.CensusRecord, ctx_ptr: ?*anyopaque) void {
            const self = @as(*@This(), @ptrCast(@alignCast(ctx_ptr.?)));
            const age: i64 = @max(rec.age, 0);
            const term: i64 = @max(rec.term, 0);
            const pv = self.calc.termImmediate(age, term, rec.sum_assured);
            self.total_pv.* += pv;
            self.results.append(self.allocator, writer.Record{
                .age = rec.age,
                .sex = rec.sex,
                .policy_type = rec.policy_type,
                .sum_assured = rec.sum_assured,
                .term = rec.term,
                .present_value = pv,
            }) catch {};
        }
    };

    var ctx = Context{
        .total_pv = &total_pv,
        .results = &results,
        .allocator = allocator,
        .calc = calc,
    };

    try reader.streamCensus(io, filepath, opts, Context.callback, &ctx);

    const end = std.Io.Timestamp.now(io, .real);
    const elapsed: u64 = @intCast(end.nanoseconds - start.nanoseconds);
    const count = results.items.len;

    try writeReadOutput(w, results.items, total_pv, @intCast(count), interest, table_path, benchmark, elapsed);
}

fn writeReadOutput(w: anytype, results: []const writer.Record, total_pv: f64, count: usize, interest: f64, table_path: ?[]const u8, benchmark: bool, elapsed_ns: u64) !void {
    _ = results;
    _ = interest;
    const elapsed_sec = @as(f64, @floatFromInt(elapsed_ns)) / 1_000_000_000.0;

    if (table_path) |tp| {
        try w.print("Processed {d} records with mortality table: {s}\n", .{ count, tp });
    } else {
        try w.print("Processed {d} records\n", .{count});
    }
    try w.print("Total Present Value: {d:.2}\n", .{total_pv});

    if (benchmark) {
        const throughput = if (elapsed_sec > 0) @as(f64, @floatFromInt(count)) / elapsed_sec else 0;
        try w.print("\n=== Benchmark Results ===\n", .{});
        try w.print("Total rows: {d}\n", .{count});
        try w.print("Duration: {d:.3}s\n", .{elapsed_sec});
        try w.print("Throughput: {d:.0} rows/sec\n", .{throughput});
        try w.print("Total Present Value: {d:.2}\n", .{total_pv});
    }
}

// ---------------------------------------------------------------------------
// Subcommand: montecarlo
// ---------------------------------------------------------------------------

fn runMonteCarlo(w: anytype, err: anytype, gpa: Allocator, io: std.Io, args: []const [:0]const u8, default_interest: f64) !void {
    _ = err;
    _ = gpa;
    const allocator = std.heap.page_allocator;

    var num_paths: usize = 100000;
    var steps: usize = 10;
    var drift: f64 = 0.02;
    var volatility: f64 = 0.15;
    var seed: i64 = -1;
    var interest: f64 = default_interest;

    var i: usize = 1;
    while (i < args.len) : (i += 1) {
        const arg = args[i];
        if (mem.startsWith(u8, arg, "--paths=")) {
            const eq_idx = mem.indexOfScalar(u8, arg, '=') orelse continue;
            num_paths = fmt.parseUnsigned(usize, arg[eq_idx + 1 ..], 10) catch continue;
        } else if (mem.startsWith(u8, arg, "--steps=")) {
            const eq_idx = mem.indexOfScalar(u8, arg, '=') orelse continue;
            steps = fmt.parseUnsigned(usize, arg[eq_idx + 1 ..], 10) catch continue;
        } else if (mem.startsWith(u8, arg, "--drift=")) {
            const eq_idx = mem.indexOfScalar(u8, arg, '=') orelse continue;
            drift = fmt.parseFloat(f64, arg[eq_idx + 1 ..]) catch continue;
        } else if (mem.startsWith(u8, arg, "--volatility=")) {
            const eq_idx = mem.indexOfScalar(u8, arg, '=') orelse continue;
            volatility = fmt.parseFloat(f64, arg[eq_idx + 1 ..]) catch continue;
        } else if (mem.startsWith(u8, arg, "--seed=")) {
            const eq_idx = mem.indexOfScalar(u8, arg, '=') orelse continue;
            seed = fmt.parseUnsigned(i64, arg[eq_idx + 1 ..], 10) catch continue;
        } else if (mem.startsWith(u8, arg, "--interest=")) {
            const eq_idx = mem.indexOfScalar(u8, arg, '=') orelse continue;
            interest = fmt.parseFloat(f64, arg[eq_idx + 1 ..]) catch continue;
        }
    }

    try w.print("Generating {d} Monte Carlo interest rate paths...\n", .{num_paths});
    try w.print("Parameters: Initial Rate={d:.2}%, Drift={d:.2}%, Volatility={d:.2}%, Steps={d}\n", .{ interest * 100, drift * 100, volatility * 100, steps });
    if (seed >= 0) {
        try w.print("Seed: {d} (deterministic)\n", .{seed});
    }

    const start = std.Io.Timestamp.now(io, .real);

    const paths = if (seed >= 0) blk: {
        const gen = stochastic.RateGenerator.initWithSeed(allocator, interest, drift, volatility, @as(u64, @intCast(seed)));
        break :blk try gen.generatePaths(num_paths, steps, 1.0);
    } else blk: {
        const gen = stochastic.RateGenerator.init(allocator, interest, drift, volatility);
        break :blk try gen.generatePaths(num_paths, steps, 1.0);
    };
    defer {
        for (paths) |p| allocator.free(p);
        allocator.free(paths);
    }

    const end = std.Io.Timestamp.now(io, .real);
    const elapsed: u64 = @intCast(end.nanoseconds - start.nanoseconds);
    const elapsed_sec = @as(f64, @floatFromInt(elapsed)) / 1_000_000_000.0;

    // Final rate statistics
    var total_rate: f64 = 0;
    var min_rate: f64 = 1e9;
    var max_rate: f64 = -1e9;
    var final_rates = try allocator.alloc(f64, num_paths);
    defer allocator.free(final_rates);

    for (paths, 0..) |path, idx| {
        const rate = path[steps];
        final_rates[idx] = rate;
        total_rate += rate;
        if (rate < min_rate) min_rate = rate;
        if (rate > max_rate) max_rate = rate;
    }

    const avg_rate = total_rate / @as(f64, @floatFromInt(num_paths));

    // Sort for percentiles
    std.sort.block(f64, final_rates, {}, comptime std.sort.asc(f64));

    try w.print("\n=== Monte Carlo Results ===\n", .{});
    try w.print("Paths Generated: {d}\n", .{num_paths});
    try w.print("Duration: {d:.3}s\n", .{elapsed_sec});
    try w.print("Throughput: {d:.0} paths/sec\n", .{@as(f64, @floatFromInt(num_paths)) / elapsed_sec});
    try w.print("\nFinal Rate Statistics:\n", .{});
    try w.print("  Average: {d:.4}%\n", .{avg_rate * 100});
    try w.print("  Minimum: {d:.4}%\n", .{min_rate * 100});
    try w.print("  Maximum: {d:.4}%\n", .{max_rate * 100});
    try w.print("\nPercentiles:\n", .{});

    const percentiles = [_]struct { name: []const u8, p: f64 }{
        .{ .name = "5th (VaR 95%)", .p = 0.05 },
        .{ .name = "25th", .p = 0.25 },
        .{ .name = "50th (Median)", .p = 0.50 },
        .{ .name = "75th", .p = 0.75 },
        .{ .name = "95th", .p = 0.95 },
    };

    for (percentiles) |p| {
        const idx_f: f64 = @as(f64, @floatFromInt(num_paths - 1)) * p.p;
        const idx: usize = @intFromFloat(idx_f);
        try w.print("  {s}: {d:.4}%\n", .{ p.name, final_rates[idx] * 100 });
    }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test {
    _ = @import("rates.zig");
    _ = @import("reader.zig");
    _ = @import("stochastic.zig");
    _ = @import("risk.zig");
    _ = @import("annuities.zig");
    _ = @import("reserves.zig");
    _ = @import("writer.zig");
    _ = @import("concurrency.zig");
    _ = @import("bench.zig");
    _ = @import("mortality.zig");
}
