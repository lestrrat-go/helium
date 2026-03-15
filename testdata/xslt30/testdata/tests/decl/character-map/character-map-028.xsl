<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test that characters produced by a character map are immune to Unicode normalization -->

  <xsl:character-map name="map01">
    <xsl:output-character character="c" string="#ç#"/>
  </xsl:character-map>
  
  <xsl:output method="xml" use-character-maps="map01" normalization-form="NFD"/>

  <xsl:template match="/">
    <out att="abcçde">abcçde</out>
  </xsl:template>

</xsl:stylesheet>