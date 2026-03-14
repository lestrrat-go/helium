<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- CaseName: numbering17 -->
  <!-- Author: David Marston -->
  <!-- Purpose: Test of proper formation of Roman numerals. -->
  <!-- SpecCitation: Rec="XSLT" VersionDrop="1.1" --><!-- Formatting of zero will become an error -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/p[1]/text()[3]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/p[2]/text()[5]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/ulist[1]/item[5]/p[1]/text()[1]" -->
  <!-- Scenario: operation="standard-XML" -->

<!-- Elaboration: The value= formula below generates groups of 12 numbers out of every 50.
     Look particularly at subtractive numbers: 4, 9, 19, 40, 49, 99, 490, 499, 900, 990, 999, etc.
     It generates a zero, which may vary in its representation. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="chapter">
  <xsl:for-each select="note">
    <xsl:number format="(I) " value="(floor(position() div 12)*50)+(position() mod 12)-1" />
    <xsl:value-of select="."/><xsl:text>&#10;</xsl:text>
  </xsl:for-each>
</xsl:template>

</xsl:stylesheet>
