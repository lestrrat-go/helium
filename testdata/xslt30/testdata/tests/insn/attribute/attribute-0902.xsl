<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test for selecting attributes with namespace. -->
  
  <xsl:template match="/docs">
    <xsl:variable name="union">
      <xsl:for-each select="//@*[namespace-uri() eq 'http://www.ped.com'] | //@*[local-name() eq 'att1']">
        <xsl:value-of select="."/>
      </xsl:for-each>
    </xsl:variable>
    <xsl:variable name="or">
      <xsl:for-each select="//@*[namespace-uri() eq 'http://www.ped.com' or local-name() eq 'att1']">
        <xsl:value-of select="."/>
      </xsl:for-each>
    </xsl:variable>
    
    <out><xsl:value-of select="$union eq $or"/></out>
    
  </xsl:template>
  
</xsl:stylesheet>
