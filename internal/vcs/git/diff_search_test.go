package git
		opt  RawLogDiffSearchOptions
		want []*LogCommitSearchResult
		opt: RawLogDiffSearchOptions{
			Query: TextSearchOptions{Pattern: "root"},
		want: []*LogCommitSearchResult{{
			Commit: Commit{
				Author:    Signature{Name: "a", Email: "a@a.com", Date: MustParseTime(time.RFC3339, "2006-01-02T15:04:06Z")},
				Committer: &Signature{Name: "a", Email: "a@a.com", Date: MustParseTime(time.RFC3339, "2006-01-02T15:04:06Z")},
			Diff:       &Diff{Raw: "diff --git a/f b/f\nindex d8649da..1193ff4 100644\n--- a/f\n+++ b/f\n@@ -1,1 +1,1 @@\n-root\n+branch1\n"},
			Commit: Commit{
				Author:    Signature{Name: "a", Email: "a@a.com", Date: MustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
				Committer: &Signature{Name: "a", Email: "a@a.com", Date: MustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
			Diff:       &Diff{Raw: "diff --git a/f b/f\nnew file mode 100644\nindex 0000000..d8649da\n--- /dev/null\n+++ b/f\n@@ -0,0 +1,1 @@\n+root\n"},
		opt: RawLogDiffSearchOptions{
			Query: TextSearchOptions{Pattern: ""},
		want: []*LogCommitSearchResult{{
			Commit: Commit{
				Author:    Signature{Name: "a", Email: "a@a.com", Date: MustParseTime(time.RFC3339, "2006-01-02T15:04:06Z")},
				Committer: &Signature{Name: "a", Email: "a@a.com", Date: MustParseTime(time.RFC3339, "2006-01-02T15:04:06Z")},
			Commit: Commit{
				Author:    Signature{Name: "a", Email: "a@a.com", Date: MustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
				Committer: &Signature{Name: "a", Email: "a@a.com", Date: MustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
		opt: RawLogDiffSearchOptions{
			Paths: PathOptions{
			results, complete, err := RawLogDiffSearch(ctx, repo, test.opt)
		want map[*RawLogDiffSearchOptions][]*LogCommitSearchResult
			want: map[*RawLogDiffSearchOptions][]*LogCommitSearchResult{
					Paths: PathOptions{IncludePatterns: []string{"/xyz.txt"}, IsRegExp: true},
			results, complete, err := RawLogDiffSearch(ctx, test.repo, *opt)