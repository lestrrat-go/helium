<?xml version="1.0"?><?spec xslt#character-maps?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test a composite character map containing duplicate characters -->
  
  <xsl:character-map name="map01" use-character-maps="map02 map03"/>

  <xsl:character-map name="map02">
    <xsl:output-character character="c" string="[C]"/>
  </xsl:character-map>
  
  <xsl:character-map name="map03">
    <xsl:output-character character="y" string="[Y]"/>
    <xsl:output-character character="c" string="{C}"/>
    <xsl:output-character character="y" string="{Y}"/>    
  </xsl:character-map>
  
  <xsl:output method="xml" use-character-maps="map01"/>

  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>

</xsl:stylesheet>