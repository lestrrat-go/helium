<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#variable-values?>
  <!-- Purpose: Use copy to set a variable to an RTF.
       REMINDER: we won't get the whole sub-tree, just the 'doc' element node. -->
  

<xsl:template match="doc">
  <xsl:variable name="z">
    <xsl:copy>
      <xsl:attribute name="status">replacement</xsl:attribute>
    </xsl:copy>
  </xsl:variable>
  <out>
    <xsl:text>$z contains </xsl:text>
    <xsl:copy-of select="$z"/>
  </out>
</xsl:template>

</xsl:stylesheet>
