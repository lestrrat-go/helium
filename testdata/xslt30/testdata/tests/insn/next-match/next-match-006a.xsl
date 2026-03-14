<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <xsl:import href="next-match-006b.xsl" />
  
  <xsl:template match="node()">
    <aa>
      <xsl:next-match />
    </aa>
  </xsl:template>
  
  <xsl:template match="doc">
    <ab>
      <xsl:next-match />
    </ab>
  </xsl:template>
  
  <xsl:template match="*[foo]">
    <ac>
      <xsl:next-match />
    </ac>
  </xsl:template>
  
</xsl:stylesheet>
