<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    version="2.0">

  <!-- Purpose: Test passing value from top-level param to top-level variable via value-of. -->
  
  <xsl:param name="toto" select="'titi'"/>

  <xsl:variable name="tata">
    <xsl:call-template name="set-tata"/>
  </xsl:variable>

  <xsl:template match="/">
    <out>
      <xsl:value-of select="$tata"/>
    </out>
  </xsl:template>

  <xsl:template name="set-tata">
    <xsl:value-of select="$toto"/>
  </xsl:template>

</xsl:stylesheet>
