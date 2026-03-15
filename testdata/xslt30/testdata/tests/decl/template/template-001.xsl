<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#applying-templates?>
  <!-- Purpose: Test apply-templates for a comment with a mode
       and moded matching template. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="comment()" mode="mode1"/>
  </out>
</xsl:template>

<xsl:template match="comment()" mode="mode1">
  <xsl:value-of select="."/>
</xsl:template>

<xsl:template match="comment()">
  This test failed to execute properly.
</xsl:template>

</xsl:stylesheet>
