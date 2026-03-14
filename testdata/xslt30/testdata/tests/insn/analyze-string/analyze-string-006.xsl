<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#analyze-string?>
  <!-- PURPOSE: test xsl:analyze-string. -->
  
  
  <xsl:template match="/">
    <out>
      <xsl:variable name="v">
        <xsl:analyze-string select="doc" regex="e">
         <xsl:matching-substring>[<xsl:value-of select="."/>]</xsl:matching-substring>
         <xsl:non-matching-substring><xsl:value-of select="."/></xsl:non-matching-substring>
        </xsl:analyze-string>
      </xsl:variable>
      <xsl:value-of select="translate($v, '&#xa;', '|')"/>
    </out>
  </xsl:template>

</xsl:stylesheet>