<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" xmlns:f="http://f.com/">

<?spec xslt#patterns?>
<?error XPST0017?>
  <!-- FileName: match9055-err -->
    <!-- Purpose: Test use of an disallowed function at the start of a pattern -->

<xsl:template match="copy-of($x)//a" priority="5">
 <out>WRONG!</out>
</xsl:template>

<xsl:template match="/">
<out>OK!</out>
</xsl:template>

<xsl:variable name="x"><a/></xsl:variable>

</xsl:stylesheet>
