<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <!-- Test with next-match where both the dominant and secondary template have 
    @match=node() and identical priorities but their import precedence is different.-->
  
  <xsl:import href="next-match-004a.xsl" />
  
  <xsl:template match="node()">
    <a>
      <xsl:next-match />
    </a>
  </xsl:template>
   
</xsl:stylesheet>
