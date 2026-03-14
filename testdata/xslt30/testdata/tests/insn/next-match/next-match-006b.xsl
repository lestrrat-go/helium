<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <xsl:template match="node()">
    <ba>
      <xsl:next-match />
    </ba>
  </xsl:template>
  
  <xsl:template match="doc">
    <bb>
      <xsl:next-match />
    </bb>
  </xsl:template>
  
  <xsl:template match="*[foo]">
    <bc>
      <xsl:next-match />
    </bc>
  </xsl:template>
  
</xsl:stylesheet>
