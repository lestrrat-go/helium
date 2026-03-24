<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    version="2.0">

 <?spec xslt#global-variables?>
  <!-- Purpose: Test passing value from top-level param to top-level variable via value-of. -->
  

  <xsl:param name="toto" select="'titi'"/>

  <xsl:variable name="tata" select="$toto"/>

  <xsl:template match="/">
    <out>
      <xsl:value-of select="$tata"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
