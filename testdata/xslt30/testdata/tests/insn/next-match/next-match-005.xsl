<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <!-- Purpose: Test next-match from an imported stylesheet using default priority and Wildcard Test -->
  
  <xsl:import href="next-match-005a.xsl" />
  
  <xsl:template match="*">
    <a>
      <xsl:next-match />
    </a>
  </xsl:template>
  
</xsl:stylesheet>