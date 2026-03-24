<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Test optimisation of true() in a filter expression -->

  <xsl:template match="doc">
    <out>
      <xsl:variable name="paras" select="//para"/>
      <xsl:for-each select="$paras[2=2][@cat='a']">
        <xsl:copy-of select="."/>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
