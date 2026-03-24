<?xml version="1.0"?><?spec xslt#character-maps?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test generation of markup using character maps.  -->

  <xsl:output method="xml" use-character-maps="map02"/>
  
  <xsl:character-map name="map02">
    <xsl:output-character character="v" string="&lt;jjj&gt;"/>
    <xsl:output-character character="z" string="&lt;/jjj&gt;"/>    
  </xsl:character-map>


  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>

</xsl:stylesheet>