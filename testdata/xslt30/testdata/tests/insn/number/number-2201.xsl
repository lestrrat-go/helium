<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- CaseName: numbering05 -->
  <!-- Author: Paul Dick, based on example in spec -->
  <!-- Purpose: Test of level (any) and nested from/count. -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[2]/item[3]/p[1]/text()[6]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[1]/item[2]/p[1]/text()[1]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[1]/item[3]/p[1]/text()[1]" -->
  <!-- Scenario: operation="standard-XML" -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="H4">
  <xsl:text>&#010;</xsl:text>
  <xsl:number level="any" from="H1" count="H2"/>
  <xsl:text>.</xsl:text>
  <xsl:number level="any" from="H2" count="H3"/>
  <xsl:text>.</xsl:text>
  <xsl:number level="any" from="H3" count="H4"/>
  <xsl:text></xsl:text>
  <xsl:text> &#010;</xsl:text>
  <xsl:apply-templates/>
</xsl:template>

<xsl:template match="text()"/>

</xsl:stylesheet>
