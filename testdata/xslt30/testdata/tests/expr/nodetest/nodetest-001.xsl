<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Test for text() node test. Should say "test" -->
  
  <xsl:template match="/doc">
    <out>
      <xsl:value-of select="./text()"/>
    </out>
  </xsl:template>

  <xsl:template match="text()">
    <xsl:value-of select="."/>
  </xsl:template>

</xsl:stylesheet>
