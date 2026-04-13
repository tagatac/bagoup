#!/usr/bin/env bash
set -e -o pipefail

OS=$(uname -s)
EXAMPLE_EXPORTS_DIR="example-exports/$OS"
TEST_EXPORTS_DIR="test-exports"
EXAMPLE_EXPORT_FILE='Novak Djokovic/iMessage;-;+3815555555555'
PDFINFO_IGNORE_PATTERN='Creator|CreationDate|File size|Producer'
MAGICK_MAJOR=$(compare --version | awk 'NR==1 { split($3, v, "."); print v[1] }')

TMPDIR_SCRIPT=$(mktemp -d)
trap 'rm -rf "$TMPDIR_SCRIPT"' EXIT

compare_pdf() {
    local file="$1"
    local expected="$EXAMPLE_EXPORTS_DIR/$file"
    local actual="$TEST_EXPORTS_DIR/$file"

    echo "==> Comparing PDF metadata: $file"
    pdfinfo "$expected" | grep -Ev "$PDFINFO_IGNORE_PATTERN" > "$TMPDIR_SCRIPT/pdfinfo-expected"
    pdfinfo "$actual" | grep -Ev "$PDFINFO_IGNORE_PATTERN" > "$TMPDIR_SCRIPT/pdfinfo-actual"
    diff "$TMPDIR_SCRIPT/pdfinfo-expected" "$TMPDIR_SCRIPT/pdfinfo-actual"

    echo "==> Comparing PDF visually: $file"
    # magick compare exits 1 even for valid comparisons (any pixel difference), so pipefail is
    # disabled for this pipeline. The awk explicitly fails if no SSIM output is produced (e.g. gs
    # not installed).
    (
        set +o pipefail
        compare -verbose -metric SSIM -density 300 -background white -alpha remove \
            "$expected" "$actual" null: 2>&1 \
            | tee /dev/stderr \
            | grep -i "all" \
            | awk -v major="$MAGICK_MAJOR" \
                'BEGIN { found=0 } { found=1; val=$2 } END { if (!found) exit 1; exit (major+0 >= 7 ? val <= 0.001 : val >= 0.999) ? 0 : 1 }'
    )
}

echo "==> Generating test exports"
rm -rf "$TEST_EXPORTS_DIR"
mkdir -p "$TEST_EXPORTS_DIR"
(cd example-exports && go run examplegen.go "../$TEST_EXPORTS_DIR")

echo "==> Comparing text export"
diff "$EXAMPLE_EXPORTS_DIR/messages-export/$EXAMPLE_EXPORT_FILE.txt" \
     "$TEST_EXPORTS_DIR/messages-export/$EXAMPLE_EXPORT_FILE.txt"

compare_pdf "messages-export-pdf/$EXAMPLE_EXPORT_FILE.pdf"
compare_pdf "messages-export-wkhtmltopdf/$EXAMPLE_EXPORT_FILE.pdf"

echo "==> All exports match"
