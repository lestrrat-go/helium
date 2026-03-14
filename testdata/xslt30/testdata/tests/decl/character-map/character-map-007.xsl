<?xml version="1.0"?><?spec xslt#character-maps?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test use of high Unicode characters in character maps.  -->

  <xsl:output method="xml" use-character-maps="map02" encoding="iso-8859-1"/>
  
  <xsl:character-map name="map02">
    <xsl:output-character character="&#x50000;" string="#####"/>
  </xsl:character-map>


  <xsl:template match="/">
    <v><xsl:value-of select="'[[[&#x50000;]]]'"/></v>
  </xsl:template>

</xsl:stylesheet>