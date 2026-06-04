// reader.zig - CSV parsing, CensusRecord, streaming
//
// Port of Go pkg/reader. Zero-allocation CSV parsing with parallel support.
// NOTE: All file I/O requires an 'io: std.Io' parameter (Zig 0.16.0 API).
// File reading uses allocRemaining() to load entire file, then line-by-line parsing.
const std = @import("std");
const mem = std.mem;
const Allocator = mem.Allocator;
const Io = std.Io;
const Dir = Io.Dir;

/// CensusRecord represents a policy record with core actuarial fields.
pub const CensusRecord = struct {
    sex: []const u8 = "",
    policy_type: []const u8 = "",
    age: i64 = 0,
    sum_assured: f64 = 0,
    term: i64 = 0,

    /// Duplicates string fields into the given allocator for persistent storage.
    pub fn clone(self: CensusRecord, allocator: Allocator) !CensusRecord {
        return .{
            .age = self.age,
            .sex = try allocator.dupe(u8, self.sex),
            .policy_type = try allocator.dupe(u8, self.policy_type),
            .sum_assured = self.sum_assured,
            .term = self.term,
        };
    }

    /// Frees string fields allocated by `clone`.
    pub fn deinit(self: *CensusRecord, allocator: Allocator) void {
        if (self.sex.len > 0) allocator.free(self.sex);
        if (self.policy_type.len > 0) allocator.free(self.policy_type);
    }
};

/// CSVOptions configures CSV reading behavior.
pub const CSVOptions = struct {
    header: bool = true,
    delimiter: u8 = ',',
    limit: usize = 0,
};

/// ColumnMap maps CSV column names to their indices.
pub const ColumnMap = std.StringHashMap(usize);

/// Builds a ColumnMap from header names.
pub fn buildColumnMap(headers: []const []const u8) ColumnMap {
    var col_map = ColumnMap.init(std.heap.page_allocator);
    for (headers, 0..) |h, i| {
        col_map.put(normalizeColumnName(h), i) catch {};
    }
    return col_map;
}

/// Normalizes column name for flexible matching.
pub fn normalizeColumnName(name: []const u8) []const u8 {
    return name;
}

/// Returns the default column map: age=0, sex=1, policy_type=2, sum_assured=3, term=4.
pub fn defaultColumnMap() ColumnMap {
    var col_map = ColumnMap.init(std.heap.page_allocator);
    col_map.put("age", 0) catch {};
    col_map.put("sex", 1) catch {};
    col_map.put("policy_type", 2) catch {};
    col_map.put("sum_assured", 3) catch {};
    col_map.put("term", 4) catch {};
    return col_map;
}

/// Parses a single CSV line into fields. No allocation - returns slices into the line.
pub fn parseFields(line: []u8, delimiter: u8) [][]u8 {
    var fields: [16][]u8 = undefined;
    var count: usize = 0;
    var start: usize = 0;
    var i: usize = 0;

    while (i < line.len) : (i += 1) {
        if (line[i] == delimiter) {
            if (count < fields.len) {
                fields[count] = line[start..i];
                count += 1;
            }
            start = i + 1;
        }
    }
    // Last field
    if (count < fields.len) {
        fields[count] = line[start..line.len];
        count += 1;
    }

    return fields[0..count];
}

/// Parses a census line directly from bytes into a CensusRecord.
/// Assumes default column order: age,sex,policy_type,sum_assured,term
/// Single-pass: finds all delimiters, then parses numbers from field slices.
/// NOTE: String fields (sex, policy_type) are slices into the input `line`.
///       If you need persistent records, call `record.clone(allocator)`.
pub fn parseCensusFastBytes(line: []const u8, delimiter: u8) !CensusRecord {
    if (line.len == 0) return error.EmptyLine;

    // Find all 4 delimiter positions in one pass
    var c1: i64 = -1;
    var c2: i64 = -1;
    var c3: i64 = -1;
    var c4: i64 = -1;
    for (line, 0..) |ch, idx| {
        if (ch == delimiter) {
            if (c1 < 0) {
                c1 = @intCast(idx);
            } else if (c2 < 0) {
                c2 = @intCast(idx);
            } else if (c3 < 0) {
                c3 = @intCast(idx);
            } else if (c4 < 0) {
                c4 = @intCast(idx);
            }
        }
    }

    if (c4 < 0) return error.InvalidLineFormat;

    const c1_usize: usize = @intCast(c1);
    const c2_usize: usize = @intCast(c2);
    const c3_usize: usize = @intCast(c3);
    const c4_usize: usize = @intCast(c4);

    const age = parseFastInt(line[0..c1_usize]) catch return error.InvalidAge;
    const sex = line[c1_usize + 1 .. c2_usize];
    const policy_type = line[c2_usize + 1 .. c3_usize];
    const sum_assured = parseFastFloat(line[c3_usize + 1 .. c4_usize]) catch return error.InvalidSumAssured;

    // term field (trim trailing \r)
    var term_field = line[c4_usize + 1 ..];
    if (term_field.len > 0 and term_field[term_field.len - 1] == '\r') {
        term_field = term_field[0 .. term_field.len - 1];
    }
    const term = parseFastInt(term_field) catch return error.InvalidTerm;

    return CensusRecord{
        .age = age,
        .sex = sex,
        .policy_type = policy_type,
        .sum_assured = sum_assured,
        .term = term,
    };
}

/// Parses an integer from a byte slice without allocation.
pub fn parseFastInt(bytes: []const u8) !i64 {
    if (bytes.len == 0) return error.InvalidInteger;
    var result: i64 = 0;
    var neg = false;
    var i: usize = 0;
    if (bytes[0] == '-') {
        neg = true;
        i = 1;
    }
    while (i < bytes.len) : (i += 1) {
        const ch = bytes[i];
        if (ch < '0' or ch > '9') return error.InvalidInteger;
        result = result * 10 + @as(i64, ch - '0');
    }
    return if (neg) -result else result;
}

/// Parses a float from a byte slice without allocation.
pub fn parseFastFloat(bytes: []const u8) !f64 {
    if (bytes.len == 0) return error.InvalidFloat;
    const trimmed = mem.trim(u8, bytes, " \t\r\n");
    return std.fmt.parseFloat(f64, trimmed);
}

/// Internal: reads entire file into a buffer.
fn readFileIntoBuffer(io: Io, filepath: []const u8, allocator: Allocator) ![]u8 {
    const cwd = Dir.cwd();
    var file = try cwd.openFile(io, filepath, .{ .mode = .read_only });
    defer file.close(io);

    var file_buf: [4096]u8 = undefined;
    var fr = file.reader(io, &file_buf);
    const content = try fr.interface.allocRemaining(allocator, .unlimited);
    return content;
}

/// Reads a CSV file header and returns header fields and byte offset past the header.
pub fn readHeadersAndOffset(io: Io, filepath: []const u8, delimiter: u8) !struct { headers: [][]const u8, offset: i64 } {
    const allocator = std.heap.page_allocator;
    const content = try readFileIntoBuffer(io, filepath, allocator);
    defer allocator.free(content);

    // Find end of first line
    var eol: usize = 0;
    while (eol < content.len and content[eol] != '\n') : (eol += 1) {}

    const header_line = content[0..eol];
    var fields_buf: [16][]const u8 = undefined;
    var count: usize = 0;
    var start: usize = 0;
    for (header_line, 0..) |ch, idx| {
        if (ch == delimiter) {
            if (count < fields_buf.len) {
                fields_buf[count] = header_line[start..idx];
                count += 1;
            }
            start = idx + 1;
        }
    }
    if (count < fields_buf.len) {
        fields_buf[count] = header_line[start..];
        count += 1;
    }

    const offset: i64 = @intCast(eol + 1); // skip past \n
    return .{ .headers = try allocator.dupe([]const u8, fields_buf[0..count]), .offset = offset };
}

/// Callback type for streamCensus.
pub const CensusCallback = *const fn (record: CensusRecord, ctx: ?*anyopaque) void;

/// Streams a census CSV file, calling `callback` for each parsed record.
/// String fields in the `CensusRecord` are borrowed from internal buffers
/// and are only valid until the callback returns.
pub fn streamCensus(io: Io, filepath: []const u8, opts: CSVOptions, callback: CensusCallback, ctx: ?*anyopaque) !void {
    const allocator = std.heap.page_allocator;
    const content = try readFileIntoBuffer(io, filepath, allocator);
    defer allocator.free(content);

    var count: usize = 0;
    const limit = opts.limit;
    var pos: usize = 0;

    // Skip header if present
    if (opts.header) {
        while (pos < content.len and content[pos] != '\n') : (pos += 1) {}
        if (pos < content.len) pos += 1; // skip \n
    }

    while (pos < content.len) {
        if (limit > 0 and count >= limit) break;

        // Find end of line
        const start = pos;
        while (pos < content.len and content[pos] != '\n') : (pos += 1) {}
        var line = content[start..pos];
        // Skip trailing \r
        if (line.len > 0 and line[line.len - 1] == '\r') {
            line = line[0 .. line.len - 1];
        }
        if (pos < content.len) pos += 1; // skip \n

        if (line.len == 0) continue;

        const record = parseCensusFastBytes(line, opts.delimiter) catch continue;
        callback(record, ctx);
        count += 1;
    }
}

/// Reads a CSV file as raw lines (no CensusRecord parsing).
/// Returns an ArrayList of lines, each a []const u8.
pub fn readAllCensusRaw(io: Io, allocator: Allocator, filepath: []const u8) !std.ArrayList([]const u8) {
    const content = try readFileIntoBuffer(io, filepath, allocator);
    defer allocator.free(content);

    var list: std.ArrayList([]const u8) = .empty;
    errdefer {
        for (list.items) |item| allocator.free(item);
        list.deinit(allocator);
    }

    var pos: usize = 0;
    while (pos < content.len) {
        // Find end of line
        const start = pos;
        while (pos < content.len and content[pos] != '\n') : (pos += 1) {}
        var line = content[start..pos];
        // Skip trailing \r
        if (line.len > 0 and line[line.len - 1] == '\r') {
            line = line[0 .. line.len - 1];
        }
        if (pos < content.len) pos += 1; // skip \n

        if (line.len == 0) continue;
        const owned = try allocator.dupe(u8, line);
        try list.append(allocator, owned);
    }

    return list;
}

/// Reads all census records from a CSV file into an ArrayList.
/// String fields are cloned so the records remain valid after the file is closed.
pub fn readAllCensus(io: Io, allocator: Allocator, filepath: []const u8, opts: CSVOptions) !std.ArrayList(CensusRecord) {
    const content = try readFileIntoBuffer(io, filepath, allocator);
    defer allocator.free(content);

    var list: std.ArrayList(CensusRecord) = .empty;
    errdefer {
        for (list.items) |*rec| rec.deinit(allocator);
        list.deinit(allocator);
    }

    var pos: usize = 0;

    // Skip header if present
    if (opts.header) {
        while (pos < content.len and content[pos] != '\n') : (pos += 1) {}
        if (pos < content.len) pos += 1;
    }

    while (pos < content.len) {
        if (opts.limit > 0 and list.items.len >= opts.limit) break;

        const start = pos;
        while (pos < content.len and content[pos] != '\n') : (pos += 1) {}
        var line = content[start..pos];
        if (line.len > 0 and line[line.len - 1] == '\r') {
            line = line[0 .. line.len - 1];
        }
        if (pos < content.len) pos += 1;

        if (line.len == 0) continue;

        const record = parseCensusFastBytes(line, opts.delimiter) catch continue;
        const cloned = try record.clone(allocator);
        try list.append(allocator, cloned);
    }

    return list;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test "parseFastInt" {
    const testing = std.testing;
    try testing.expectEqual(@as(i64, 42), try parseFastInt("42"));
    try testing.expectEqual(@as(i64, 0), try parseFastInt("0"));
    try testing.expectEqual(@as(i64, -7), try parseFastInt("-7"));
    try testing.expectError(error.InvalidInteger, parseFastInt(""));
    try testing.expectError(error.InvalidInteger, parseFastInt("abc"));
}

test "parseFastFloat" {
    const testing = std.testing;
    try testing.expectApproxEqAbs(@as(f64, 42.5), try parseFastFloat("42.5"), 1e-10);
    try testing.expectApproxEqAbs(@as(f64, 100000.0), try parseFastFloat("100000"), 1e-10);
}

test "parseCensusFastBytes" {
    const testing = std.testing;
    const line = "30,M,term,100000,20";
    const record = try parseCensusFastBytes(line, ',');
    try testing.expectEqual(@as(i64, 30), record.age);
    try testing.expectEqualStrings("M", record.sex);
    try testing.expectEqualStrings("term", record.policy_type);
    try testing.expectApproxEqAbs(@as(f64, 100000), record.sum_assured, 1e-10);
    try testing.expectEqual(@as(i64, 20), record.term);
}

test "parseFields" {
    const testing = std.testing;
    var line = [_]u8{ 'a', ',', 'b', ',', 'c' };
    const fields = parseFields(&line, ',');
    try testing.expectEqual(@as(usize, 3), fields.len);
    try testing.expectEqualStrings("a", fields[0]);
    try testing.expectEqualStrings("b", fields[1]);
    try testing.expectEqualStrings("c", fields[2]);
}
