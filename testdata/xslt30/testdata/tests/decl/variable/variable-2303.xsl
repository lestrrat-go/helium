<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    version="2.0">

 <?spec xslt#global-variables?>
  <!-- Purpose: Set top-level variable with a forward reference to see what happens. -->
  

  <xsl:variable name="tata" select="$toto"/>

  <xsl:param name="toto" select="'titi'"/>

  <xsl:template match="/">
    <out>
      <xsl:value-of select="$tata"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
