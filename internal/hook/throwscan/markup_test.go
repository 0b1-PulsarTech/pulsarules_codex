package throwscan

import "testing"

func TestScan_MarkupComments(t *testing.T) {
	t.Parallel()

	testCases := []scanCase{
		{
			"svelte commented-out script", ".svelte",
			"<!-- <script>throw 1</script> -->\n<p>throw</p>\n", nil,
		},
		{
			"svelte script after a comment", ".svelte",
			"<!-- note -->\n<script>\n  throw error(1)\n</script>\n",
			[]int{3},
		},
		{
			"svelte unclosed comment opener in a script string", ".svelte",
			"<script>\n  const a = \"<!--\"\n</script>\n<!-- <script>throw 1</script> -->\n",
			nil,
		},
		{
			"svelte html comment opener inside a script", ".svelte",
			"<script>\n  const a = 1 // <!--\n  throw error(1)\n</script>\n",
			[]int{3},
		},
		{
			"svelte script opener inside a comment, no script close", ".svelte",
			"<!-- <script> -->\n<p>hi</p>\n<!-- <script>throw 1</script> -->\n" +
				"<script>let a</script>\n",
			nil,
		},
		{
			"svelte usage comment names a script, then a commented-out script", ".svelte",
			"<!-- Usage: <script> import Modal from \"./Modal.svelte\" -->\n" +
				"<p>Do not forget the modal</p>\n<!--\n<script>\n" +
				"  throw new Error(\"disabled\")\n</script>\n-->\n<script>let a = 1</script>\n",
			nil,
		},
	}
	runScanCases(t, testCases)
}
