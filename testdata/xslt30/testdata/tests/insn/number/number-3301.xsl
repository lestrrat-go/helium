<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- CaseName: numbering45 -->
  <!-- Author: David Marston -->
  <!-- Purpose: Test more than one xsl:number counter active at the same time. -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[1]/item[2]/p[1]/text()[1]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/p[2]/text()[5]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/ulist[1]/item[1]/p[1]/text()[1]" -->
  <!-- Scenario: operation="standard-XML" -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="chapter">
  <xsl:for-each select="note">
    <xsl:number format="A-" count="note" />
    <xsl:if test="position() mod 2 = 0">
      <xsl:number format="(1) " count="note[position() mod 2 = 0]" />
    </xsl:if>
    <xsl:if test="position() mod 2 = 1">
      <xsl:number format="(1) " count="note[position() mod 2 = 1]" />
    </xsl:if>
    <xsl:value-of select="."/><xsl:text>&#10;</xsl:text>
  </xsl:for-each>
</xsl:template>

</xsl:stylesheet>
