<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Test optimisation of string-length()=0 -->


  <xsl:template match="doc">
    <out>
      <a><xsl:value-of select="string-length((//item)[1])=0"/> should be true</a>
      <a><xsl:value-of select="string-length((//item)[1]) &gt; 0"/> should be false</a>
      <a><xsl:value-of select="string-length((//argh)[1])=0"/> should be true</a>
      <a><xsl:value-of select="string-length((//argh)[1]) &gt; 0"/> should be false</a>
    </out>
  </xsl:template>


</xsl:stylesheet>
