<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test xf:tokenize() function applied to an empty sequence. -->
  
  <xsl:template match="/">
    <out>
        <xsl:variable name="t" select="tokenize((), '\s+')"/>
        <ans count="{count($t)}" length="{string-length($t)}"/>
    </out>
  </xsl:template>

</xsl:stylesheet>