// writer.zig - Output formatting (JSON, CSV, text reports)
//
// Provides records and formatting utilities for valuation output.
const std = @import("std");
const Allocator = std.mem.Allocator;
const risk = @import("risk.zig");

/// Record represents a valuation result record.
pub const Record = struct {
    sex: []const u8 = "",
    policy_type: []const u8 = "",
    age: i64 = 0,
    sum_assured: f64 = 0,
    term: i64 = 0,
    present_value: f64 = 0,
};

/// ReportData contains data for generating a text report.
pub const ReportData = struct {
    title: []const u8 = "Actuarial Valuation Report",
    interest_rate: f64 = 0,
    record_count: usize = 0,
    total_present_value: f64 = 0,
    generated_at: []const u8 = "",
    assumptions: std.StringHashMap([]const u8),
    risk_report: ?risk.RiskReport = null,
};

/// SanitizeString removes newlines and other problematic characters.
pub fn sanitizeString(s: []const u8, buf: []u8) []const u8 {
    var len: usize = 0;
    for (s) |ch| {
        if (ch != '\n' and ch != '\r' and ch != '\t') {
            if (len < buf.len) {
                buf[len] = ch;
                len += 1;
            }
        }
    }
    return buf[0..len];
}

test "sanitizeString" {
    const testing = std.testing;
    var buf: [100]u8 = undefined;
    const result = sanitizeString("hello\nworld\r\t!", &buf);
    try testing.expectEqualStrings("helloworld!", result);
}
