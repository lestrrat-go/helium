<?xml version="1.0"?><?spec xslt#character-maps?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test xsl:output referencing two character maps. Use forwards references -->

  <xsl:output method="xml" use-character-maps="map03 map02"/>
  
  <xsl:character-map name="map02">
    <xsl:output-character character="c" string="[C]"/>
  </xsl:character-map>
  
  <xsl:character-map name="map03">
    <xsl:output-character character="y" string="[Y]"/>
    <xsl:output-character character="c" string="{C}"/>
    <xsl:output-character character="y" string="{Y}"/>    
  </xsl:character-map>

  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>

</xsl:stylesheet>