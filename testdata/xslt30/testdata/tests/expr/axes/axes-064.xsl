<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Test for generate-id() (on each tree model) 
       The test is written to produce reproducible results -->
  <xsl:template match="/">
    <out>
      <xsl:value-of select="generate-id(/*) = generate-id(/)"/>
  	</out>
  </xsl:template>
 
</xsl:stylesheet>
