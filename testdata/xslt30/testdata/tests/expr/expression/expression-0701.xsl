<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Test optimisation of count()=0 -->
  
  <xsl:template match="doc">
    <out>
      <a><xsl:value-of select="count(//item)=0"/> should be false</a>
      <a><xsl:value-of select="count(//item) &gt; 0"/> should be true</a>
      <a><xsl:value-of select="count(//argh)=0"/> should be true</a>
      <a><xsl:value-of select="count(//argh) &gt; 0"/> should be false</a>
    </out>
  </xsl:template>


</xsl:stylesheet>
