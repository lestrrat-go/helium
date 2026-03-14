<?xml version="1.0"?><?spec xslt#character-maps?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test multiple xsl:outputs referencing character maps.  -->

  <xsl:output method="xml" use-character-maps="map02"/>
  
  <xsl:character-map name="map02">
    <xsl:output-character character="c" string="[C]"/>
    <xsl:output-character character="a" string="[A]"/>    
  </xsl:character-map>
  
  <xsl:character-map name="map03">
    <xsl:output-character character="c" string="{C}"/>
    <xsl:output-character character="y" string="{Y}"/>    
  </xsl:character-map>
  
  <xsl:output method="xml" use-character-maps="map03"/>  

  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>

</xsl:stylesheet>