<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<?spec xslt#patterns?>

  <!-- FileName: MATCH87 -->
    <!-- Purpose: Test the "union" keyword in a pattern (permitted in XSLT 2.1) -->

<xsl:template match="and union or" priority="5">
 <yes>MATCH</yes>
</xsl:template>

<xsl:variable name="x"><and/></xsl:variable>

<xsl:template match="/">
<out><xsl:apply-templates select="$x/*"/></out>
</xsl:template>

</xsl:stylesheet>
