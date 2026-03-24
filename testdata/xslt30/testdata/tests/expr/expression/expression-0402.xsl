<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- DOCUMENT: http://www.w3.org/TR/xpath -->
  <!-- PURPOSE: Test optimisation of false() in a filter expression -->
  <?spec xpath#id-predicates?>
<xsl:template match="doc">
    <out>
      <xsl:variable name="paras" select="//para"/>
      <xsl:for-each select="$paras[2=3][@cat='a']">
        <xsl:copy-of select="."/>;
      </xsl:for-each>  
    </out>
</xsl:template>
 
</xsl:stylesheet>
