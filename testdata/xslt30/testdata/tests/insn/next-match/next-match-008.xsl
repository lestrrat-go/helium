<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <!-- Purpose: Test of apply-imports for two templates, one with pattern "element()" residing 
    in primary stylesheet and the other one with pattern "element(*, xs:untyped)" 
    residing in an imported stylesheet.-->
  
  <xsl:import href="next-match-008a.xsl"/>
  
  <xsl:template match="element()">
    <a>
      <xsl:apply-imports />
    </a>
  </xsl:template>
  
</xsl:stylesheet>