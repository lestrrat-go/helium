<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#global-variables?>
  <!-- Purpose: Try to catch lazy-evaluation scheme picking up local instead of global. -->
  

  <xsl:variable name="tata" select="$toto"/>

  <xsl:param name="toto" select="'titi'"/>

  <xsl:template match="/">
    <xsl:param name="toto" select="'templ'"/>
    <out>
      <xsl:value-of select="$toto"/><xsl:text>, </xsl:text>
      <xsl:value-of select="$tata"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
