<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- CaseName: numbering02 -->
  <!-- Author: Paul Dick, based on example in spec -->
  <!-- Purpose: Test of simple numbering, no attributes specified. -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="DocFrag" place="id(number)" -->
  <!-- Scenario: operation="standard-XML" -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="ol/item"/>
  </out>
</xsl:template>

<xsl:template match="ol/item">
  <xsl:number/><xsl:text>. </xsl:text><xsl:value-of select="."/><xsl:text>
</xsl:text>
</xsl:template>
 
</xsl:stylesheet>
