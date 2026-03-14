<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
   
  <xsl:import href="next-match-019b.xsl"/>
  
  <xsl:template match="tag">
    <xsl:param name="p1" select="'default'"/>
    <imp1-t><xsl:value-of select="$p1"/></imp1-t>
  </xsl:template>
  
  <xsl:template match="bag">
    <xsl:param name="p1" select="'default'"/>
    <imp-b><xsl:value-of select="$p1"/></imp-b>
  </xsl:template>
  
</xsl:stylesheet>