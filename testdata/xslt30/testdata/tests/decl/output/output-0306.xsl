<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
 
  <xsl:import href="output-0306y.xsl" />
  
  <xsl:output use-character-maps="format1" method="xml" indent="no" encoding="UTF-8"/>
  <xsl:output use-character-maps="format2" method="xml" indent="no" encoding="UTF-8"/>
  
  <xsl:character-map name="format1">
    <xsl:output-character character="&#036;" string="B" />
    <xsl:output-character character="&#037;" string="&#040;" />
    <xsl:output-character character="&#037;" string="&#042;" />
  </xsl:character-map>
  
  <xsl:template match="/doc">
    <out>
      <xsl:for-each select="*">
        <p><xsl:value-of select="."/></p><xsl:text>&#10;</xsl:text>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>