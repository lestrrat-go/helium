<?xml version="1.0"?><?spec xslt#character-maps?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test circular reference from xsl:character-map to another character map.  -->
  <?error XTSE1600?>

  <xsl:output method="xml" use-character-maps="map01"/>
  
  <xsl:character-map name="map02" use-character-maps="map01">
    <xsl:output-character character="c" string="[C]"/>
    <xsl:output-character character="a" string="[A]"/>    
  </xsl:character-map>
  
  <xsl:character-map name="map03" use-character-maps="map02">
    <xsl:output-character character="c" string="{C}"/>
    <xsl:output-character character="y" string="{Y}"/>    
  </xsl:character-map>
  
  <xsl:character-map name="map01" use-character-maps="map03"/>  

  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>

</xsl:stylesheet>