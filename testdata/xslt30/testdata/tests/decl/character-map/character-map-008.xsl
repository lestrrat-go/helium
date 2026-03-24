<?xml version="1.0"?><?spec xslt#character-maps?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test character maps in a namespace  -->

  <xsl:output method="xml" use-character-maps="z:map02" xmlns:z="http://charmap/"/>
  
  <xsl:character-map name="y:map02" xmlns:y="http://charmap/" use-character-maps="y:map03">
    <xsl:output-character character="a" string="[A]"/>
  </xsl:character-map>
  
  <xsl:character-map name="x:map03" xmlns:x="http://charmap/" >
    <xsl:output-character character="b" string="[B]"/>
  </xsl:character-map>  


  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>

</xsl:stylesheet>