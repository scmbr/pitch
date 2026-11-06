param([Parameter(Mandatory=$true)][string]$Directory)
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.IO.Compression
$pptx = Join-Path $Directory 'talk.pptx'
$stream = [IO.File]::Open($pptx, [IO.FileMode]::CreateNew)
$zip = [IO.Compression.ZipArchive]::new($stream, [IO.Compression.ZipArchiveMode]::Create)
try {
    $entry = $zip.CreateEntry('ppt/slides/slide1.xml')
    $writer = [IO.StreamWriter]::new($entry.Open())
    try { $writer.Write('<slide>offline fixture</slide>') } finally { $writer.Dispose() }
} finally { $zip.Dispose(); $stream.Dispose() }

$first = 'BT /F1 36 Tf 60 220 Td (Opening slide) Tj ET'
$second = 'BT /F1 36 Tf 60 220 Td (Closing slide) Tj ET'
$objects = @(
    '<< /Type /Catalog /Pages 2 0 R >>',
    '<< /Type /Pages /Kids [3 0 R 5 0 R] /Count 2 >>',
    '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 640 360] /Resources << /Font << /F1 7 0 R >> >> /Contents 4 0 R >>',
    ("<< /Length {0} >>`nstream`n{1}`nendstream" -f $first.Length, $first),
    '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 640 360] /Resources << /Font << /F1 7 0 R >> >> /Contents 6 0 R >>',
    ("<< /Length {0} >>`nstream`n{1}`nendstream" -f $second.Length, $second),
    '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>'
)
$pdf = [Text.StringBuilder]::new("%PDF-1.4`n")
$offsets = [Collections.Generic.List[int]]::new()
for ($i = 0; $i -lt $objects.Count; $i++) {
    $offsets.Add($pdf.Length)
    $null = $pdf.AppendFormat("{0} 0 obj`n{1}`nendobj`n", $i+1, $objects[$i])
}
$xref = $pdf.Length
$null = $pdf.AppendFormat("xref`n0 {0}`n0000000000 65535 f `n", $objects.Count+1)
foreach ($offset in $offsets) { $null = $pdf.Append($offset.ToString('0000000000') + " 00000 n `n") }
$null = $pdf.AppendFormat("trailer`n<< /Size {0} /Root 1 0 R >>`nstartxref`n{1}`n%%EOF`n", $objects.Count+1, $xref)
[IO.File]::WriteAllText((Join-Path $Directory 'talk.pdf'), $pdf.ToString(), [Text.ASCIIEncoding]::new())
