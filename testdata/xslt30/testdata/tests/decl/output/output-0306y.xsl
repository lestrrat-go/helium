<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
 
  <xsl:import href="output-0306z.xsl" />
  
  <xsl:character-map name="format1">
    <xsl:output-character character="&#036;" string="A" />
    <xsl:output-character character="&#037;" string="&#041;" />
  </xsl:character-map>
  
  <xsl:character-map name="format2">
    <xsl:output-character character="&#035;" string="&amp;" />
  </xsl:character-map>

</xsl:stylesheet>