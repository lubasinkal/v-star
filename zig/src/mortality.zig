// mortality.zig - Mortality tables, qx/px calculations, CSV loading
//
// Port of Go pkg/mortality. Provides Table, DecrementTable, and CSV loading.
const std = @import("std");
const mem = std.mem;
const Allocator = mem.Allocator;
const reader = @import("reader.zig");

/// MortalityTable provides mortality rates (qx), survival probabilities (px),
/// curtate life expectancy (ex), survivor count (lx), and maximum age.
pub const MortalityTable = struct {
    name: []const u8,
    qx: []f64,
    lx: []f64,
    ex: []f64,
    max_age: i64,
    allocator: Allocator,

    /// Constructs a Table from a slice of qx values.
    /// Computes lx internally using radix 100000.
    /// Pre-computes curtate expectation of life ex via recurrence.
    pub fn init(allocator: Allocator, name: []const u8, qx: []const f64) !MortalityTable {
        const max_age: i64 = if (qx.len > 0) @intCast(qx.len - 1) else -1;

        const lx = try allocator.alloc(f64, @max(qx.len, 1));
        lx[0] = 100000.0;
        for (1..@max(qx.len, 1)) |i| {
            lx[i] = lx[i - 1] * (1.0 - qx[i - 1]);
        }

        const ex = try allocator.alloc(f64, lx.len);
        @memset(ex, 0.0);

        // Compute via recurrence (backward)
        if (max_age >= 0) {
            var age: i64 = max_age - 1;
            while (age >= 0) : (age -= 1) {
                const age_usize: usize = @intCast(age);
                const p = lx[age_usize + 1] / lx[age_usize];
                ex[age_usize] = p * (1.0 + ex[age_usize + 1]);
            }
        }

        // Dupe the name
        const name_dup = try allocator.dupe(u8, name);

        return .{
            .name = name_dup,
            .qx = try allocator.dupe(f64, qx),
            .lx = lx,
            .ex = ex,
            .max_age = max_age,
            .allocator = allocator,
        };
    }

    pub fn deinit(self: *MortalityTable) void {
        self.allocator.free(self.name);
        self.allocator.free(self.qx);
        self.allocator.free(self.lx);
        self.allocator.free(self.ex);
    }

    /// Returns the probability of death between age x and x+1.
    pub fn qxAt(self: *const MortalityTable, age: i64) f64 {
        if (age < 0 or age > self.max_age) return 0.0;
        const a: usize = @intCast(age);
        if (a >= self.qx.len) return 0.0;
        return self.qx[a];
    }

    /// Returns the cumulative survival probability over term years from age.
    /// Uses pre-computed lx table for O(1) lookup.
    pub fn px(self: *const MortalityTable, age: i64, term: i64) f64 {
        if (age < 0 or term <= 0) return 1.0;
        const end_age = age + term;
        if (end_age > self.max_age) return 0.0;
        const a: usize = @intCast(age);
        const e: usize = @intCast(end_age);
        if (a >= self.lx.len or self.lx[a] == 0) return 0.0;
        return self.lx[e] / self.lx[a];
    }

    /// Returns the curtate expectation of life at the given age.
    pub fn exAt(self: *const MortalityTable, age: i64) f64 {
        if (age < 0 or age > self.max_age) return 0.0;
        const a: usize = @intCast(age);
        return self.ex[a];
    }

    /// Returns the maximum age defined in the table.
    pub fn maxAge(self: *const MortalityTable) i64 {
        return self.max_age;
    }

    /// Returns the number of lives surviving to the given age from radix 100000.
    pub fn lxAt(self: *const MortalityTable, age: i64) f64 {
        if (age < 0 or age > self.max_age) return 0.0;
        const a: usize = @intCast(age);
        return self.lx[a];
    }
};

/// DecrementTable combines multiple causes of decrement into a single table.
pub const DecrementTable = struct {
    tables: []*const MortalityTable,
    names: []const []const u8,
    qx: []f64,
    lx: []f64,
    max_age: i64,
    allocator: Allocator,

    /// Creates a combined decrement table from multiple single-decrement tables.
    pub fn init(allocator: Allocator, tables: []*const MortalityTable, names: []const []const u8) !DecrementTable {
        if (tables.len == 0) {
            return .{
                .tables = &.{},
                .names = &.{},
                .qx = &.{},
                .lx = &.{},
                .max_age = -1,
                .allocator = allocator,
            };
        }

        var max_age = tables[0].maxAge();
        for (tables[1..]) |t| {
            if (t.maxAge() < max_age) max_age = t.maxAge();
        }

        const qx_len: usize = @intCast(max_age + 1);
        const qx = try allocator.alloc(f64, qx_len);
        for (0..qx_len) |age_f| {
            const age: i64 = @intCast(age_f);
            var survival: f64 = 1.0;
            for (tables) |t| {
                survival *= 1.0 - t.qxAt(age);
            }
            qx[age_f] = 1.0 - survival;
        }

        const lx = try allocator.alloc(f64, qx_len + 1);
        lx[0] = 100000.0;
        for (1..lx.len) |i| {
            lx[i] = lx[i - 1] * (1.0 - qx[i - 1]);
        }

        return .{
            .tables = tables,
            .names = names,
            .qx = qx,
            .lx = lx,
            .max_age = max_age,
            .allocator = allocator,
        };
    }

    pub fn deinit(self: *DecrementTable) void {
        self.allocator.free(self.qx);
        self.allocator.free(self.lx);
    }

    /// Total probability of decrement at age: 1 - prod(1 - qx_i).
    pub fn qxAt(self: *const DecrementTable, age: i64) f64 {
        if (age < 0 or age > self.max_age) return 0.0;
        const a: usize = @intCast(age);
        return self.qx[a];
    }

    /// Total survival probability over term years using pre-computed lx.
    pub fn px(self: *const DecrementTable, age: i64, term: i64) f64 {
        if (age < 0 or term <= 0 or age > self.max_age) return 0.0;
        const end_age = age + term;
        const a: usize = @intCast(age);
        if (a >= self.lx.len or self.lx[a] == 0) return 0.0;
        const e: usize = @intCast(end_age);
        if (e >= self.lx.len) return 0.0;
        return self.lx[e] / self.lx[a];
    }

    pub fn maxAge(self: *const DecrementTable) i64 {
        return self.max_age;
    }
};

// ---------------------------------------------------------------------------
// CSV Loading
// ---------------------------------------------------------------------------

/// Loads qx values from a CSV file with "age" and "qx" columns.
pub fn loadQxFromCSV(io: std.Io, allocator: Allocator, filepath: []const u8) !MortalityTable {
    // Read the entire file
    const cwd = std.Io.Dir.cwd();
    var file = try cwd.openFile(io, filepath, .{ .mode = .read_only });
    defer file.close(io);

    var file_buf: [4096]u8 = undefined;
    var fr = file.reader(io, &file_buf);
    const content = try fr.interface.allocRemaining(allocator, .unlimited);
    defer allocator.free(content);

    var qx: std.ArrayList(f64) = .empty;
    errdefer qx.deinit(allocator);

    // First line: header
    var pos: usize = 0;
    while (pos < content.len and content[pos] != '\n') : (pos += 1) {}
    if (pos < content.len) pos += 1; // skip \n

    // Parse data lines: "age,qx" format
    while (pos < content.len) {
        const start = pos;
        while (pos < content.len and content[pos] != '\n') : (pos += 1) {}
        var line = content[start..pos];
        if (pos < content.len) pos += 1;

        // Trim trailing \r
        if (line.len > 0 and line[line.len - 1] == '\r') {
            line = line[0 .. line.len - 1];
        }
        if (line.len == 0) continue;

        // Find comma
        const comma = mem.indexOfScalar(u8, line, ',') orelse continue;
        const age_str = line[0..comma];
        const qx_str = line[comma + 1 ..];

        const age = reader.parseFastInt(age_str) catch continue;
        const q = reader.parseFastFloat(qx_str) catch 0.0;

        const a: usize = @intCast(age);
        while (qx.items.len <= a) {
            try qx.append(allocator, 0.0);
        }
        qx.items[a] = q;
    }

    return try MortalityTable.init(allocator, filepath, qx.items);
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test "MortalityTable basics" {
    const testing = std.testing;
    const alloc = testing.allocator;

    // Simple qx table: age 0 has 0.001 death prob, age 1 has 0.002, age 2 has 0.003
    const qx = [_]f64{ 0.001, 0.002, 0.003 };
    var table = try MortalityTable.init(alloc, "test", &qx);
    defer table.deinit();

    try testing.expectEqual(@as(i64, 2), table.maxAge());
    try testing.expectApproxEqAbs(@as(f64, 0.001), table.qxAt(0), 1e-10);
    try testing.expectApproxEqAbs(@as(f64, 0.002), table.qxAt(1), 1e-10);
    try testing.expectApproxEqAbs(@as(f64, 0.003), table.qxAt(2), 1e-10);

    // Out of range
    try testing.expectApproxEqAbs(@as(f64, 0.0), table.qxAt(-1), 1e-10);
    try testing.expectApproxEqAbs(@as(f64, 0.0), table.qxAt(3), 1e-10);

    // lx
    try testing.expectApproxEqAbs(@as(f64, 100000.0), table.lxAt(0), 1e-10);
    try testing.expect(table.lxAt(1) < 100000.0);

    // Px (survival probability)
    const px_0_1 = table.px(0, 1);
    try testing.expectApproxEqAbs(@as(f64, 0.999), px_0_1, 1e-10);
}

test "MortalityTable single entry" {
    const testing = std.testing;
    const alloc = testing.allocator;
    const qx = [_]f64{0.001};
    var table = try MortalityTable.init(alloc, "single", &qx);
    defer table.deinit();
    try testing.expectEqual(@as(i64, 0), table.maxAge());
    try testing.expectApproxEqAbs(@as(f64, 0.001), table.qxAt(0), 1e-10);
}
