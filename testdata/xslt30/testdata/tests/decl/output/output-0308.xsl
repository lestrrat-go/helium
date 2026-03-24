<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <xsl:import href="output-0308a.xsl"/>

  <xsl:output method="text" encoding="UTF-8"/>
  
  <xsl:output name="temp1" method="xml" indent="no" byte-order-mark="yes" include-content-type="yes"
    undeclare-prefixes="no" use-character-maps="format1" encoding="UTF-8"/>

  <xsl:character-map name="format1">
    <xsl:output-character character="&#035;" string="B"/>
  </xsl:character-map>

  <xsl:character-map name="format2">
    <xsl:output-character character="&#035;" string="A"/>
  </xsl:character-map>


  <xsl:template match="/doc">
    <xsl:result-document format="temp1">
      <out>
        <xsl:for-each select="*">
          <p>
            <xsl:value-of select="."/>
          </p>
          <xsl:text>&#10;</xsl:text>
        </xsl:for-each>
        <br/>
      </out>
    </xsl:result-document>
  </xsl:template>

</xsl:stylesheet>
