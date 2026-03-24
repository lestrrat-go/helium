<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test xf:tokenize() function. -->
  <?spec fo#func-tokenize?>
  
  <xsl:template match="/">
    <out>
    <xsl:variable name="s" select="'12, 14, 3, 17, 24,,5 '"/>
    <xsl:text/>[<xsl:value-of select="tokenize($s, ',\s*')" separator="|"/>]<xsl:text/>
    </out>
  </xsl:template>

</xsl:stylesheet>